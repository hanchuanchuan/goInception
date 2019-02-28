// Copyright 2013 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package session

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	// "github.com/hanchuanchuan/goInception/ast"
	// "github.com/hanchuanchuan/goInception/server"
	"github.com/hanchuanchuan/goInception/util"
	"github.com/hanchuanchuan/goInception/util/auth"
	// "github.com/pingcap/errors"
	log "github.com/sirupsen/logrus"
)

type ChanOscData struct {
	out string
	p   *util.OscProcessInfo
}

// Copying `test`.`t1`:  99% 00:00 remain
// 匹配osc执行进度
var regOscPercent *regexp.Regexp = regexp.MustCompile(`^Copying .*? (\d+)% (\d+:\d+) remain`)

func (s *session) checkAlterUseOsc(t *TableInfo) {
	if s.Osc.OscOn && (s.Osc.OscMinTableSize == 0 || t.TableSize >= s.Osc.OscMinTableSize) {
		s.myRecord.useOsc = true
	} else {
		s.myRecord.useOsc = false
	}
}

func (s *session) mysqlComputeSqlSha1(r *Record) {
	if !r.useOsc || r.Sqlsha1 != "" {
		return
	}

	buf := bytes.NewBufferString(r.DBName)

	buf.WriteString(s.opt.password)
	buf.WriteString(s.opt.host)
	buf.WriteString(s.opt.user)
	buf.WriteString(strconv.Itoa(s.opt.port))
	buf.WriteString(strconv.Itoa(r.SeqNo))
	buf.WriteString(r.Sql)

	r.Sqlsha1 = auth.EncodePassword(buf.String())
}

func (s *session) mysqlExecuteAlterTableOsc(r *Record) {

	err := os.Setenv("PATH", fmt.Sprintf("%s%s%s",
		s.Osc.OscBinDir, string(os.PathListSeparator), os.Getenv("PATH")))
	if err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
		return
	}

	if _, err := exec.LookPath("pt-online-schema-change"); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
		return
	}

	buf := bytes.NewBufferString("pt-online-schema-change")

	buf.WriteString(" --alter \"")
	buf.WriteString(s.getAlterTablePostPart(r.Sql))

	if s.hasError() {
		return
	}

	buf.WriteString("\" ")
	if s.Osc.OscPrintSql {
		buf.WriteString(" --print ")
	}
	buf.WriteString(" --charset=utf8 ")
	buf.WriteString(" --chunk-time ")
	buf.WriteString(fmt.Sprintf("%g ", s.Osc.OscChunkTime))

	buf.WriteString(" --critical-load ")
	buf.WriteString("Threads_connected:")
	buf.WriteString(strconv.Itoa(s.Osc.OscCriticalConnected))
	buf.WriteString(",Threads_running:")
	buf.WriteString(strconv.Itoa(s.Osc.OscCriticalRunning))
	buf.WriteString(" ")

	buf.WriteString(" --max-load ")
	buf.WriteString("Threads_connected:")
	buf.WriteString(strconv.Itoa(s.Osc.OscMaxConnected))
	buf.WriteString(",Threads_running:")
	buf.WriteString(strconv.Itoa(s.Osc.OscMaxRunning))
	buf.WriteString(" ")

	buf.WriteString(" --recurse=1 ")

	buf.WriteString(" --check-interval ")
	buf.WriteString(fmt.Sprintf("%d ", s.Osc.OscCheckInterval))

	if !s.Osc.OscDropNewTable {
		buf.WriteString(" --no-drop-new-table ")
	}

	if !s.Osc.OscDropOldTable {
		buf.WriteString(" --no-drop-old-table ")
	}

	if !s.Osc.OscCheckReplicationFilters {
		buf.WriteString(" --no-check-replication-filters ")
	}

	if !s.Osc.OscCheckAlter {
		buf.WriteString(" --no-check-alter ")
	}

	buf.WriteString(" --alter-foreign-keys-method=")
	buf.WriteString(s.Osc.OscAlterForeignKeysMethod)

	if s.Osc.OscAlterForeignKeysMethod == "none" {
		buf.WriteString(" --force ")
	}

	buf.WriteString(" --execute ")
	buf.WriteString(" --statistics ")
	buf.WriteString(" --max-lag=")
	buf.WriteString(fmt.Sprintf("%d", s.Osc.OscMaxLag))

	buf.WriteString(" --no-version-check ")
	buf.WriteString(" --recursion-method=")
	buf.WriteString(s.Osc.OscRecursionMethod)

	buf.WriteString(" --progress ")
	buf.WriteString("percentage,1 ")

	buf.WriteString(" --user=")
	buf.WriteString(s.opt.user)
	buf.WriteString(" --password=")
	buf.WriteString(s.opt.password)
	buf.WriteString(" --host=")
	buf.WriteString(s.opt.host)
	buf.WriteString(" --port=")
	buf.WriteString(strconv.Itoa(s.opt.port))

	buf.WriteString(" D=")
	buf.WriteString(r.TableInfo.Schema)
	buf.WriteString(",t=")
	buf.WriteString(r.TableInfo.Name)

	str := buf.String()

	s.execCommand(r, "bash", []string{"-c", str})

}

func (s *session) execCommand(r *Record, commandName string, params []string) bool {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command(commandName, params...)

	//StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.AppendErrorMessage(err.Error())
		return false
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.AppendErrorMessage(err.Error())
		return false
	}

	// 保证关闭输出流
	defer stdout.Close()
	defer stderr.Close()

	// 运行命令
	if err := cmd.Start(); err != nil {
		s.AppendErrorMessage(err.Error())
		return false
	}

	p := &util.OscProcessInfo{
		Schema:     r.TableInfo.Schema,
		Table:      r.TableInfo.Name,
		Command:    r.Sql,
		Sqlsha1:    r.Sqlsha1,
		Percent:    0,
		RemainTime: "",
		Info:       "",
	}
	s.sessionManager.AddOscProcess(p)

	// 消息
	reader := bufio.NewReader(stdout)

	// 进度
	reader2 := bufio.NewReader(stderr)

	buf := bytes.NewBufferString("")

	//实时循环读取输出流中的一行内容
	f := func(reader *bufio.Reader) {
		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			buf.WriteString(line)
			buf.WriteString("\n")
			s.mysqlAnalyzeOscOutput(line, p)
			if p.Killed {
				if err := cmd.Process.Kill(); err != nil {
					s.AppendErrorMessage(err.Error())
				} else {
					s.AppendErrorMessage(fmt.Sprintf("Execute has been abort in percent: %d, remain time: %s",
						p.Percent, p.RemainTime))
				}
			}
		}
	}

	go f(reader)
	go f(reader2)

	//阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	err = cmd.Wait()
	if err != nil {
		s.AppendErrorMessage(err.Error())
	}
	if p.Percent < 100 || s.hasError() {
		r.StageStatus = StatusExecFail
	} else {
		r.StageStatus = StatusExecOK
		r.ExecComplete = true
	}

	if p.Percent < 100 || s.Osc.OscPrintNone {
		r.Buf.WriteString(buf.String())
		r.Buf.WriteString("\n")
	}

	// 执行完成或中止后清理osc进程信息
	pl := s.sessionManager.ShowOscProcessList()
	delete(pl, p.Sqlsha1)

	return true
}

func (s *session) mysqlAnalyzeOscOutput(out string, p *util.OscProcessInfo) {
	firsts := regOscPercent.FindStringSubmatch(out)
	// log.Info(p.Killed)
	if len(firsts) < 3 {
		if strings.HasPrefix(out, "Successfully altered") {

			p.Percent = 100
			p.RemainTime = ""
			p.Info = strings.TrimSpace(out)
		}
		return
	}

	pct, _ := strconv.Atoi(firsts[1])
	remain := firsts[2]
	p.Info = strings.TrimSpace(out)

	p.Percent = pct
	p.RemainTime = remain

}

func (s *session) getAlterTablePostPart(sql string) string {

	var buf []string
	for _, line := range strings.Split(sql, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-- ") || strings.HasPrefix(line, "/*") {
			continue
		}
		buf = append(buf, line)
	}
	sql = strings.Join(buf, "\n")
	index := strings.Index(strings.ToUpper(sql), "ALTER")
	if index == -1 {
		s.AppendErrorMessage("无效alter语句!")
		return sql
	}

	sql = string(sql[index:])
	parts := strings.SplitN(sql, " ", 4)
	if len(parts) != 4 {
		s.AppendErrorMessage("无效alter语句!")
		return sql
	}

	sql = parts[3]

	return sql
}

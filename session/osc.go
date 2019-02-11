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
	"fmt"
	// "strings"
	"bytes"
	"strconv"

	// "github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/util/auth"
	// "github.com/hanchuanchuan/goInception/mysql"
	// "github.com/pingcap/errors"
	// log "github.com/sirupsen/logrus"
)

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

	buf := bytes.NewBufferString(fmt.Sprintf("PATH=%s:$PATH && ", s.Osc.OscBinDir))

	buf.WriteString("pt-online-schema-change ")
	buf.WriteString("--alter ")
	buf.WriteString(r.Sql)
	buf.WriteString(" ")
	if s.Osc.OscPrintSql {
		buf.WriteString("--print ")
	}
	buf.WriteString("--charset=utf8 ")
	buf.WriteString("--chunk-time ")
	buf.WriteString(strconv.Itoa(s.Osc.OscChunkTime))

	buf.WriteString("--critical-load ")
	buf.WriteString("Threads_connected:")
	buf.WriteString(strconv.Itoa(s.Osc.OscCriticalConnected))
	buf.WriteString("Threads_running:")
	buf.WriteString(strconv.Itoa(s.Osc.OscCriticalRunning))

	buf.WriteString("--max-load ")
	buf.WriteString("Threads_connected:")
	buf.WriteString(strconv.Itoa(s.Osc.OscMaxConnected))
	buf.WriteString("Threads_running:")
	buf.WriteString(strconv.Itoa(s.Osc.OscMaxRunning))

	buf.WriteString("--recurse=1")

	buf.WriteString("--check-interval")

	buf.WriteString(s.opt.user)
	buf.WriteString(strconv.Itoa(s.opt.port))
	buf.WriteString(strconv.Itoa(r.SeqNo))
	buf.WriteString(r.Sql)

	r.Sqlsha1 = auth.EncodePassword(buf.String())
}

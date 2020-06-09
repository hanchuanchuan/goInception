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
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/parser"
	"github.com/hanchuanchuan/goInception/sessionctx/variable"
	"github.com/hanchuanchuan/goInception/util"
	"github.com/hanchuanchuan/goInception/util/sqlexec"
	"github.com/hanchuanchuan/goInception/util/timeutil"
	"github.com/jinzhu/gorm"
	"github.com/pingcap/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	// "vitess.io/vitess/go/vt/sqlparser"
)

func (s *session) makeNewResult() ([]Record, error) {
	// if s.opt != nil && s.opt.Print && s.printSets != nil {
	// 	return s.printSets.Rows(), nil
	// } else if s.opt != nil && s.opt.split && s.splitSets != nil {
	// 	s.addNewSplitNode()
	// 	// log.Infof("%#v", s.splitSets)
	// 	return s.splitSets.Rows(), nil
	// } else {

	// }
	records := make([]Record, len(s.recordSets.records))
	for i, r := range s.recordSets.records {
		// log.Info(r.SeqNo, "		", i)
		r.SeqNo = i

		r.cut()
		records[i] = *r
	}
	return records, nil
}

func NewInception() *session {
	se := &session{
		parser:              parser.New(),
		sessionVars:         variable.NewSessionVars(),
		lowerCaseTableNames: 1,
		isAPI:               true,
	}

	// cluster := mocktikv.NewCluster()
	// mocktikv.BootstrapWithSingleStore(cluster)
	// mvccStore := mocktikv.MustNewMVCCStore()
	// store, err := mockstore.NewMockTikvStore(
	// 	mockstore.WithCluster(cluster),
	// 	mockstore.WithMVCCStore(mvccStore),
	// )
	// se.store = store

	// session.SetSchemaLease(0)
	// session.SetStatsLease(0)

	se.sessionVars.GlobalVarsAccessor = se

	tz := timeutil.InferSystemTZ()
	// log.Errorf("tz: %v", tz)
	timeutil.SetSystemTZ(tz)

	return se
}

// init 初始化map
func (s *session) init() {
	s.dbName = ""
	s.haveBegin = false
	s.haveCommit = false
	s.threadID = 0
	s.isClusterNode = false

	s.tableCacheList = make(map[string]*TableInfo)
	s.dbCacheList = make(map[string]*DBInfo)

	s.backupDBCacheList = make(map[string]bool)
	s.backupTableCacheList = make(map[string]bool)

	s.inc = config.GetGlobalConfig().Inc
	s.osc = config.GetGlobalConfig().Osc
	s.ghost = config.GetGlobalConfig().Ghost

	s.inc.Lang = strings.Replace(strings.ToLower(s.inc.Lang), "-", "_", 1)

	s.sqlFingerprint = nil

	s.dbType = DBTypeMysql
	s.dbVersion = 0

	// 自定义审核级别,通过解析config.GetGlobalConfig().IncLevel生成
	s.parseIncLevel()
}

// clear 清理变量或map等信息
func (s *session) clear() {
	if s.db != nil {
		defer s.db.Close()
	}
	if s.ddlDB != nil {
		defer s.ddlDB.Close()
	}
	if s.backupdb != nil {
		defer s.backupdb.Close()
	}

	s.dbName = ""
	s.haveBegin = false
	s.haveCommit = false
	s.threadID = 0
	s.isClusterNode = false

	s.tableCacheList = nil
	s.dbCacheList = nil
	s.backupDBCacheList = nil
	s.backupTableCacheList = nil
	s.sqlFingerprint = nil

	s.incLevel = nil

	s.recordSets = nil
	s.printSets = nil
	s.splitSets = nil
}

func (s *session) Audit(ctx context.Context, sql string) ([]Record, error) {

	if s.opt == nil {
		return nil, errors.New("未配置数据源信息!")
	}

	s.init()
	defer s.clear()
	s.opt.Check = true
	err := s.audit(ctx, sql)
	if err != nil {
		log.Error(err)
	}
	return s.makeNewResult()
	// return s.recordSets.records, nil
	// return s.makeResult()
}

func (s *session) RunExecute(ctx context.Context, sql string) ([]Record, error) {

	if s.opt == nil {
		return nil, errors.New("未配置数据源信息!")
	}

	s.init()
	defer s.clear()

	s.opt.Check = false
	s.opt.Execute = true
	s.audit(ctx, sql)

	if s.hasErrorBefore() {
		return s.makeNewResult()
	}
	s.executeCommit(ctx)

	return s.makeNewResult()
	// return s.makeResult()
}

func (s *session) LoadOptions(opt SourceOptions) error {
	s.opt = &opt
	// return s.parseOptions()
	return nil
}

func (s *session) audit(ctx context.Context, sql string) (err error) {

	sqlList := strings.Split(sql, "\n")

	// tidb执行的SQL关闭general日志
	logging := s.inc.GeneralLog

	defer func() {
		if s.sessionVars.StmtCtx.AffectedRows() == 0 {
			if s.opt != nil && s.opt.Print {
				s.sessionVars.StmtCtx.AddAffectedRows(uint64(s.printSets.rc.count))
			} else if s.opt != nil && s.opt.split {
				s.sessionVars.StmtCtx.AddAffectedRows(uint64(s.splitSets.rc.count))
			} else {
				s.sessionVars.StmtCtx.AddAffectedRows(uint64(len(s.recordSets.records)))
			}
		}

		if logging {
			logQuery(sql, s.sessionVars)
		}
	}()

	// s.PrepareTxnCtx(ctx)
	connID := s.sessionVars.ConnectionID
	// connID := 1
	// err = s.loadCommonGlobalVariablesIfNeeded()
	// if err != nil {
	// 	return nil, errors.Trace(err)
	// }

	charsetInfo, collation := s.sessionVars.GetCharsetInfo()

	lineCount := len(sqlList) - 1
	// batchSize := 1

	tmp := s.processInfo.Load()
	if tmp != nil {
		pi := tmp.(util.ProcessInfo)
		pi.OperState = "CHECKING"
		pi.Percent = 0
		s.processInfo.Store(pi)
	}

	s.stage = StageCheck

	err = s.checkOptions()
	if err != nil {
		return err
	}

	if s.opt.Print {
		s.printSets = NewPrintSets()
	} else if s.opt.split {
		s.splitSets = NewSplitSets()
	} else {
		s.recordSets = NewRecordSets()
	}

	// sql指纹设置取并集
	if s.opt.fingerprint {
		s.inc.EnableFingerprint = true
	}

	if s.inc.EnableFingerprint {
		s.sqlFingerprint = make(map[string]*Record, 64)
	}

	var buf []string

	quotaIsDouble := true
	for i, sql_line := range sqlList {

		// 100行解析一次
		// 如果以分号结尾,或者是最后一行,就做解析
		// strings.HasSuffix(sql_line, ";")
		// && batchSize >= 100)

		if strings.Count(sql_line, "'")%2 == 1 {
			quotaIsDouble = !quotaIsDouble
		}

		if ((strings.HasSuffix(sql_line, ";") || strings.HasSuffix(sql_line, ";\r")) &&
			quotaIsDouble) || i == lineCount {
			// batchSize = 1
			buf = append(buf, sql_line)
			s1 := strings.Join(buf, "\n")

			s1 = strings.TrimRight(s1, ";")

			stmtNodes, err := s.ParseSQL(ctx, s1, charsetInfo, collation)

			if err == nil && len(stmtNodes) == 0 {
				tmpSQL := strings.TrimSpace(s1)
				// 未成功解析时，添加异常判断
				if tmpSQL != "" &&
					!strings.HasPrefix(tmpSQL, "#") &&
					!strings.HasPrefix(tmpSQL, "--") &&
					!strings.HasPrefix(tmpSQL, "/*") {
					err = errors.New("解析失败! 可能是解析器bug,请联系作者.")
				}
			}

			if err != nil {
				log.Errorf("con:%d 解析失败! %s", connID, err)
				log.Error(s1)
				if s.opt != nil && s.opt.Print {
					s.printSets.Append(2, strings.TrimSpace(s1), "", err.Error())
				} else if s.opt != nil && s.opt.split {
					s.addNewSplitNode()
					s.splitSets.Append(strings.TrimSpace(s1), err.Error())
				} else {
					s.recordSets.Append(&Record{
						Sql:          strings.TrimSpace(s1),
						ErrLevel:     2,
						ErrorMessage: err.Error(),
					})
				}
				return err
			}

			for i, stmtNode := range stmtNodes {
				//  是ASCII码160的特殊空格
				currentSql := strings.Trim(stmtNode.Text(), " ;\t\n\v\f\r ")

				s.myRecord = &Record{
					Sql:   currentSql,
					Buf:   new(bytes.Buffer),
					Type:  stmtNode,
					Stage: StageCheck,
				}

				s.SetMyProcessInfo(currentSql, time.Now(), float64(i)/float64(lineCount+1))

				var result []sqlexec.RecordSet
				var err error
				if s.opt != nil && s.opt.Print {
					result, err = s.printCommand(ctx, stmtNode, currentSql)
				} else if s.opt != nil && s.opt.split {
					result, err = s.splitCommand(ctx, stmtNode, currentSql)
				} else {
					result, err = s.processCommand(ctx, stmtNode, currentSql)
				}
				if err != nil {
					return err
				}
				if result != nil {
					return nil
				}

				// 进程Killed
				if err := checkClose(ctx); err != nil {
					log.Warn("Killed: ", err)
					s.appendErrorMessage("Operation has been killed!")
					if s.opt != nil && s.opt.Print {
						s.printSets.Append(2, "", "", strings.TrimSpace(s.myRecord.Buf.String()))
					} else if s.opt != nil && s.opt.split {
						s.addNewSplitNode()
						s.splitSets.Append("", strings.TrimSpace(s.myRecord.Buf.String()))
					} else {
						s.recordSets.Append(s.myRecord)
					}
					return err
				}

				if s.opt != nil && s.opt.Print {
					// s.printSets.Append(2, "", "", strings.TrimSpace(s.myRecord.Buf.String()))
				} else {
					// 远程操作时隐藏本地的set命令
					if _, ok := stmtNode.(*ast.InceptionSetStmt); ok && s.myRecord.ErrLevel == 0 {
						log.Info(currentSql)
					} else {
						s.recordSets.Append(s.myRecord)
					}
				}
			}

			buf = nil

		} else if i < lineCount {
			buf = append(buf, sql_line)
			// batchSize++
		}
	}

	return nil

}

// checkOptions 校验配置信息
func (s *session) checkOptions() error {

	if s.opt == nil {
		return errors.New("未配置数据源信息!")
	}

	if s.opt.split || s.opt.Check || s.opt.Print {
		s.opt.Execute = false
		s.opt.Backup = false

		// 审核阶段自动忽略警告,以免审核过早中止
		s.opt.IgnoreWarnings = true
	}

	if s.opt.sleep <= 0 {
		s.opt.sleepRows = 0
	} else if s.opt.sleepRows < 1 {
		s.opt.sleepRows = 1
	}

	if s.opt.split || s.opt.Print {
		s.opt.Check = false
	}

	// 不再检查密码是否为空
	if s.opt.Host == "" || s.opt.Port == 0 || s.opt.User == "" {
		log.Warningf("%#v", s.opt)
		msg := ""
		if s.opt.Host == "" {
			msg += "主机名为空,"
		}
		if s.opt.Port == 0 {
			msg += "端口为0,"
		}
		if s.opt.User == "" {
			msg += "用户名为空,"
		}
		return fmt.Errorf(s.getErrorMessage(ER_SQL_INVALID_SOURCE), strings.TrimRight(msg, ","))
	}

	var addr string
	if s.opt.middlewareExtend == "" {
		tlsValue, err := s.getTLSConfig()
		if err != nil {
			return err
		}
		addr = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&maxAllowedPacket=%d&tls=%s",
			s.opt.User, s.opt.Password, s.opt.Host, s.opt.Port, s.opt.db,
			s.inc.DefaultCharset, s.inc.MaxAllowedPacket, tlsValue)
	} else {
		s.opt.middlewareExtend = fmt.Sprintf("/*%s*/",
			strings.Replace(s.opt.middlewareExtend, ": ", "=", 1))

		addr = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&maxAllowedPacket=%d&maxOpen=100&maxLifetime=60",
			s.opt.User, s.opt.Password, s.opt.Host, s.opt.Port,
			s.opt.middlewareDB, s.inc.DefaultCharset, s.inc.MaxAllowedPacket)
	}

	db, err := gorm.Open("mysql", fmt.Sprintf("%s&autocommit=1", addr))

	if err != nil {
		return fmt.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
	}

	if s.opt.tranBatch > 1 {
		s.ddlDB, _ = gorm.Open("mysql", fmt.Sprintf("%s&autocommit=1", addr))
		s.ddlDB.LogMode(false)
	}

	// 禁用日志记录器，不显示任何日志
	db.LogMode(false)

	s.db = db

	if s.opt.Execute {
		if s.opt.Backup && !s.checkBinlogIsOn() {
			return errors.New("binlog日志未开启,无法备份!")
		}
	}

	if s.opt.Backup {
		// 不再检查密码是否为空
		if s.inc.BackupHost == "" || s.inc.BackupPort == 0 || s.inc.BackupUser == "" {
			return errors.New(s.getErrorMessage(ER_INVALID_BACKUP_HOST_INFO))
		} else {
			addr = fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=%s&parseTime=True&loc=Local&autocommit=1",
				s.inc.BackupUser, s.inc.BackupPassword, s.inc.BackupHost, s.inc.BackupPort,
				s.inc.DefaultCharset)
			backupdb, err := gorm.Open("mysql", addr)

			if err != nil {
				return fmt.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
			}

			backupdb.LogMode(false)
			s.backupdb = backupdb
		}
	}

	tmp := s.processInfo.Load()
	if tmp != nil {
		pi := tmp.(util.ProcessInfo)
		pi.DestHost = s.opt.Host
		pi.DestPort = s.opt.Port
		pi.DestUser = s.opt.User

		if s.opt.Check {
			pi.Command = "CHECK"
		} else if s.opt.Execute {
			pi.Command = "EXECUTE"
		}
		s.processInfo.Store(pi)
	}

	s.mysqlServerVersion()
	s.setSqlSafeUpdates()
	s.setLockWaitTimeout()

	if s.opt.Backup && s.dbType == DBTypeTiDB {
		s.appendErrorMessage("TiDB暂不支持备份功能.")
	}

	return nil
}

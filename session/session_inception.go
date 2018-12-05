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
	"time"

	"github.com/hanchuanchuan/tidb/ast"
	"github.com/hanchuanchuan/tidb/metrics"
	"github.com/hanchuanchuan/tidb/model"
	"github.com/hanchuanchuan/tidb/mysql"
	// "github.com/hanchuanchuan/tidb/table"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"regexp"
	"strconv"
	"strings"

	// "database/sql/driver"
	mysqlDriver "github.com/go-sql-driver/mysql"

	// "github.com/hanchuanchuan/tidb/config"
	"github.com/jinzhu/gorm"
	// _ "github.com/jinzhu/gorm/dialects/mysql"
)

type MasterStatus struct {
	gorm.Model
	File              string `gorm:"Column:File"`
	Position          int    `gorm:"Column:Position"`
	Binlog_Do_DB      string `gorm:"Column:Binlog_Do_DB"`
	Binlog_Ignore_DB  string `gorm:"Column:Binlog_Ignore_DB"`
	Executed_Gtid_Set string `gorm:"Column:Executed_Gtid_Set"`
}

type sourceOptions struct {
	host           string
	port           int
	user           string
	password       string
	check          bool
	execute        bool
	backup         bool
	ignoreWarnings bool
}

type ExplainInfo struct {
	gorm.Model

	SelectType   string  `gorm:"Column:select_type"`
	Table        string  `gorm:"Column:table"`
	Partitions   string  `gorm:"Column:partitions"`
	Type         string  `gorm:"Column:type"`
	PossibleKeys string  `gorm:"Column:possible_keys"`
	Key          string  `gorm:"Column:key"`
	KeyLen       string  `gorm:"Column:key_len"`
	Ref          string  `gorm:"Column:ref"`
	Rows         int     `gorm:"Column:rows"`
	Filtered     float32 `gorm:"Column:filtered"`
	Extra        string  `gorm:"Column:Extra"`
}

type FieldInfo struct {
	gorm.Model

	Field      string `gorm:"Column:Field"`
	Type       string `gorm:"Column:Type"`
	Collation  string `gorm:"Column:Collation"`
	Null       string `gorm:"Column:Null"`
	Key        string `gorm:"Column:Key"`
	Default    string `gorm:"Column:Default"`
	Extra      string `gorm:"Column:Extra"`
	Privileges string `gorm:"Column:Privileges"`
	Comment    string `gorm:"Column:Comment"`
}

type TableInfo struct {
	Schema string
	Name   string
	Fields []FieldInfo
}

type IndexInfo struct {
	gorm.Model

	Table      string `gorm:"Column:Table"`
	IndexName  string `gorm:"Column:Key_name"`
	Seq        string `gorm:"Column:Seq_in_index"`
	ColumnName string `gorm:"Column:Column_name"`
	IndexType  string `gorm:"Column:Index_type"`
}

var reg *regexp.Regexp
var regFieldLength *regexp.Regexp

const (
	MaxKeyLength      = 767
	RemoteBackupTable = "$_$Inception_backup_information$_$"
)

func init() {

	// 正则匹配sql的option设置
	reg = regexp.MustCompile(`^\/\*(.*?)\*\/`)

	regFieldLength = regexp.MustCompile(`^.*?\((\d)`)

}

func (s *session) ExecuteInc(ctx context.Context, sql string) (recordSets []ast.RecordSet, err error) {
	if recordSets, err = s.executeInc(ctx, sql); err != nil {
		err = errors.Trace(err)
		s.sessionVars.StmtCtx.AppendError(err)
	}
	return
}

func (s *session) executeInc(ctx context.Context, sql string) (recordSets []ast.RecordSet, err error) {

	// log.Info("%+v", s.Inc)
	// log.Info("---------------===============")
	sqlList := strings.Split(sql, "\n")
	// log.Info(len(sqlList))
	// for i, a := range sqlList {
	// 	log.Info(i, a)
	// }
	if !s.haveBegin && sql == "select @@version_comment limit 1" {
		return s.Execute(ctx, sql)
	}
	if !s.haveBegin && sql == "SET AUTOCOMMIT = 0" {
		return s.Execute(ctx, sql)
	}

	s.PrepareTxnCtx(ctx)
	connID := s.sessionVars.ConnectionID
	err = s.loadCommonGlobalVariablesIfNeeded()
	if err != nil {
		return nil, errors.Trace(err)
	}

	charsetInfo, collation := s.sessionVars.GetCharsetInfo()

	// Step1: Compile query string to abstract syntax trees(ASTs).
	startTS := time.Now()

	lineCount := len(sqlList) - 1

	var buf []string
	for i, sql_line := range sqlList {
		// 如果以分号结尾,或者是最后一行,就做解析
		if strings.HasSuffix(sql_line, ";") || i == lineCount {

			buf = append(buf, sql_line)

			s1 := strings.Join(buf, "\n")
			stmtNodes, err := s.ParseSQL(ctx, s1, charsetInfo, collation)

			// log.Info(len(stmtNodes))
			// log.Info("语句数", len(stmtNodes))

			if err != nil {
				// log.Info(err)
				// log.Info(fmt.Sprintf("解析失败! %s", err))
				s.recordSets.Append(&Record{
					Sql:          s1,
					ErrLevel:     2,
					ErrorMessage: err.Error(),
				})
				return s.recordSets.Rows(), nil

				s.rollbackOnError(ctx)
				log.Warnf("con:%d parse error:\n%v\n%s", connID, err, s1)
				return nil, errors.Trace(err)
			}

			for _, stmtNode := range stmtNodes {
				// log.Info("当前语句: ", stmtNode)
				// log.Infof("%T\n", stmtNode)

				currentSql := stmtNode.Text()

				// var checkResult *Record = nil
				s.myRecord = &Record{
					Sql:   currentSql,
					Buf:   new(bytes.Buffer),
					Type:  stmtNode,
					Stage: StageCheck,
				}

				switch node := stmtNode.(type) {
				case *ast.InceptionStartStmt:
					s.haveBegin = true

					s.parseOptions(sql)

					if s.db != nil {
						defer s.db.Close()
					}
					if s.backupdb != nil {
						defer s.backupdb.Close()
					}

					if s.myRecord.ErrLevel == 2 {
						s.myRecord.Sql = ""
						s.recordSets.Append(s.myRecord)

						return s.recordSets.Rows(), nil
					}

					continue
				case *ast.InceptionCommitStmt:
					s.haveCommit = true

					s.executeCommit()

					return s.recordSets.Rows(), nil

				case *ast.InsertStmt:
					s.checkInsert(node, currentSql)
				case *ast.DeleteStmt:
					s.checkDelete(node, currentSql)
				case *ast.UpdateStmt:
					s.checkUpdate(node, currentSql)

				case *ast.UseStmt:
					s.checkChangeDB(node)

				case *ast.CreateDatabaseStmt:
					s.checkCreateDB(node)
				case *ast.DropDatabaseStmt:
					s.checkDropDB(node)

				case *ast.CreateTableStmt:
					s.checkCreateTable(node, currentSql)
				case *ast.AlterTableStmt:
					s.checkAlterTable(node, currentSql)
				case *ast.DropTableStmt:
					s.checkDropTable(node, currentSql)
				case *ast.RenameTableStmt:
					s.checkRenametable(node, currentSql)
				case *ast.TruncateTableStmt:
					s.checkTruncateTable(node, currentSql)

				case *ast.CreateIndexStmt:
					// s.checkCreateIndex(node, currentSql)
					// table *ast.TableName, IndexName string,
					// IndexColNames []*ast.IndexColName, IndexOption *ast.IndexOption
					s.checkCreateIndex(node.Table, node.IndexName,
						node.IndexColNames, node.IndexOption, nil, node.Unique)

				case *ast.DropIndexStmt:
					s.checkDropIndex(node, currentSql)

				case *ast.CreateViewStmt:
					s.AppendErrorMessage(fmt.Sprintf("命令禁止! 无法创建视图'%s'.", node.ViewName.Name))

				default:
					log.Info("无匹配类型...")
					log.Infof("%T\n", stmtNode)
					s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
				}

				// log.Info(stmtNode.Text())
				// log.Info("---")

				if !s.haveBegin {
					s.AppendErrorMessage("Must start as begin statement.")
					s.recordSets.Append(s.myRecord)
					// s.recordSets.Append(&Record{
					// 	Sql:          currentSql,
					// 	ErrLevel:     2,
					// 	ErrorMessage: "Must start as begin statement.",
					// })
					return s.recordSets.Rows(), nil
				}

				s.recordSets.Append(s.myRecord)
			}

			buf = nil

		} else if i < lineCount {
			buf = append(buf, sql_line)
		}
	}

	label := metrics.LblGeneral
	if s.sessionVars.InRestrictedSQL {
		label = metrics.LblInternal
	}
	metrics.SessionExecuteParseDuration.WithLabelValues(label).Observe(time.Since(startTS).Seconds())

	if !s.haveCommit {
		s.recordSets.Append(&Record{
			Sql:          "",
			ErrLevel:     2,
			ErrorMessage: "Must end with commit.",
		})
	}

	return s.recordSets.Rows(), nil

	if s.sessionVars.ClientCapability&mysql.ClientMultiResults == 0 && len(recordSets) > 1 {
		// return the first recordset if client doesn't support ClientMultiResults.
		recordSets = recordSets[:1]
	}

	// t := &testStatisticsSuite{}

	return recordSets, nil
}

func (s *session) executeCommit() {
	if s.opt.check {
		return
	}

	// log.Info("执行最大错误等级", s.recordSets.MaxLevel)

	if s.recordSets.MaxLevel == 2 ||
		(s.recordSets.MaxLevel == 1 && !s.opt.ignoreWarnings) {
		return
	}

	// 如果有错误时,把错误输出放在第一行
	s.myRecord = s.recordSets.All()[0]

	if s.opt.backup {
		if !s.checkBinlogIsOn() {
			s.AppendErrorMessage("binlog日志未开启,无法备份!")
			return
		}

		if !s.checkBinlogFormatIsRow() {
			s.modifyBinlogFormatRow()
		}
	}

	if s.recordSets.MaxLevel == 2 ||
		(s.recordSets.MaxLevel == 1 && !s.opt.ignoreWarnings) {
		return
	}

	s.executeAllStatement()

	if s.recordSets.MaxLevel == 2 ||
		(s.recordSets.MaxLevel == 1 && !s.opt.ignoreWarnings) {
		return
	}

	if s.opt.backup {

		// 如果有错误时,把错误输出放在第一行
		// s.myRecord = s.recordSets.All()[0]

		for _, record := range s.recordSets.All() {

			if s.checkSqlIsDML(record) || s.checkSqlIsDDL(record) {
				s.myRecord = record

				errno := s.mysqlCreateBackupTables(record)
				if errno == 2 {
					break
				}

				if record.TableInfo == nil {
					s.AppendErrorMessage("无表结构信息,生成备份失败!")
				} else {
					s.mysqlBackupSql(record)
				}
			}
		}

	}
}

func (s *session) mysqlBackupSql(record *Record) {
	if s.checkSqlIsDDL(record) {
		s.mysqlExecuteBackupInfoInsertSql(record)

		s.mysqlExecuteBackupSqlForDDL(record)
	} else if s.checkSqlIsDML(record) {
		s.mysqlExecuteBackupInfoInsertSql(record)
	}
}

func makeOPIDByTime(execTime int64, threadId int, seqNo int) string {
	return fmt.Sprintf("'%d_%d_%08d'", execTime, threadId, seqNo)
}

func (s *session) mysqlExecuteBackupSqlForDDL(record *Record) {
	if record.DDLRollback == "" {
		return
	}

	buf := bytes.NewBufferString("INSERT INTO ")
	dbname := s.getRemoteBackupDBName(record)
	buf.WriteString(fmt.Sprintf("`%s`.`%s`", dbname, record.TableInfo.Name))
	buf.WriteString("(rollback_statement, opid_time) VALUES('")
	buf.WriteString(record.DDLRollback)
	buf.WriteString("',")
	buf.WriteString(makeOPIDByTime(
		record.ExecTimestamp, record.ThreadId, record.SeqNo))
	buf.WriteString(")")

	sql := buf.String()

	if err := s.backupdb.Exec(sql).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
			record.StageStatus = StatusBackupFail
		}
	}
	record.StageStatus = StatusBackupOK
}

func (s *session) mysqlExecuteBackupInfoInsertSql(record *Record) int {

	buf := bytes.NewBufferString("INSERT INTO ")
	dbname := s.getRemoteBackupDBName(record)
	buf.WriteString(fmt.Sprintf("`%s`.`%s`", dbname, RemoteBackupTable))
	buf.WriteString(" VALUES(")
	buf.WriteString(makeOPIDByTime(record.ExecTimestamp, record.ThreadId, record.SeqNo))
	buf.WriteString(",'")
	buf.WriteString(record.StartFile)
	buf.WriteString("',")
	buf.WriteString(strconv.Itoa(record.StartPosition))
	buf.WriteString(",'")
	buf.WriteString(record.EndFile)
	buf.WriteString("',")
	buf.WriteString(strconv.Itoa(record.EndPosition))
	buf.WriteString(",'")
	buf.WriteString(record.Sql)
	buf.WriteString("','")
	buf.WriteString(s.opt.host)
	buf.WriteString("','")
	buf.WriteString(record.TableInfo.Schema)
	buf.WriteString("','")
	buf.WriteString(record.TableInfo.Name)
	buf.WriteString("',")
	buf.WriteString(strconv.Itoa(s.opt.port))
	buf.WriteString(",NOW(),'UNKNOWN')")

	sql := buf.String()

	if err := s.backupdb.Exec(sql).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)

			record.StageStatus = StatusBackupFail
			return 2
		}
	}

	record.StageStatus = StatusBackupOK
	return 0
}

func (s *session) mysqlBackupSingleDDLStatement(record *Record) {

}

func (s *session) mysqlBackupSingleStatement(record *Record) {

}

func (s *session) checkSqlIsDML(record *Record) bool {
	switch record.Type.(type) {
	case *ast.InsertStmt, *ast.DeleteStmt, *ast.UpdateStmt:
		return true
	default:
		return false
	}
}

func (s *session) mysqlCreateBackupTables(record *Record) int {

	if record.TableInfo == nil {
		return 0
	}

	backupDBName := s.getRemoteBackupDBName(record)
	if backupDBName == "" {
		return 2
	}

	if _, ok := s.backupDBCacheList[backupDBName]; !ok {
		sql := fmt.Sprintf("create database if not exists `%s`;", backupDBName)
		if err := s.backupdb.Exec(sql).Error; err != nil {
			log.Error(err)
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
				if myErr.Number != 1007 { /*ER_DB_CREATE_EXISTS*/
					s.AppendErrorMessage(myErr.Message)
					return 2
				}
			}
		}
		s.backupDBCacheList[backupDBName] = true
	}

	key := fmt.Sprintf("%s.%s", backupDBName, record.TableInfo.Name)

	if _, ok := s.backupTableCacheList[key]; !ok {
		createSql := s.mysqlCreateSqlFromTableInfo(backupDBName, record.TableInfo)
		if err := s.backupdb.Exec(createSql).Error; err != nil {
			log.Error(err)
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
				if myErr.Number != 1050 { /*ER_TABLE_EXISTS_ERROR*/
					s.AppendErrorMessage(myErr.Message)
					return 2
				}
			}
		}
		s.backupTableCacheList[key] = true
	}

	key = fmt.Sprintf("%s.%s", backupDBName, RemoteBackupTable)
	if _, ok := s.backupTableCacheList[key]; !ok {
		createSql := s.mysqlCreateSqlBackupTable(backupDBName)
		if err := s.backupdb.Exec(createSql).Error; err != nil {
			log.Error(err)
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
				if myErr.Number != 1050 { /*ER_TABLE_EXISTS_ERROR*/
					s.AppendErrorMessage(myErr.Message)
					return 2
				}
			}
		}
		s.backupTableCacheList[key] = true
	}

	return int(record.ErrLevel)
}

func (s *session) mysqlCreateSqlBackupTable(dbname string) string {

	buf := bytes.NewBufferString("CREATE TABLE if not exists ")

	buf.WriteString(fmt.Sprintf("`%s`.`%s`", dbname, RemoteBackupTable))
	buf.WriteString("(")

	buf.WriteString("opid_time varchar(50),")
	buf.WriteString("start_binlog_file varchar(512),")
	buf.WriteString("start_binlog_pos int,")
	buf.WriteString("end_binlog_file varchar(512),")
	buf.WriteString("end_binlog_pos int,")
	buf.WriteString("sql_statement text,")
	buf.WriteString("host VARCHAR(64),")
	buf.WriteString("dbname VARCHAR(64),")
	buf.WriteString("tablename VARCHAR(64),")
	buf.WriteString("port INT,")
	buf.WriteString("time TIMESTAMP,")
	buf.WriteString("type VARCHAR(20),")
	buf.WriteString("PRIMARY KEY(opid_time)")

	buf.WriteString(")ENGINE INNODB DEFAULT CHARSET UTF8;")

	return buf.String()
}
func (s *session) mysqlCreateSqlFromTableInfo(dbname string, ti *TableInfo) string {

	buf := bytes.NewBufferString("CREATE TABLE if not exists ")
	buf.WriteString(fmt.Sprintf("`%s`.`%s`", dbname, ti.Name))
	buf.WriteString("(")

	buf.WriteString("id bigint auto_increment primary key, ")
	buf.WriteString("rollback_statement mediumtext, ")
	buf.WriteString("opid_time varchar(50)")

	buf.WriteString(") ENGINE INNODB DEFAULT CHARSET UTF8;")

	return buf.String()
}

func (s *session) mysqlRealQueryBackup(sql string) error {
	res := s.db.Exec(sql)
	err := res.Error
	if err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	}
	return err
}

func (s *session) getRemoteBackupDBName(record *Record) (v string) {

	v = fmt.Sprintf("%s_%d_%s", s.opt.host, s.opt.port, record.TableInfo.Schema)

	if len(v) > mysql.MaxDatabaseNameLength {
		s.AppendErrorNo(ER_TOO_LONG_BAKDB_NAME, s.opt.host, s.opt.port, record.TableInfo.Schema)
		return ""
	}

	v = strings.Replace(v, "-", "_", -1)
	v = strings.Replace(v, ".", "_", -1)

	return
}

func (s *session) checkSqlIsDDL(record *Record) bool {

	switch record.Type.(type) {
	case *ast.CreateTableStmt,
		*ast.AlterTableStmt,
		*ast.DropTableStmt,

		*ast.CreateIndexStmt,
		*ast.DropIndexStmt:
		return true

	default:
		return false
	}
}

func (s *session) executeAllStatement() {

	log.Info("---------------------")
	for _, record := range s.recordSets.All() {
		errno := s.executeRemoteCommand(record)
		if errno == 2 {
			break
		}
	}
}

func (s *session) executeRemoteCommand(record *Record) int {

	s.myRecord = record
	record.Stage = StageExec

	log.Infof("%T", record.Type)
	switch node := record.Type.(type) {

	case *ast.InsertStmt, *ast.DeleteStmt, *ast.UpdateStmt:
		s.executeRemoteStatementAndBackup(record)

	case *ast.UseStmt,
		*ast.DropDatabaseStmt,

		*ast.CreateTableStmt,
		*ast.AlterTableStmt,
		*ast.DropTableStmt,
		*ast.RenameTableStmt,
		*ast.TruncateTableStmt,

		*ast.CreateIndexStmt,
		*ast.DropIndexStmt:
		s.executeRemoteStatement(record)

	default:
		log.Info("无匹配类型...")
		log.Infof("%T\n", node)
		s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
	}

	return int(record.ErrLevel)
}

func (s *session) executeRemoteStatement(record *Record) {
	log.Info("executeRemoteStatement")
	sql := record.Sql

	start := time.Now()

	res := s.db.Exec(sql)

	record.ExecTime = fmt.Sprintf("%.3f", time.Since(start).Seconds())

	record.ExecTimestamp = time.Now().Unix()

	err := res.Error
	if err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
		log.Error(err)
		record.StageStatus = StatusExecFail
	} else {
		record.AffectedRows = int(res.RowsAffected)
		record.StageStatus = StatusExecOK

		record.ThreadId = s.fetchThreadID()

		if _, ok := record.Type.(*ast.CreateTableStmt); ok &&
			record.TableInfo == nil && record.DBName != "" && record.TableName != "" {
			record.TableInfo = s.getTableFromCache(record.DBName, record.TableName, true)
		}
	}
}

func (s *session) executeRemoteStatementAndBackup(record *Record) {
	log.Info("executeRemoteStatementAndBackup")

	if s.opt.backup {
		masterStatus := s.mysqlFetchMasterBinlogPosition()
		if masterStatus != nil {
			record.StartFile = masterStatus.File
			record.StartPosition = masterStatus.Position
		}
	}

	s.executeRemoteStatement(record)

	if s.opt.backup {
		masterStatus := s.mysqlFetchMasterBinlogPosition()
		if masterStatus != nil {
			record.EndFile = masterStatus.File
			record.EndPosition = masterStatus.Position
		}
	}
}

func (s *session) mysqlFetchMasterBinlogPosition() *MasterStatus {
	log.Info("mysqlFetchMasterBinlogPosition")

	r := MasterStatus{}
	if err := s.db.Raw("SHOW MASTER STATUS").Scan(&r).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
		log.Error(err)
	}
	return &r
}

func (s *session) checkBinlogFormatIsRow() bool {
	log.Info("checkBinlogFormatIsRow")

	sql := "show variables like 'binlog_format';"

	var format string

	rows, err := s.db.Raw(sql).Rows()
	defer rows.Close()
	if err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else {
		for rows.Next() {
			rows.Scan(&format, &format)
		}
	}

	log.Infof("binlog format: %s", format)
	return format == "ROW"
}

func (s *session) fetchThreadID() (threadId int) {
	log.Info("fetchThreadID")

	rows, err := s.db.Raw("select connection_id()").Rows()
	defer rows.Close()
	if err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else {
		for rows.Next() {
			rows.Scan(&threadId)
			return
		}
	}
	return
}

func (s *session) modifyBinlogFormatRow() {
	log.Info("modifyBinlogFormatRow")

	sql := "set session binlog_format=row;"

	res := s.db.Exec(sql)

	if err := res.Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
		log.Error(err)
	}
}

func (s *session) checkBinlogIsOn() bool {
	log.Info("checkBinlogIsOn")

	sql := "show variables like 'log_bin';"

	var format string

	rows, err := s.db.Raw(sql).Rows()
	defer rows.Close()
	if err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else {
		for rows.Next() {
			rows.Scan(&format, &format)
		}
	}

	log.Infof("log_bin is %s", format)
	return format == "ON"
}

func (s *session) parseOptions(sql string) {

	firsts := reg.FindStringSubmatch(sql)
	if len(firsts) < 2 {
		s.AppendErrorNo(ER_SQL_INVALID_SOURCE)
		return
	}

	options := strings.Replace(strings.Replace(firsts[1], "-", "", -1), "_", "", -1)
	options = strings.Replace(options, ";", "\n", -1)
	options = strings.Replace(options, "=", ": ", -1)

	// log.Info(options)
	// viper.SetConfigType("toml")
	viper.SetConfigType("yaml")

	viper.ReadConfig(bytes.NewBuffer([]byte(options)))

	s.opt = &sourceOptions{
		host:           viper.GetString("host"),
		port:           viper.GetInt("port"),
		user:           viper.GetString("user"),
		password:       viper.GetString("password"),
		check:          viper.GetBool("check"),
		execute:        viper.GetBool("execute"),
		backup:         viper.GetBool("backup"),
		ignoreWarnings: viper.GetBool("ignoreWarnings"),
	}

	if s.opt.check {
		s.opt.execute = false
		s.opt.backup = false
	}

	if s.opt.host == "" || s.opt.port == 0 ||
		s.opt.user == "" || s.opt.password == "" {
		log.Info(s.opt)

		s.AppendErrorNo(ER_SQL_INVALID_SOURCE)
		return
	}

	addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local",
		s.opt.user, s.opt.password, s.opt.host, s.opt.port)
	db, err := gorm.Open("mysql", addr)

	// 禁用日志记录器，不显示任何日志
	// db.LogMode(false)

	if err != nil {
		log.Info(err)
		log.Info(err.Error())
		s.AppendErrorMessage(err.Error())
		return
	}

	s.db = db

	if s.opt.execute {
		if s.opt.backup && !s.checkBinlogIsOn() {
			s.AppendErrorMessage("binlog日志未开启,无法备份!")
		}
	}

	if s.opt.backup {
		if s.Inc.BackupHost == "" || s.Inc.BackupPort == 0 ||
			s.Inc.BackupUser == "" || s.Inc.BackupPassword == "" {
			s.AppendErrorNo(ER_INVALID_BACKUP_HOST_INFO)
		} else {
			addr = fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local",
				s.Inc.BackupUser, s.Inc.BackupPassword, s.Inc.BackupHost, s.Inc.BackupPort)
			backupdb, err := gorm.Open("mysql", addr)

			if err != nil {
				log.Info(err)
				s.AppendErrorMessage(err.Error())
				return
			}

			s.backupdb = backupdb
		}
	}
}

func (s *session) checkTruncateTable(node *ast.TruncateTableStmt, sql string) {

	log.Info("checkTruncateTable")

	// log.Infof("%#v \n", node)

	t := node.Table

	if !s.Inc.EnableDropTable {
		s.AppendErrorNo(ER_CANT_DROP_TABLE, t.Name)
	} else {

		table := s.getTableFromCache(t.Schema.O, t.Name.O, false)

		if table == nil {
			s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, t.Name)
		}
	}
}

func (s *session) checkDropTable(node *ast.DropTableStmt, sql string) {

	log.Info("checkDropTable")

	// log.Infof("%#v \n", node)
	for _, t := range node.Tables {

		if !s.Inc.EnableDropTable {
			s.AppendErrorNo(ER_CANT_DROP_TABLE, t.Name)
		} else {

			table := s.getTableFromCache(t.Schema.O, t.Name.O, false)

			s.myRecord.TableInfo = table

			//如果表不存在，但存在if existed，则跳过
			if table == nil && !node.IfExists {
				s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, t.Name)
			} else {
				s.mysqlShowCreateTable(table.Schema, table.Name)
			}
		}
	}
}

func (s *session) mysqlShowCreateTable(dbname string, tableName string) {
	sql := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`;", dbname, tableName)

	var res string

	rows, err := s.db.Raw(sql).Rows()
	defer rows.Close()
	if err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else {
		for rows.Next() {
			rows.Scan(&res)
		}
		s.myRecord.DDLRollback = res
		s.myRecord.DDLRollback += ";"
	}
}

func (s *session) checkRenametable(node *ast.RenameTableStmt, sql string) {

	log.Info("checkRenametable")

	log.Infof("%#v \n", node)

	s.getTableFromCache(node.OldTable.Schema.O, node.OldTable.Name.O, true)

	table := s.getTableFromCache(node.NewTable.Schema.O, node.NewTable.Name.O, false)
	if table != nil {
		s.AppendErrorNo(ER_TABLE_EXISTS_ERROR, node.NewTable.Name.O)
	}

}

func (s *session) checkCreateTable(node *ast.CreateTableStmt, sql string) {

	log.Info("checkCreateTable")

	// tidb暂不支持临时表 create temporary table t1

	log.Infof("%#v", node)
	// log.Infof("%#v", node.Options)
	// log.Infof("%#v", node.ReferTable)

	// Table       *TableName
	// ReferTable  *TableName
	// Cols        []*ColumnDef
	// Constraints []*Constraint
	// Options     []*TableOption
	// Partition   *PartitionOptions
	// OnDuplicate OnDuplicateCreateTableSelectType
	// Select      ResultSetNode

	s.checkDBExists(node.Table.Schema.O, true)

	table := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, false)

	if table != nil {
		s.AppendErrorNo(ER_TABLE_EXISTS_ERROR, node.Table.Name.O)
		return
	}

	if node.Table.Schema.O == "" {
		s.myRecord.DBName = s.DBName
	} else {
		s.myRecord.DBName = node.Table.Schema.O
	}
	s.myRecord.TableName = node.Table.Name.O

	// 缓存表结构 CREATE TABLE LIKE
	if node.ReferTable != nil {
		originTable := s.getTableFromCache(node.ReferTable.Schema.O, node.ReferTable.Name.O, true)
		if originTable != nil {
			table = copyTableInfo(originTable)

			table.Name = node.Table.Name.O
			if node.Table.Schema.O == "" {
				table.Schema = s.DBName
			} else {
				table.Schema = node.Table.Schema.O
			}
			s.addTableCache(table)
			s.myRecord.TableInfo = table
		}
	} else {

		hasComment := false
		for _, opt := range node.Options {
			// log.Infof("%#v", opt)
			switch opt.Tp {
			case ast.TableOptionEngine:
				if !strings.EqualFold(opt.StrValue, "innodb") {
					s.AppendErrorNo(ER_TABLE_MUST_INNODB, node.Table.Name.O)
				}
			case ast.TableOptionCharset, ast.TableOptionCollate:
				s.AppendErrorNo(ER_TABLE_CHARSET_MUST_NULL, node.Table.Name.O)
			case ast.TableOptionComment:
				if opt.StrValue != "" {
					hasComment = true
				}
			}
		}

		hasPrimary := false
		for _, ct := range node.Constraints {
			log.Infof("%#v", ct)

			switch ct.Tp {
			case ast.ConstraintPrimaryKey:
				hasPrimary = len(ct.Keys) > 0
				break
			}
		}

		if !hasPrimary {
			for _, field := range node.Cols {
				for _, op := range field.Options {
					if op.Tp == ast.ColumnOptionPrimaryKey {
						hasPrimary = true
						break
					}
				}
				if hasPrimary {
					break
				}
			}
		}

		if !hasPrimary {
			s.AppendErrorNo(ER_TABLE_MUST_HAVE_PK, node.Table.Name.O)
		}

		if !hasComment && s.Inc.CheckTableComment {
			s.AppendErrorNo(ER_TABLE_MUST_HAVE_COMMENT, node.Table.Name.O)
		}

		table = s.buildTableInfo(node)

		for _, nc := range node.Cols {
			s.mysqlCheckField(table, nc)
		}

		s.myRecord.TableInfo = table
	}

	if node.Partition != nil {
		s.AppendErrorNo(ER_PARTITION_NOT_ALLOWED)
	}

	if s.opt.execute {
		s.myRecord.DDLRollback = fmt.Sprintf("DROP TABLE `%s`.`%s`;", table.Schema, table.Name)
	}

}

func (s *session) buildTableInfo(node *ast.CreateTableStmt) *TableInfo {
	log.Info("buildTableInfo")

	table := &TableInfo{}

	if node.Table.Schema.O == "" {
		table.Schema = s.DBName
	} else {
		table.Schema = node.Table.Schema.O
	}

	table.Name = node.Table.Name.O
	table.Fields = make([]FieldInfo, 0, len(node.Cols))
	// 	Field      string `gorm:"Column:Field"`
	// Type       string `gorm:"Column:Type"`
	// Collation  string `gorm:"Column:Collation"`
	// Null       string `gorm:"Column:Null"`
	// Key        string `gorm:"Column:Key"`
	// Default    string `gorm:"Column:Default"`
	// Extra      string `gorm:"Column:Extra"`
	// Privileges string `gorm:"Column:Privileges"`
	// Comment    string `gorm:"Column:Comment"`

	for _, field := range node.Cols {
		c := &FieldInfo{}

		c.Field = field.Name.Name.String()
		c.Type = field.Tp.CompactStr()
		c.Null = "YES"

		for _, op := range field.Options {
			switch op.Tp {
			case ast.ColumnOptionComment:
				c.Comment = op.Expr.GetDatum().GetString()
			case ast.ColumnOptionNotNull:
				c.Null = "NO"
			case ast.ColumnOptionPrimaryKey:
				c.Key = "PRI"
			case ast.ColumnOptionDefaultValue:
				c.Default = op.Expr.GetDatum().GetString()
			}
		}

		table.Fields = append(table.Fields, *c)
	}

	// log.Infof("%#v", table)

	return table
}

func (s *session) checkAlterTable(node *ast.AlterTableStmt, sql string) {

	log.Info("checkAlterTable")

	// Table *TableName
	// Specs []*AlterTableSpec

	// Tp              AlterTableType
	// Name            string
	// Constraint      *Constraint
	// Options         []*TableOption
	// NewTable        *TableName
	// NewColumns      []*ColumnDef
	// OldColumnName   *ColumnName
	// Position        *ColumnPosition
	// LockType        LockType
	// Comment         string
	// FromKey         model.CIStr
	// ToKey           model.CIStr
	// PartDefinitions []*PartitionDefinition

	// AlterTableOption = 1
	// AlterTableAddColumns = 2
	// AlterTableAddConstraint = 3
	// AlterTableDropColumn = 4
	// AlterTableDropPrimaryKey = 5
	// AlterTableDropIndex = 6
	// AlterTableDropForeignKey = 7
	// AlterTableModifyColumn = 8
	// AlterTableChangeColumn = 9
	// AlterTableRenameTable = 10
	// AlterTableAlterColumn = 11
	// AlterTableLock = 12
	// AlterTableAlgorithm = 13
	// AlterTableRenameIndex = 14
	// AlterTableForce = 15
	// AlterTableAddPartitions = 16
	// AlterTableDropPartition = 17

	s.checkDBExists(node.Table.Schema.O, true)

	// table := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, true)
	table := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, true)
	if table == nil {
		return
	}

	s.myRecord.TableInfo = table

	if s.opt.execute {
		s.myRecord.DDLRollback += fmt.Sprintf("ALTER TABLE `%s`.`%s` ",
			table.Schema, table.Name)
	}
	// log.Infof("%s \n", table)
	for _, alter := range node.Specs {
		// log.Infof("%s \n", alter)
		// log.Info(alter.Tp)
		switch alter.Tp {
		case ast.AlterTableAddColumns:
			s.checkAddColumn(table, alter)
		case ast.AlterTableDropColumn:
			s.checkDropColumn(table, alter)

		case ast.AlterTableAddConstraint:
			s.checkAddConstraint(table, alter)

		case ast.AlterTableDropPrimaryKey:
			s.checkDropPrimaryKey(table, alter)
		case ast.AlterTableDropIndex:
			s.checkAlterTableDropIndex(table, alter.Name)

		case ast.AlterTableDropForeignKey:
			s.checkDropForeignKey(table, alter)

		case ast.AlterTableModifyColumn:
			s.checkModifyColumn(table, alter)
		case ast.AlterTableChangeColumn:
			s.checkChangeColumn(table, alter)

		case ast.AlterTableRenameTable:
			s.checkAlterTableRenameTable(table, alter)

		default:
			s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
			log.Info("未定义的解析: ", alter.Tp)
		}
	}
}

func (s *session) checkAlterTableRenameTable(t *TableInfo, c *ast.AlterTableSpec) {
	// log.Info("checkAlterTableRenameTable")

	table := s.getTableFromCache(c.NewTable.Schema.O, c.NewTable.Name.O, false)
	if table != nil {
		s.AppendErrorNo(ER_TABLE_EXISTS_ERROR, c.NewTable.Name.O)
	}

}

func (s *session) checkChangeColumn(t *TableInfo, c *ast.AlterTableSpec) {
	log.Info("checkChangeColumn")

	// log.Infof("%#v \n", c)

	log.Info(c.OldColumnName, len(c.NewColumns))
	found := false
	for _, nc := range c.NewColumns {
		if nc.Name.Name.L == c.OldColumnName.Name.O {
			found = true
		}
	}
	// 更新了列名
	if !found {
		s.AppendErrorMessage(
			fmt.Sprintf("列'%s'.'%s'禁止变更列名.",
				t.Name, c.OldColumnName.Name))
	}
	s.checkModifyColumn(t, c)
}

func (s *session) checkModifyColumn(t *TableInfo, c *ast.AlterTableSpec) {
	log.Info("checkModifyColumn")

	log.Infof("%#v \n", c)

	for _, nc := range c.NewColumns {
		// log.Infof("%s \n", nc)
		log.Infof("%s --- %s \n", nc.Name, nc.Tp)
		found := false
		var foundField FieldInfo

		if c.OldColumnName == nil || c.OldColumnName.Name.L == nc.Name.Name.L {

			for _, field := range t.Fields {
				if strings.EqualFold(field.Field, nc.Name.Name.O) {
					found = true
					foundField = field
					break
				}
			}

			if !found {
				s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name))
			}
		} else {
			oldFound := false
			newFound := false
			for _, field := range t.Fields {
				if strings.EqualFold(field.Field, c.OldColumnName.Name.L) {
					oldFound = true
					foundField = field
				}
				if strings.EqualFold(field.Field, nc.Name.Name.L) {
					newFound = true
				}
			}

			if newFound {
				s.AppendErrorNo(ER_COLUMN_EXISTED, fmt.Sprintf("%s.%s", t.Name, foundField.Field))
			} else if !oldFound {
				s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name))
			}
		}

		// 未变更列名时,列需要存在
		// 变更列名后,新列名不能存在
		// if c.OldColumnName == nil && !found {
		// 	s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name))
		// } else if c.OldColumnName != nil &&
		// 	!strings.EqualFold(c.OldColumnName.Name.L, foundField.Field) && found {
		// 	s.AppendErrorNo(ER_COLUMN_EXISTED, fmt.Sprintf("%s.%s", t.Name, foundField.Field))
		// }

		if nc.Tp.Charset != "" || nc.Tp.Collate != "" {
			s.AppendErrorNo(ER_CHARSET_ON_COLUMN, t.Name, nc.Name.Name)
		}

		fieldType := nc.Tp.CompactStr()
		log.Info("--------------", nc.Name, fieldType, foundField.Type, foundField)

		switch nc.Tp.Tp {
		case mysql.TypeDecimal, mysql.TypeNewDecimal,
			mysql.TypeVarchar,
			mysql.TypeVarString, mysql.TypeString:

			str := string([]byte(foundField.Type)[:7])
			if strings.Index(fieldType, str) == -1 {
				s.AppendErrorNo(ER_CHANGE_COLUMN_TYPE,
					fmt.Sprintf("%s.%s", t.Name, nc.Name.Name),
					foundField.Type, fieldType)
			}
		default:
			if fieldType != foundField.Type {
				s.AppendErrorNo(ER_CHANGE_COLUMN_TYPE,
					fmt.Sprintf("%s.%s", t.Name, nc.Name.Name),
					foundField.Type, fieldType)
			}
		}

		s.mysqlCheckField(t, nc)
	}

	// if c.Position.Tp != ast.ColumnPositionNone {
	// 	found := false
	// 	for _, field := range t.Fields {
	// 		if strings.EqualFold(field.Field, c.Position.RelativeColumn.Name.O) {
	// 			found = true
	// 			break
	// 		}
	// 	}
	// 	if !found {
	// 		s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
	// 			fmt.Sprintf("%s.%s", t.Name, c.Position.RelativeColumn.Name))
	// 	}
	// }
}

func mysqlFiledIsBlob(tp byte) bool {

	if tp == mysql.TypeTinyBlob || tp == mysql.TypeMediumBlob ||
		tp == mysql.TypeLongBlob || tp == mysql.TypeBlob {
		return true
	}
	return false
}

func (s *session) mysqlCheckField(t *TableInfo, field *ast.ColumnDef) {
	log.Info("mysqlCheckField")

	// log.Infof("%#v", field.Tp)

	tableName := t.Name
	if field.Tp.Tp == mysql.TypeEnum ||
		field.Tp.Tp == mysql.TypeSet ||
		field.Tp.Tp == mysql.TypeBit {
		s.AppendErrorNo(ER_INVALID_DATA_TYPE, field.Name.Name)
	}

	if field.Tp.Tp == mysql.TypeString && field.Tp.Flen > 10 {
		s.AppendErrorNo(ER_CHAR_TO_VARCHAR_LEN, field.Name.Name)
	}

	if field.Tp.Charset != "" || field.Tp.Collate != "" {
		s.AppendErrorNo(ER_CHARSET_ON_COLUMN, tableName, field.Name.Name)
	}

	// notNullFlag := mysql.HasNotNullFlag(field.Tp.Flag)
	// autoIncrement := mysql.HasAutoIncrementFlag(field.Tp.Flag)

	hasComment := false
	notNullFlag := false
	autoIncrement := false

	if len(field.Options) > 0 {
		for _, op := range field.Options {
			// log.Infof("%#v", op)

			switch op.Tp {
			case ast.ColumnOptionComment:
				if op.Expr.GetDatum().GetString() != "" {
					hasComment = true
				}
			case ast.ColumnOptionNotNull:
				notNullFlag = true
			case ast.ColumnOptionNull:
				notNullFlag = false
			case ast.ColumnOptionAutoIncrement:
				autoIncrement = true
			}
		}
	}
	if !hasComment && s.Inc.CheckColumnComment {
		s.AppendErrorNo(ER_COLUMN_HAVE_NO_COMMENT, field.Name.Name, tableName)
	}

	if mysqlFiledIsBlob(field.Tp.Tp) {
		s.AppendErrorNo(ER_USE_TEXT_OR_BLOB, field.Name.Name)
	} else {
		if !notNullFlag && !s.Inc.EnableNullable {
			s.AppendErrorNo(ER_NOT_ALLOWED_NULLABLE, field.Name.Name, tableName)
		}
	}

	if len(field.Name.Name.O) > mysql.MaxColumnNameLength {
		s.AppendErrorNo(ER_WRONG_COLUMN_NAME, field.Name.Name)
	}

	if mysqlFiledIsBlob(field.Tp.Tp) && notNullFlag {
		s.AppendErrorNo(ER_TEXT_NOT_NULLABLE_ERROR, field.Name.Name, tableName)
	}

	if autoIncrement {
		if !mysql.HasUnsignedFlag(field.Tp.Flag) {
			s.AppendErrorNo(ER_AUTOINC_UNSIGNED, tableName)
		}

		if field.Tp.Tp != mysql.TypeLong &&
			field.Tp.Tp != mysql.TypeLonglong &&
			field.Tp.Tp != mysql.TypeInt24 {
			s.AppendErrorNo(ER_SET_DATA_TYPE_INT_BIGINT)
		}
	}

	if field.Tp.Tp == mysql.TypeTimestamp {
		if !mysql.HasNoDefaultValueFlag(field.Tp.Flag) {
			s.AppendErrorNo(ER_TIMESTAMP_DEFAULT, tableName)
		}
	}

}

func (s *session) checkDropForeignKey(t *TableInfo, c *ast.AlterTableSpec) {
	log.Info("checkDropForeignKey")

	log.Infof("%s \n", c)

	s.AppendErrorNo(ER_NOT_SUPPORTED_YET)

}
func (s *session) checkAlterTableDropIndex(t *TableInfo, indexName string) bool {
	log.Info("checkAlterTableDropIndex")

	sql := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s` where key_name=?", t.Schema, t.Name)

	var rows []IndexInfo

	if err := s.db.Raw(sql, indexName).Scan(&rows).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
		return false
	} else {

		if len(rows) == 0 {
			s.AppendErrorNo(ER_CANT_DROP_FIELD_OR_KEY, fmt.Sprintf("%s.%s", t.Name, indexName))
			return false
		}

		if s.opt.execute {
			s.myRecord.DDLRollback += fmt.Sprintf("ADD INDEX `%s`(", indexName)
			for i, row := range rows {
				if i == 0 {
					s.myRecord.DDLRollback += fmt.Sprintf("`%s`", row.ColumnName)
				} else {
					s.myRecord.DDLRollback += fmt.Sprintf(",`%s`", row.ColumnName)
				}
			}
			s.myRecord.DDLRollback += ");"
		}
		return true
	}

}

func (s *session) checkDropPrimaryKey(t *TableInfo, c *ast.AlterTableSpec) {
	log.Info("checkDropPrimaryKey")

	s.checkAlterTableDropIndex(t, "PRIMARY")
}

func (s *session) checkAddColumn(t *TableInfo, c *ast.AlterTableSpec) {
	log.Infof("%s \n", c)
	// log.Infof("%s \n", c.NewColumns)

	for _, nc := range c.NewColumns {
		log.Infof("%s \n", nc)
		found := false
		for _, field := range t.Fields {
			if strings.EqualFold(field.Field, nc.Name.Name.O) {
				found = true
				break
			}
		}
		if found {
			s.AppendErrorNo(ER_COLUMN_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name))
		} else {
			s.mysqlCheckField(t, nc)

			if s.opt.execute {
				s.myRecord.DDLRollback += fmt.Sprintf("ALTER TABLE `%s`.`%s` DROP COLUMN `%s`;",
					t.Schema, t.Name, nc.Name.Name.O)
			}
		}
	}

	if c.Position.Tp != ast.ColumnPositionNone {
		found := false
		for _, field := range t.Fields {
			if strings.EqualFold(field.Field, c.Position.RelativeColumn.Name.O) {
				found = true
				break
			}
		}
		if !found {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
				fmt.Sprintf("%s.%s", t.Name, c.Position.RelativeColumn.Name))
		}
	}
}

func (s *session) checkDropColumn(t *TableInfo, c *ast.AlterTableSpec) {
	log.Infof("%s \n", c)
	// log.Infof("%s \n", c.NewColumns)

	found := false
	for _, field := range t.Fields {
		if strings.EqualFold(field.Field, c.OldColumnName.Name.O) {
			found = true
			s.mysqlDropColumnRollback(field)
			break
		}
	}
	if !found {
		s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
			fmt.Sprintf("%s.%s", t.Name, c.OldColumnName.Name.O))
	}
}

func (s *session) mysqlDropColumnRollback(field FieldInfo) {
	if s.opt.check {
		return
	}

	buf := bytes.NewBufferString("ADD COLUMN `")
	buf.WriteString(field.Field)
	buf.WriteString("` ")
	buf.WriteString(field.Type)
	buf.WriteString(" ")
	if field.Null == "NO" {
		buf.WriteString("NOT NULL ")
	}
	if field.Default != "" {
		buf.WriteString("'")
		buf.WriteString(field.Default)
		buf.WriteString("' ")
	}
	if field.Comment != "" {
		buf.WriteString("'")
		buf.WriteString(field.Comment)
		buf.WriteString("' ")
	}
	buf.WriteString(";")

	s.myRecord.DDLRollback += buf.String()

}

func (s *session) checkDropIndex(node *ast.DropIndexStmt, sql string) {
	log.Info("checkDropIndex")
	// log.Infof("%#v \n", node)

	t := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, true)
	if t == nil {
		return
	}

	s.checkAlterTableDropIndex(t, node.IndexName)

}

func (s *session) checkCreateIndex1(node *ast.CreateIndexStmt, sql string) {

	log.Info("checkCreateIndex1")
	log.Infof("%#v \n", node)

	t := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, true)
	if t == nil {
		return
	}

	keyMaxLen := 0
	for _, col := range node.IndexColNames {
		found := false
		var foundField FieldInfo
		for _, field := range t.Fields {
			if strings.EqualFold(field.Field, col.Column.Name.O) {
				found = true
				foundField = field
				keyMaxLen += FieldLengthWithType(field.Type)
				break
			}
		}
		if !found {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, col.Column.Name.O))
		} else {
			tp := foundField.Type
			if strings.Index(tp, "bolb") > -1 {
				s.AppendErrorNo(ER_BLOB_USED_AS_KEY, foundField.Field)
			}
		}
	}

	if len(node.IndexName) > mysql.MaxIndexIdentifierLen {
		s.AppendErrorMessage(fmt.Sprintf("表'%s'的索引'%s'名称过长", t.Name, node.IndexName))
	}

	if keyMaxLen > MaxKeyLength {
		s.AppendErrorNo(ER_TOO_LONG_KEY, node.IndexName, MaxKeyLength)
	}

	// if len(node.IndexColNames) > s.Inc.MaxPrimaryKeyParts {
	// 	s.AppendErrorNo(ER_PK_TOO_MANY_PARTS, t.Name, col.Column.Name.O, s.Inc.MaxPrimaryKeyParts)
	// }

	querySql := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", t.Schema, t.Name)

	var rows []IndexInfo
	// log.Info("++++++++")
	if err := s.db.Raw(querySql).Scan(&rows).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	}

	if len(rows) > 0 {
		for _, row := range rows {
			if strings.EqualFold(row.IndexName, node.IndexName) {
				s.AppendErrorNo(ER_DUP_INDEX, node.IndexName, t.Schema, t.Name)
				break
			}
		}
	}

	if s.Inc.MaxKeys > 0 && len(rows) > int(s.Inc.MaxKeys) {
		s.AppendErrorNo(ER_TOO_MANY_KEYS, t.Name, s.Inc.MaxKeys)
	}

}

func (s *session) checkCreateIndex(table *ast.TableName, IndexName string,
	IndexColNames []*ast.IndexColName, IndexOption *ast.IndexOption,
	t *TableInfo, unique bool) {
	log.Info("checkCreateIndex")

	if t == nil {
		t = s.getTableFromCache(table.Schema.O, table.Name.O, true)
		if t == nil {
			return
		}
	}

	keyMaxLen := 0
	for _, col := range IndexColNames {
		found := false
		var foundField FieldInfo
		for _, field := range t.Fields {
			if strings.EqualFold(field.Field, col.Column.Name.O) {
				found = true
				foundField = field
				keyMaxLen += FieldLengthWithType(field.Type)
				break
			}
		}
		if !found {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, col.Column.Name.O))
		} else {
			tp := foundField.Type
			if strings.Index(tp, "bolb") > -1 {
				s.AppendErrorNo(ER_BLOB_USED_AS_KEY, foundField.Field)
			}
		}
	}

	if len(IndexName) > mysql.MaxIndexIdentifierLen {
		s.AppendErrorMessage(fmt.Sprintf("表'%s'的索引'%s'名称过长", t.Name, IndexName))
	}

	if keyMaxLen > MaxKeyLength {
		s.AppendErrorNo(ER_TOO_LONG_KEY, IndexName, MaxKeyLength)
	}

	// if len(IndexColNames) > s.Inc.MaxPrimaryKeyParts {
	// 	s.AppendErrorNo(ER_PK_TOO_MANY_PARTS, t.Name, col.Column.Name.O, s.Inc.MaxPrimaryKeyParts)
	// }

	querySql := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", t.Schema, t.Name)

	var rows []IndexInfo
	// log.Info("++++++++")
	if err := s.db.Raw(querySql).Scan(&rows).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	}

	if len(rows) > 0 {
		for _, row := range rows {
			if strings.EqualFold(row.IndexName, IndexName) {
				s.AppendErrorNo(ER_DUP_INDEX, IndexName, t.Schema, t.Name)
				break
			}
		}
	}

	if s.Inc.MaxKeys > 0 && len(rows) > int(s.Inc.MaxKeys) {
		s.AppendErrorNo(ER_TOO_MANY_KEYS, t.Name, s.Inc.MaxKeys)
	}

	if s.opt.execute {
		s.myRecord.DDLRollback += fmt.Sprintf("DROP INDEX `%s`;", IndexName)
	}
}

func (s *session) checkAddIndex(t *TableInfo, ct *ast.Constraint) {
	log.Info("checkAddIndex")

	log.Infof("%#v \n", ct)

	for _, col := range ct.Keys {
		found := false
		for _, field := range t.Fields {
			if strings.EqualFold(field.Field, col.Column.Name.O) {
				found = true
				break
			}
		}
		if !found {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, col.Column.Name.O))
		}
	}

}

func (s *session) checkAddConstraint(t *TableInfo, c *ast.AlterTableSpec) {
	log.Info("checkAddConstraint")

	// log.Infof("%s \n", c.Constraint)
	// log.Infof("%s \n", c.Constraint.Keys)

	switch c.Constraint.Tp {
	case ast.ConstraintIndex:
		// s.checkAddIndex(t, c.Constraint)
		s.checkCreateIndex(nil, c.Constraint.Name,
			c.Constraint.Keys, c.Constraint.Option, t, false)
	case ast.ConstraintUniq:
		s.checkCreateIndex(nil, c.Constraint.Name,
			c.Constraint.Keys, c.Constraint.Option, t, true)
	// case ast.ConstraintForeignKey:
	// 	s.checkDropColumn(table, alter)
	// case ast.AlterTableAddConstraint:
	// 	s.checkAddConstraint(table, alter)
	default:
		s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
		log.Info("未定义的解析: ", c.Constraint.Tp)
	}

	// ConstraintNoConstraint ConstraintType = iota
	// ConstraintPrimaryKey
	// ConstraintKey
	// ConstraintIndex
	// ConstraintUniq
	// ConstraintUniqKey
	// ConstraintUniqIndex
	// ConstraintForeignKey
	// ConstraintFulltext

	// Tp   ConstraintType
	// Name string

	// Keys []*IndexColName // Used for PRIMARY KEY, UNIQUE, ......

	// Refer *ReferenceDef // Used for foreign key.

	// Option *IndexOption // Index Options

	// for _, nc := range c.NewColumns {
	// 	log.Infof("%s \n", nc)
	// 	found := false
	// 	for _, field := range t.Fields {
	// 		if strings.EqualFold(field.Field, nc.Name.Name.O) {
	// 			found = true
	// 			break
	// 		}
	// 	}
	// 	if found {
	// 		s.AppendErrorNo(ER_COLUMN_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name))
	// 	}
	// }

	// if c.Position.Tp != ast.ColumnPositionNone {
	// 	found := false
	// 	for _, field := range t.Fields {
	// 		if strings.EqualFold(field.Field, c.Position.RelativeColumn.Name.O) {
	// 			found = true
	// 			break
	// 		}
	// 	}
	// 	if !found {
	// 		s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
	// 			fmt.Sprintf("%s.%s", t.Name, c.Position.RelativeColumn.Name))
	// 	}
	// }
}

func (s *session) checkDBExists(db string, reportNotExists bool) bool {

	if db == "" {
		db = s.DBName
	}

	if _, ok := s.dbCacheList[db]; ok {
		return true
	}

	sql := "show databases like '%s';"

	// count:= s.db.Exec(fmt.Sprintf(sql,db)).AffectedRows
	var name string

	rows, err := s.db.Raw(fmt.Sprintf(sql, db)).Rows()
	defer rows.Close()
	if err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else {
		for rows.Next() {
			rows.Scan(&name)
		}
	}

	if name == "" {
		if reportNotExists {
			s.AppendErrorNo(ER_DB_NOT_EXISTED_ERROR, db)
		}
		return false
	} else {
		s.dbCacheList[db] = true
		return true
	}

}

func (s *session) checkInsert(node *ast.InsertStmt, sql string) {

	log.Info("checkInsert")
	// IsReplace   bool
	// IgnoreErr   bool
	// Table       *TableRefsClause
	// Columns     []*ColumnName
	// Lists       [][]ExprNode
	// Setlist     []*Assignment
	// Priority    mysql.PriorityEnum
	// OnDuplicate []*Assignment
	// Select      ResultSetNode
	x := node

	fieldCount := len(x.Columns)

	if fieldCount == 0 {
		s.AppendErrorNo(ER_WITH_INSERT_FIELD)
	}

	t := getSingleTableName(x.Table)

	for _, c := range x.Columns {
		if c.Schema.O == "" {
			c.Schema = model.NewCIStr(s.DBName)
		}
		if c.Table.O == "" {
			c.Table = model.NewCIStr(t.Name.O)
		}
	}

	// if t.Schema.O != "" {
	// 	s.myRecord.DBName = t.Schema.O
	// } else {
	// 	s.myRecord.DBName = s.DBName
	// }

	table := s.getTableFromCache(t.Schema.O, t.Name.O, true)
	// log.Infof("%#v", table)
	s.myRecord.TableInfo = table

	s.checkFieldsValid(x.Columns, table)

	if len(x.Lists) > 0 {
		if fieldCount == 0 {
			fieldCount = len(x.Lists[0])
		}
		for i, list := range x.Lists {
			if len(list) != fieldCount {
				s.AppendErrorNo(ER_WRONG_VALUE_COUNT_ON_ROW, i+1)
			}
		}

		s.myRecord.AffectedRows = len(x.Lists)
	}

	// insert select 语句
	if x.Select != nil {
		if sel, ok := x.Select.(*ast.SelectStmt); ok {
			// log.Info(sel.Text())

			// log.Info(len(sel.Fields.Fields), sel.Fields)

			checkWildCard := false
			if len(sel.Fields.Fields) > 0 {
				f := sel.Fields.Fields[0]
				if f.WildCard != nil {
					checkWildCard = true
				}
			}

			// 判断字段数是否匹配, *星号时跳过
			if fieldCount > 0 && !checkWildCard && len(sel.Fields.Fields) != fieldCount {
				s.AppendErrorNo(ER_WRONG_VALUE_COUNT_ON_ROW, 1)
			}

			if len(sel.Fields.Fields) > 0 {
				for _, f := range sel.Fields.Fields {
					// log.Info("--------")

					// log.Info(fmt.Sprintf("%T", f.Expr))

					// if c, ok := f.Expr.(*ast.ColumnNameExpr); ok {
					// 	// log.Info(c.Name)
					// }
					if f.Expr == nil {
						log.Info(node.Text())
						log.Info("Expr is NULL", f.WildCard, f.Expr, f.AsName)
						log.Info(f.Text())
						log.Info("是个*号")
					}
				}
			}

			i := strings.Index(strings.ToLower(sql), "select")
			selectSql := sql[i:]
			var explain []string

			explain = append(explain, "EXPLAIN ")
			explain = append(explain, selectSql)

			// log.Info(explain)

			rows := s.getExplainInfo(strings.Join(explain, ""))

			s.myRecord.AnlyzeExplain(rows)

		}
	}

	// log.Info(len(node.Setlist), "----------------")
	if len(node.Setlist) > 0 {
		// Process `set` type column.
		// columns := make([]string, 0, len(node.Setlist))
		for _, v := range node.Setlist {
			log.Info(v.Column)
			// columns = append(columns, v.Column)
		}

		// cols, err = table.FindCols(tableCols, columns, node.Table.Meta().PKIsHandle)
		// if err != nil {
		// 	return nil, errors.Errorf("INSERT INTO %s: %s", node.Table.Meta().Name.O, err)
		// }
		// if len(cols) == 0 {
		// 	return nil, errors.Errorf("INSERT INTO %s: empty column", node.Table.Meta().Name.O)
		// }
	}
}

func (s *session) checkDropDB(node *ast.DropDatabaseStmt) {

	log.Info("checkDropDB")

	log.Infof("%#v \n", node)

	s.AppendErrorMessage(fmt.Sprintf("命令禁止! 无法删除数据库'%s'.", node.Name))
}

func (s *session) checkCreateDB(node *ast.CreateDatabaseStmt) {

	log.Info("checkCreateDB")

	// log.Infof("%#v \n", node)

	if s.checkDBExists(node.Name, false) {
		s.AppendErrorMessage(fmt.Sprintf("数据库'%s'已存在.", node.Name))
	}
}

func (s *session) checkChangeDB(node *ast.UseStmt) {

	log.Info("checkChangeDB", node.DBName)

	s.DBName = node.DBName
	if s.checkDBExists(node.DBName, true) {
		s.db.Exec(fmt.Sprintf("USE `%s`", node.DBName))
	}
}

func getSingleTableName(tableRefs *ast.TableRefsClause) *ast.TableName {
	if tableRefs == nil || tableRefs.TableRefs == nil || tableRefs.TableRefs.Right != nil {
		return nil
	}
	tblSrc, ok := tableRefs.TableRefs.Left.(*ast.TableSource)
	if !ok {
		return nil
	}
	if tblSrc.AsName.L != "" {
		return nil
	}
	tblName, ok := tblSrc.Source.(*ast.TableName)
	if !ok {
		return nil
	}
	return tblName
}

func (s *session) getExplainInfo(sql string) []ExplainInfo {
	var rows []ExplainInfo
	if err := s.db.Raw(sql).Scan(&rows).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	}

	return rows
}

func (s *session) explainOrAnalyzeSql(sql string) {

	var explain []string

	explain = append(explain, "EXPLAIN ")
	explain = append(explain, sql)

	log.Info(explain)

	rows := s.getExplainInfo(strings.Join(explain, ""))

	s.myRecord.AnlyzeExplain(rows)

}

func (s *session) checkUpdate(node *ast.UpdateStmt, sql string) {

	log.Info("checkUpdate")

	log.Infof("%#v", node)
	log.Infof("%#v", node.TableRefs)
	log.Infof("%#v", node.List)

	// 从set列表读取要更新的表
	var originTable string
	var firstColumnName string
	if node.List != nil {
		for _, l := range node.List {
			log.Infof("%#v", l.Column)
			originTable = l.Column.Table.L
			firstColumnName = l.Column.Name.L
		}
	}

	// log.Infof("%s \n", node.TableRefs)
	// log.Infof("%s \n", node.List)
	// TableRefs     *TableRefsClause
	// List          []*Assignment
	// Where         ExprNode
	// Order         *OrderByClause
	// Limit         *Limit
	// Priority      mysql.PriorityEnum
	// IgnoreErr     bool
	// MultipleTable bool
	// TableHints    []*TableOptimizerHint

	var tableList []*ast.TableName
	tableList = extractTableList(node.TableRefs.TableRefs, tableList)

	catchError := false
	for _, tblName := range tableList {
		log.Info(tblName.Schema)
		log.Info(tblName.Name)

		t := s.getTableFromCache(tblName.Schema.O, tblName.Name.O, true)

		if t == nil {
			catchError = true
		} else if s.myRecord.TableInfo == nil {

			// 如果set没有指定列名,则需要根据列名遍历所有访问表的列,看操作的表是哪一个
			if originTable == "" {
				for _, field := range t.Fields {
					if strings.EqualFold(field.Field, firstColumnName) {
						s.myRecord.TableInfo = t
						break
					}
				}
			} else {
				if originTable == tblName.Name.L {
					s.myRecord.TableInfo = t
				}
			}
		}
	}

	if !catchError {
		s.explainOrAnalyzeSql(sql)
	}
}

func (s *session) checkDelete(node *ast.DeleteStmt, sql string) {

	log.Info("checkDelete")

	log.Infof("%#v", node)

	log.Infof("%#v", node.TableRefs.TableRefs)

	if node.Tables != nil {
		for _, a := range node.Tables.Tables {
			s.myRecord.TableInfo = s.getTableFromCache(a.Schema.O, a.Name.O, true)
		}
	}

	var tableList []*ast.TableName
	tableList = extractTableList(node.TableRefs.TableRefs, tableList)

	for _, tblName := range tableList {
		t := s.getTableFromCache(tblName.Schema.O, tblName.Name.O, true)
		if s.myRecord.TableInfo == nil {
			s.myRecord.TableInfo = t
		}
	}

	s.explainOrAnalyzeSql(sql)

	// if node.TableRefs != nil {
	// 	a := node.TableRefs.TableRefs
	// 	// log.Info(a)
	// 	log.Infof("%T,%s  \n", a, a)
	// 	log.Info("===================")
	// 	log.Infof("%T , %s  \n", a.Left, a.Left)
	// 	log.Infof("%T , %s  \n", a.Right, a.Right)
	// 	if a.Left != nil {
	// 		if tblSrc, ok := a.Left.(*ast.Join); ok {
	// 			log.Info("---left")
	// 			// log.Info(tblSrc)
	// 			// log.Info(tblSrc.AsName)
	// 			// log.Info(tblSrc.Source)
	// 			// log.Info(fmt.Sprintf("%T", tblSrc.Source))

	// 			if tblName, ok := tblSrc.Source.(*ast.TableName); ok {
	// 				log.Info(tblName.Schema)
	// 				log.Info(tblName.Name)

	// 				s.QueryTableFromDB(tblName.Schema.O, tblName.Name.O, true)
	// 			}
	// 		}

	// 		if tblSrc, ok := a.Left.(*ast.TableSource); ok {
	// 			log.Info("---left")
	// 			// log.Info(tblSrc)
	// 			// log.Info(tblSrc.AsName)
	// 			// log.Info(tblSrc.Source)
	// 			// log.Info(fmt.Sprintf("%T", tblSrc.Source))

	// 			if tblName, ok := tblSrc.Source.(*ast.TableName); ok {
	// 				log.Info(tblName.Schema)
	// 				log.Info(tblName.Name)

	// 				s.QueryTableFromDB(tblName.Schema.O, tblName.Name.O, true)
	// 			}
	// 		}
	// 	}
	// 	if a.Right != nil {
	// 		if tblSrc, ok := a.Right.(*ast.TableSource); ok {
	// 			log.Info("+++right")
	// 			// log.Info(tblSrc)
	// 			// log.Info(tblSrc.AsName)
	// 			// log.Info(tblSrc.Source)
	// 			// log.Info(fmt.Sprintf("%T", tblSrc.Source))

	// 			if tblName, ok := tblSrc.Source.(*ast.TableName); ok {
	// 				log.Info(tblName.Schema)
	// 				log.Info(tblName.Name)

	// 				s.QueryTableFromDB(tblName.Schema.O, tblName.Name.O, true)
	// 			}
	// 		}
	// 	}
	// 	// log.Info(a.Left, a.Left.Text())
	// 	// log.Info(fmt.Sprintf("%T", a.Left))
	// 	// log.Info(a.Right)
	// }
}

func (s *session) checkFieldsValid(columns []*ast.ColumnName, table *TableInfo) {

	if table == nil {
		return
	}
	for _, c := range columns {
		found := false
		for _, field := range table.Fields {
			if strings.EqualFold(field.Field, c.Name.O) {
				found = true
				break
			}
		}
		if !found {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", c.Table, c.Name))
		}
	}

}

func (s *session) QueryTableFromDB(db string, tableName string, reportNotExists bool) []FieldInfo {
	if db == "" {
		db = s.DBName
	}
	sql := fmt.Sprintf("SHOW FULL FIELDS FROM `%s`.`%s`", db, tableName)

	var rows []FieldInfo
	// log.Info("++++++++")
	if err := s.db.Raw(sql).Scan(&rows).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			if myErr.Number != 1146 || reportNotExists {
				s.AppendErrorMessage(myErr.Message + ".")
			}
		} else {
			s.AppendErrorMessage(err.Error() + ".")
		}

		return nil
	}
	// errs := a.GetErrors()

	return rows
}

func (r *Record) AppendErrorMessage(msg string) {
	r.ErrLevel = 2

	r.Buf.WriteString(msg)
	r.Buf.WriteString("\n")
}

func (r *Record) AppendErrorNo(number int, values ...interface{}) {
	r.ErrLevel = uint8(Max(int(r.ErrLevel), int(GetErrorLevel(number))))

	// if number == ER_CANT_DROP_TABLE {
	// 	log.Info("ER_CANT_DROP_TABLE", GetErrorLevel(number), r.ErrLevel)
	// }
	if len(values) == 0 {
		r.Buf.WriteString(GetErrorMessage(number))
	} else {
		r.Buf.WriteString(fmt.Sprintf(GetErrorMessage(number), values...))
	}
	r.Buf.WriteString("\n")
}

func (s *session) AppendErrorMessage(msg string) {
	s.recordSets.MaxLevel = 2
	s.myRecord.AppendErrorMessage(msg)
}

func (s *session) AppendErrorNo(number int, values ...interface{}) {
	s.myRecord.AppendErrorNo(number, values...)
	s.recordSets.MaxLevel = uint8(Max(int(s.recordSets.MaxLevel), int(s.myRecord.ErrLevel)))
}

func extractTableList(node ast.ResultSetNode, input []*ast.TableName) []*ast.TableName {
	switch x := node.(type) {
	case *ast.Join:
		input = extractTableList(x.Left, input)
		input = extractTableList(x.Right, input)
	case *ast.TableSource:
		if s, ok := x.Source.(*ast.TableName); ok {
			if x.AsName.L != "" {
				newTableName := *s
				newTableName.Name = x.AsName
				s.Name = x.AsName
				input = append(input, &newTableName)
			} else {
				input = append(input, s)
			}
		}
	}
	return input
}

func (s *session) getTableFromCache(db string, tableName string, reportNotExists bool) *TableInfo {
	if db == "" {
		db = s.DBName
	}
	key := fmt.Sprintf("%s.%s", db, tableName)

	if t, ok := s.tableCacheList[key]; ok {
		return t
	} else {
		rows := s.QueryTableFromDB(db, tableName, reportNotExists)

		if rows != nil {
			newT := &TableInfo{
				Schema: db,
				Name:   tableName,
				Fields: rows,
			}

			s.tableCacheList[key] = newT

			return newT
		}
	}

	return nil
}

func (s *session) addTableCache(t *TableInfo) {
	if t.Schema == "" {
		t.Schema = s.DBName
	}
	key := fmt.Sprintf("%s.%s", t.Schema, t.Name)

	if t, ok := s.tableCacheList[key]; !ok {
		s.tableCacheList[key] = t
	}
}

func (s *session) CacheNewTable(dbname string, tablename string) {

}

func FieldLengthWithType(tp string) int {

	var p string
	if strings.Index(tp, "(") > -1 {
		p = tp[0:strings.Index(tp, "(")]
	} else {
		p = tp
	}

	var l int

	firsts := regFieldLength.FindStringSubmatch(tp)
	if len(firsts) < 2 {
		l = 0
	} else {
		l, _ = strconv.Atoi(firsts[1])
	}

	switch p {
	case "bit", "tinyint", "bool", "year":
		l = 1
	case "small":
		l = 2
	case "date", "int", "integer", "timestamp", "time":
		l = 4
	case "bigint", "datetime":
		l = 8
	}

	return l
}

func Max(x, y int) int {
	if x >= y {
		return x
	}
	return y
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func copyTableInfo(t *TableInfo) *TableInfo {
	p := &TableInfo{}

	p.Schema = t.Schema
	p.Name = t.Name
	p.Fields = make([]FieldInfo, len(t.Fields))
	copy(p.Fields, t.Fields)

	return p
}

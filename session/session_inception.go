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
	"database/sql/driver"
	"fmt"
	// "io"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/executor"
	"github.com/hanchuanchuan/goInception/expression"
	"github.com/hanchuanchuan/goInception/metrics"
	"github.com/hanchuanchuan/goInception/model"
	"github.com/hanchuanchuan/goInception/mysql"
	"github.com/hanchuanchuan/goInception/parser/opcode"
	"github.com/hanchuanchuan/goInception/types"
	"github.com/hanchuanchuan/goInception/util"
	"github.com/hanchuanchuan/goInception/util/auth"
	"github.com/hanchuanchuan/goInception/util/stringutil"
	"github.com/jinzhu/gorm"
	"github.com/pingcap/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

// MasterStatus 主库状态信息,包括当前日志文件,位置等
type MasterStatus struct {
	gorm.Model
	File            string `gorm:"Column:File"`
	Position        int    `gorm:"Column:Position"`
	BinlogDoDB      string `gorm:"Column:Binlog_Do_DB"`
	BinlogIgnoreDB  string `gorm:"Column:Binlog_Ignore_DB"`
	ExecutedGtidSet string `gorm:"Column:Executed_Gtid_Set"`
}

// sourceOptions 线上数据库信息和审核或执行的参数
type sourceOptions struct {
	host           string
	port           int
	user           string
	password       string
	check          bool
	execute        bool
	backup         bool
	ignoreWarnings bool

	// 仅供第三方扩展使用! 设置该字符串会跳过binlog解析!
	middlewareExtend string
	middlewareDB     string
}

// ExplainInfo 执行计划信息
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

// FieldInfo 字段信息
type FieldInfo struct {
	gorm.Model

	Field      string  `gorm:"Column:Field"`
	Type       string  `gorm:"Column:Type"`
	Collation  string  `gorm:"Column:Collation"`
	Null       string  `gorm:"Column:Null"`
	Key        string  `gorm:"Column:Key"`
	Default    *string `gorm:"Column:Default"`
	Extra      string  `gorm:"Column:Extra"`
	Privileges string  `gorm:"Column:Privileges"`
	Comment    string  `gorm:"Column:Comment"`

	IsDeleted bool `gorm:"-"`
	IsNew     bool `gorm:"-"`

	Tp *types.FieldType `gorm:"-"`
}

// TableInfo 表结构.
// 表结构实现了快照功能,在表结构变更前,会复制快照,在快照上做变更
// 在解析binlog时,基于执行时的快照做binlog解析,以实现删除列时的binlog解析
type TableInfo struct {
	Schema string
	Name   string
	// 表别名,仅用于update,delete多表
	AsName string
	Fields []FieldInfo

	// 索引
	Indexes []*IndexInfo

	// 是否已删除
	IsDeleted bool
	// 备份库是否已创建
	IsCreated bool

	// 表是否为新增
	IsNew bool
	// 列是否为新增
	IsNewColumns bool

	// 主键信息,用以备份
	hasPrimary bool
	primarys   map[int]bool

	AlterCount int

	// 是否已清除已删除的列[解析binlog时会自动清除已删除的列]
	IsClear bool

	// 表大小.单位MB
	TableSize uint
}

// IndexInfo 索引信息
type IndexInfo struct {
	gorm.Model

	Table      string `gorm:"Column:Table"`
	NonUnique  int    `gorm:"Column:Non_unique"`
	IndexName  string `gorm:"Column:Key_name"`
	Seq        int    `gorm:"Column:Seq_in_index"`
	ColumnName string `gorm:"Column:Column_name"`
	IndexType  string `gorm:"Column:Index_type"`

	IsDeleted bool `gorm:"-"`
}

var (
	regParseOption *regexp.Regexp
	regFieldLength *regexp.Regexp
	regIdentified  *regexp.Regexp
)

// var Keywords map[string]int = parser.GetKeywords()

const (
	MaxKeyLength   = 767
	MaxKeyLength57 = 3072

	RemoteBackupTable              = "$_$Inception_backup_information$_$"
	TABLE_COMMENT_MAXLEN           = 2048
	COLUMN_COMMENT_MAXLEN          = 1024
	INDEX_COMMENT_MAXLEN           = 1024
	TABLE_PARTITION_COMMENT_MAXLEN = 1024
)

func init() {

	// 匹配sql的option设置
	regParseOption = regexp.MustCompile(`^\/\*(.*?)\*\/`)

	// 匹配字段长度
	regFieldLength = regexp.MustCompile(`^.*?\((\d)`)

	// 匹配标识符,只能包含字母数字和下划线
	regIdentified = regexp.MustCompile(`^[0-9a-zA-Z\_]*$`)

}

func (s *session) ExecuteInc(ctx context.Context, sql string) (recordSets []ast.RecordSet, err error) {

	// 跳过mysql客户端发送的sql
	// 跳过tidb测试时发送的sql
	if sql == "select @@version_comment limit 1" || sql == "SELECT @@max_allowed_packet" {
		return s.execute(ctx, sql)
	} else if sql == "SET AUTOCOMMIT = 0" {
		return s.execute(ctx, sql)
	} else if sql == "show warnings" {
		return s.execute(ctx, sql)
	} else if strings.HasPrefix(sql, "select HIGH_PRIORITY") {
		return s.execute(ctx, sql)
	} else if strings.HasPrefix(sql,
		`select variable_value from mysql.tidb where variable_name = "system_tz"`) {
		return s.execute(ctx, sql)
	} else if strings.HasPrefix(sql, "SELECT HIGH_PRIORITY") {
		return s.execute(ctx, sql)
	}

	s.DBName = ""
	s.haveBegin = false
	s.haveCommit = false

	s.tableCacheList = make(map[string]*TableInfo)
	s.dbCacheList = make(map[string]bool)

	s.backupDBCacheList = make(map[string]bool)
	s.backupTableCacheList = make(map[string]bool)

	s.Inc = config.GetGlobalConfig().Inc
	s.Osc = config.GetGlobalConfig().Osc
	s.Ghost = config.GetGlobalConfig().Ghost

	s.recordSets = NewRecordSets()

	if recordSets, err = s.executeInc(ctx, sql); err != nil {
		err = errors.Trace(err)
		s.sessionVars.StmtCtx.AppendError(err)
	}
	// else {
	// 	if s.sessionVars.StmtCtx.AffectedRows() == 0 {
	// 		log.Info(s.recordSets.rc.count)
	// 		s.sessionVars.StmtCtx.AddAffectedRows(uint64(s.recordSets.rc.count))
	// 	}
	// }
	// else {
	// 	fmt.Println("---------------")
	// 	fmt.Println(len(recordSets))
	// 	fmt.Printf("%#v", recordSets)
	// 	s.sessionVars.StmtCtx.AddAffectedRows(uint64(len(recordSets)))
	// }
	return
}

func (s *session) executeInc(ctx context.Context, sql string) (recordSets []ast.RecordSet, err error) {

	sqlList := strings.Split(sql, "\n")

	defer func() {
		if s.sessionVars.StmtCtx.AffectedRows() == 0 {
			s.sessionVars.StmtCtx.AddAffectedRows(uint64(s.recordSets.rc.count))
		}
	}()

	defer logQuery(sql, s.sessionVars)

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
			s1 = strings.TrimRight(s1, ";")
			stmtNodes, err := s.ParseSQL(ctx, s1, charsetInfo, collation)

			if err != nil {
				log.Error(fmt.Sprintf("解析失败! %s", err))
				log.Error(s1)
				// 移除config配置信息/*user=...*/
				if !s.haveBegin && strings.Contains(s1, "*/") {
					s1 = s1[strings.Index(s1, "*/")+2:]
				}
				s.recordSets.Append(&Record{
					Sql:          strings.TrimSpace(s1),
					ErrLevel:     2,
					ErrorMessage: err.Error(),
				})
				return s.recordSets.Rows(), nil

				s.rollbackOnError(ctx)
				log.Warnf("con:%d parse error:\n%v\n%s", connID, err, s1)
				return nil, errors.Trace(err)
			}

			tmp := s.processInfo.Load()
			if tmp != nil {
				pi := tmp.(util.ProcessInfo)
				pi.OperState = "CHECKING"
				pi.Percent = 0
				s.processInfo.Store(pi)
			}

			for i, stmtNode := range stmtNodes {

				currentSql := strings.TrimSpace(stmtNode.Text())

				s.myRecord = &Record{
					Sql:   currentSql,
					Buf:   new(bytes.Buffer),
					Type:  stmtNode,
					Stage: StageCheck,
				}

				switch stmtNode.(type) {
				case *ast.InceptionStartStmt:
					if s.haveBegin {
						s.AppendErrorNo(ER_HAVE_BEGIN)
						s.myRecord.Sql = ""
						s.recordSets.Append(s.myRecord)

						log.Error(sql)
						return s.recordSets.Rows(), nil
					}

					// 操作前重设上下文
					if err := executor.ResetContextOfStmt(s, stmtNode); err != nil {
						return nil, errors.Trace(err)
					}

					s.haveBegin = true
					s.parseOptions(currentSql)

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

					s.mysqlServerVersion()

					continue
				case *ast.InceptionCommitStmt:

					if !s.haveBegin {
						s.AppendErrorMessage("Must start as begin statement.")
						s.recordSets.Append(s.myRecord)
						return s.recordSets.Rows(), nil
					}

					s.haveCommit = true
					s.executeCommit(ctx)
					return s.recordSets.Rows(), nil
				default:
					need := s.needDataSource(stmtNode)

					if !s.haveBegin && need {
						log.Warnf("%#v", stmtNode)
						s.AppendErrorMessage("Must start as begin statement.")
						s.recordSets.Append(s.myRecord)
						return s.recordSets.Rows(), nil
					}

					s.SetMyProcessInfo(currentSql, time.Now(), float64(i)/float64(lineCount+1))

					// 交互式命令行
					if !need {
						if s.opt != nil {
							// log.Error(s.opt)
							return nil, errors.New("无效操作!不支持本地操作和远程操作混用!")
						}

						// 操作前重设上下文
						if err := executor.ResetContextOfStmt(s, stmtNode); err != nil {
							return nil, errors.Trace(err)
						}

						return s.processCommand(ctx, stmtNode)
						// if err != nil {
						// 	return nil, err
						// }
						// if result != nil {
						// 	return result, nil
						// }
					} else {
						result, err := s.processCommand(ctx, stmtNode)
						if err != nil {
							return nil, err
						}
						if result != nil {
							return result, nil
						}
					}

					// 进程Killed
					if err := checkClose(ctx); err != nil {
						log.Warn("Killed: ", err)
						s.AppendErrorMessage("Operation has been killed!")
						s.recordSets.Append(s.myRecord)
						return s.recordSets.Rows(), nil
					}
				}

				if !s.haveBegin && s.needDataSource(stmtNode) {
					log.Warnf("%#v", stmtNode)
					s.AppendErrorMessage("Must start as begin statement.")
					s.recordSets.Append(s.myRecord)
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

func (s *session) needDataSource(stmtNode ast.StmtNode) bool {
	switch node := stmtNode.(type) {
	case *ast.ShowStmt:
		if node.IsInception {
			return false
		}
	case *ast.InceptionSetStmt, *ast.ShowOscStmt, *ast.KillStmt:
		return false
	}

	return true
}

func (s *session) processCommand(ctx context.Context, stmtNode ast.StmtNode) ([]ast.RecordSet, error) {
	log.Debug("processCommand")

	currentSql := strings.TrimSpace(stmtNode.Text())

	switch node := stmtNode.(type) {
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
		s.checkRenameTable(node, currentSql)
	case *ast.TruncateTableStmt:
		s.checkTruncateTable(node, currentSql)

	case *ast.CreateIndexStmt:
		s.checkCreateIndex(node.Table, node.IndexName,
			node.IndexColNames, node.IndexOption, nil, node.Unique, ast.ConstraintIndex)

	case *ast.DropIndexStmt:
		s.checkDropIndex(node, currentSql)

	case *ast.CreateViewStmt:
		s.AppendErrorMessage(fmt.Sprintf("命令禁止! 无法创建视图'%s'.", node.ViewName.Name))

	case *ast.ShowStmt:
		if node.IsInception {
			switch node.Tp {
			case ast.ShowVariables:
				return s.executeLocalShowVariables(node)
			case ast.ShowProcessList:
				return s.executeLocalShowProcesslist(node)
			default:
				log.Infof("%#v", node)
				return nil, errors.New("不支持的语法类型")
			}
		} else {
			s.executeInceptionShow(currentSql)
		}

	case *ast.InceptionSetStmt:
		return s.executeInceptionSet(node, currentSql)

	case *ast.ExplainStmt:
		s.executeInceptionShow(currentSql)

	case *ast.ShowOscStmt:
		switch node.Tp {
		case ast.OscOptionKill:
			return s.executeLocalOscKill(node)
		case ast.OscOptionPause:
			return s.executeLocalOscPause(node)
		case ast.OscOptionResume:
			return s.executeLocalOscResume(node)
		default:
			return s.executeLocalShowOscProcesslist(node)
		}

	case *ast.KillStmt:
		return s.executeKillStmt(node)
	default:
		log.Info("无匹配类型...")
		log.Infof("%T\n", stmtNode)
		s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
	}

	s.mysqlComputeSqlSha1(s.myRecord)

	return nil, nil
}

func (s *session) executeCommit(ctx context.Context) {
	if s.opt.check {
		return
	}

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

	s.executeAllStatement(ctx)

	// 只要有执行成功的,就添加备份
	// if s.recordSets.MaxLevel == 2 ||
	// 	(s.recordSets.MaxLevel == 1 && !s.opt.ignoreWarnings) {
	// 	return
	// }

	if s.opt.backup {

		// 如果连接已断开
		if err := s.backupdb.DB().Ping(); err != nil {
			log.Error(err)
			addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local",
				s.Inc.BackupUser, s.Inc.BackupPassword, s.Inc.BackupHost, s.Inc.BackupPort)
			db, err := gorm.Open("mysql", addr)
			if err != nil {
				log.Error(err)
				s.AppendErrorMessage(err.Error())
				return
			}
			// 禁用日志记录器，不显示任何日志
			db.LogMode(false)
			s.backupdb = db
		}

		log.Debug("开始备份")

		tmp := s.processInfo.Load()
		if tmp != nil {
			pi := tmp.(util.ProcessInfo)
			pi.OperState = "BACKUP"
			pi.Percent = 0
			s.processInfo.Store(pi)
		}

		s.runBackup(ctx)

		// for _, record := range s.recordSets.All() {

		// 	if s.checkSqlIsDML(record) || s.checkSqlIsDDL(record) {
		// 		s.myRecord = record

		// 		errno := s.mysqlCreateBackupTable(record)
		// 		if errno == 2 {
		// 			break
		// 		}
		// 		if record.TableInfo == nil {
		// 			s.AppendErrorNo(ErrNotFoundTableInfo)
		// 		} else {
		// 			s.mysqlBackupSql(record)
		// 		}

		// 		if s.hasError() {
		// 			break
		// 		}
		// 	}
		// }

		if !s.isMiddleware() {
			// 解析binlog生成回滚语句
			s.Parser(ctx)
		}
	}
}

func (s *session) mysqlBackupSql(record *Record) {
	if s.checkSqlIsDDL(record) {
		s.mysqlExecuteBackupInfoInsertSql(record)

		if s.isMiddleware() {
			s.mysqlExecuteBackupSqlForDDL(record)
		}
	} else if s.checkSqlIsDML(record) {
		s.mysqlExecuteBackupInfoInsertSql(record)
	}
}

func makeOPIDByTime(execTime int64, threadId uint32, seqNo int) string {
	return fmt.Sprintf("%d_%d_%08d", execTime, threadId, seqNo)
}

func (s *session) mysqlExecuteBackupSqlForDDL(record *Record) {
	if record.DDLRollback == "" {
		return
	}

	var buf strings.Builder
	buf.WriteString("INSERT INTO ")
	dbname := s.getRemoteBackupDBName(record)
	buf.WriteString(fmt.Sprintf("`%s`.`%s`", dbname, record.TableInfo.Name))
	buf.WriteString("(rollback_statement, opid_time) VALUES('")
	buf.WriteString(HTMLEscapeString(record.DDLRollback))
	buf.WriteString("','")
	buf.WriteString(record.OPID)
	buf.WriteString("')")

	sql := buf.String()

	if err := s.backupdb.Exec(sql).Error; err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
			record.StageStatus = StatusBackupFail
		}
	}
	record.StageStatus = StatusBackupOK
}

func (s *session) mysqlExecuteBackupInfoInsertSql(record *Record) int {

	record.OPID = makeOPIDByTime(record.ExecTimestamp, record.ThreadId, record.SeqNo)

	typeStr := "UNKNOWN"
	switch record.Type.(type) {
	case *ast.InsertStmt:
		typeStr = "INSERT"
	case *ast.DeleteStmt:
		typeStr = "DELETE"
	case *ast.UpdateStmt:
		typeStr = "UPDATE"
	case *ast.CreateDatabaseStmt:
		typeStr = "CREATEDB"
	case *ast.CreateTableStmt:
		typeStr = "CREATETABLE"
	case *ast.AlterTableStmt:
		typeStr = "ALTERTABLE"
	case *ast.DropTableStmt:
		typeStr = "DROPTABLE"
	case *ast.RenameTableStmt:
		typeStr = "RENAMETABLE"
	case *ast.CreateIndexStmt:
		typeStr = "CREATEINDEX"
	case *ast.DropIndexStmt:
		typeStr = "DROPINDEX"
	default:
		log.Warning("类型未知: ", record.Type)
	}

	values := []interface{}{
		record.OPID,
		record.StartFile,
		strconv.Itoa(record.StartPosition),
		record.EndFile,
		strconv.Itoa(record.EndPosition),
		HTMLEscapeString(record.Sql),
		s.opt.host,
		record.TableInfo.Schema,
		record.TableInfo.Name,
		strconv.Itoa(s.opt.port),
		typeStr,
	}

	dbName := s.getRemoteBackupDBName(record)

	if s.lastBackupTable == "" {
		s.lastBackupTable = dbName
	}
	// 库名改变时强制flush
	if s.lastBackupTable != dbName {
		s.chBackupRecord <- &chanBackup{
			dbname: dbName,
			record: record,
			values: nil,
		}
		s.lastBackupTable = dbName
	}

	s.chBackupRecord <- &chanBackup{
		dbname: dbName,
		record: record,
		values: values,
	}

	// s.lastBackupTable = lastBackupTable

	// var buf strings.Builder

	// buf.WriteString("INSERT INTO ")
	// dbname := s.getRemoteBackupDBName(record)
	// buf.WriteString(fmt.Sprintf("`%s`.`%s`", dbname, RemoteBackupTable))
	// buf.WriteString(" VALUES('")
	// buf.WriteString(record.OPID)
	// buf.WriteString("','")
	// buf.WriteString(record.StartFile)
	// buf.WriteString("',")
	// buf.WriteString(strconv.Itoa(record.StartPosition))
	// buf.WriteString(",'")
	// buf.WriteString(record.EndFile)
	// buf.WriteString("',")
	// buf.WriteString(strconv.Itoa(record.EndPosition))
	// // buf.WriteString(",?,'")
	// buf.WriteString(",'")
	// buf.WriteString(HTMLEscapeString(record.Sql))
	// buf.WriteString("','")
	// buf.WriteString(s.opt.host)
	// buf.WriteString("','")
	// buf.WriteString(record.TableInfo.Schema)
	// buf.WriteString("','")
	// buf.WriteString(record.TableInfo.Name)
	// buf.WriteString("',")
	// buf.WriteString(strconv.Itoa(s.opt.port))
	// buf.WriteString(",NOW(),'")

	// switch record.Type.(type) {
	// case *ast.InsertStmt:
	// 	buf.WriteString("INSERT")
	// case *ast.DeleteStmt:
	// 	buf.WriteString("DELETE")
	// case *ast.UpdateStmt:
	// 	buf.WriteString("UPDATE")
	// case *ast.CreateDatabaseStmt:
	// 	buf.WriteString("CREATEDB")
	// case *ast.CreateTableStmt:
	// 	buf.WriteString("CREATETABLE")
	// case *ast.AlterTableStmt:
	// 	buf.WriteString("ALTERTABLE")
	// case *ast.DropTableStmt:
	// 	buf.WriteString("DROPTABLE")
	// case *ast.RenameTableStmt:
	// 	buf.WriteString("RENAMETABLE")
	// case *ast.CreateIndexStmt:
	// 	buf.WriteString("CREATEINDEX")
	// case *ast.DropIndexStmt:
	// 	buf.WriteString("DROPINDEX")
	// default:
	// 	buf.WriteString("UNKNOWN")
	// }

	// buf.WriteString("')")

	// if err := s.backupdb.Exec(buf.String()).Error; err != nil {
	// 	log.Error(err)
	// 	if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
	// 		s.AppendErrorMessage(myErr.Message)

	// 		record.StageStatus = StatusBackupFail
	// 		return 2
	// 	}
	// }

	return 0
}

func (s *session) mysqlBackupSingleDDLStatement(record *Record) {

}

func (s *session) mysqlBackupSingleStatement(record *Record) {

}

func (s *session) checkSqlIsDML(record *Record) bool {
	switch record.Type.(type) {
	case *ast.InsertStmt, *ast.DeleteStmt, *ast.UpdateStmt:
		if record.ExecComplete {
			return true
		}
		return false
	default:
		return false
	}
}

func (s *session) mysqlCreateBackupTable(record *Record) int {

	if record.TableInfo == nil || record.TableInfo.IsCreated {
		return 0
	}

	// configPrimaryKey(record.TableInfo)

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

	record.TableInfo.IsCreated = true
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
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	}
	return err
}

func (s *session) getRemoteBackupDBName(record *Record) string {

	if record.BackupDBName != "" {
		return record.BackupDBName
	}

	v := fmt.Sprintf("%s_%d_%s", s.opt.host, s.opt.port, record.TableInfo.Schema)

	if len(v) > mysql.MaxDatabaseNameLength {
		s.AppendErrorNo(ER_TOO_LONG_BAKDB_NAME, s.opt.host, s.opt.port, record.TableInfo.Schema)
		return ""
	}

	v = strings.Replace(v, "-", "_", -1)
	v = strings.Replace(v, ".", "_", -1)
	record.BackupDBName = v
	return record.BackupDBName
}

func (s *session) checkSqlIsDDL(record *Record) bool {

	switch record.Type.(type) {
	case *ast.CreateTableStmt,
		*ast.AlterTableStmt,
		*ast.DropTableStmt,
		*ast.RenameTableStmt,
		*ast.TruncateTableStmt,

		// *ast.CreateDatabaseStmt,
		// *ast.DropDatabaseStmt,

		*ast.CreateIndexStmt,
		*ast.DropIndexStmt:
		if record.ExecComplete {
			return true
		}
		return false

	default:
		return false
	}
}

func (s *session) executeAllStatement(ctx context.Context) {

	tmp := s.processInfo.Load()
	if tmp != nil {
		pi := tmp.(util.ProcessInfo)
		pi.OperState = "EXECUTING"
		pi.Percent = 0
		s.processInfo.Store(pi)
	}

	count := len(s.recordSets.All())
	for i, record := range s.recordSets.All() {

		// 忽略不需要备份的类型
		switch record.Type.(type) {
		case *ast.ShowStmt, *ast.ExplainStmt:
			continue
		}

		s.SetMyProcessInfo(record.Sql, time.Now(), float64(i)/float64(count))

		errno := s.executeRemoteCommand(record)
		if errno == 2 {
			break
		}

		// 进程Killed
		if err := checkClose(ctx); err != nil {
			log.Warn("Killed: ", err)
			s.AppendErrorMessage("Operation has been killed!")
			break
		}
	}
}

func (s *session) executeRemoteCommand(record *Record) int {

	s.myRecord = record
	record.Stage = StageExec

	// log.Infof("%T", record.Type)
	switch node := record.Type.(type) {

	case *ast.InsertStmt, *ast.DeleteStmt, *ast.UpdateStmt:

		s.executeRemoteStatementAndBackup(record)

	case *ast.UseStmt,
		*ast.CreateDatabaseStmt,
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
		log.Infof("无匹配类型: %T\n", node)
		s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
	}

	return int(record.ErrLevel)
}

func (s *session) executeRemoteStatement(record *Record) {
	log.Debug("executeRemoteStatement")

	sql := record.Sql

	start := time.Now()

	if record.useOsc {
		if s.Ghost.GhostOn {
			s.mysqlExecuteAlterTableGhost(record)
		} else {
			s.mysqlExecuteAlterTableOsc(record)
		}
		record.ExecTimestamp = time.Now().Unix()
		record.ThreadId = s.fetchThreadID()
		if record.ThreadId == 0 {
			record.ThreadId = s.fetchThreadID()
		}
		if record.ThreadId == 0 {
			s.AppendErrorMessage("无法获取线程号")
		}
		record.ExecTime = fmt.Sprintf("%.3f", time.Since(start).Seconds())
	} else {
		res := s.db.Exec(sql)

		record.ExecTime = fmt.Sprintf("%.3f", time.Since(start).Seconds())

		record.ExecTimestamp = time.Now().Unix()

		err := res.Error
		if err != nil {
			log.Error(err)
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
				s.AppendErrorMessage(myErr.Message)
			} else {
				s.AppendErrorMessage(err.Error())
			}
			record.StageStatus = StatusExecFail
		} else {
			record.AffectedRows = int(res.RowsAffected)
			if record.ThreadId == 0 {
				record.ThreadId = s.fetchThreadID()
			}
			if record.ThreadId == 0 {
				s.AppendErrorMessage("无法获取线程号")
			} else {
				record.StageStatus = StatusExecOK
				record.ExecComplete = true
			}

			switch record.Type.(type) {
			case *ast.InsertStmt, *ast.DeleteStmt, *ast.UpdateStmt:
				s.TotalChangeRows += record.AffectedRows
			}

			if _, ok := record.Type.(*ast.CreateTableStmt); ok &&
				record.TableInfo == nil && record.DBName != "" && record.TableName != "" {
				record.TableInfo = s.getTableFromCache(record.DBName, record.TableName, true)
			}
		}
	}
}

func (s *session) executeRemoteStatementAndBackup(record *Record) {
	log.Debug("executeRemoteStatementAndBackup")

	if s.opt.backup {
		masterStatus := s.mysqlFetchMasterBinlogPosition()
		if masterStatus == nil {
			s.AppendErrorNo(ErrNotFoundMasterStatus)
			return
		} else {
			record.StartFile = masterStatus.File
			record.StartPosition = masterStatus.Position
		}
	}

	s.executeRemoteStatement(record)

	if s.opt.backup {
		masterStatus := s.mysqlFetchMasterBinlogPosition()
		if masterStatus == nil {
			s.AppendErrorNo(ErrNotFoundMasterStatus)
			return
		} else {
			record.EndFile = masterStatus.File
			record.EndPosition = masterStatus.Position

			// 开始位置和结束位置一样,无变更
			if record.StartFile == record.EndFile &&
				record.StartPosition == record.EndPosition {
				return
			}
		}
	}

	record.ExecComplete = true
}

func (s *session) mysqlFetchMasterBinlogPosition() *MasterStatus {
	log.Debug("mysqlFetchMasterBinlogPosition")

	sql := "SHOW MASTER STATUS;"
	if s.isMiddleware() {
		sql = s.opt.middlewareExtend + sql
	}

	var r MasterStatus
	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		log.Error(err)
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else {
		for rows.Next() {
			s.db.ScanRows(rows, &r)
			return &r
		}
	}

	log.Info(sql)
	log.Infof("%#v", r)
	return nil
}

func (s *session) checkBinlogFormatIsRow() bool {
	log.Debug("checkBinlogFormatIsRow")

	sql := "show variables like 'binlog_format';"

	var format string

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}

	if err != nil {
		log.Error(err)
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

	// log.Infof("binlog format: %s", format)
	return format == "ROW"
}

func (s *session) mysqlServerVersion() {
	log.Debug("mysqlServerVersion")

	if s.DBVersion > 0 {
		return
	}

	var value string
	// sql := "select @@version;"
	sql := "show variables like 'version';"

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}

	if err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else {
		for rows.Next() {
			rows.Scan(&value, &value)
		}
	}

	if strings.Contains(strings.ToLower(value), "mariadb") {
		s.DBType = DBTypeMariaDB
	}
	versionStr := strings.Split(value, "-")[0]
	versionSeg := strings.Split(versionStr, ".")
	if len(versionSeg) == 3 {
		versionStr = fmt.Sprintf("%s%02s%02s", versionSeg[0], versionSeg[1], versionSeg[2])
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			s.AppendErrorMessage(err.Error())
		}
		s.DBVersion = version
	} else {
		s.AppendErrorMessage(fmt.Sprintf("无法解析版本号:%s", value))
	}

	log.Debug("db version: ", s.DBVersion)
}

func (s *session) fetchThreadID() (threadId uint32) {
	log.Debug("fetchThreadID")

	sql := "select connection_id();"
	if s.isMiddleware() {
		sql = s.opt.middlewareExtend + sql
	}

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()

		for rows.Next() {
			rows.Scan(&threadId)
		}
	}
	if err != nil {
		log.Error(err, threadId)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	}

	return
}

func (s *session) modifyBinlogFormatRow() {
	log.Debug("modifyBinlogFormatRow")

	sql := "set session binlog_format=row;"

	res := s.db.Exec(sql)

	if err := res.Error; err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
		log.Error(err)
	}
}

func (s *session) setSqlSafeUpdates() {
	log.Debug("setSqlSafeUpdates")

	var sql string
	if s.Inc.SqlSafeUpdates == 1 {
		sql = "set session sql_safe_updates=1;"
	} else if s.Inc.SqlSafeUpdates == 0 {
		sql = "set session sql_safe_updates=0;"
	} else {
		return
	}

	res := s.db.Exec(sql)

	if err := res.Error; err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
		log.Error(err)
	}
}

func (s *session) checkBinlogIsOn() bool {
	log.Debug("checkBinlogIsOn")

	sql := "show variables like 'log_bin';"

	var format string

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		log.Error(err)
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

	// log.Infof("log_bin is %s", format)
	return format == "ON"
}

func (s *session) parseOptions(sql string) {

	firsts := regParseOption.FindStringSubmatch(sql)
	if len(firsts) < 2 {
		log.Warning(sql)
		s.AppendErrorNo(ER_SQL_INVALID_SOURCE)
		return
	}

	// options := strings.Replace(strings.Replace(firsts[1], "-", "", -1), "_", "", -1)
	// options := strings.Replace(firsts[1], "=", ": ", -1)
	options := strings.Replace(firsts[1], "remote", "", -1)

	var buf strings.Builder

	for _, line := range strings.Split(options, ";") {
		if strings.Contains(line, "=") {
			v := strings.SplitN(line, "=", 2)
			param, value := v[0], v[1]
			param = strings.Replace(strings.Replace(param, "-", "", -1), "_", "", -1)

			buf.WriteString(param)
			buf.WriteString(": ")
			buf.WriteString(value)

		} else {
			line = strings.Replace(strings.Replace(line, "-", "", -1), "_", "", -1)
			if strings.HasPrefix(line, "enable") {
				buf.WriteString(line[6:])
				buf.WriteString(": true")
			} else if strings.HasPrefix(line, "disable") {
				buf.WriteString(line[7:])
				buf.WriteString(": false")
			} else {
				buf.WriteString(line)
			}
		}
		buf.WriteString("\n")
	}

	opt := buf.String()
	viper.SetConfigType("yaml")

	viper.ReadConfig(bytes.NewBuffer([]byte(opt)))

	s.opt = &sourceOptions{
		host:           viper.GetString("host"),
		port:           viper.GetInt("port"),
		user:           viper.GetString("user"),
		password:       viper.GetString("password"),
		check:          viper.GetBool("check"),
		execute:        viper.GetBool("execute"),
		backup:         viper.GetBool("backup"),
		ignoreWarnings: viper.GetBool("ignoreWarnings"),

		middlewareExtend: viper.GetString("middlewareExtend"),
		middlewareDB:     viper.GetString("middlewareDB"),
	}

	if s.opt.check {
		s.opt.execute = false
		s.opt.backup = false
	}

	log.Infof("%#v", s.opt)

	// 不再检查密码是否为空
	if s.opt.host == "" || s.opt.port == 0 || s.opt.user == "" {
		log.Warning(s.opt)
		s.AppendErrorNo(ER_SQL_INVALID_SOURCE)
		return
	}

	var addr string
	if s.opt.middlewareExtend == "" {
		addr = fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local&maxAllowedPacket=4194304",
			s.opt.user, s.opt.password, s.opt.host, s.opt.port)
	} else {
		s.opt.middlewareExtend = fmt.Sprintf("/*%s*/",
			strings.Replace(s.opt.middlewareExtend, ": ", "=", 1))

		addr = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&maxAllowedPacket=4194304&maxOpen=100&maxLifetime=60",
			s.opt.user, s.opt.password, s.opt.host, s.opt.port, s.opt.middlewareDB)

	}

	db, err := gorm.Open("mysql", addr)

	if err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
		return
	}

	// 禁用日志记录器，不显示任何日志
	db.LogMode(false)

	s.db = db

	if s.opt.execute {
		if s.opt.backup && !s.checkBinlogIsOn() {
			s.AppendErrorMessage("binlog日志未开启,无法备份!")
		}
	}

	if s.opt.backup {
		// 不再检查密码是否为空
		if s.Inc.BackupHost == "" || s.Inc.BackupPort == 0 || s.Inc.BackupUser == "" {
			s.AppendErrorNo(ER_INVALID_BACKUP_HOST_INFO)
		} else {
			addr = fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local",
				s.Inc.BackupUser, s.Inc.BackupPassword, s.Inc.BackupHost, s.Inc.BackupPort)
			backupdb, err := gorm.Open("mysql", addr)

			if err != nil {
				log.Error(err)
				s.AppendErrorMessage(err.Error())
				return
			}

			s.backupdb = backupdb
		}
	}

	tmp := s.processInfo.Load()
	if tmp != nil {
		pi := tmp.(util.ProcessInfo)
		pi.DestHost = s.opt.host
		pi.DestPort = s.opt.port
		pi.DestUser = s.opt.user

		if s.opt.check {
			pi.Command = "CHECK"
		} else if s.opt.execute {
			pi.Command = "EXECUTE"
		}
		s.processInfo.Store(pi)
	}

	s.setSqlSafeUpdates()
}

// createNewConnection 用来创建新的连接
// 注意: 该方法可能导致driver: bad connection异常
func (s *session) createNewConnection(dbName string) {
	addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&maxAllowedPacket=4194304",
		s.opt.user, s.opt.password, s.opt.host, s.opt.port, dbName)

	db, err := gorm.Open("mysql", addr)

	if err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
		return
	}

	if s.db != nil {
		s.db.Close()
	}

	// 禁用日志记录器，不显示任何日志
	db.LogMode(false)

	// 为保证连接成功关闭,此处等待10ms
	time.Sleep(10 * time.Millisecond)

	s.db = db
}

func (s *session) checkTruncateTable(node *ast.TruncateTableStmt, sql string) {

	log.Debug("checkTruncateTable")

	t := node.Table

	if !s.Inc.EnableDropTable {
		s.AppendErrorNo(ER_CANT_DROP_TABLE, t.Name)
	} else {

		if t.Schema.O == "" {
			t.Schema = model.NewCIStr(s.DBName)
		}

		table := s.getTableFromCache(t.Schema.O, t.Name.O, false)

		if table == nil {
			s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, fmt.Sprintf("%s.%s", t.Schema, t.Name))
		} else {
			s.mysqlShowTableStatus(table)
		}
	}
}

func (s *session) checkDropTable(node *ast.DropTableStmt, sql string) {

	log.Debug("checkDropTable")

	// log.Infof("%#v \n", node)
	for _, t := range node.Tables {

		if !s.Inc.EnableDropTable {
			s.AppendErrorNo(ER_CANT_DROP_TABLE, t.Name)
		} else {

			if t.Schema.O == "" {
				t.Schema = model.NewCIStr(s.DBName)
			}

			table := s.getTableFromCache(t.Schema.O, t.Name.O, false)

			//如果表不存在，但存在if existed，则跳过
			if table == nil {
				if !node.IfExists {
					s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, fmt.Sprintf("%s.%s", t.Schema, t.Name))
				}
			} else {
				if s.opt.execute {
					// 生成回滚语句
					s.mysqlShowCreateTable(table)
				}

				if s.opt.check {
					// 获取表估计的受影响行数
					s.mysqlShowTableStatus(table)
				}

				s.myRecord.TableInfo = table

				s.myRecord.TableInfo.IsDeleted = true
			}
		}
	}
}

// mysqlShowTableStatus 获取表估计的受影响行数
func (s *session) mysqlShowTableStatus(t *TableInfo) {

	if t.IsNew {
		return
	}

	// sql := fmt.Sprintf("show table status from `%s` where name = '%s';", dbname, tableName)
	sql := fmt.Sprintf(`select TABLE_ROWS from information_schema.tables
		where table_schema='%s' and table_name='%s';`, t.Schema, t.Name)

	var res uint

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else if rows != nil {
		for rows.Next() {
			rows.Scan(&res)
		}
		s.myRecord.AffectedRows = int(res)
	}
}

// mysqlGetTableSize 获取表估计的受影响行数
func (s *session) mysqlGetTableSize(t *TableInfo) {

	if t.IsNew || t.TableSize > 0 {
		return
	}

	sql := fmt.Sprintf(`select (DATA_LENGTH + INDEX_LENGTH)/1024/1024
		from information_schema.tables
		where table_schema='%s' and table_name='%s';`, t.Schema, t.Name)

	var res uint

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else if rows != nil {
		for rows.Next() {
			rows.Scan(&res)
		}
		t.TableSize = res
	}
}

// mysqlShowCreateTable 生成回滚语句
func (s *session) mysqlShowCreateTable(t *TableInfo) {

	if t.IsNew {
		return
	}

	sql := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`;", t.Schema, t.Name)

	var res string

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}

	if err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else if rows != nil {

		for rows.Next() {
			rows.Scan(&res, &res)
		}
		s.myRecord.DDLRollback = res
		s.myRecord.DDLRollback += ";"
	}
}

// mysqlShowCreateDatabase 生成回滚语句
func (s *session) mysqlShowCreateDatabase(name string) {

	sql := fmt.Sprintf("SHOW CREATE DATABASE `%s`;", name)

	var res string

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}

	if err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		} else {
			s.AppendErrorMessage(err.Error())
		}
	} else if rows != nil {
		for rows.Next() {
			rows.Scan(&res, &res)
		}
		s.myRecord.DDLRollback = res
		s.myRecord.DDLRollback += ";"
	}
}

func (s *session) checkRenameTable(node *ast.RenameTableStmt, sql string) {

	log.Debug("checkRenameTable")

	// log.Infof("%#v \n", node)

	originTable := s.getTableFromCache(node.OldTable.Schema.O, node.OldTable.Name.O, true)
	if originTable == nil {
		s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, node.OldTable.Name.O)
	}

	table := s.getTableFromCache(node.NewTable.Schema.O, node.NewTable.Name.O, false)
	if table != nil {
		s.AppendErrorNo(ER_TABLE_EXISTS_ERROR, node.NewTable.Name.O)
	}

	s.checkKeyWords(node.NewTable.Schema.O)

	if s.hasError() {
		return
	}

	// 旧表存在,新建不存在时
	if originTable != nil && table == nil {
		table = s.copyTableInfo(originTable)

		table.Name = node.NewTable.Name.O
		if node.NewTable.Schema.O == "" {
			table.Schema = s.DBName
		} else {
			table.Schema = node.NewTable.Schema.O
		}
		s.cacheNewTable(table)
		s.myRecord.TableInfo = table

		if s.opt.execute {
			s.myRecord.DDLRollback = fmt.Sprintf("RENAME TABLE `%s`.`%s` TO `%s`.`%s`;",
				table.Schema, table.Name, originTable.Schema, originTable.Name)
		}
	}

	if originTable != nil {
		// rename后旧表标记删除
		originTable.IsDeleted = true
	}
}

func (s *session) checkCreateTable(node *ast.CreateTableStmt, sql string) {

	log.Debug("checkCreateTable")

	// tidb暂不支持临时表 create temporary table t1

	if node.Table.Schema.O == "" {
		node.Table.Schema = model.NewCIStr(s.DBName)
	}

	if !s.checkDBExists(node.Table.Schema.O, true) {
		return
	}

	s.checkKeyWords(node.Table.Name.O)
	// 如果列名有错误的话,则直接跳出
	if s.myRecord.ErrLevel == 2 {
		return
	}

	table := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, false)

	if table != nil {
		if !node.IfNotExists {
			s.AppendErrorNo(ER_TABLE_EXISTS_ERROR, node.Table.Name.O)
		}
		s.myRecord.DBName = node.Table.Schema.O
		s.myRecord.TableName = node.Table.Name.O
	} else {

		s.myRecord.DBName = node.Table.Schema.O
		s.myRecord.TableName = node.Table.Name.O

		s.checkCreateTableGrammar(node)

		s.checkAutoIncrement(node)
		s.checkContainDotColumn(node)

		// 缓存表结构 CREATE TABLE LIKE
		if node.ReferTable != nil {
			originTable := s.getTableFromCache(node.ReferTable.Schema.O, node.ReferTable.Name.O, true)
			if originTable != nil {
				table = s.copyTableInfo(originTable)

				table.Name = node.Table.Name.O
				table.Schema = node.Table.Schema.O

				s.cacheNewTable(table)
				s.myRecord.TableInfo = table
			}
		} else {

			// 校验列是否重复指定
			checkDup := map[string]bool{}
			for _, c := range node.Cols {
				if _, ok := checkDup[c.Name.Name.L]; ok {
					s.AppendErrorNo(ER_DUP_FIELDNAME, c.Name.Name)
				}
				checkDup[c.Name.Name.L] = true
			}

			hasComment := false
			for _, opt := range node.Options {
				// log.Infof("%#v", opt)
				switch opt.Tp {
				case ast.TableOptionEngine:
					if !strings.EqualFold(opt.StrValue, "innodb") {
						s.AppendErrorNo(ER_TABLE_MUST_INNODB, node.Table.Name.O)
					}
				case ast.TableOptionCharset:
					if !s.Inc.EnableSetCharset {
						s.AppendErrorNo(ER_TABLE_CHARSET_MUST_NULL, node.Table.Name.O)
					}
					s.checkCharset(opt.StrValue)
				case ast.TableOptionCollate:
					if !s.Inc.EnableSetCharset {
						s.AppendErrorNo(ER_TABLE_CHARSET_MUST_NULL, node.Table.Name.O)
					}
				case ast.TableOptionComment:
					if opt.StrValue != "" {
						hasComment = true
					}
					if len(opt.StrValue) > TABLE_COMMENT_MAXLEN {
						s.AppendErrorMessage(fmt.Sprintf("Comment for table '%s' is too long (max = %d)",
							node.Table.Name.O, TABLE_COMMENT_MAXLEN))
					}
				case ast.TableOptionAutoIncrement:
					// log.Infof("%#v", opt)
					if opt.UintValue > 1 {
						s.AppendErrorNo(ER_INC_INIT_ERR)
					}
				}
			}

			hasPrimary := false
			for _, ct := range node.Constraints {
				// log.Infof("%#v", ct)
				switch ct.Tp {
				case ast.ConstraintPrimaryKey:
					hasPrimary = len(ct.Keys) > 0
					// for _, col := range ct.Keys {
					// 	found := false
					// 	for _, field := range node.Cols {
					// 		if field.Name.Name.L == col.Column.Name.L {
					// 			found = true
					// 			break
					// 		}
					// 	}
					// 	if !found {
					// 		s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
					// 			fmt.Sprintf("%s.%s", node.Table.Name.O, col.Column.Name.O))
					// 	}
					// }
					break
				}
			}

			if !hasPrimary {
				for _, field := range node.Cols {
					hasNullFlag := false
					defaultNullValue := false
					for _, op := range field.Options {
						switch op.Tp {
						case ast.ColumnOptionNull:
							hasNullFlag = true
						case ast.ColumnOptionPrimaryKey:
							hasPrimary = true

							if field.Tp.Tp != mysql.TypeInt24 &&
								field.Tp.Tp != mysql.TypeLong &&
								field.Tp.Tp != mysql.TypeLonglong {
								s.AppendErrorNo(ER_PK_COLS_NOT_INT,
									field.Name.Name.O,
									node.Table.Schema, node.Table.Name)
							}
						case ast.ColumnOptionDefaultValue:
							// log.Infof("%#v", op)
							// log.Info(op.Expr.GetDatum().GetString())
							// log.Info(op.Expr.GetDatum().IsNull())

							// op.Expr.GetDatum().GetString()
							if op.Expr.GetDatum().IsNull() {
								defaultNullValue = true
							}
						}
					}

					if hasPrimary && (hasNullFlag || defaultNullValue) {
						s.AppendErrorNo(ER_PRIMARY_CANT_HAVE_NULL)
					}
					if hasPrimary {
						break
					}
				}
			}

			if !hasPrimary {
				s.AppendErrorNo(ER_TABLE_MUST_HAVE_PK, node.Table.Name.O)
			}

			if !hasComment {
				s.AppendErrorNo(ER_TABLE_MUST_HAVE_COMMENT, node.Table.Name.O)
			}

			if len(node.Cols) > 0 {
				table = s.buildTableInfo(node)

				currentTimestampCount := 0
				onUpdateTimestampCount := 0
				for _, field := range node.Cols {
					s.mysqlCheckField(table, field)

					if field.Tp.Tp == mysql.TypeTimestamp {
						for _, op := range field.Options {
							if op.Tp == ast.ColumnOptionDefaultValue {
								if f, ok := op.Expr.(*ast.FuncCallExpr); ok {
									if f.FnName.L == ast.CurrentTimestamp {
										currentTimestampCount += 1
									}
								}
							} else if op.Tp == ast.ColumnOptionOnUpdate {
								if f, ok := op.Expr.(*ast.FuncCallExpr); ok {
									if f.FnName.L == ast.CurrentTimestamp {
										onUpdateTimestampCount += 1
									}
								} else {

								}
							}
						}
					}
				}

				if currentTimestampCount > 1 || onUpdateTimestampCount > 1 {
					s.AppendErrorNo(ER_TOO_MUCH_AUTO_TIMESTAMP_COLS)
				}

				s.cacheNewTable(table)
				s.myRecord.TableInfo = table
			}
		}

		if node.Partition != nil {
			s.AppendErrorNo(ER_PARTITION_NOT_ALLOWED)
		}

		if node.ReferTable != nil || len(node.Cols) > 0 {
			dupIndexes := map[string]bool{}
			for _, ct := range node.Constraints {
				s.checkCreateIndex(nil, ct.Name,
					ct.Keys, ct.Option, table, false, ct.Tp)

				switch ct.Tp {
				case ast.ConstraintKey, ast.ConstraintUniq,
					ast.ConstraintIndex, ast.ConstraintUniqKey,
					ast.ConstraintUniqIndex:
					if ct.Name == "" {
						ct.Name = ct.Keys[0].Column.Name.O
					}
					if _, ok := dupIndexes[strings.ToLower(ct.Name)]; ok {
						s.AppendErrorNo(ER_DUP_KEYNAME, ct.Name)
					}
					dupIndexes[strings.ToLower(ct.Name)] = true
				}
			}

			if len(node.Cols) > 0 && table == nil {
				s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, node.Table.Name.O)
				return
			}
		}
	}

	if !s.hasError() && s.opt.execute {
		s.myRecord.DDLRollback = fmt.Sprintf("DROP TABLE `%s`.`%s`;", table.Schema, table.Name)
	}
}

func (s *session) buildTableInfo(node *ast.CreateTableStmt) *TableInfo {
	log.Debug("buildTableInfo")

	table := &TableInfo{}

	if node.Table.Schema.O == "" {
		table.Schema = s.DBName
	} else {
		table.Schema = node.Table.Schema.O
	}

	table.Name = node.Table.Name.O
	table.Fields = make([]FieldInfo, 0, len(node.Cols))

	for _, field := range node.Cols {
		c := s.buildNewColumnToCache(table, field)
		table.Fields = append(table.Fields, *c)
	}
	table.IsNewColumns = true

	return table
}

func (s *session) checkAlterTable(node *ast.AlterTableStmt, sql string) {

	log.Debug("checkAlterTable")

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

	if node.Table.Schema.O == "" {
		node.Table.Schema = model.NewCIStr(s.DBName)
	}

	if !s.checkDBExists(node.Table.Schema.O, true) {
		return
	}

	// table := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, true)
	table := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, true)
	if table == nil {
		return
	}

	table.AlterCount += 1

	if table.AlterCount > 1 {
		s.AppendErrorNo(ER_ALTER_TABLE_ONCE, node.Table.Name.O)
	}

	for _, sepc := range node.Specs {
		if sepc.Options != nil {
			hasComment := false
			for _, opt := range sepc.Options {
				// log.Infof("%#v", opt)
				switch opt.Tp {
				case ast.TableOptionEngine:
					if !strings.EqualFold(opt.StrValue, "innodb") {
						s.AppendErrorNo(ER_TABLE_MUST_INNODB, node.Table.Name.O)
					}
				case ast.TableOptionCharset:
					if !s.Inc.EnableSetCharset {
						s.AppendErrorNo(ER_TABLE_CHARSET_MUST_NULL, node.Table.Name.O)
					}
					s.checkCharset(opt.StrValue)
				case ast.TableOptionCollate:
					if !s.Inc.EnableSetCharset {
						s.AppendErrorNo(ER_TABLE_CHARSET_MUST_NULL, node.Table.Name.O)
					}
				case ast.TableOptionComment:
					if opt.StrValue != "" {
						hasComment = true
					}
				}
			}
			if !hasComment {
				s.AppendErrorNo(ER_TABLE_MUST_HAVE_COMMENT, node.Table.Name.O)
			}
		}
	}

	s.mysqlShowTableStatus(table)
	s.mysqlGetTableSize(table)

	for _, alter := range node.Specs {
		if alter.Tp != ast.AlterTableRenameTable {
			s.checkAlterUseOsc(table)
		} else {
			s.myRecord.useOsc = false
		}
	}

	s.myRecord.TableInfo = table

	if s.opt.execute {
		s.myRecord.DDLRollback += fmt.Sprintf("ALTER TABLE `%s`.`%s` ",
			table.Schema, table.Name)
	}

	for _, alter := range node.Specs {
		switch alter.Tp {
		case ast.AlterTableAddColumns:
			s.checkAddColumn(table, alter)
		case ast.AlterTableDropColumn:
			// s.AppendErrorNo(ER_CANT_DROP_FIELD_OR_KEY, alter.OldColumnName.Name.O)
			s.checkDropColumn(table, alter)

		case ast.AlterTableAddConstraint:
			s.checkAddConstraint(table, alter)

		case ast.AlterTableDropPrimaryKey:
			// s.AppendErrorNo(ER_CANT_DROP_FIELD_OR_KEY, alter.Name)
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

		case ast.AlterTableAlterColumn:
			s.checkAlterTableAlterColumn(table, alter)
		default:
			s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
			log.Info("未定义的解析: ", alter.Tp)
		}
	}

	if s.opt.execute {
		if strings.HasSuffix(s.myRecord.DDLRollback, ",") {
			s.myRecord.DDLRollback = strings.TrimSuffix(s.myRecord.DDLRollback, ",") + ";"
		}
	}

}

func (s *session) checkAlterTableAlterColumn(t *TableInfo, c *ast.AlterTableSpec) {
	// log.Info("checkAlterTableAlterColumn")

	for _, nc := range c.NewColumns {
		found := false
		var foundField FieldInfo
		for _, field := range t.Fields {
			if strings.EqualFold(field.Field, nc.Name.Name.O) {
				found = true
				foundField = field
				break
			}
		}

		if !found {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name))
		} else {
			if s.opt.execute {
				if foundField.Default == nil {
					s.myRecord.DDLRollback += "DROP DEFAULT,"
				} else {
					s.myRecord.DDLRollback += fmt.Sprintf("SET DEFAULT '%s',", *foundField.Default)
				}
			}

			if nc.Options == nil {
				// drop default . 不需要判断,可以删除本身为null的默认值
				foundField.Default = nil
			} else {
				// "SET" "DEFAULT" SignedLiteral
				for _, op := range nc.Options {
					defaultValue := fmt.Sprint(op.Expr.GetValue())

					if len(defaultValue) == 0 {
						switch strings.Split(foundField.Type, "(")[0] {
						case "bit", "smallint", "mediumint", "int",
							"bigint", "decimal", "float", "double", "year":
							s.AppendErrorNo(ER_INVALID_DEFAULT, nc.Name.Name)
						}
					}

					foundField.Default = &defaultValue
				}
			}

		}
	}

}

func (s *session) checkAlterTableRenameTable(t *TableInfo, c *ast.AlterTableSpec) {
	// log.Info("checkAlterTableRenameTable")

	table := s.getTableFromCache(c.NewTable.Schema.O, c.NewTable.Name.O, false)
	if table != nil {
		s.AppendErrorNo(ER_TABLE_EXISTS_ERROR, c.NewTable.Name.O)
	} else {
		// 旧表存在,新建不存在时

		table = s.copyTableInfo(t)

		table.Name = c.NewTable.Name.O
		if c.NewTable.Schema.O == "" {
			table.Schema = s.DBName
		} else {
			table.Schema = c.NewTable.Schema.O
		}
		s.cacheNewTable(table)
		s.myRecord.TableInfo = table

		if s.opt.execute {
			s.myRecord.DDLRollback = fmt.Sprintf("RENAME TABLE `%s`.`%s` TO `%s`.`%s`;",
				table.Schema, table.Name, t.Schema, t.Name)
		}

		// rename后旧表标记删除
		t.IsDeleted = true
	}
}

func (s *session) checkChangeColumn(t *TableInfo, c *ast.AlterTableSpec) {
	log.Debug("checkChangeColumn")

	s.checkModifyColumn(t, c)
}

func (s *session) checkModifyColumn(t *TableInfo, c *ast.AlterTableSpec) {
	log.Debug("checkModifyColumn")

	for _, nc := range c.NewColumns {

		found := false
		foundIndexOld := -1
		var foundField FieldInfo

		if nc.Name.Schema.L != "" && !strings.EqualFold(nc.Name.Schema.L, t.Schema) {
			s.AppendErrorNo(ER_WRONG_DB_NAME, nc.Name.Schema.O)
		} else if nc.Name.Table.L != "" && !strings.EqualFold(nc.Name.Table.L, t.Name) {
			s.AppendErrorNo(ER_WRONG_TABLE_NAME, nc.Name.Table.O)
		}

		if s.myRecord.ErrLevel == 2 {
			continue
		}

		// 列名未变
		if c.OldColumnName == nil || c.OldColumnName.Name.L == nc.Name.Name.L {
			for i, field := range t.Fields {
				if strings.EqualFold(field.Field, nc.Name.Name.O) {
					found = true
					foundIndexOld = i
					foundField = field
					break
				}
			}

			if !found {
				s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name))
			} else {

				if c.Position.Tp != ast.ColumnPositionNone {

					// 在新的快照上变更表结构
					t := s.cacheTableSnapshot(t)

					if c.Position.Tp == ast.ColumnPositionFirst {
						tmp := make([]FieldInfo, 0, len(t.Fields))
						tmp = append(tmp, foundField)
						if foundIndexOld > 0 {
							tmp = append(tmp, t.Fields[:foundIndexOld]...)
						}
						tmp = append(tmp, t.Fields[foundIndexOld+1:]...)

						t.Fields = tmp
					} else if c.Position.Tp == ast.ColumnPositionAfter {
						foundIndex := -1
						for i, field := range t.Fields {
							if strings.EqualFold(field.Field, c.Position.RelativeColumn.Name.O) && !field.IsDeleted {
								foundIndex = i
								break
							}
						}
						if foundIndex == -1 {
							s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
								fmt.Sprintf("%s.%s", t.Name, c.Position.RelativeColumn.Name))
						} else if foundIndex == foundIndexOld-1 {
							// 原位置和新位置一样,不做操作
						} else {

							tmp := make([]FieldInfo, 0, len(t.Fields)+3)
							// 先把列移除
							// tmp = append(t.Fields[:foundIndexOld], t.Fields[foundIndexOld+1:]...)

							if foundIndex > foundIndexOld {
								tmp = append(tmp, t.Fields[:foundIndexOld]...)
								tmp = append(tmp, t.Fields[foundIndexOld+1:foundIndex+1]...)
								tmp = append(tmp, foundField)
								tmp = append(tmp, t.Fields[foundIndex+1:]...)
							} else {
								tmp = append(tmp, t.Fields[:foundIndex+1]...)
								tmp = append(tmp, foundField)
								tmp = append(tmp, t.Fields[foundIndex+1:foundIndexOld]...)
								tmp = append(tmp, t.Fields[foundIndexOld+1:]...)
							}

							t.Fields = tmp
						}
					}
				}

				if s.opt.execute {
					buf := bytes.NewBufferString("MODIFY COLUMN `")
					buf.WriteString(foundField.Field)
					buf.WriteString("` ")
					buf.WriteString(foundField.Type)
					if foundField.Null == "NO" {
						buf.WriteString(" NOT NULL")
					}

					if foundField.Default != nil {
						buf.WriteString(" DEFAULT '")
						buf.WriteString(*foundField.Default)
						buf.WriteString("'")
					}

					if foundField.Comment != "" {
						buf.WriteString(" COMMENT '")
						buf.WriteString(foundField.Comment)
						buf.WriteString("'")
					}
					buf.WriteString(",")

					s.myRecord.DDLRollback += buf.String()
				}
			}
		} else { // 列名改变

			oldFound := false
			newFound := false
			foundIndexOld := -1
			for i, field := range t.Fields {
				if strings.EqualFold(field.Field, c.OldColumnName.Name.L) {
					oldFound = true
					foundIndexOld = i
					foundField = field
				}
				if strings.EqualFold(field.Field, nc.Name.Name.L) {
					newFound = true
				}
			}

			// 未变更列名时,列需要存在
			// 变更列名后,新列名不能存在
			if newFound {
				s.AppendErrorNo(ER_COLUMN_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name.O))
			}
			if !oldFound {
				s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, c.OldColumnName.Name.O))
			}

			s.checkKeyWords(nc.Name.Name.O)

			if !s.hasError() {

				// 在新的快照上变更表结构
				t := s.cacheTableSnapshot(t)

				t.Fields[foundIndexOld].Field = nc.Name.Name.O
				// 修改列名后标记有新列
				t.IsNewColumns = true

				if c.Position.Tp != ast.ColumnPositionNone {

					if c.Position.Tp == ast.ColumnPositionFirst {
						tmp := make([]FieldInfo, 0, len(t.Fields))
						tmp = append(tmp, foundField)
						if foundIndexOld > 0 {
							tmp = append(tmp, t.Fields[:foundIndexOld]...)
						}
						tmp = append(tmp, t.Fields[foundIndexOld+1:]...)

						t.Fields = tmp
					} else if c.Position.Tp == ast.ColumnPositionAfter {
						foundIndex := -1
						for i, field := range t.Fields {
							if strings.EqualFold(field.Field, c.Position.RelativeColumn.Name.O) && !field.IsDeleted {
								foundIndex = i
								break
							}
						}
						if foundIndex == -1 {
							s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
								fmt.Sprintf("%s.%s", t.Name, c.Position.RelativeColumn.Name))
						} else if foundIndex == foundIndexOld-1 {
							// 原位置和新位置一样,不做操作
						} else {

							tmp := make([]FieldInfo, 0, len(t.Fields)+3)
							// 先把列移除
							// tmp = append(t.Fields[:foundIndexOld], t.Fields[foundIndexOld+1:]...)

							if foundIndex > foundIndexOld {
								tmp = append(tmp, t.Fields[:foundIndexOld]...)
								tmp = append(tmp, t.Fields[foundIndexOld+1:foundIndex+1]...)
								tmp = append(tmp, foundField)
								tmp = append(tmp, t.Fields[foundIndex+1:]...)
							} else {
								tmp = append(tmp, t.Fields[:foundIndex+1]...)
								tmp = append(tmp, foundField)
								tmp = append(tmp, t.Fields[foundIndex+1:foundIndexOld]...)
								tmp = append(tmp, t.Fields[foundIndexOld+1:]...)
							}

							t.Fields = tmp
						}
					}
				}

				if s.opt.execute {
					buf := bytes.NewBufferString("CHANGE COLUMN `")
					buf.WriteString(nc.Name.Name.O)
					buf.WriteString("` `")
					buf.WriteString(foundField.Field)
					buf.WriteString("` ")
					buf.WriteString(foundField.Type)
					if foundField.Null == "NO" {
						buf.WriteString(" NOT NULL")
					}
					if foundField.Default != nil {
						buf.WriteString(" DEFAULT '")
						buf.WriteString(*foundField.Default)
						buf.WriteString("'")
					}
					if foundField.Comment != "" {
						buf.WriteString(" COMMENT '")
						buf.WriteString(foundField.Comment)
						buf.WriteString("'")
					}
					buf.WriteString(",")

					s.myRecord.DDLRollback += buf.String()
				}
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

		// if !types.IsTypeBlob(nc.Tp.Tp) && (nc.Tp.Charset != "" || nc.Tp.Collate != "") {
		// 	s.AppendErrorNo(ER_CHARSET_ON_COLUMN, t.Name, nc.Name.Name)
		// }

		s.mysqlCheckField(t, nc)

		// 列(或旧列)未找到时结束
		if s.hasError() {
			return
		}

		fieldType := nc.Tp.CompactStr()

		switch nc.Tp.Tp {
		case mysql.TypeDecimal, mysql.TypeNewDecimal,
			mysql.TypeVarchar,
			mysql.TypeVarString:
			str := string([]byte(foundField.Type)[:7])
			if !strings.Contains(fieldType, str) {
				s.AppendErrorNo(ER_CHANGE_COLUMN_TYPE,
					fmt.Sprintf("%s.%s", t.Name, nc.Name.Name),
					foundField.Type, fieldType)
			}
		case mysql.TypeString:
			str := string([]byte(foundField.Type)[:4])
			if !strings.Contains(fieldType, str) {
				s.AppendErrorNo(ER_CHANGE_COLUMN_TYPE,
					fmt.Sprintf("%s.%s", t.Name, nc.Name.Name),
					foundField.Type, fieldType)
			}
		default:
			if strings.Contains(fieldType, "(") && strings.Contains(foundField.Type, "(") {
				if fieldType[:strings.Index(fieldType, "(")] !=
					foundField.Type[:strings.Index(foundField.Type, "(")] {
					s.AppendErrorNo(ER_CHANGE_COLUMN_TYPE,
						fmt.Sprintf("%s.%s", t.Name, nc.Name.Name),
						foundField.Type, fieldType)
				}
			} else if fieldType != foundField.Type {
				s.AppendErrorNo(ER_CHANGE_COLUMN_TYPE,
					fmt.Sprintf("%s.%s", t.Name, nc.Name.Name),
					foundField.Type, fieldType)
			}
		}
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

// hasError return current sql has errors or warnings
func (s *session) hasError() bool {
	if s.myRecord.ErrLevel == 2 ||
		(s.myRecord.ErrLevel == 1 && !s.opt.ignoreWarnings) {
		return true
	}

	return false
}

// hasError return all sql has errors or warnings
func (s *session) hasErrorBefore() bool {
	if s.recordSets.MaxLevel == 2 ||
		(s.recordSets.MaxLevel == 1 && !s.opt.ignoreWarnings) {
		return true
	}

	return false
}

func (s *session) mysqlCheckField(t *TableInfo, field *ast.ColumnDef) {
	log.Debug("mysqlCheckField")

	// log.Infof("%#v", field.Tp)

	tableName := t.Name
	if field.Tp.Tp == mysql.TypeEnum ||
		field.Tp.Tp == mysql.TypeSet ||
		field.Tp.Tp == mysql.TypeBit {
		s.AppendErrorNo(ER_INVALID_DATA_TYPE, field.Name.Name)
	}

	if field.Tp.Tp == mysql.TypeString && field.Tp.Flen > int(s.Inc.MaxCharLength) {
		s.AppendErrorNo(ER_CHAR_TO_VARCHAR_LEN, field.Name.Name)
	}

	s.checkKeyWords(field.Name.Name.O)

	// notNullFlag := mysql.HasNotNullFlag(field.Tp.Flag)
	// autoIncrement := mysql.HasAutoIncrementFlag(field.Tp.Flag)

	hasComment := false
	notNullFlag := false
	autoIncrement := false
	hasDefaultValue := false
	hasGenerated := false
	var defaultValue *types.Datum
	var defaultExpr ast.ExprNode

	isPrimary := false

	if len(field.Options) > 0 {
		for _, op := range field.Options {

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
			case ast.ColumnOptionDefaultValue:
				defaultExpr = op.Expr
				defaultValue = op.Expr.GetDatum()
				hasDefaultValue = true
			case ast.ColumnOptionPrimaryKey:
				isPrimary = true
			case ast.ColumnOptionGenerated:
				hasGenerated = true
			}
		}
	}

	if !hasComment {
		s.AppendErrorNo(ER_COLUMN_HAVE_NO_COMMENT, field.Name.Name, tableName)
	}

	//有默认值，且归类无效，如(default CURRENT_TIMESTAMP)
	if hasDefaultValue && isInvalidDefaultValue(field) {
		s.AppendErrorNo(ER_INVALID_DEFAULT, field.Name.Name.O)
	}

	//有默认值，且为NULL，且有NOT NULL约束，如(not null default null)
	if _, ok := defaultExpr.(*ast.ValueExpr); ok && hasDefaultValue && defaultValue.IsNull() && notNullFlag {
		s.AppendErrorNo(ER_INVALID_DEFAULT, field.Name.Name.O)
	}

	//有默认值，且不为NULL
	if _, ok := defaultExpr.(*ast.ValueExpr); ok && hasDefaultValue && !defaultValue.IsNull() {
		switch field.Tp.Tp {
		case mysql.TypeTiny, mysql.TypeShort, mysql.TypeInt24,
			mysql.TypeLong, mysql.TypeLonglong,
			mysql.TypeBit, mysql.TypeYear,
			mysql.TypeFloat, mysql.TypeDouble, mysql.TypeNewDecimal:
			//验证string型默认值的合法性
			if v, ok := defaultValue.GetValue().(string); ok {
				if v == "" {
					s.AppendErrorNo(ER_INVALID_DEFAULT, field.Name.Name)
				} else {
					_, intErr := strconv.ParseInt(defaultValue.GetString(), 10, 64)
					_, floatErr := strconv.ParseFloat(defaultValue.GetString(), 64)
					if intErr != nil && floatErr != nil {
						s.AppendErrorNo(ER_INVALID_DEFAULT, field.Name.Name)
					}
				}

			}

		}
	}

	//不可设置default值的部分字段类型
	if hasDefaultValue && !defaultValue.IsNull() && (field.Tp.Tp == mysql.TypeJSON || types.IsTypeBlob(field.Tp.Tp)) {
		s.AppendErrorNo(ER_BLOB_CANT_HAVE_DEFAULT, field.Name.Name.O)
	}

	if types.IsTypeBlob(field.Tp.Tp) {
		s.AppendErrorNo(ER_USE_TEXT_OR_BLOB, field.Name.Name)
	} else {
		if !notNullFlag {
			s.AppendErrorNo(ER_NOT_ALLOWED_NULLABLE, field.Name.Name, tableName)
		}

		if field.Tp.Charset != "" || field.Tp.Collate != "" {
			if field.Tp.Charset != "binary" {
				s.AppendErrorNo(ER_CHARSET_ON_COLUMN, tableName, field.Name.Name)
			}
		}
	}

	if isIncorrectName(field.Name.Name.O) {
		s.AppendErrorNo(ER_WRONG_COLUMN_NAME, field.Name.Name)
	}

	if types.IsTypeBlob(field.Tp.Tp) && notNullFlag {
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
		// if !mysql.HasNoDefaultValueFlag(field.Tp.Flag) {
		if !hasDefaultValue {
			s.AppendErrorNo(ER_TIMESTAMP_DEFAULT, field.Name.Name.O)
		}
	}

	if !hasDefaultValue && field.Tp.Tp != mysql.TypeTimestamp &&
		!types.IsTypeBlob(field.Tp.Tp) && !autoIncrement && !isPrimary && field.Tp.Tp != mysql.TypeJSON && !hasGenerated {
		s.AppendErrorNo(ER_WITH_DEFAULT_ADD_COLUMN, field.Name.Name.O, tableName)
	}

	// if (thd->variables.sql_mode & MODE_NO_ZERO_DATE &&
	//        is_timestamp_type(field->sql_type) && !field->def &&
	//        (field->flags & NOT_NULL_FLAG) &&
	//        (field->unireg_check == Field::NONE ||
	//         field->unireg_check == Field::TIMESTAMP_UN_FIELD))
	//    {
	//        my_error(ER_INVALID_DEFAULT, MYF(0), field->field_name);
	//        mysql_errmsg_append(thd);
	//    }
}

func (s *session) checkIndexAttr(tp ast.ConstraintType, name string,
	keys []*ast.IndexColName, table *TableInfo) {

	if tp == ast.ConstraintPrimaryKey {

		if s.Inc.MaxPrimaryKeyParts > 0 && len(keys) > int(s.Inc.MaxPrimaryKeyParts) {
			s.AppendErrorNo(ER_TOO_MANY_KEY_PARTS, table.Schema, table.Name, s.Inc.MaxPrimaryKeyParts)
		}

		s.checkDuplicateColumnName(keys)

		return
	}

	if name == "" {
		s.AppendErrorNo(ER_WRONG_NAME_FOR_INDEX, "NULL", table.Name)
	} else {
		// found := false
		// for _, field := range table.Fields {
		// 	if strings.EqualFold(field.Field, name) {
		// 		found = true
		// 		break
		// 	}
		// }
		if isIncorrectName(name) {
			s.AppendErrorNo(ER_WRONG_NAME_FOR_INDEX, name, table.Name)
		} else {
			if len(name) > mysql.MaxIndexIdentifierLen {
				s.AppendErrorNo(ER_TOO_LONG_IDENT, name)
			}
		}
	}

	if tp != ast.ConstraintPrimaryKey && strings.ToUpper(name) == "PRIMARY" {
		s.AppendErrorNo(ER_WRONG_NAME_FOR_INDEX, name, table.Name)
	}

	s.checkDuplicateColumnName(keys)

	switch tp {
	case ast.ConstraintForeignKey:
		s.AppendErrorNo(ER_FOREIGN_KEY, table.Name)

	case ast.ConstraintUniq:
		if !strings.HasPrefix(name, "uniq_") {
			s.AppendErrorNo(ER_INDEX_NAME_UNIQ_PREFIX, name, table.Name)
		}

	default:
		if !strings.HasPrefix(name, "idx_") {
			s.AppendErrorNo(ER_INDEX_NAME_IDX_PREFIX, name, table.Name)
		}
	}

	if s.Inc.MaxKeyParts > 0 && len(keys) > int(s.Inc.MaxKeyParts) {
		s.AppendErrorNo(ER_TOO_MANY_KEY_PARTS, table.Name, s.Inc.MaxKeyParts)
	}

}

func (s *session) checkDropForeignKey(t *TableInfo, c *ast.AlterTableSpec) {
	log.Debug("checkDropForeignKey")

	// log.Infof("%s \n", c)

	s.AppendErrorNo(ER_NOT_SUPPORTED_YET)

}
func (s *session) checkAlterTableDropIndex(t *TableInfo, indexName string) bool {
	log.Debug("checkAlterTableDropIndex")

	// var rows []*IndexInfo

	// if !t.IsNew {
	// 	// 删除索引时回库查询
	// 	sql := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s` where key_name=?", t.Schema, t.Name)
	// 	if err := s.db.Raw(sql, indexName).Scan(&rows).Error; err != nil {
	// 		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
	// 			s.AppendErrorMessage(myErr.Message)
	// 		} else {
	// 			s.AppendErrorMessage(err.Error())
	// 		}
	// 		return false
	// 	}
	// } else {
	// 	rows = t.Indexes
	// }
	if len(t.Indexes) == 0 {
		s.AppendErrorNo(ER_CANT_DROP_FIELD_OR_KEY, fmt.Sprintf("%s.%s", t.Name, indexName))
		return false
	}

	var foundRows []*IndexInfo
	for _, row := range t.Indexes {
		if row.IndexName == indexName && !row.IsDeleted {
			foundRows = append(foundRows, row)
			row.IsDeleted = true
		}
	}

	if len(foundRows) == 0 {
		s.AppendErrorNo(ER_CANT_DROP_FIELD_OR_KEY, fmt.Sprintf("%s.%s", t.Name, indexName))
		return false
	}

	if s.opt.execute {
		for i, row := range foundRows {
			if i == 0 {
				if indexName == "PRIMARY" {
					s.myRecord.DDLRollback += "ADD PRIMARY KEY("
				} else {
					if row.NonUnique == 0 {
						s.myRecord.DDLRollback += fmt.Sprintf("ADD UNIQUE INDEX `%s`(", indexName)
					} else {
						s.myRecord.DDLRollback += fmt.Sprintf("ADD INDEX `%s`(", indexName)
					}
				}

				s.myRecord.DDLRollback += fmt.Sprintf("`%s`", row.ColumnName)
			} else {
				s.myRecord.DDLRollback += fmt.Sprintf(",`%s`", row.ColumnName)
			}
		}
		s.myRecord.DDLRollback += "),"
	}
	return true
}

func (s *session) checkDropPrimaryKey(t *TableInfo, c *ast.AlterTableSpec) {
	log.Debug("checkDropPrimaryKey")

	s.checkAlterTableDropIndex(t, "PRIMARY")
}

func (s *session) checkAddColumn(t *TableInfo, c *ast.AlterTableSpec) {
	// log.Infof("%s \n", c)
	// log.Infof("%s \n", c.NewColumns)

	for _, nc := range c.NewColumns {
		// log.Infof("%s \n", nc)
		found := false
		for _, field := range t.Fields {
			if strings.EqualFold(field.Field, nc.Name.Name.O) && !field.IsDeleted {
				found = true
				break
			}
		}
		if found {
			s.AppendErrorNo(ER_COLUMN_EXISTED, fmt.Sprintf("%s.%s", t.Name, nc.Name.Name))
		} else {
			s.mysqlCheckField(t, nc)

			newColumn := s.buildNewColumnToCache(t, nc)

			// 在新的快照上变更表结构
			t := s.cacheTableSnapshot(t)
			t.IsNewColumns = true

			if c.Position.Tp == ast.ColumnPositionFirst {
				tmp := make([]FieldInfo, 0, len(t.Fields)+1)
				tmp = append(tmp, *newColumn)
				tmp = append(tmp, t.Fields...)
				t.Fields = tmp

			} else if c.Position.Tp == ast.ColumnPositionAfter {
				foundIndex := -1
				for i, field := range t.Fields {
					if strings.EqualFold(field.Field, c.Position.RelativeColumn.Name.O) {
						foundIndex = i
						break
					}
				}
				if foundIndex == -1 {
					s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
						fmt.Sprintf("%s.%s", t.Name, c.Position.RelativeColumn.Name))
				} else if foundIndex == len(t.Fields)-1 {
					t.Fields = append(t.Fields, *newColumn)
				} else {
					tmp := make([]FieldInfo, 0, len(t.Fields)+1)
					tmp = append(tmp, t.Fields[:foundIndex+1]...)
					tmp = append(tmp, *newColumn)
					tmp = append(tmp, t.Fields[foundIndex+1:]...)
					t.Fields = tmp
					// log.Infof("%#v", t.Fields)
				}
			} else {
				t.Fields = append(t.Fields, *newColumn)
			}

			if s.opt.execute {
				s.myRecord.DDLRollback += fmt.Sprintf("DROP COLUMN `%s`,",
					nc.Name.Name.O)
			}
		}
	}
}

// checkExistsColumns 获取总列数,以避免删除最后一列
func checkExistsColumns(t *TableInfo) (count int) {
	for _, field := range t.Fields {
		if !field.IsDeleted {
			count++
		}
	}
	return
}

func (s *session) checkDropColumn(t *TableInfo, c *ast.AlterTableSpec) {

	found := false
	for i, field := range t.Fields {
		if strings.EqualFold(field.Field, c.OldColumnName.Name.O) && !field.IsDeleted {
			found = true
			s.mysqlDropColumnRollback(field)

			if checkExistsColumns(t) > 1 {
				// 在新的快照上删除字段
				newTable := s.cacheTableSnapshot(t)
				(&(newTable.Fields[i])).IsDeleted = true
			} else {
				s.AppendErrorNo(ErrCantRemoveAllFields)
			}

			break
		}
	}
	if !found {
		s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
			fmt.Sprintf("%s.%s", t.Name, c.OldColumnName.Name.O))
	}
}

// cacheTableSnapshot 保存表快照,用以解析binlog
// 当删除列,变更列顺序时, 重新保存表结构
func (s *session) cacheTableSnapshot(t *TableInfo) *TableInfo {
	newT := s.copyTableInfo(t)

	s.cacheNewTable(newT)

	newT.IsNew = t.IsNew

	return newT
}

func (s *session) mysqlDropColumnRollback(field FieldInfo) {
	if s.opt.check {
		return
	}

	buf := bytes.NewBufferString("ADD COLUMN `")
	buf.WriteString(field.Field)
	buf.WriteString("` ")
	buf.WriteString(field.Type)
	if field.Null == "NO" {
		buf.WriteString(" NOT NULL")
	}
	if field.Default != nil {
		buf.WriteString(" DEFAULT '")
		buf.WriteString(*field.Default)
		buf.WriteString("'")
	}
	if field.Comment != "" {
		buf.WriteString(" COMMENT '")
		buf.WriteString(field.Comment)
		buf.WriteString("'")
	}
	buf.WriteString(",")

	s.myRecord.DDLRollback += buf.String()

}

func (s *session) checkDropIndex(node *ast.DropIndexStmt, sql string) {
	log.Debug("checkDropIndex")
	// log.Infof("%#v \n", node)

	t := s.getTableFromCache(node.Table.Schema.O, node.Table.Name.O, true)
	if t == nil {
		return
	}

	s.checkAlterTableDropIndex(t, node.IndexName)

}

func (s *session) checkCreateIndex(table *ast.TableName, IndexName string,
	IndexColNames []*ast.IndexColName, IndexOption *ast.IndexOption,
	t *TableInfo, unique bool, tp ast.ConstraintType) {
	log.Debug("checkCreateIndex")

	if t == nil {
		t = s.getTableFromCache(table.Schema.O, table.Name.O, true)
		if t == nil {
			return
		}
	}

	s.checkIndexAttr(tp, IndexName, IndexColNames, t)

	keyMaxLen := 0
	// 禁止使用blob列当索引,所以不再检测blob字段时列是否过长
	isBlobColumn := false

	for _, col := range IndexColNames {
		found := false
		var foundField FieldInfo

		for _, field := range t.Fields {
			if strings.EqualFold(field.Field, col.Column.Name.O) {
				found = true
				foundField = field
				break
			}
		}
		if !found {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", t.Name, col.Column.Name.O))
		} else {

			if strings.ToLower(foundField.Type) == "json" {
				s.AppendErrorMessage(
					fmt.Sprintf("JSON column '%-.192s' cannot be used in key specification.", foundField.Field))
			}

			if strings.Contains(strings.ToLower(foundField.Type), "blob") {
				isBlobColumn = true
				s.AppendErrorNo(ER_BLOB_USED_AS_KEY, foundField.Field)
			}

			maxLength := foundField.GetDataBytes(s.DBVersion)

			// Length must be specified for BLOB and TEXT column indexes.
			// if types.IsTypeBlob(col.FieldType.Tp) && ic.Length == types.UnspecifiedLength {
			// 	return nil, errors.Trace(errBlobKeyWithoutLength)
			// }

			if col.Length != types.UnspecifiedLength {
				if !strings.Contains(strings.ToLower(foundField.Type), "blob") &&
					!strings.Contains(strings.ToLower(foundField.Type), "char") &&
					!strings.Contains(strings.ToLower(foundField.Type), "text") {
					s.AppendErrorNo(ER_WRONG_SUB_KEY)
					col.Length = types.UnspecifiedLength
				}

				if (strings.Contains(strings.ToLower(foundField.Type), "char") ||
					strings.Contains(strings.ToLower(foundField.Type), "text")) &&
					col.Length > maxLength {
					s.AppendErrorNo(ER_WRONG_SUB_KEY)
					col.Length = maxLength
				}
			}

			if col.Length == types.UnspecifiedLength {
				keyMaxLen += maxLength
			} else {
				// log.Info(foundField)
				if foundField.Collation == "" || strings.HasPrefix(foundField.Collation, "utf8mb4") {
					keyMaxLen += col.Length * 4
				} else {
					keyMaxLen += col.Length * 3
				}
			}

			if tp == ast.ConstraintPrimaryKey {
				if strings.Contains(strings.ToLower(foundField.Type), "int") {
					s.AppendErrorNo(ER_PK_COLS_NOT_INT, foundField.Field, t.Schema, t.Name)
				}

				if foundField.Null == "YES" {
					s.AppendErrorNo(ER_PRIMARY_CANT_HAVE_NULL)
				}
			}
		}
	}

	if len(IndexName) > mysql.MaxIndexIdentifierLen {
		s.AppendErrorMessage(fmt.Sprintf("表'%s'的索引'%s'名称过长", t.Name, IndexName))
	}

	if !isBlobColumn {
		mysqlVersion := s.DBVersion
		// mysql 5.6版本索引长度限制是767,5.7及之后变为3072
		if mysqlVersion < 50700 && keyMaxLen > MaxKeyLength {
			s.AppendErrorNo(ER_TOO_LONG_KEY, IndexName, MaxKeyLength)
		} else if (mysqlVersion >= 50700) && keyMaxLen > MaxKeyLength57 {
			s.AppendErrorNo(ER_TOO_LONG_KEY, IndexName, MaxKeyLength57)
		}
	}

	if IndexOption != nil {
		// 注释长度校验
		if len(IndexOption.Comment) > INDEX_COMMENT_MAXLEN {
			s.AppendErrorNo(ER_TOO_LONG_INDEX_COMMENT, IndexName, INDEX_COMMENT_MAXLEN)
		}
	}

	rows := t.Indexes

	if len(rows) > 0 {
		for _, row := range rows {
			if strings.EqualFold(row.IndexName, IndexName) && !row.IsDeleted {
				s.AppendErrorNo(ER_DUP_INDEX, IndexName, t.Schema, t.Name)
				break
			}
		}
	}

	key_count := 0
	for _, row := range rows {
		if row.Seq == 1 && !row.IsDeleted {
			key_count += 1
		}
	}

	if s.Inc.MaxKeys > 0 && key_count >= int(s.Inc.MaxKeys) {
		s.AppendErrorNo(ER_TOO_MANY_KEYS, t.Name, s.Inc.MaxKeys)
	}
	// }

	if s.hasError() {
		return
	}

	// cache new index
	for i, col := range IndexColNames {
		index := &IndexInfo{
			Table: t.Name,
			// NonUnique:  unique  ,
			IndexName:  IndexName,
			Seq:        i + 1,
			ColumnName: col.Column.Name.O,
			IndexType:  "BTREE",
		}
		if unique {
			index.NonUnique = 1
		}
		t.Indexes = append(t.Indexes, index)
	}

	if !t.IsNew && s.opt.execute {
		if IndexName == "PRIMARY" {
			s.myRecord.DDLRollback += fmt.Sprintf("DROP PRIMARY KEY,")
		} else {
			s.myRecord.DDLRollback += fmt.Sprintf("DROP INDEX `%s`,", IndexName)
		}
	}
}

func (s *session) checkAddConstraint(t *TableInfo, c *ast.AlterTableSpec) {
	log.Debug("checkAddConstraint")

	// log.Infof("%s \n", c.Constraint)
	// log.Infof("%s \n", c.Constraint.Keys)

	switch c.Constraint.Tp {
	case ast.ConstraintKey, ast.ConstraintIndex, ast.ConstraintUniq, ast.ConstraintUniqIndex,
		ast.ConstraintUniqKey:
		s.checkCreateIndex(nil, c.Constraint.Name,
			c.Constraint.Keys, c.Constraint.Option, t, false, c.Constraint.Tp)

	case ast.ConstraintPrimaryKey:
		s.checkCreateIndex(nil, "PRIMARY",
			c.Constraint.Keys, c.Constraint.Option, t, true, c.Constraint.Tp)

	default:
		s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
		log.Info("未定义的解析: ", c.Constraint.Tp)
	}
}

func (s *session) checkDBExists(db string, reportNotExists bool) bool {

	if db == "" {
		db = s.DBName
	}

	if db == "" {
		s.AppendErrorNo(ER_WRONG_DB_NAME, "")
		return false
	}

	if v, ok := s.dbCacheList[strings.ToLower(db)]; ok {
		return v
	}

	sql := "show databases like '%s';"

	// count:= s.db.Exec(fmt.Sprintf(sql,db)).AffectedRows
	var name string

	rows, err := s.db.Raw(fmt.Sprintf(sql, db)).Rows()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		log.Error(err)
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
		s.dbCacheList[strings.ToLower(db)] = true
		return true
	}

}

func (s *session) checkInsert(node *ast.InsertStmt, sql string) {

	log.Debug("checkInsert")

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

	table := s.getTableFromCache(t.Schema.O, t.Name.O, true)
	if table == nil {
		return
	}

	// 校验列是否重复指定
	if fieldCount > 0 {
		checkDup := map[string]bool{}
		for _, c := range x.Columns {
			if _, ok := checkDup[c.Name.L]; ok {
				s.AppendErrorNo(ER_FIELD_SPECIFIED_TWICE, c.Name, c.Table)
			}
			checkDup[c.Name.L] = true
		}
	}

	s.myRecord.TableInfo = table

	columnsCannotNull := map[string]bool{}

	for _, c := range x.Columns {
		found := false
		for _, field := range table.Fields {
			if strings.EqualFold(field.Field, c.Name.O) && !field.IsDeleted {
				found = true

				if field.Null == "NO" {
					columnsCannotNull[c.Name.L] = true
				}
				break
			}
		}
		if !found {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", c.Table, c.Name))
		}
	}

	if len(x.Lists) > 0 {
		if fieldCount == 0 {
			fieldCount = len(table.Fields)
		}
		for i, list := range x.Lists {
			if len(list) == 0 {
				s.AppendErrorNo(ER_WITH_INSERT_VALUES)
			} else if len(list) != fieldCount {
				s.AppendErrorNo(ER_WRONG_VALUE_COUNT_ON_ROW, i+1)
			} else if len(x.Columns) > 0 {
				for colIndex, vv := range list {
					if v, ok := vv.(*ast.ValueExpr); ok {
						name := x.Columns[colIndex].Name.L
						if _, ok := columnsCannotNull[name]; ok && v.Type.Tp == mysql.TypeNull {
							s.AppendErrorNo(ER_BAD_NULL_ERROR, x.Columns[colIndex], i+1)
						}
					}
				}
				// if i == 0 {

				// 	for _, c := range x.Columns {
				// 		found := false
				// 		for _, field := range table.Fields {
				// 			if strings.EqualFold(field.Field, c.Name.O) {
				// 				found = true
				// 				break
				// 			}
				// 		}
				// 		if !found {
				// 			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, fmt.Sprintf("%s.%s", c.Table, c.Name))
				// 		}
				// 	}
				// }
			}
		}
		s.myRecord.AffectedRows = len(x.Lists)
	} else if x.Select == nil {
		s.AppendErrorNo(ER_WITH_INSERT_VALUES)
	}

	// insert select 语句
	if x.Select != nil {
		// log.Info(x.Select)
		// log.Infof("%#v", x.Select)

		sel, ok := x.Select.(*ast.SelectStmt)
		if !ok {
			if u, ok := x.Select.(*ast.UnionStmt); ok {
				sel = u.SelectList.Selects[0]
			}
		}

		if sel != nil {

			// 只考虑insert select单表时,表不存在的情况
			from := getSingleTableName(sel.From)
			var fromTable *TableInfo
			if from != nil {
				fromTable = s.getTableFromCache(from.Schema.O, from.Name.O, true)
			}

			isWildCard := false
			if len(sel.Fields.Fields) > 0 {
				f := sel.Fields.Fields[0]
				if f.WildCard != nil {
					isWildCard = true
				}
			}

			if isWildCard {
				s.AppendErrorNo(ER_SELECT_ONLY_STAR)
			}

			// 判断字段数是否匹配, *星号时跳过
			if fieldCount > 0 && !isWildCard && len(sel.Fields.Fields) != fieldCount {
				s.AppendErrorNo(ER_WRONG_VALUE_COUNT_ON_ROW, 1)
			}

			// if len(sel.Fields.Fields) > 0 {
			// 	for _, f := range sel.Fields.Fields {
			// 		// if c, ok := f.Expr.(*ast.ColumnNameExpr); ok {
			// 		// 	// log.Info(c.Name)
			// 		// }
			// 		if f.Expr == nil {
			// 			log.Info(node.Text())
			// 			log.Info("Expr is NULL", f.WildCard, f.Expr, f.AsName)
			// 			log.Info(f.Text())
			// 			log.Info("是个*号")
			// 		}
			// 	}
			// }

			if !s.hasError() {
				// s.checkSelectItem(sel)
				s.checkSelectItem(x.Select)
			}

			if from == nil || (fromTable != nil && !fromTable.IsNew) {
				i := strings.Index(strings.ToLower(sql), "select")
				selectSql := sql[i:]

				s.explainOrAnalyzeSql(selectSql)

				if from == nil && s.myRecord.AffectedRows == 0 {
					s.myRecord.AffectedRows = 1
				}
			}

			if sel.Where == nil {
				s.AppendErrorNo(ER_NO_WHERE_CONDITION)
			}

			if sel.Limit != nil {
				s.AppendErrorNo(ER_WITH_LIMIT_CONDITION)
			}

			if sel.OrderBy != nil {
				for _, item := range sel.OrderBy.Items {
					if f, ok := item.Expr.(*ast.FuncCallExpr); ok {
						if f.FnName.L == "rand" {
							s.AppendErrorNo(ER_ORDERY_BY_RAND)
						}
					}
				}
			}
		}
	}

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
	log.Debug("checkDropDB")

	if !s.Inc.EnableDropDatabase {
		s.AppendErrorNo(ER_CANT_DROP_DATABASE, node.Name)
		return
	}

	if s.checkDBExists(node.Name, !node.IfExists) {
		// if s.opt.execute {
		// 	// 生成回滚语句
		// 	s.mysqlShowCreateDatabase(node.Name)
		// }
		s.dbCacheList[strings.ToLower(node.Name)] = false
	}
}

func (s *session) executeInceptionSet(node *ast.InceptionSetStmt, sql string) ([]ast.RecordSet, error) {
	log.Debug("executeInceptionSet")

	for _, v := range node.Variables {
		if !v.IsSystem {
			return nil, errors.New("无效参数")
		}

		var value *ast.ValueExpr

		switch expr := v.Value.(type) {
		case *ast.ValueExpr:
			value = expr
		case *ast.UnaryOperationExpr:
			value, _ = expr.V.(*ast.ValueExpr)
			if expr.Op == opcode.Minus {
				value.Datum = types.NewIntDatum(value.GetInt64() * -1)
			}
		default:
			return nil, errors.New("参数值无效")
		}

		cnf := config.GetGlobalConfig()

		// t := reflect.TypeOf(cnf.Inc)
		// values := reflect.ValueOf(&cnf.Inc).Elem()
		prefix := strings.ToLower(v.Name)
		if strings.Contains(prefix, "_") {
			prefix = strings.Split(prefix, "_")[0]
		}

		var err error
		switch prefix {
		case "osc":
			err = s.setVariableValue(reflect.TypeOf(cnf.Osc), reflect.ValueOf(&cnf.Osc).Elem(), v.Name, value)
			if err != nil {
				return nil, err
			}
		case "ghost":
			err = s.setVariableValue(reflect.TypeOf(cnf.Ghost), reflect.ValueOf(&cnf.Ghost).Elem(), v.Name, value)
			if err != nil {
				return nil, err
			}
		default:
			err = s.setVariableValue(reflect.TypeOf(cnf.Inc), reflect.ValueOf(&cnf.Inc).Elem(), v.Name, value)
			if err != nil {
				return nil, err
			}

			// 错误信息语言设置
			if prefix == "lang" {
				SetLanguage(value.GetString())
			}
		}
	}

	return nil, nil
}

func (s *session) setVariableValue(t reflect.Type, values reflect.Value,
	name string, value *ast.ValueExpr) error {

	// t := reflect.TypeOf(*(obj))
	// // values := reflect.ValueOf(obj).Elem()
	// values := reflect.ValueOf(obj).Elem()

	found := false
	for i := 0; i < values.NumField(); i++ {
		if values.Field(i).CanInterface() { //判断是否为可导出字段
			if k := t.Field(i).Tag.Get("toml"); strings.EqualFold(k, name) ||
				strings.EqualFold(t.Field(i).Name, name) {
				err := s.setConfigValue(name, values.Field(i), &(value.Datum))
				if err != nil {
					return err
				}
				found = true
				break
			}
		}
	}
	if !found {
		return errors.New("无效参数")
	}
	return nil
}

func (s *session) checkUInt64SystemVar(name, value string, min, max uint64) (string, error) {
	if value[0] == '-' {
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return value, ErrWrongTypeForVar.GenWithStackByArgs(name)
		}
		s.sessionVars.StmtCtx.AppendWarning(ErrTruncatedWrongValue.GenWithStackByArgs(name, value))
		return fmt.Sprintf("%d", min), nil
	}
	val, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return value, ErrWrongTypeForVar.GenWithStackByArgs(name)
	}
	if val < min {
		s.sessionVars.StmtCtx.AppendWarning(ErrTruncatedWrongValue.GenWithStackByArgs(name, value))
		return fmt.Sprintf("%d", min), nil
	}
	if val > max {
		s.sessionVars.StmtCtx.AppendWarning(ErrTruncatedWrongValue.GenWithStackByArgs(name, value))
		return fmt.Sprintf("%d", max), nil
	}
	return value, nil
}

func (s *session) checkInt64SystemVar(name, value string, min, max int64) (string, error) {
	val, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return value, ErrWrongTypeForVar.GenWithStackByArgs(name)
	}
	if val < min {
		s.sessionVars.StmtCtx.AppendWarning(ErrTruncatedWrongValue.GenWithStackByArgs(name, value))
		return fmt.Sprintf("%d", min), nil
	}
	if val > max {
		s.sessionVars.StmtCtx.AppendWarning(ErrTruncatedWrongValue.GenWithStackByArgs(name, value))
		return fmt.Sprintf("%d", max), nil
	}
	return value, nil
}

func (s *session) setConfigValue(name string, field reflect.Value, value *types.Datum) error {

	sVal := ""
	var err error
	if !value.IsNull() {
		sVal, err = value.ToString()
	}
	if err != nil {
		return err
	}

	switch field.Type().String() {
	case reflect.String.String():
		field.SetString(sVal)

	case reflect.Uint.String():
		// field.SetUint(value.GetUint64())
		v, err := s.checkUInt64SystemVar(name, sVal, 0, math.MaxUint64)
		if err != nil {
			return err
		}

		v1, _ := strconv.ParseUint(v, 10, 64)
		field.SetUint(v1)

	case reflect.Int.String(), reflect.Int64.String():
		// field.SetInt(value.GetInt64())
		v, err := s.checkInt64SystemVar(name, sVal, math.MinInt64, math.MaxInt64)
		if err != nil {
			return err
		}

		v1, _ := strconv.ParseInt(v, 10, 64)
		field.SetInt(v1)

	case reflect.Bool.String():
		if strings.EqualFold(sVal, "ON") || sVal == "1" ||
			strings.EqualFold(sVal, "OFF") || sVal == "0" ||
			strings.EqualFold(sVal, "TRUE") || strings.EqualFold(sVal, "FALSE") {
			if strings.EqualFold(sVal, "ON") || sVal == "1" || strings.EqualFold(sVal, "TRUE") {
				field.SetBool(true)
			} else {
				field.SetBool(false)
			}
		} else {
			// s.sessionVars.StmtCtx.AppendError(ErrWrongValueForVar.GenWithStackByArgs(name, sVal))
			return ErrWrongValueForVar.GenWithStackByArgs(name, sVal)
		}
	default:
		field.SetString(sVal)
	}
	return nil
}

func (s *session) showVariables(node *ast.ShowStmt, obj interface{}, res *VariableSets) {

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	var (
		like     string
		patChars []byte
		patTypes []byte
	)
	if node.Pattern != nil {
		if node.Pattern.Pattern != nil {
			va, _ := node.Pattern.Pattern.(*ast.ValueExpr)
			like = va.GetString()
		}
		patChars, patTypes = stringutil.CompilePattern(like, node.Pattern.Escape)
	}

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).CanInterface() { //判断是否为可导出字段
			if len(like) == 0 {
				if k := t.Field(i).Tag.Get("toml"); k != "" {
					if k == "backup_password" {
						p := auth.EncodePassword(
							fmt.Sprintf("%v", v.Field(i).Interface()))
						res.Append(k, p)
					} else {
						res.Append(k, fmt.Sprintf("%v", v.Field(i).Interface()))
					}
				}
			} else {
				if k := t.Field(i).Tag.Get("toml"); k != "" {
					match := stringutil.DoMatch(k, patChars, patTypes)
					if match && !node.Pattern.Not {
						if k == "backup_password" {
							p := auth.EncodePassword(
								fmt.Sprintf("%v", v.Field(i).Interface()))
							res.Append(k, p)
						} else {
							res.Append(k, fmt.Sprintf("%v", v.Field(i).Interface()))
						}
					} else if !match && node.Pattern.Not {
						if k == "backup_password" {
							p := auth.EncodePassword(
								fmt.Sprintf("%v", v.Field(i).Interface()))
							res.Append(k, p)
						} else {
							res.Append(k, fmt.Sprintf("%v", v.Field(i).Interface()))
						}
					}
				}
			}
		}
	}
}

func (s *session) executeLocalShowVariables(node *ast.ShowStmt) ([]ast.RecordSet, error) {

	res := NewVariableSets(120)
	s.showVariables(node, s.Inc, res)
	s.showVariables(node, s.Osc, res)
	s.showVariables(node, s.Ghost, res)

	s.sessionVars.StmtCtx.AddAffectedRows(uint64(res.rc.count))

	return res.Rows(), nil

}

func (s *session) executeLocalShowProcesslist(node *ast.ShowStmt) ([]ast.RecordSet, error) {
	pl := s.sessionManager.ShowProcessList()

	var keys []int
	for k := range pl {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	res := NewProcessListSets(len(pl))

	for _, k := range keys {
		pi := pl[uint64(k)]

		var info string
		if node.Full {
			info = pi.Info
		} else {
			info = fmt.Sprintf("%.100v", pi.Info)
		}

		data := []interface{}{
			pi.ID,
			pi.DestUser,
			pi.DestHost,
			pi.DestPort,
			pi.Host,
			pi.Command,
			pi.OperState,
			int64(time.Since(pi.Time) / time.Second),
			info,
		}
		if pi.Percent > 0 {
			data = append(data, fmt.Sprintf("%.2f%%", pi.Percent*100))
		}
		res.appendRow(data)
	}

	s.sessionVars.StmtCtx.AddAffectedRows(uint64(res.rc.count))
	return res.Rows(), nil
}

func (s *session) executeLocalShowOscProcesslist(node *ast.ShowOscStmt) ([]ast.RecordSet, error) {
	pl := s.sessionManager.ShowOscProcessList()

	// 根据是否指定sqlsha1控制显示command列
	res := NewOscProcessListSets(len(pl), node.Sqlsha1 != "")

	if node.Sqlsha1 == "" {
		for _, pi := range pl {
			data := []interface{}{
				pi.Schema,
				pi.Table,
				pi.Command,
				pi.Sqlsha1,
				pi.Percent,
				pi.RemainTime,
				pi.Info,
			}
			res.appendRow(data)
		}
	} else if pi, ok := pl[node.Sqlsha1]; ok {
		data := []interface{}{
			pi.Schema,
			pi.Table,
			// pi.Command,
			pi.Sqlsha1,
			pi.Percent,
			pi.RemainTime,
			pi.Info,
		}
		res.appendRow(data)
	} else {
		s.sessionVars.StmtCtx.AppendWarning(errors.New("osc process not found"))
	}

	s.sessionVars.StmtCtx.AddAffectedRows(uint64(res.rc.count))
	return res.Rows(), nil
}

func (s *session) executeLocalOscKill(node *ast.ShowOscStmt) ([]ast.RecordSet, error) {
	pl := s.sessionManager.ShowOscProcessList()

	if pi, ok := pl[node.Sqlsha1]; ok {
		if pi.Killed {
			s.sessionVars.StmtCtx.AppendWarning(errors.New("osc process has been aborted"))
		} else {
			pi.Killed = true
		}
	} else {
		return nil, errors.New("osc process not found")
	}

	return nil, nil
}

func (s *session) executeLocalOscPause(node *ast.ShowOscStmt) ([]ast.RecordSet, error) {
	pl := s.sessionManager.ShowOscProcessList()

	if pi, ok := pl[node.Sqlsha1]; ok {
		if !pi.IsGhost {
			return nil, errors.New("pt-osc process not support pause")
		}

		if pi.Pause {
			s.sessionVars.StmtCtx.AppendWarning(errors.New("osc process has been paused"))
		} else {
			pi.Pause = true
		}
	} else {
		return nil, errors.New("osc process not found")
	}

	return nil, nil
}

func (s *session) executeLocalOscResume(node *ast.ShowOscStmt) ([]ast.RecordSet, error) {
	pl := s.sessionManager.ShowOscProcessList()

	if pi, ok := pl[node.Sqlsha1]; ok {
		if !pi.IsGhost {
			return nil, errors.New("pt-osc process not support resume")
		}

		if pi.Pause {
			pi.Pause = false
		} else {
			s.sessionVars.StmtCtx.AppendWarning(errors.New("osc process not paused"))
		}
	} else {
		return nil, errors.New("osc process not found")
	}

	return nil, nil
}

func (s *session) executeInceptionShow(sql string) ([]ast.RecordSet, error) {
	log.Debug("executeInceptionShow")

	rows, err := s.db.Raw(sql).Rows()
	if rows != nil {
		defer rows.Close()
	}
	if err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		}
	} else if rows != nil {

		cols, _ := rows.Columns()
		colLength := len(cols)

		var buf strings.Builder
		buf.WriteString(sql)
		buf.WriteString(":\n")

		paramValues := strings.Repeat("? | ", colLength)
		paramValues = strings.TrimRight(paramValues, " | ")

		for rows.Next() {
			// https://kylewbanks.com/blog/query-result-to-map-in-golang
			// Create a slice of interface{}'s to represent each column,
			// and a second slice to contain pointers to each item in the columns slice.
			columns := make([]interface{}, colLength)
			columnPointers := make([]interface{}, colLength)
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			// Scan the result into the column pointers...
			if err := rows.Scan(columnPointers...); err != nil {
				s.AppendErrorMessage(err.Error())
				return nil, nil
			}

			var vv []driver.Value
			for i := range cols {
				val := columnPointers[i].(*interface{})
				vv = append(vv, *val)
			}

			res, err := InterpolateParams(paramValues, vv)
			if err != nil {
				s.AppendErrorMessage(err.Error())
				return nil, nil
			}

			buf.Write(res)
			buf.WriteString("\n")
		}
		s.myRecord.Sql = strings.TrimSpace(buf.String())
	}

	return nil, nil
}

func (s *session) checkCreateDB(node *ast.CreateDatabaseStmt) {
	log.Debug("checkCreateDB")

	if s.checkDBExists(node.Name, false) {
		if !node.IfNotExists {
			s.AppendErrorMessage(fmt.Sprintf("数据库'%s'已存在.", node.Name))
		}
	} else {
		s.checkKeyWords(node.Name)

		for _, opt := range node.Options {
			switch opt.Tp {
			case ast.DatabaseOptionCharset:
				if !s.Inc.EnableSetCharset {
					s.AppendErrorNo(ER_CANT_SET_CHARSET, opt.Value)
				}

				s.checkCharset(opt.Value)
			case ast.DatabaseOptionCollate:
				s.AppendErrorNo(ER_CANT_SET_COLLATION, opt.Value)
			}
		}

		if s.hasError() {
			return
		}

		s.dbCacheList[strings.ToLower(node.Name)] = true

		// if s.opt.execute {
		// 	s.myRecord.DDLRollback = fmt.Sprintf("DROP DATABASE `%s`;", node.Name)
		// }
	}
}

func (s *session) checkCharset(charset string) bool {
	if s.Inc.SupportCharset != "" {
		for _, item := range strings.Split(s.Inc.SupportCharset, ",") {
			if strings.EqualFold(item, charset) {
				return true
			}
		}
		s.AppendErrorNo(ER_NAMES_MUST_UTF8, s.Inc.SupportCharset)
		return false
	}
	return true
}

func (s *session) checkChangeDB(node *ast.UseStmt) {
	log.Debug("checkChangeDB")

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

func (s *session) getExplainInfo(sql string) {

	// selectType   string  `gorm:"Column:select_type"`
	// table        string  `gorm:"Column:table"`
	// partitions   string  `gorm:"Column:partitions"`
	// type         string  `gorm:"Column:type"`
	// possibleKeys string  `gorm:"Column:possible_keys"`
	// key          string  `gorm:"Column:key"`
	// keyLen       string  `gorm:"Column:key_len"`
	// ref          string  `gorm:"Column:ref"`
	// rows         int     `gorm:"Column:rows"`
	// filtered     float32 `gorm:"Column:filtered"`
	// extra        string  `gorm:"Column:Extra"`

	rows, err := s.db.DB().Query(sql)
	defer rows.Close()

	var rowLength int

	if err != nil {
		log.Error(err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.AppendErrorMessage(myErr.Message)
		}
	} else {
		for rows.Next() {
			var str interface{}
			// | id | select_type | table | partitions | type  | possible_keys | key     | key_len | ref   | rows | filtered | Extra
			if err := rows.Scan(&str, &str, &str, &str, &str, &str, &str, &str, &str, &rowLength, &str, &str); err != nil {
				log.Error(err)
				if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
					s.AppendErrorMessage(myErr.Message)
				}
			}
			break
		}
		rows.Close()
	}

	r := s.myRecord
	r.AffectedRows = rowLength

	if s.Inc.MaxUpdateRows > 0 && r.AffectedRows >= int(s.Inc.MaxUpdateRows) {
		switch r.Type.(type) {
		case *ast.DeleteStmt, *ast.UpdateStmt:
			s.AppendErrorNo(ER_UDPATE_TOO_MUCH_ROWS, s.Inc.MaxUpdateRows)
		}
	}

	// var rows []ExplainInfo
	// if err := s.db.Raw(sql).Scan(&rows).Error; err != nil {
	// 	log.Error(err)
	// 	if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
	// 		s.AppendErrorMessage(myErr.Message)
	// 	} else {
	// 		s.AppendErrorMessage(err.Error())
	// 	}
	// }
	// return rows
}

func (s *session) explainOrAnalyzeSql(sql string) {

	// 如果没有表结构,或者新增表 or 新增列时,不做explain
	if s.myRecord.TableInfo == nil || s.myRecord.TableInfo.IsNew ||
		s.myRecord.TableInfo.IsNewColumns {
		return
	}

	var explain []string

	if s.isMiddleware() {
		explain = append(explain, s.opt.middlewareExtend)
	}

	explain = append(explain, "EXPLAIN ")
	explain = append(explain, sql)

	// rows := s.getExplainInfo(strings.Join(explain, ""))
	s.getExplainInfo(strings.Join(explain, ""))

	// s.AnlyzeExplain(rows)
}

func (s *session) AnlyzeExplain(rows []ExplainInfo) {
	r := s.myRecord
	if len(rows) > 0 {
		r.AffectedRows = rows[0].Rows
	}
	if s.Inc.MaxUpdateRows > 0 && r.AffectedRows >= int(s.Inc.MaxUpdateRows) {
		switch r.Type.(type) {
		case *ast.DeleteStmt, *ast.UpdateStmt:
			s.AppendErrorNo(ER_UDPATE_TOO_MUCH_ROWS, s.Inc.MaxUpdateRows)
		}
	}
}

func (s *session) checkUpdate(node *ast.UpdateStmt, sql string) {
	log.Debug("checkUpdate")

	// 从set列表读取要更新的表
	var originTable string
	var firstColumnName string
	if node.List != nil {
		for _, l := range node.List {
			originTable = l.Column.Table.L
			firstColumnName = l.Column.Name.O
			break
		}
	}

	var tableList []*ast.TableSource
	tableList = extractTableList(node.TableRefs.TableRefs, tableList)

	var tableInfoList []*TableInfo

	catchError := false
	for _, tblSource := range tableList {
		tblName, ok := tblSource.Source.(*ast.TableName)
		if !ok {
			continue
		}
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
			} else if originTable == tblName.Name.L || originTable == tblSource.AsName.L {
				s.myRecord.TableInfo = t
			}
		}

		if t != nil {
			if tblSource.AsName.L != "" {
				t.AsName = tblSource.AsName.O
			}
			tableInfoList = append(tableInfoList, t)
		}

		// if i == len(tableList) - 1 && s.myRecord.TableInfo == nil {
		// 	s.myRecord.TableInfo = t
		// }
	}

	if !catchError && s.myRecord.TableInfo == nil {
		if originTable == "" {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, firstColumnName)
		} else {
			s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
				fmt.Sprintf("%s.%s", originTable, firstColumnName))
		}
	} else if !catchError && (s.myRecord.TableInfo.IsNew || s.myRecord.TableInfo.IsNewColumns) {
		for _, l := range node.List {
			found := false
			for _, field := range s.myRecord.TableInfo.Fields {
				if strings.EqualFold(field.Field, l.Column.Name.L) && !field.IsDeleted {
					found = true
					break
				}
			}
			if !found {
				s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
					fmt.Sprintf("%s.%s", s.myRecord.TableInfo.Name, l.Column.Name.L))
			}

			s.checkItem(l.Expr, tableInfoList)
		}
	} else if !catchError {
		s.explainOrAnalyzeSql(sql)
	}

	if node.TableRefs.TableRefs.On != nil {
		s.checkItem(node.TableRefs.TableRefs.On.Expr, tableInfoList)
	}

	s.checkItem(node.Where, tableInfoList)

	if node.Where == nil {
		s.AppendErrorNo(ER_NO_WHERE_CONDITION)
	}

	if node.Limit != nil {
		s.AppendErrorNo(ER_WITH_LIMIT_CONDITION)
	}

	if node.Order != nil {
		s.AppendErrorNo(ER_WITH_ORDERBY_CONDITION)
	}
}

func (s *session) checkItem(expr ast.ExprNode, tables []*TableInfo) bool {

	if expr == nil {
		return true
	}

	switch e := expr.(type) {
	case *ast.BinaryOperationExpr:
		return s.checkItem(e.L, tables) && s.checkItem(e.R, tables)
	case *ast.ColumnNameExpr:
		found := false

		db := e.Name.Schema.L
		if db == "" {
			db = s.DBName
		}

		for _, t := range tables {
			var tName string
			if t.AsName != "" {
				tName = t.AsName
			} else {
				tName = t.Name
			}

			if e.Name.Table.L != "" && strings.EqualFold(t.Schema, db) &&
				(strings.EqualFold(tName, e.Name.Table.L)) ||
				e.Name.Table.L == "" {
				for _, field := range t.Fields {
					if strings.EqualFold(field.Field, e.Name.Name.L) && !field.IsDeleted {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
		if found {
			return true
		} else {
			if e.Name.Table.L == "" {
				s.AppendErrorNo(ER_COLUMN_NOT_EXISTED, e.Name.Name.O)
			} else {
				s.AppendErrorNo(ER_COLUMN_NOT_EXISTED,
					fmt.Sprintf("%s.%s", e.Name.Table.O, e.Name.Name.O))
			}
			return false
		}
	default:
		// log.Infof("checkItem: %#v", e)
		return true
	}
}

func (s *session) checkDelete(node *ast.DeleteStmt, sql string) {
	log.Debug("checkDelete")

	if node.Tables != nil {
		for _, a := range node.Tables.Tables {
			s.myRecord.TableInfo = s.getTableFromCache(a.Schema.O, a.Name.O, true)
			break
		}
	}

	var tableList []*ast.TableSource
	tableList = extractTableList(node.TableRefs.TableRefs, tableList)

	var tableInfoList []*TableInfo
	for _, tblSource := range tableList {
		tblName, _ := tblSource.Source.(*ast.TableName)

		t := s.getTableFromCache(tblName.Schema.O, tblName.Name.O, true)
		if t != nil {
			if tblSource.AsName.L != "" {
				t.AsName = tblSource.AsName.O
			}
			tableInfoList = append(tableInfoList, t)

			if s.myRecord.TableInfo == nil {
				s.myRecord.TableInfo = t
			}
		}
	}

	if node.TableRefs.TableRefs.On != nil {
		s.checkItem(node.TableRefs.TableRefs.On.Expr, tableInfoList)
	}
	// if node.BeforeFrom {
	// 	s.checkItem(node.TableRefs.TableRefs.On.Expr, tableInfoList)
	// }
	if s.myRecord.TableInfo != nil && !s.hasError() {
		s.checkItem(node.Where, tableInfoList)
	}

	if !s.hasError() {
		// 如果没有表结构,或者新增表 or 新增列时,不做explain
		s.explainOrAnalyzeSql(sql)
	}

	if node.Where == nil {
		s.AppendErrorNo(ER_NO_WHERE_CONDITION)
	}

	if node.Limit != nil {
		s.AppendErrorNo(ER_WITH_LIMIT_CONDITION)
	}

	if node.Order != nil {
		s.AppendErrorNo(ER_WITH_ORDERBY_CONDITION)
	}
}

func (s *session) QueryTableFromDB(db string, tableName string, reportNotExists bool) []FieldInfo {
	if db == "" {
		db = s.DBName
	}
	var rows []FieldInfo
	sql := fmt.Sprintf("SHOW FULL FIELDS FROM `%s`.`%s`", db, tableName)

	if err := s.db.Raw(sql).Scan(&rows).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			if myErr.Number != 1146 {
				log.Error(err)
				s.AppendErrorMessage(myErr.Message + ".")
			} else if reportNotExists {
				s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, fmt.Sprintf("%s.%s", db, tableName))
			}
		} else {
			s.AppendErrorMessage(err.Error() + ".")
		}
		return nil
	}
	return rows
}

func (s *session) QueryIndexFromDB(db string, tableName string, reportNotExists bool) []*IndexInfo {
	if db == "" {
		db = s.DBName
	}
	var rows []*IndexInfo
	sql := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", db, tableName)

	if err := s.db.Raw(sql).Scan(&rows).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			if myErr.Number != 1146 {
				log.Error(err)
				s.AppendErrorMessage(myErr.Message + ".")
			} else if reportNotExists {
				s.AppendErrorMessage(myErr.Message + ".")
				// s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, fmt.Sprintf("%s.%s", db, tableName))
			}

		} else {
			s.AppendErrorMessage(err.Error() + ".")
		}
		return nil
	}
	return rows
}

func (r *Record) AppendErrorMessage(msg string) {
	r.ErrLevel = 2

	r.Buf.WriteString(msg)
	r.Buf.WriteString("\n")
}

func (r *Record) AppendErrorNo(number int, values ...interface{}) {
	r.ErrLevel = uint8(Max(int(r.ErrLevel), int(GetErrorLevel(number))))

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
	if s.checkInceptionVariables(number) {
		s.myRecord.AppendErrorNo(number, values...)
		s.recordSets.MaxLevel = uint8(Max(int(s.recordSets.MaxLevel), int(s.myRecord.ErrLevel)))
	}
}

func (s *session) checkKeyWords(name string) {
	if !regIdentified.MatchString(name) {
		s.AppendErrorNo(ER_INVALID_IDENT, name)
	} else if _, ok := Keywords[strings.ToUpper(name)]; ok {
		s.AppendErrorNo(ER_IDENT_USE_KEYWORD, name)
	}

	if len(name) > mysql.MaxTableNameLength {
		s.AppendErrorNo(ER_TOO_LONG_IDENT, name)
	}
}

func (s *session) checkInceptionVariables(number int) bool {
	switch number {
	case ER_WITH_INSERT_FIELD:
		return s.Inc.CheckInsertField

	case ER_NO_WHERE_CONDITION:
		return s.Inc.CheckDMLWhere

	case ER_WITH_LIMIT_CONDITION:
		return s.Inc.CheckDMLLimit

	case ER_WITH_ORDERBY_CONDITION:
		return s.Inc.CheckDMLOrderBy

	case ER_SELECT_ONLY_STAR:
		if s.Inc.EnableSelectStar {
			return false
		}
	case ER_ORDERY_BY_RAND:
		if s.Inc.EnableOrderByRand {
			return false
		}
	case ER_NOT_ALLOWED_NULLABLE:
		if s.Inc.EnableNullable {
			return false
		}

	case ER_FOREIGN_KEY:
		if s.Inc.EnableForeignKey {
			return false
		}
	case ER_USE_TEXT_OR_BLOB:
		if s.Inc.EnableBlobType {
			return false
		}
	case ER_TABLE_MUST_INNODB:
		if s.Inc.EnableNotInnodb {
			return false
		}
	case ER_PK_COLS_NOT_INT:
		return s.Inc.EnablePKColumnsOnlyInt

	case ER_TABLE_MUST_HAVE_COMMENT:
		return s.Inc.CheckTableComment

	case ER_COLUMN_HAVE_NO_COMMENT:
		return s.Inc.CheckColumnComment

	case ER_TABLE_MUST_HAVE_PK:
		return s.Inc.CheckPrimaryKey

	case ER_PARTITION_NOT_ALLOWED:
		if s.Inc.EnablePartitionTable {
			return false
		}
	case ER_USE_ENUM, ER_INVALID_DATA_TYPE:
		if s.Inc.EnableEnumSetBit {
			return false
		}
	case ER_INDEX_NAME_IDX_PREFIX, ER_INDEX_NAME_UNIQ_PREFIX:
		return s.Inc.CheckIndexPrefix

	case ER_AUTOINC_UNSIGNED:
		return s.Inc.EnableAutoIncrementUnsigned

	case ER_INC_INIT_ERR:
		return s.Inc.CheckAutoIncrementInitValue

	case ER_INVALID_IDENT:
		return s.Inc.CheckIdentifier

	case ER_SET_DATA_TYPE_INT_BIGINT:
		return s.Inc.CheckAutoIncrementDataType

	case ER_TIMESTAMP_DEFAULT:
		return s.Inc.CheckTimestampDefault

	case ER_CHARSET_ON_COLUMN:
		if s.Inc.EnableColumnCharset {
			return false
		}
	case ER_IDENT_USE_KEYWORD:
		if s.Inc.EnableIdentiferKeyword {
			return false
		}
	case ER_AUTO_INCR_ID_WARNING:
		return s.Inc.CheckAutoIncrementName

	case ER_ALTER_TABLE_ONCE:
		return s.Inc.MergeAlterTable

	case ER_WITH_DEFAULT_ADD_COLUMN:
		return s.Inc.CheckColumnDefaultValue

	}

	return true
}

func extractTableList(node ast.ResultSetNode, input []*ast.TableSource) []*ast.TableSource {
	if node == nil {
		return input
	}

	switch x := node.(type) {
	case *ast.Join:
		input = extractTableList(x.Left, input)
		input = extractTableList(x.Right, input)
	case *ast.TableSource:
		// if s, ok := x.Source.(*ast.TableName); ok {
		// 	if x.AsName.L != "" {
		// 		newTableName := *s
		// 		newTableName.Name = x.AsName
		// 		s.Name = x.AsName
		// 		input = append(input, &newTableName)
		// 	} else {
		// 		input = append(input, s)
		// 	}
		// }
		input = append(input, x)
	default:
		log.Info(x)
		log.Infof("%#v", x)
	}
	return input
}

func (s *session) getTableFromCache(db string, tableName string, reportNotExists bool) *TableInfo {
	if db == "" {
		db = s.DBName
	}

	if db == "" {
		s.AppendErrorNo(ER_WRONG_DB_NAME, "")
		return nil
	}

	key := fmt.Sprintf("%s.%s", db, tableName)

	if t, ok := s.tableCacheList[key]; ok {
		// 如果表已删除, 之后又使用到,则报错
		if t.IsDeleted {
			if reportNotExists {
				s.AppendErrorNo(ER_TABLE_NOT_EXISTED_ERROR, fmt.Sprintf("%s.%s", t.Schema, t.Name))
			}
			return nil
		}
		t.AsName = ""
		return t
	} else {
		rows := s.QueryTableFromDB(db, tableName, reportNotExists)
		if rows != nil {
			newT := &TableInfo{
				Schema: db,
				Name:   tableName,
				Fields: rows,
			}
			if rows := s.QueryIndexFromDB(db, tableName, reportNotExists); rows != nil {
				newT.Indexes = rows
			}
			s.tableCacheList[key] = newT

			return newT
		}
	}

	return nil
}

func (s *session) cacheNewTable(t *TableInfo) {
	if t.Schema == "" {
		t.Schema = s.DBName
	}
	key := fmt.Sprintf("%s.%s", t.Schema, t.Name)

	t.IsNew = true
	// 如果表删除后新建,直接覆盖即可
	s.tableCacheList[key] = t

	// if t, ok := s.tableCacheList[key]; !ok {
	// 	s.tableCacheList[key] = t
	// }
}

func (s *session) buildNewColumnToCache(t *TableInfo, field *ast.ColumnDef) *FieldInfo {

	c := &FieldInfo{}

	c.Field = field.Name.Name.String()
	c.Type = field.Tp.CompactStr()
	// c.Null = "YES"
	c.Null = ""
	c.Tp = field.Tp

	// if !isExplicitTimeStamp() {
	// 	// Check and set TimestampFlag, OnUpdateNowFlag and NotNullFlag.
	// 	if col.Tp == mysql.TypeTimestamp {
	// 		col.Flag |= mysql.TimestampFlag
	// 		col.Flag |= mysql.OnUpdateNowFlag
	// 		col.Flag |= mysql.NotNullFlag
	// 	}
	// }

	for _, op := range field.Options {
		switch op.Tp {
		case ast.ColumnOptionComment:
			c.Comment = op.Expr.GetDatum().GetString()
		case ast.ColumnOptionNull:
			c.Null = "YES"

			field.Tp.Flag &= ^mysql.NotNullFlag
		case ast.ColumnOptionNotNull:
			c.Null = "NO"

			field.Tp.Flag |= mysql.NotNullFlag
		case ast.ColumnOptionPrimaryKey:
			c.Key = "PRI"

			field.Tp.Flag |= mysql.PriKeyFlag
		case ast.ColumnOptionUniqKey:
			field.Tp.Flag |= mysql.UniqueKeyFlag

		case ast.ColumnOptionDefaultValue:

			if op.Expr.GetDatum().IsNull() {
				c.Null = "YES"
				// *c.Default = "NULL"
				c.Default = nil
			} else {
				c.Default = new(string)
				*c.Default = fmt.Sprint(op.Expr.GetValue())
			}
		case ast.ColumnOptionAutoIncrement:
			if strings.ToLower(c.Field) != "id" {
				s.AppendErrorNo(ER_AUTO_INCR_ID_WARNING, c.Field)
			}
			field.Tp.Flag |= mysql.AutoIncrementFlag
		case ast.ColumnOptionOnUpdate:
			if field.Tp.Tp == mysql.TypeTimestamp || field.Tp.Tp == mysql.TypeDatetime {
				if !expression.IsCurrentTimestampExpr(op.Expr) {
					s.AppendErrorNo(ER_INVALID_ON_UPDATE, c.Field)
				}
			} else {
				s.AppendErrorNo(ER_INVALID_ON_UPDATE, c.Field)
			}
			field.Tp.Flag |= mysql.OnUpdateNowFlag
		}
	}

	if c.Default == nil {
		field.Tp.Flag |= mysql.NoDefaultValueFlag
	}
	c.IsNew = true
	return c
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

func (s *session) copyTableInfo(t *TableInfo) *TableInfo {
	p := &TableInfo{}

	p.Schema = t.Schema
	p.Name = t.Name

	p.Fields = make([]FieldInfo, len(t.Fields))
	copy(p.Fields, t.Fields)

	// 移除已删除的列
	// newFields := make([]FieldInfo, len(t.Fields))
	// copy(newFields, t.Fields)

	// for _, f := range newFields {
	// 	if !f.IsDeleted {
	// 		p.Fields = append(p.Fields, f)
	// 	}
	// }

	if len(t.Indexes) > 0 {
		originIndexes := make([]IndexInfo, 0, len(t.Indexes))
		p.Indexes = make([]*IndexInfo, 0, len(t.Indexes))

		for i := range t.Indexes {
			originIndexes = append(originIndexes, *(t.Indexes[i]))
		}

		newIndexes := make([]IndexInfo, len(t.Indexes))
		copy(newIndexes, originIndexes)

		for i, r := range newIndexes {
			if !r.IsDeleted {
				p.Indexes = append(p.Indexes, &newIndexes[i])
			}
		}
	}

	// querySql := fmt.Sprintf("SHOW INDEX FROM `%s`.`%s`", t.Schema, t.Name)
	// var rows []*IndexInfo
	// if err := s.db.Raw(querySql).Scan(&rows).Error; err != nil {
	// 	if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
	// 		s.AppendErrorMessage(myErr.Message)
	// 	} else {
	// 		s.AppendErrorMessage(err.Error())
	// 	}
	// }
	// p.Indexes = rows

	return p
}

func (s *session) checkSelectItem(node ast.ResultSetNode) bool {
	switch x := node.(type) {
	case *ast.UnionStmt:
		stmt := x.SelectList
		for _, sel := range stmt.Selects[:len(stmt.Selects)-1] {
			if sel.Limit != nil {
				s.AppendErrorNo(ErrWrongUsage, "UNION", "LIMIT")
			}
			if sel.OrderBy != nil {
				s.AppendErrorNo(ErrWrongUsage, "UNION", "ORDER BY")
			}
		}

		for _, sel := range stmt.Selects {
			s.checkSubSelectItem(sel)
		}

	case *ast.SelectStmt:
		s.checkSubSelectItem(x)
	default:
		log.Info(x)
		log.Infof("%#v", x)
	}
	return !s.hasError()
}

func (s *session) checkSubSelectItem(node *ast.SelectStmt) bool {
	log.Debug("checkSubSelectItem")

	var tableList []*ast.TableSource
	if node.From != nil {
		tableList = extractTableList(node.From.TableRefs, tableList)

		s.checkTableAliasDuplicate(node.From.TableRefs, make(map[string]interface{}))
	}

	var tableInfoList []*TableInfo
	for _, tblSource := range tableList {
		tblName, _ := tblSource.Source.(*ast.TableName)
		t := s.getTableFromCache(tblName.Schema.O, tblName.Name.O, true)
		if t != nil {
			if tblSource.AsName.L != "" {
				t.AsName = tblSource.AsName.O
			}
			tableInfoList = append(tableInfoList, t)
		}
	}

	if node.Fields != nil {
		for _, field := range node.Fields.Fields {
			if field.WildCard == nil {
				s.checkItem(field.Expr, tableInfoList)
			}
		}
	}

	if node.GroupBy != nil {
		for _, item := range node.GroupBy.Items {
			s.checkItem(item.Expr, tableInfoList)
		}
	}

	if node.Having != nil {
		s.checkItem(node.Having.Expr, tableInfoList)
	}

	if node.OrderBy != nil {
		for _, item := range node.OrderBy.Items {
			s.checkItem(item.Expr, tableInfoList)
		}
	}

	return !s.hasError()
}

func (s *session) isMiddleware() bool {
	return s.opt.middlewareExtend != ""
}

func (s *session) executeKillStmt(node *ast.KillStmt) ([]ast.RecordSet, error) {
	sm := s.GetSessionManager()
	if sm == nil {
		return nil, nil
	}
	sm.Kill(node.ConnectionID, node.Query)
	// conf := config.GetGlobalConfig()
	// if node.TiDBExtension || conf.CompatibleKillQuery {
	// 	sm := s.GetSessionManager()
	// 	if sm == nil {
	// 		return nil, nil
	// 	}
	// 	sm.Kill(node.ConnectionID, node.Query)
	// } else {
	// 	err := errors.New("Invalid operation. Please use 'KILL TIDB [CONNECTION | QUERY] connectionID' instead")
	// 	s.sessionVars.StmtCtx.AppendWarning(err)
	// }
	return nil, nil
}

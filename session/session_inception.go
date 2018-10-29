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
	"github.com/hanchuanchuan/tidb/mysql"
	// "github.com/hanchuanchuan/tidb/table"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"regexp"
	"strings"

	// "database/sql/driver"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	// _ "github.com/jinzhu/gorm/dialects/mysql"
)

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
	Fileds []FieldInfo
}

var reg *regexp.Regexp

func init() {

	// 正则匹配sql的option设置
	reg = regexp.MustCompile(`^\/\*(.*?)\*\/`)
}

func (s *session) ExecuteInc(ctx context.Context, sql string) (recordSets []ast.RecordSet, err error) {
	if recordSets, err = s.executeInc(ctx, sql); err != nil {
		err = errors.Trace(err)
		s.sessionVars.StmtCtx.AppendError(err)
	}
	return
}

func (s *session) executeInc(ctx context.Context, sql string) (recordSets []ast.RecordSet, err error) {

	// fmt.Println("---------------===============")
	sqlList := strings.Split(sql, "\n")
	// fmt.Println(len(sqlList))
	// for i, a := range sqlList {
	// 	fmt.Println(i, a)
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

			if err != nil {

				s.recordSets.Append(&Record{
					Sql:          s1,
					Errlevel:     2,
					ErrorMessage: err.Error(),
				})
				return s.recordSets.Rows(), nil

				s.rollbackOnError(ctx)
				log.Warnf("con:%d parse error:\n%v\n%s", connID, err, s1)
				return nil, errors.Trace(err)
			}

			for _, stmtNode := range stmtNodes {

				currentSql := stmtNode.Text()

				// var checkResult *Record = nil
				s.myRecord = &Record{
					Sql: currentSql,
					Buf: new(bytes.Buffer),
				}

				switch node := stmtNode.(type) {
				case *ast.InceptionStartStmt:
					s.haveBegin = true

					s.parseOptions(sql)
					if s.myRecord.Errlevel == 2 {
						return s.recordSets.Rows(), nil
					}
					// if err != nil {
					// 	s.recordSets.Append(&Record{
					// 		Sql:          currentSql,
					// 		Errlevel:     2,
					// 		ErrorMessage: err.Error(),
					// 	})
					// 	return s.recordSets.Rows(), nil
					// }
					continue
				case *ast.InceptionCommitStmt:
					s.haveCommit = true
					return s.recordSets.Rows(), nil

				case *ast.UseStmt:
					s.checkChangeDB(node)
				case *ast.InsertStmt:
					s.checkInsert(node, currentSql)
				case *ast.DeleteStmt:
					s.checkDelete(node, currentSql)
					// default:
					// s.recordSets.Append(&Record{
					// 	Sql:          currentSql,
					// 	Errlevel:     0,
					// 	ErrorMessage: fmt.Sprintf("%T", node),
					// })
				}

				// fmt.Println(stmtNode.Text())
				// fmt.Println("---")

				if !s.haveBegin {
					s.recordSets.Append(&Record{
						Sql:          currentSql,
						Errlevel:     2,
						ErrorMessage: "Must start as begin statement.",
					})
					return s.recordSets.Rows(), nil
				}

				// if checkResult != nil {
				// 	checkResult.Sql = currentSql

				// 	s.recordSets.Append(checkResult)
				// }

				s.recordSets.Append(s.myRecord)

				// s.recordSets.Append(&Record{
				// 	Sql:      currentSql,
				// 	Errlevel: 0,
				// })
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
			Errlevel:     2,
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

func (s *session) parseOptions(sql string) {

	firsts := reg.FindStringSubmatch(sql)
	if len(firsts) < 2 {
		s.AppendErrorNo(ER_SQL_INVALID_SOURCE)
		return
	}

	options := strings.Replace(strings.Replace(firsts[1], "-", "", -1), "_", "", -1)
	options = strings.Replace(options, ";", "\n", -1)
	options = strings.Replace(options, "=", ": ", -1)

	// fmt.Println(options)
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
		fmt.Println(s.opt)

		s.AppendErrorNo(ER_SQL_INVALID_SOURCE)
		return
	}

	addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local",
		s.opt.user, s.opt.password, s.opt.host, s.opt.port)
	db, err := gorm.Open("mysql", addr)
	if err != nil {
		s.AppendErrorMessage(err.Error())
		return
	}
	s.db = db

}

func (s *session) checkCommand() {

}

func (s *session) checkInsert(node *ast.InsertStmt, sql string) {

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
		return
	}

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
			fmt.Println(sel.Text())

			fmt.Println(len(sel.Fields.Fields), sel.Fields)

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
					fmt.Println("--------")

					fmt.Println(fmt.Sprintf("%T", f.Expr))

					if c, ok := f.Expr.(*ast.ColumnNameExpr); ok {
						fmt.Println(c.Name)
					}
					if f.Expr == nil {
						fmt.Println(node.Text())
						fmt.Println("Expr is NULL", f.WildCard, f.Expr, f.AsName)
						fmt.Println(f.Text())
						fmt.Println("是个*号")
					}
				}
			}

			i := strings.Index(strings.ToLower(sql), "select")
			selectSql := sql[i:]
			var explain []string

			explain = append(explain, "EXPLAIN ")
			explain = append(explain, selectSql)

			fmt.Println(explain)

			rows := s.getExplainInfo(strings.Join(explain, ""))

			s.myRecord.AnlyzeExplain(rows)

		}
	}

	// fmt.Println(len(node.Setlist), "----------------")
	if len(node.Setlist) > 0 {
		// Process `set` type column.
		// columns := make([]string, 0, len(node.Setlist))
		for _, v := range node.Setlist {
			fmt.Println(v.Column)
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

func (s *session) checkChangeDB(node *ast.UseStmt) {

	fmt.Println("checkChangeDB", node.DBName)

	s.DBName = node.DBName

	s.db.Exec(fmt.Sprintf("USE `%s`", node.DBName))
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
	s.db.Raw(sql).Scan(&rows)

	fmt.Println(rows)

	return rows
}

func (s *session) checkDelete(node *ast.DeleteStmt, sql string) *Record {

	result := &Record{
		Errlevel: 0,
	}
	// // TableRefs is used in both single table and multiple table delete statement.
	// TableRefs *TableRefsClause
	// // Tables is only used in multiple table delete statement.
	// Tables       *DeleteTableList
	// Where        ExprNode
	// Order        *OrderByClause
	// Limit        *Limit
	// Priority     mysql.PriorityEnum
	// IgnoreErr    bool
	// Quick        bool
	// IsMultiTable bool
	// BeforeFrom   bool
	// // TableHints represents the table level Optimizer Hint for join type.
	// TableHints []*TableOptimizerHint

	fmt.Println("checkDelete")
	fmt.Println(node.Tables, node.TableRefs)
	if node.TableRefs != nil {
		a := node.TableRefs.TableRefs
		fmt.Println(a)
		if a.Left != nil {
			if tblSrc, ok := a.Left.(*ast.TableSource); ok {
				fmt.Println("----------------------------------")
				// fmt.Println(tblSrc)
				// fmt.Println(tblSrc.AsName)
				// fmt.Println(tblSrc.Source)
				// fmt.Println(fmt.Sprintf("%T", tblSrc.Source))

				if tblName, ok := tblSrc.Source.(*ast.TableName); ok {
					fmt.Println(tblName.Schema)
					fmt.Println(tblName.Name)

					s.QueryTableFromDB(tblName.Schema.O, tblName.Name.O, true)
				}
			}

		}
		// fmt.Println(a.Left, a.Left.Text())
		// fmt.Println(fmt.Sprintf("%T", a.Left))
		// fmt.Println(a.Right)
	}
	return result
}

func (s *session) QueryTableFromDB(db string, tableName string, reportNotExists bool) {
	if db == "" {
		db = s.DBName
	}
	sql := fmt.Sprintf("SHOW FULL FIELDS FROM `%s`.`%s`", db, tableName)

	var rows []FieldInfo
	fmt.Println("++++++++")
	if err := s.db.Raw(sql).Scan(&rows).Error; err != nil {
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			if myErr.Number != 1146 || reportNotExists {
				fmt.Println(myErr.Number)
				fmt.Println(myErr.Message)
				s.AppendErrorMessage(myErr.Message)
			}
		} else {
			fmt.Println(fmt.Sprintf("%T", err))
		}
	}
	// errs := a.GetErrors()

	fmt.Println(rows)
}

func (r *Record) AppendErrorMessage(msg string) {
	r.Errlevel = 2

	r.Buf.WriteString(msg)
	r.Buf.WriteString("\n")
}

func (r *Record) AppendErrorNo(number int, values ...interface{}) {
	r.Errlevel |= GetErrorLevel(number)
	if len(values) == 0 {
		r.Buf.WriteString(GetErrorMessage(number))
	} else {
		r.Buf.WriteString(fmt.Sprintf(GetErrorMessage(number), values...))
	}
	r.Buf.WriteString("\n")
}

func (s *session) AppendErrorMessage(msg string) {
	s.myRecord.AppendErrorMessage(msg)
}

func (s *session) AppendErrorNo(number int, values ...interface{}) {
	s.myRecord.AppendErrorNo(number)
}

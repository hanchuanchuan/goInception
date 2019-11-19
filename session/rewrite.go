/*
 * Copyright 2018 Xiaomi, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package session

import (
	"fmt"
	"vitess.io/vitess/go/vt/sqlparser"
)

// Rewrite 用于重写SQL
type Rewrite struct {
	SQL  string
	Stmt sqlparser.Statement
}

// NewRewrite 返回一个*Rewrite对象，如果SQL无法被正常解析，将错误输出到日志中，返回一个nil
func NewRewrite(sql string) (*Rewrite, error) {
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return nil, err
	}

	return &Rewrite{
		SQL:  sql,
		Stmt: stmt,
	}, err
}

// Rewrite 入口函数
func (rw *Rewrite) Rewrite() error {
	return rw.RewriteDML2Select()
}

// RewriteDML2Select dml2select: DML 转成 SELECT，兼容低版本的 EXPLAIN
func (rw *Rewrite) RewriteDML2Select() error {
	if rw.Stmt == nil {
		return nil
	}
	switch stmt := rw.Stmt.(type) {
	case *sqlparser.Select:
		return nil
	case *sqlparser.Delete: // Multi DELETE not support yet.
		rw.SQL = delete2Select(stmt)
	case *sqlparser.Insert:
		rw.SQL = insert2Select(stmt)
	case *sqlparser.Update: // Multi UPDATE not support yet.
		rw.SQL = update2Select(stmt)
	}
	var err error
	rw.Stmt, err = sqlparser.Parse(rw.SQL)
	return err
}

// delete2Select 将 Delete 语句改写成 Select
func delete2Select(stmt *sqlparser.Delete) string {
	newSQL := &sqlparser.Select{
		SelectExprs: []sqlparser.SelectExpr{
			new(sqlparser.StarExpr),
		},
		From:    stmt.TableExprs,
		Where:   stmt.Where,
		OrderBy: stmt.OrderBy,
		Limit:   stmt.Limit,
	}
	return sqlparser.String(newSQL)
}

// update2Select 将 Update 语句改写成 Select
func update2Select(stmt *sqlparser.Update) string {
	newSQL := &sqlparser.Select{
		SelectExprs: []sqlparser.SelectExpr{
			new(sqlparser.StarExpr),
		},
		From:    stmt.TableExprs,
		Where:   stmt.Where,
		OrderBy: stmt.OrderBy,
		Limit:   stmt.Limit,
	}
	return sqlparser.String(newSQL)
}

// insert2Select 将 Insert 语句改写成 Select
func insert2Select(stmt *sqlparser.Insert) string {
	switch row := stmt.Rows.(type) {
	// 如果insert包含子查询，只需要explain该子树
	case *sqlparser.Select, *sqlparser.Union, *sqlparser.ParenSelect:
		return sqlparser.String(row)
	}

	return "select 1 from DUAL"
}

func (rw *Rewrite) TestSelect2Count() string {
	return rw.select2Count()
}

// select2Count : SELECT 转成 COUNT语句
func (rw *Rewrite) select2Count() string {
	if rw.Stmt == nil {
		return fmt.Sprintf("SELECT COUNT(1) FROM (%s)t", rw.SQL)
	}
	// log.Infof("%#v", rw.Stmt)

	switch stmt := rw.Stmt.(type) {
	case *sqlparser.Select:
		if stmt.Distinct != "" || stmt.GroupBy != nil || stmt.Having != nil {
			return fmt.Sprintf("SELECT COUNT(1) FROM (%s)t", rw.SQL)
		}

		newSQL := &sqlparser.Select{
			SelectExprs: []sqlparser.SelectExpr{
				&sqlparser.AliasedExpr{
					Expr: &sqlparser.FuncExpr{
						Name: sqlparser.NewColIdent("count"),
						Exprs: []sqlparser.SelectExpr{
							new(sqlparser.StarExpr),
						},
					},
				},
			},
			Distinct: stmt.Distinct,
			From:     stmt.From,
			Where:    stmt.Where,
			GroupBy:  stmt.GroupBy,
			Having:   stmt.Having,
		}

		//fix the bug when the 'limit' we assign will not be the condition which what we exactly want
		//for example we want to get the affected rows from the sql 'select * from table where id < 100 limit 10'
		//before the bug fixed bug, the rewrited sql is 'select count(*) from table where id < 100 limit 10', which is incorrect
		//what we exactly want is 'select count(*) from (select 1 from table where id < 100 limit 10)'
		if stmt.Limit != nil {
			newSQL.Limit = stmt.Limit
			newSQL.SelectExprs = []sqlparser.SelectExpr{
				new(sqlparser.StarExpr),
			}
			return fmt.Sprintf("SELECT COUNT(1) FROM (%s)t", sqlparser.String(newSQL))
		}

		return sqlparser.String(newSQL)
	// case *sqlparser.Union, *sqlparser.ParenSelect:
	// 	return fmt.Sprintf("SELECT COUNT(1) FROM (%s)t", rw.SQL)
	default:
		return fmt.Sprintf("SELECT COUNT(1) FROM (%s)t", rw.SQL)
	}
}

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
	"strings"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/mysql"
	"github.com/pingcap/errors"
	// log "github.com/sirupsen/logrus"
)

// checkContainDotColumn checks field contains the table name.
// for example :create table t (c1.c2 int default null).
func (s *session) checkContainDotColumn(stmt *ast.CreateTableStmt) {
	tName := stmt.Table.Name.String()
	sName := stmt.Table.Schema.String()

	for _, colDef := range stmt.Cols {
		// check schema and table names.
		if colDef.Name.Schema.O != sName && len(colDef.Name.Schema.O) != 0 {
			s.AppendErrorNo(ER_WRONG_DB_NAME, colDef.Name.Schema.O)
		}
		if colDef.Name.Table.O != tName && len(colDef.Name.Table.O) != 0 {
			s.AppendErrorNo(ER_WRONG_TABLE_NAME, colDef.Name.Table.O)
		}
	}
}

func (s *session) checkAutoIncrement(stmt *ast.CreateTableStmt) {
	var (
		isKey            bool
		count            int
		autoIncrementCol *ast.ColumnDef
	)

	for _, colDef := range stmt.Cols {
		var hasAutoIncrement bool
		for i, op := range colDef.Options {
			ok, err := checkAutoIncrementOp(colDef, i)
			if err != nil {
				s.AppendErrorMessage(err.Error())
				// return
			}
			if ok {
				hasAutoIncrement = true
			}
			switch op.Tp {
			case ast.ColumnOptionPrimaryKey, ast.ColumnOptionUniqKey:
				isKey = true
			}
		}
		if hasAutoIncrement {
			count++
			autoIncrementCol = colDef
		}
	}

	if count < 1 {
		return
	}
	if !isKey {
		isKey = isConstraintKeyTp(stmt.Constraints, autoIncrementCol)
	}
	autoIncrementMustBeKey := true
	for _, opt := range stmt.Options {
		if opt.Tp == ast.TableOptionEngine && strings.EqualFold(opt.StrValue, "MyISAM") {
			autoIncrementMustBeKey = false
		}
	}
	if (autoIncrementMustBeKey && !isKey) || count > 1 {
		s.AppendErrorNo(ER_WRONG_AUTO_KEY)
	}

	switch autoIncrementCol.Tp.Tp {
	case mysql.TypeTiny, mysql.TypeShort, mysql.TypeLong,
		mysql.TypeFloat, mysql.TypeDouble, mysql.TypeLonglong, mysql.TypeInt24:
	default:
		s.AppendErrorMessage(
			fmt.Sprintf("Incorrect column specifier for column '%s'", autoIncrementCol.Name.Name.O))
	}
}

func isConstraintKeyTp(constraints []*ast.Constraint, colDef *ast.ColumnDef) bool {
	for _, c := range constraints {
		// If the constraint as follows: primary key(c1, c2)
		// we only support c1 column can be auto_increment.
		if colDef.Name.Name.L != c.Keys[0].Column.Name.L {
			continue
		}
		switch c.Tp {
		case ast.ConstraintPrimaryKey, ast.ConstraintKey, ast.ConstraintIndex,
			ast.ConstraintUniq, ast.ConstraintUniqIndex, ast.ConstraintUniqKey:
			return true
		}
	}

	return false
}

func checkAutoIncrementOp(colDef *ast.ColumnDef, num int) (bool, error) {
	var hasAutoIncrement bool

	if colDef.Options[num].Tp == ast.ColumnOptionAutoIncrement {
		hasAutoIncrement = true
		if len(colDef.Options) == num+1 {
			return hasAutoIncrement, nil
		}
		for _, op := range colDef.Options[num+1:] {
			if op.Tp == ast.ColumnOptionDefaultValue && !op.Expr.GetDatum().IsNull() {
				return hasAutoIncrement,
					errors.Errorf(fmt.Sprintf(GetErrorMessage(ER_INVALID_DEFAULT), colDef.Name.Name.O))
			}
		}
	}
	if colDef.Options[num].Tp == ast.ColumnOptionDefaultValue && len(colDef.Options) != num+1 {
		if colDef.Options[num].Expr.GetDatum().IsNull() {
			return hasAutoIncrement, nil
		}
		for _, op := range colDef.Options[num+1:] {
			if op.Tp == ast.ColumnOptionAutoIncrement {
				return hasAutoIncrement,
					errors.Errorf(fmt.Sprintf(GetErrorMessage(ER_INVALID_DEFAULT), colDef.Name.Name.O))
			}
		}
	}

	return hasAutoIncrement, nil
}

func (s *session) checkDuplicateColumnName(indexColNames []*ast.IndexColName) error {
	colNames := make(map[string]struct{}, len(indexColNames))
	for _, indexColName := range indexColNames {
		name := indexColName.Column.Name
		if _, ok := colNames[name.L]; ok {
			s.AppendErrorNo(ER_DUP_FIELDNAME, name)
			return errors.New("")
		}
		colNames[name.L] = struct{}{}
	}
	return nil
}

// checkIndexInfo checks index name and index column names.
func (s *session) checkIndexInfo(indexName string, indexColNames []*ast.IndexColName) error {
	if strings.EqualFold(indexName, mysql.PrimaryKeyName) {
		s.AppendErrorNo(ER_WRONG_NAME_FOR_INDEX, indexName, "")
		return errors.New("")
	}
	if s.Inc.MaxKeyParts > 0 && len(indexColNames) > int(s.Inc.MaxKeyParts) {
		s.AppendErrorNo(ER_TOO_MANY_KEY_PARTS, indexName, "", s.Inc.MaxKeyParts)
		return errors.New("")
	}
	return s.checkDuplicateColumnName(indexColNames)
}

// isDefaultValNowSymFunc checks whether defaul value is a NOW() builtin function.
func isDefaultValNowSymFunc(expr ast.ExprNode) bool {
	if funcCall, ok := expr.(*ast.FuncCallExpr); ok {
		// Default value NOW() is transformed to CURRENT_TIMESTAMP() in parser.
		if funcCall.FnName.L == ast.CurrentTimestamp {
			return true
		}
	}
	return false
}

func isInvalidDefaultValue(colDef *ast.ColumnDef) bool {
	tp := colDef.Tp
	// Check the last default value.
	for i := len(colDef.Options) - 1; i >= 0; i-- {
		columnOpt := colDef.Options[i]
		if columnOpt.Tp == ast.ColumnOptionDefaultValue {
			// log.Infof("%#v", columnOpt.Expr)
			if !(tp.Tp == mysql.TypeTimestamp || tp.Tp == mysql.TypeDatetime) && isDefaultValNowSymFunc(columnOpt.Expr) {
				return true
			}
			break
		}
	}

	return false
}

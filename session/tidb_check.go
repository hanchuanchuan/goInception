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
	"math"
	"strings"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/mysql"
	"github.com/hanchuanchuan/goInception/types"
	"github.com/hanchuanchuan/goInception/util/charset"
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

func (s *session) checkDuplicateColumnName(indexColNames []*ast.IndexColName) {
	colNames := make(map[string]struct{}, len(indexColNames))
	for _, indexColName := range indexColNames {
		name := indexColName.Column.Name
		if _, ok := colNames[name.L]; ok {
			s.AppendErrorNo(ER_DUP_FIELDNAME, name)
		}
		colNames[name.L] = struct{}{}
	}
}

// checkIndexInfo checks index name and index column names.
func (s *session) checkIndexInfo(tableName, indexName string, indexColNames []*ast.IndexColName) {
	// if strings.EqualFold(indexName, mysql.PrimaryKeyName) {
	// 	s.AppendErrorNo(ER_WRONG_NAME_FOR_INDEX, indexName, tableName)
	// }
	if s.Inc.MaxKeyParts > 0 && len(indexColNames) > int(s.Inc.MaxKeyParts) {
		s.AppendErrorNo(ER_TOO_MANY_KEY_PARTS, indexName, "", s.Inc.MaxKeyParts)
	}
	s.checkDuplicateColumnName(indexColNames)
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

// func checkDefaultValue( c *table.Column ) error {

// 	if c.GetDefaultValue() != nil {
// 		if _, err := table.GetColDefaultValue(ctx, c.ToInfo()); err != nil {
// 			return types.ErrInvalidDefault.GenWithStackByArgs(c.Name)
// 		}
// 		return nil
// 	}
// 	// Primary key default null is invalid.
// 	if mysql.HasPriKeyFlag(c.Flag) {
// 		return ErrPrimaryCantHaveNull
// 	}

// 	// Set not null but default null is invalid.
// 	if mysql.HasNotNullFlag(c.Flag) {
// 		return types.ErrInvalidDefault.GenWithStackByArgs(c.Name)
// 	}

// 	return nil
// }

// isIncorrectName checks if the identifier is incorrect.
// See https://dev.mysql.com/doc/refman/5.7/en/identifiers.html
func isIncorrectName(name string) bool {
	if len(name) == 0 {
		return true
	}
	if name[len(name)-1] == ' ' {
		return true
	}
	return false
}

func (s *session) checkCreateTableGrammar(stmt *ast.CreateTableStmt) {
	tName := stmt.Table.Name.String()
	if isIncorrectName(tName) {
		s.AppendErrorNo(ER_WRONG_TABLE_NAME, tName)
	}
	countPrimaryKey := 0
	for _, colDef := range stmt.Cols {
		if err := s.checkColumn(colDef); err != nil {
			s.AppendErrorMessage(err.Error())
		}
		countPrimaryKey += isPrimary(colDef.Options)
	}
	for _, constraint := range stmt.Constraints {
		switch tp := constraint.Tp; tp {
		// case ast.ConstraintKey, ast.ConstraintIndex, ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
		// 	s.checkIndexInfo(tName, constraint.Name, constraint.Keys)
		case ast.ConstraintPrimaryKey:
			countPrimaryKey++
			// s.checkIndexInfo(tName, constraint.Name, constraint.Keys)
		}
	}

	if countPrimaryKey > 1 {
		s.AppendErrorNo(ER_MULTIPLE_PRI_KEY)
	}

	if len(stmt.Cols) == 0 && stmt.ReferTable == nil {
		s.AppendErrorNo(ER_MUST_HAVE_COLUMNS)
	}
}

func (s *session) checkColumn(colDef *ast.ColumnDef) error {
	// Check column name.
	cName := colDef.Name.Name.String()
	if isIncorrectName(cName) {
		s.AppendErrorNo(ER_WRONG_COLUMN_NAME, colDef.Name.Name.O)
	}

	// if isInvalidDefaultValue(colDef) {
	// 	s.AppendErrorNo(ER_INVALID_DEFAULT, colDef.Name.Name.O)
	// }

	// Check column type.
	tp := colDef.Tp
	if tp == nil {
		return nil
	}
	if tp.Flen > math.MaxUint32 {
		s.AppendErrorMessage(fmt.Sprintf("Display width out of range for column '%s' (max = %d)", colDef.Name.Name.O, math.MaxUint32))
	}

	switch tp.Tp {
	case mysql.TypeString:
		if tp.Flen != types.UnspecifiedLength && tp.Flen > mysql.MaxFieldCharLength {
			s.AppendErrorMessage(fmt.Sprintf("Column length too big for column '%s' (max = %d); use BLOB or TEXT instead", colDef.Name.Name.O, mysql.MaxFieldCharLength))
		}
	case mysql.TypeVarchar:
		maxFlen := mysql.MaxFieldVarCharLength
		cs := tp.Charset
		// TODO: TableDefaultCharset-->DatabaseDefaultCharset-->SystemDefaultCharset.
		// TODO: Change TableOption parser to parse collate.
		// Reference https://github.com/hanchuanchuan/goInception/blob/b091e828cfa1d506b014345fb8337e424a4ab905/ddl/ddl_api.go#L185-L204
		if len(tp.Charset) == 0 {
			cs = mysql.DefaultCharset
		}
		desc, err := charset.GetCharsetDesc(cs)
		if err != nil {
			return errors.Trace(err)
		}
		maxFlen /= desc.Maxlen
		if tp.Flen != types.UnspecifiedLength && tp.Flen > maxFlen {
			s.AppendErrorMessage(fmt.Sprintf("Column length too big for column '%s' (max = %d); use BLOB or TEXT instead", colDef.Name.Name.O, maxFlen))
		}
	case mysql.TypeFloat, mysql.TypeDouble:
		if tp.Decimal > mysql.MaxFloatingTypeScale {
			s.AppendErrorMessage(fmt.Sprintf("Too big scale %d specified for column '%-.192s'. Maximum is %d.", tp.Decimal, colDef.Name.Name.O, mysql.MaxFloatingTypeScale))
		}
		if tp.Flen > mysql.MaxFloatingTypeWidth {

			s.AppendErrorMessage(fmt.Sprintf("Too big precision %d specified for column '%-.192s'. Maximum is %d.", tp.Flen, colDef.Name.Name.O, mysql.MaxFloatingTypeWidth))
		}
	case mysql.TypeSet:
		if len(tp.Elems) > mysql.MaxTypeSetMembers {
			s.AppendErrorMessage(fmt.Sprintf("Too many strings for column %s and SET", colDef.Name.Name.O))
		}
		// Check set elements. See https://dev.mysql.com/doc/refman/5.7/en/set.html .
		for _, str := range colDef.Tp.Elems {
			if strings.Contains(str, ",") {
				s.AppendErrorMessage(fmt.Sprintf("Illegal %s '%-.192s' value found during parsing", types.TypeStr(tp.Tp), str))
			}
		}
	case mysql.TypeNewDecimal:
		if tp.Decimal > mysql.MaxDecimalScale {
			s.AppendErrorMessage(fmt.Sprintf("Too big scale %d specified for column '%-.192s'. Maximum is %d.", tp.Decimal, colDef.Name.Name.O, mysql.MaxDecimalScale))
		}

		if tp.Flen > mysql.MaxDecimalWidth {
			s.AppendErrorMessage(fmt.Sprintf("Too big precision %d specified for column '%-.192s'. Maximum is %d.", tp.Flen, colDef.Name.Name.O, mysql.MaxDecimalWidth))
		}
	default:
		// TODO: Add more types.
	}
	return nil
}

// func (s *session) checkNonUniqTableAlias(stmt *ast.Join, tableAliases map[string]interface{}) {
// 	if err := isTableAliasDuplicate(stmt.Left, tableAliases); err != nil {
// 		return
// 	}
// 	if err := isTableAliasDuplicate(stmt.Right, tableAliases); err != nil {
// 		return
// 	}
// }

// checkTableAliasDuplicate 检查表名/别名是否重复
func (s *session) checkTableAliasDuplicate(node ast.ResultSetNode, tableAliases map[string]interface{}) {
	switch x := node.(type) {
	case *ast.Join:
		s.checkTableAliasDuplicate(x.Left, tableAliases)
		s.checkTableAliasDuplicate(x.Right, tableAliases)
	case *ast.TableSource:
		name := ""
		if x.AsName.L != "" {
			name = x.AsName.L
		} else if s, ok := x.Source.(*ast.TableName); ok {
			name = s.Name.L
		}

		_, ok := tableAliases[name]
		if len(name) != 0 && ok {
			s.AppendErrorNo(ErrNonUniqTable, name)
			return
		}
		tableAliases[name] = nil
	}
}

func isPrimary(ops []*ast.ColumnOption) int {
	for _, op := range ops {
		if op.Tp == ast.ColumnOptionPrimaryKey {
			return 1
		}
	}
	return 0
}

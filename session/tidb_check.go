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
	"math"
	"reflect"
	"strings"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/format"
	"github.com/hanchuanchuan/goInception/model"
	"github.com/hanchuanchuan/goInception/mysql"
	"github.com/hanchuanchuan/goInception/types"
	"github.com/hanchuanchuan/goInception/util/charset"
	"github.com/pingcap/errors"
)

const (
	// ErrExprInSelect  is in select fields for the error of ErrFieldNotInGroupBy
	ErrExprInSelect = "SELECT list"
	// ErrExprInOrderBy  is in order by items for the error of ErrFieldNotInGroupBy
	ErrExprInOrderBy = "ORDER BY"
)

// ErrExprLoc is for generate the ErrFieldNotInGroupBy error info
type ErrExprLoc struct {
	Offset int
	Loc    string
}

// checkContainDotColumn checks field contains the table name.
// for example :create table t (c1.c2 int default null).
func (s *session) checkContainDotColumn(stmt *ast.CreateTableStmt) {
	tName := stmt.Table.Name.String()
	sName := stmt.Table.Schema.String()

	for _, colDef := range stmt.Cols {
		// check schema and table names.
		if colDef.Name.Schema.O != sName && len(colDef.Name.Schema.O) != 0 {
			s.appendErrorNo(ER_WRONG_DB_NAME, colDef.Name.Schema.O)
		}
		if colDef.Name.Table.O != tName && len(colDef.Name.Table.O) != 0 {
			s.appendErrorNo(ER_WRONG_TABLE_NAME, colDef.Name.Table.O)
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
			ok, err := s.checkAutoIncrementOp(colDef, i)
			if err != nil {
				s.appendErrorMsg(err.Error())
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
		s.appendErrorNo(ER_WRONG_AUTO_KEY)
	}

	switch autoIncrementCol.Tp.Tp {
	case mysql.TypeTiny, mysql.TypeShort, mysql.TypeLong,
		mysql.TypeFloat, mysql.TypeDouble, mysql.TypeLonglong, mysql.TypeInt24:
	default:
		s.appendErrorMsg(
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

func (s *session) checkAutoIncrementOp(colDef *ast.ColumnDef, num int) (bool, error) {
	var hasAutoIncrement bool

	if colDef.Options[num].Tp == ast.ColumnOptionAutoIncrement {
		hasAutoIncrement = true
		if len(colDef.Options) == num+1 {
			return hasAutoIncrement, nil
		}
		for _, op := range colDef.Options[num+1:] {
			if op.Tp == ast.ColumnOptionDefaultValue && !op.Expr.GetDatum().IsNull() {
				return hasAutoIncrement,
					errors.Errorf(fmt.Sprintf(s.getErrorMessage(ER_INVALID_DEFAULT), colDef.Name.Name.O))
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
					errors.Errorf(fmt.Sprintf(s.getErrorMessage(ER_INVALID_DEFAULT), colDef.Name.Name.O))
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
			s.appendErrorNo(ER_DUP_FIELDNAME, name)
		}
		colNames[name.L] = struct{}{}
	}
}

// checkIndexInfo checks index name and index column names.
func (s *session) checkIndexInfo(tableName, indexName string, indexColNames []*ast.IndexColName) {
	// if strings.EqualFold(indexName, mysql.PrimaryKeyName) {
	// 	s.AppendErrorNo(ER_WRONG_NAME_FOR_INDEX, indexName, tableName)
	// }
	if s.inc.MaxKeyParts > 0 && len(indexColNames) > int(s.inc.MaxKeyParts) {
		s.appendErrorNo(ER_TOO_MANY_KEY_PARTS, indexName, "", s.inc.MaxKeyParts)
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

func (s *session) isInvalidDefaultValue(colDef *ast.ColumnDef) bool {
	tp := colDef.Tp

	// utc := &stmtctx.StatementContext{TimeZone: time.UTC}
	// Check the last default value.
	for i := len(colDef.Options) - 1; i >= 0; i-- {
		columnOpt := colDef.Options[i]
		if columnOpt.Tp == ast.ColumnOptionDefaultValue {

			if !(tp.Tp == mysql.TypeTimestamp || tp.Tp == mysql.TypeDatetime) && isDefaultValNowSymFunc(columnOpt.Expr) {
				return true
			}
			if !types.IsTypeTime(tp.Tp) ||
				columnOpt.Expr.GetDatum().IsNull() || isDefaultValNowSymFunc(columnOpt.Expr) {
				return false
			}

			d, err := GetTimeValue(s, columnOpt.Expr, tp.Tp, tp.Decimal)
			if err != nil {
				// log.Warning(err)
				return true
			}

			vars := s.sessionVars

			// 根据服务器sql_mode设置处理零值日期
			t := d.GetMysqlTime()
			// log.Info(vars.StrictSQLMode, vars.SQLMode.HasNoZeroDateMode(), t.IsZero())
			// log.Info(vars.StrictSQLMode, vars.SQLMode.HasNoZeroInDateMode(), t.InvalidZero())
			if t.IsZero() {
				if s.inc.EnableZeroDate {
					return vars.StrictSQLMode && vars.SQLMode.HasNoZeroDateMode()
				}
				return true
			} else if t.InvalidZero() {
				return vars.StrictSQLMode && vars.SQLMode.HasNoZeroInDateMode()
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
		s.appendErrorNo(ER_WRONG_TABLE_NAME, tName)
	}
	countPrimaryKey := 0
	for _, colDef := range stmt.Cols {
		// 放在mysqlCheckField函数统一检查(create/alter)
		// if err := s.checkColumn(colDef); err != nil {
		// 	s.AppendErrorMessage(err.Error())
		// }
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
		s.appendErrorNo(ER_MULTIPLE_PRI_KEY)
	}

	if len(stmt.Cols) == 0 && stmt.ReferTable == nil && stmt.Select == nil {
		s.appendErrorNo(ER_MUST_AT_LEAST_ONE_COLUMN)
	}
}

func (s *session) checkColumn(colDef *ast.ColumnDef, tableCharset string, alterTableType ast.AlterTableType) {
	// Check column name.
	cName := colDef.Name.Name.String()
	if isIncorrectName(cName) {
		s.appendErrorNo(ER_WRONG_COLUMN_NAME, cName)
	}

	// if isInvalidDefaultValue(colDef) {
	// 	s.AppendErrorNo(ER_INVALID_DEFAULT, cName)
	// }

	// Check column type.
	tp := colDef.Tp
	if tp == nil {
		return
	}
	if tp.Flen > math.MaxUint32 {
		s.appendErrorMsg(fmt.Sprintf("Display width out of range for column '%s' (max = %d)", cName, math.MaxUint32))
	}

	switch tp.Tp {
	case mysql.TypeString:
		if tp.Flen != types.UnspecifiedLength && tp.Flen > mysql.MaxFieldCharLength {
			s.appendErrorMsg(fmt.Sprintf("Column length too big for column '%s' (max = %d); use BLOB or TEXT instead", cName, mysql.MaxFieldCharLength))
		}
	case mysql.TypeVarchar:
		maxFlen := mysql.MaxFieldVarCharLength
		cs := tp.Charset
		// TODO: TableDefaultCharset-->DatabaseDefaultCharset-->SystemDefaultCharset.
		// TODO: Change TableOption parser to parse collate.
		// Reference https://github.com/hanchuanchuan/goInception/blob/b091e828cfa1d506b014345fb8337e424a4ab905/ddl/ddl_api.go#L185-L204
		if len(tp.Charset) == 0 {
			if tableCharset != "" {
				if strings.Contains(tableCharset, "_") {
					tableCharset = strings.SplitN(tableCharset, "_", 2)[0]
				}
				cs = tableCharset
			} else {
				cs = s.databaseCharset
			}
		}
		// log.Errorf("field: %#v,cs: %v", colDef, cs)
		if _, ok := charSets[strings.ToLower(cs)]; ok {
			bysPerChar := charSets[strings.ToLower(cs)]
			maxFlen /= bysPerChar
		} else {
			desc, err := charset.GetCharsetDesc(cs)
			if err != nil {
				s.appendErrorMsg(err.Error())
				return
			}
			maxFlen /= desc.Maxlen
		}

		if tp.Flen != types.UnspecifiedLength && tp.Flen > maxFlen {
			s.appendErrorMsg(fmt.Sprintf("Column length too big for column '%s' (max = %d); use BLOB or TEXT instead", cName, maxFlen))
		}
		// check varchar length and ignore other alter table operation
		if alterTableType == ast.AlterTableAddColumns && tp.Flen != types.UnspecifiedLength &&
			s.inc.MaxVarcharLength > 0 && tp.Flen > int(s.inc.MaxVarcharLength) {
			s.appendErrorNo(ErrMaxVarcharLength, cName, s.inc.MaxVarcharLength)
		}

	case mysql.TypeFloat, mysql.TypeDouble:
		if tp.Decimal > mysql.MaxFloatingTypeScale {
			s.appendErrorMsg(fmt.Sprintf("Too big scale %d specified for column '%-.192s'. Maximum is %d.", tp.Decimal, cName, mysql.MaxFloatingTypeScale))
		}
		if tp.Flen > mysql.MaxFloatingTypeWidth {

			s.appendErrorMsg(fmt.Sprintf("Too big precision %d specified for column '%-.192s'. Maximum is %d.", tp.Flen, cName, mysql.MaxFloatingTypeWidth))
		}
	case mysql.TypeSet:
		if len(tp.Elems) > mysql.MaxTypeSetMembers {
			s.appendErrorMsg(fmt.Sprintf("Too many strings for column %s and SET", cName))
		}
		// Check set elements. See https://dev.mysql.com/doc/refman/5.7/en/set.html .
		for _, str := range colDef.Tp.Elems {
			if strings.Contains(str, ",") {
				s.appendErrorMsg(fmt.Sprintf("Illegal %s '%-.192s' value found during parsing", types.TypeStr(tp.Tp), str))
			}
		}
	case mysql.TypeNewDecimal:
		if tp.Decimal > mysql.MaxDecimalScale {
			s.appendErrorMsg(fmt.Sprintf("Too big scale %d specified for column '%-.192s'. Maximum is %d.", tp.Decimal, cName, mysql.MaxDecimalScale))
		}

		if tp.Flen > mysql.MaxDecimalWidth {
			s.appendErrorMsg(fmt.Sprintf("Too big precision %d specified for column '%-.192s'. Maximum is %d.", tp.Flen, cName, mysql.MaxDecimalWidth))
		}
	case mysql.TypeBit:
		if tp.Flen <= 0 {
			s.appendErrorMsg(fmt.Sprintf("Invalid size for column '%s'.", cName))
		}
		if tp.Flen > mysql.MaxBitDisplayWidth {
			s.appendErrorMsg(fmt.Sprintf("Too big display width for column '%s' (max = %d).",
				cName, mysql.MaxBitDisplayWidth))
		}
	case mysql.TypeTiny, mysql.TypeInt24, mysql.TypeLong,
		mysql.TypeShort, mysql.TypeLonglong:
		if tp.Flen > mysql.MaxFloatingTypeWidth {
			s.appendErrorMsg(fmt.Sprintf("Too big display width for column '%-.192s' (max = %d).",
				cName, mysql.MaxFloatingTypeWidth))
		}
	default:
		// TODO: Add more types.
	}
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
			s.appendErrorNo(ErrNonUniqTable, name)
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

func (s *session) checkOnlyFullGroupByWithOutGroupClause(fields []*ast.SelectField) error {
	var (
		firstNonAggCol *ast.ColumnName
		// exprIdx           int
		firstNonAggColIdx int
		hasAggFunc        bool
	)
	for idx, field := range fields {
		switch t := field.Expr.(type) {
		case *ast.AggregateFuncExpr:
			hasAggFunc = true
		case *ast.ColumnNameExpr:
			if firstNonAggCol == nil {
				firstNonAggCol, firstNonAggColIdx = t.Name, idx
			}
		}
		if hasAggFunc && firstNonAggCol != nil {
			s.appendErrorNo(ErrMixOfGroupFuncAndFields, firstNonAggColIdx+1, firstNonAggCol.Name.O)
			return nil
		}
	}
	return nil
}

func (s *session) checkOnlyFullGroupByWithGroupClause(sel *ast.SelectStmt, tables []*TableInfo) error {
	gbyCols := make(map[string]struct{}, len(sel.Fields.Fields))
	gbyExprs := make([]ast.ExprNode, 0, len(sel.Fields.Fields))

	for _, byItem := range sel.GroupBy.Items {
		if colExpr, ok := byItem.Expr.(*ast.ColumnNameExpr); ok {

			col := findColumnWithList(colExpr, tables)
			if col == nil {
				continue
			}
			if s.IgnoreCase() {
				gbyCols[strings.ToLower(col.getName())] = struct{}{}
			} else {
				gbyCols[col.getName()] = struct{}{}
			}
		} else {
			gbyExprs = append(gbyExprs, byItem.Expr)
		}
	}

	notInGbyCols := make(map[string]ErrExprLoc, len(sel.Fields.Fields))
	for offset, field := range sel.Fields.Fields {
		// log.Info(field.Auxiliary, "---", field.AsName)
		if field.Auxiliary {
			continue
		}
		if _, ok := field.Expr.(*ast.AggregateFuncExpr); ok {
			if field.AsName.L != "" {
				c := &ast.ColumnNameExpr{Name: &ast.ColumnName{
					Name: model.NewCIStr(field.AsName.O),
				}}
				col := findColumnWithList(c, tables)
				if col != nil {
					if _, ok := gbyCols[col.getName()]; ok {
						s.appendErrorMsgf("Can't group on '%s'", field.AsName.String())
						break
					}
				}
			}
			continue
		}
		if f, ok := field.Expr.(*ast.FuncCallExpr); ok {
			if f.FnName.L == ast.AnyValue {
				continue
			}
		}
		if field.AsName.L != "" {
			var name string
			if s.IgnoreCase() {
				name = field.AsName.L
			} else {
				name = field.AsName.O
			}

			if _, ok1 := gbyCols[name]; ok1 {
				continue
			}

			s.checkExprInGroupBy(field.Expr, offset, ErrExprInSelect, gbyCols, gbyExprs, notInGbyCols, tables)

			// c := &ast.ColumnNameExpr{Name: &ast.ColumnName{
			// 	Name: model.NewCIStr(field.AsName.O),
			// }}
			// checkExprInGroupBy(c, offset, ErrExprInSelect, gbyCols, gbyExprs, notInGbyCols, tables)
		} else {
			s.checkExprInGroupBy(field.Expr, offset, ErrExprInSelect, gbyCols, gbyExprs, notInGbyCols, tables)
		}
		// checkExprInGroupBy(field.Expr, offset, ErrExprInSelect, gbyCols, gbyExprs, notInGbyCols, tables)
	}

	if sel.OrderBy != nil {
		for offset, item := range sel.OrderBy.Items {
			s.checkExprInGroupBy(item.Expr, offset, ErrExprInOrderBy, gbyCols, gbyExprs, notInGbyCols, tables)
		}
	}

	if len(notInGbyCols) == 0 {
		return nil
	}

	for field, errExprLoc := range notInGbyCols {
		switch errExprLoc.Loc {
		case ErrExprInSelect:
			// getName(sel.Fields.Fields[errExprLoc.Offset])
			s.appendErrorNo(ErrFieldNotInGroupBy, errExprLoc.Offset+1, errExprLoc.Loc,
				field)
			return nil
		case ErrExprInOrderBy:
			// sel.OrderBy.Items[errExprLoc.Offset].Expr.Text()
			s.appendErrorNo(ErrFieldNotInGroupBy, errExprLoc.Offset+1, errExprLoc.Loc,
				field)
			return nil
		}
		return nil
	}
	return nil
}

func (s *session) checkExprInGroupBy(expr ast.ExprNode, offset int, loc string,
	gbyCols map[string]struct{}, gbyExprs []ast.ExprNode, notInGbyCols map[string]ErrExprLoc, tables []*TableInfo) {
	// if _, ok := expr.(*ast.AggregateFuncExpr); ok {
	// 	return
	// }
	// AnyValue可以跳过only_full_group_by检查
	// if f, ok := expr.(*ast.FuncCallExpr); ok {
	// 	if f.FnName.L == ast.AnyValue {
	// 		return
	// 	}
	// }
	if c, ok := expr.(*ast.ColumnNameExpr); ok {
		col := findColumnWithList(c, tables)
		if col != nil {
			if _, ok := gbyCols[col.getName()]; !ok {
				notInGbyCols[col.getName()] = ErrExprLoc{Offset: offset, Loc: loc}
			}
		}
	} else {
		for _, gbyExpr := range gbyExprs {
			if reflect.DeepEqual(gbyExpr, expr) {
				return
			}
		}

		// todo: 待优化，从表达式中返回一列
		// colNames := s.getSubSelectColumns(expr)
		// log.Errorf("colNames: %#v", colNames)
		// if len(colNames) > 0 {
		// 	notInGbyCols[colNames[0]] = ErrExprLoc{Offset: offset, Loc: loc}
		// }

		var builder strings.Builder
		_ = expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &builder))
		notInGbyCols[builder.String()] = ErrExprLoc{Offset: offset, Loc: loc}
	}
}

func (s *session) checkPartitionNameUnique(defs []*ast.PartitionDefinition) {
	partNames := make(map[string]struct{})
	listRanges := make(map[string]struct{})
	for _, oldPar := range defs {
		switch clause := oldPar.Clause.(type) {
		case *ast.PartitionDefinitionClauseIn:
			if clause.Values == nil {
				continue
			}
			for _, values := range clause.Values {
				for _, v := range values {
					// key := fmt.Sprintf("%v", v.GetValue())
					var buf bytes.Buffer
					v.Format(&buf)
					key := buf.String()
					if _, ok := listRanges[key]; !ok {
						listRanges[key] = struct{}{}
					} else {
						s.appendErrorNo(ErrRepeatConstDefinition, key)
						break
					}
				}
			}
		case *ast.PartitionDefinitionClauseLessThan:
			for _, v := range clause.Exprs {
				var buf bytes.Buffer
				v.Format(&buf)
				key := buf.String()
				if _, ok := listRanges[key]; !ok {
					listRanges[key] = struct{}{}
				} else {
					s.appendErrorNo(ErrRepeatConstDefinition, key)
					break
				}
			}
		}

		if _, ok := partNames[oldPar.Name.L]; !ok {
			partNames[oldPar.Name.L] = struct{}{}
		} else {
			s.appendErrorNo(ErrSameNamePartition, oldPar.Name.String())
			break
		}
	}
}

func (s *session) checkPartitionNameExists(t *TableInfo, defs []*ast.PartitionDefinition) {
	for _, part := range defs {
		partDescription := ""
		switch clause := part.Clause.(type) {
		case *ast.PartitionDefinitionClauseIn:
			if clause.Values == nil {
				continue
			}
			partValues := make([]string, 0)
			for _, values := range clause.Values {
				for _, v := range values {
					key := fmt.Sprintf("%v", v.GetValue())
					partValues = append(partValues, key)
				}
			}
			partDescription = strings.Join(partValues, ",")

		case *ast.PartitionDefinitionClauseLessThan:
			for _, v := range clause.Exprs {
				partDescription = fmt.Sprintf("%v", v.GetValue())
				break
			}
		}

		for _, oldPart := range t.Partitions {
			if strings.EqualFold(part.Name.String(), oldPart.PartName) {
				s.appendErrorNo(ErrSameNamePartition, oldPart.PartName)
				break
			}
			if partDescription != "" &&
				strings.EqualFold(partDescription, oldPart.PartDescription) {
				s.appendErrorNo(ErrRepeatConstDefinition, partDescription)
			}
		}
	}
}

func (s *session) checkPartitionDrop(t *TableInfo, parts []model.CIStr) {
	for _, part := range parts {
		found := false
		for _, oldPart := range t.Partitions {
			if strings.EqualFold(part.String(), oldPart.PartName) {
				found = true
				break
			}
		}
		if !found {
			s.appendErrorNo(ErrPartitionNotExisted, part.String())
		}
	}
}

// checkPartitionConvert 检查普通表转为分区表
func (s *session) checkPartitionConvert(t *TableInfo, opts *ast.PartitionOptions) {
	if opts == nil {
		return
	}
	// if len(t.Partitions) > 0 {
	// 	s.appendErrorMsg(fmt.Sprintf("Table '%s' is already a partitioned table", t.Name))
	// }
	s.checkPartitionNameUnique(opts.Definitions)
	// s.checkPartitionNameExists(t, opts.Definitions)
}

// checkPartitionConvert 检查分区表转为普通表
func (s *session) checkPartitionRemove(t *TableInfo) {
	if len(t.Partitions) == 0 {
		s.appendErrorMsg("Partition management on a not partitioned table is not possible")
	}
}

// checkPartitionFuncType checks partition function return type.
func (s *session) checkPartitionFuncType(table *ast.CreateTableStmt) {
	if table.Partition.Expr == nil {
		return
	}
	buf := new(bytes.Buffer)
	table.Partition.Expr.Format(buf)
	exprStr := buf.String()
	if table.Partition.Tp == model.PartitionTypeRange {
		// if partition by columnExpr, check the column type
		if _, ok := table.Partition.Expr.(*ast.ColumnNameExpr); ok {
			for _, col := range table.Cols {
				name := strings.Replace(col.Name.String(), ".", "`.`", -1)
				// Range partitioning key supported types: tinyint, smallint, mediumint, int and bigint.
				if !validRangePartitionType(col) && fmt.Sprintf("`%s`", name) == exprStr {
					s.appendErrorNo(ErrNotAllowedTypeInPartition, name)
				}
			}
		}
	}
}

// validRangePartitionType checks the type supported by the range partitioning key.
func validRangePartitionType(col *ast.ColumnDef) bool {
	switch col.Tp.EvalType() {
	case types.ETInt:
		return true
	default:
		return false
	}
}

// checkRangePartitioningKeysConstraints checks that the range partitioning key is included in the table constraint.
func (s *session) checkRangePartitioningKeysConstraints(table *ast.CreateTableStmt) {
	// Returns directly if there is no constraint in the partition table.
	// TODO: Remove the test 's.Partition.Expr == nil' when we support 'PARTITION BY RANGE COLUMNS'
	if len(table.Constraints) == 0 || table.Partition.Expr == nil {
		return
	}
	// Extract the column names in table constraints to []map[string]struct{}.
	consColNames := extractConstraintsColumnNames(table.Constraints)
	// Parse partitioning key, extract the column names in the partitioning key to slice.
	buf := new(bytes.Buffer)
	table.Partition.Expr.Format(buf)
	var partkeys []string
	partkeys = append(partkeys, buf.String())
	// Checks that the partitioning key is included in the constraint.
	for _, con := range consColNames {
		// Every unique key on the table must use every column in the table's partitioning expression.
		// See https://dev.mysql.com/doc/refman/5.7/en/partitioning-limitations-partitioning-keys-unique-keys.html.
		if !checkConstraintIncludePartKey(partkeys, con) {
			s.appendErrorNo(ErrUniqueKeyNeedAllFieldsInPf, "PRIMARY KEY")
		}
	}
}

// extractConstraintsColumnNames extract the column names in table constraints to []map[string]struct{}.
func extractConstraintsColumnNames(cons []*ast.Constraint) []map[string]struct{} {
	var constraints []map[string]struct{}
	for _, v := range cons {
		if v.Tp == ast.ConstraintUniq || v.Tp == ast.ConstraintPrimaryKey {
			uniKeys := make(map[string]struct{})
			for _, key := range v.Keys {
				uniKeys[key.Column.Name.L] = struct{}{}
			}
			// Extract every unique key and primary key.
			if len(uniKeys) != 0 {
				constraints = append(constraints, uniKeys)
			}
		}
	}
	return constraints
}

// checkConstraintIncludePartKey checks that the partitioning key is included in the constraint.
func checkConstraintIncludePartKey(partkeys []string, constraints map[string]struct{}) bool {
	for _, pk := range partkeys {
		name := strings.Replace(pk, "`", "", -1)
		if _, ok := constraints[name]; !ok {
			return false
		}
	}
	return true
}

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package session

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	json "github.com/CorgiMan/json2"
	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/model"
	"github.com/hanchuanchuan/goInception/util/sqlexec"
	log "github.com/sirupsen/logrus"
)

type masking struct {
	maskingFields []MaskingFieldInfo
	session       *session
	colIndex      int
	buf           *bytes.Buffer
}

func (s *session) maskingCommand(ctx context.Context, stmtNode ast.StmtNode,
	currentSql string) ([]sqlexec.RecordSet, error) {
	log.Debug("maskingCommand")

	s.maskingFields = make([]MaskingFieldInfo, 0)
	switch node := stmtNode.(type) {
	case *ast.UseStmt:
		s.dbName = node.DBName
	case *ast.UnionStmt, *ast.SelectStmt:
		p := masking{
			session:       s,
			maskingFields: make([]MaskingFieldInfo, 0),
			buf:           new(bytes.Buffer),
		}
		_, _ = p.checkSelectItem(node, 0)
		fields := p.maskingFields
		tree, err := json.Marshal(fields)
		// tree1, _ := json.MarshalIndent(fields, "    ", "")
		// log.Errorf("%#v", string(tree1))
		if err != nil {
			log.Error(err)
			s.printSets.Append(2, currentSql, "", err.Error())
		} else {
			if p.buf.Len() > 0 {
				s.printSets.Append(0, currentSql, string(tree), p.buf.String())
			} else {
				s.printSets.Append(0, currentSql, string(tree), "")
			}
		}
	default:
		s.printSets.Append(2, currentSql, "", "not support")
	}

	return nil, nil
}

func (s *masking) checkSelectItem(node ast.ResultSetNode, level int) (
	tables []*TableInfo, fields []MaskingFieldInfo) {
	if node == nil {
		return
	}

	switch x := node.(type) {
	case *ast.UnionStmt:
		for _, sel := range x.SelectList.Selects {
			tmpTables, tmpFields := s.checkSubSelectItem(sel, level)
			if tmpTables != nil {
				tables = append(tables, tmpTables...)
			}
			fields = append(fields, tmpFields...)
		}
		return
	case *ast.SelectStmt:
		return s.checkSubSelectItem(x, level)

	case *ast.Join:
		tmpTables, tmpFields := s.checkSelectItem(x.Left, level+1)
		tables = append(tables, tmpTables...)
		fields = append(fields, tmpFields...)

		tmpTables, tmpFields = s.checkSelectItem(x.Right, level+1)
		tables = append(tables, tmpTables...)
		fields = append(fields, tmpFields...)
		return
	case *ast.TableSource:
		switch tblSource := x.Source.(type) {
		case *ast.TableName:
			dbName := tblSource.Schema.O
			if dbName == "" {
				dbName = s.session.dbName
			}
			t := s.session.getTableFromCache(dbName, tblSource.Name.O, false)
			if t != nil {
				if x.AsName.L != "" {
					t.AsName = x.AsName.O
					return []*TableInfo{t.copy()}, nil
				}
				return []*TableInfo{t}, nil
			}
			return
		case *ast.SelectStmt:
			return s.checkSubSelectItem(tblSource, level+1)

		case *ast.UnionStmt:
			return s.checkSelectItem(tblSource, level+1)

		default:
			return s.checkSelectItem(tblSource, level+1)
		}

	default:
		log.Infof("%T", x)
	}
	return
}

func (s *masking) checkSubSelectItem(node *ast.SelectStmt, level int) (tableInfoList []*TableInfo, fields []MaskingFieldInfo) {
	log.Debug("checkSubSelectItem")

	var tableList []*ast.TableSource
	if node.From != nil {
		tableList = extractTableList(node.From.TableRefs, tableList)
	}

	for _, tblSource := range tableList {
		switch x := tblSource.Source.(type) {
		case *ast.TableName:
			tblName := x
			dbName := tblName.Schema.O
			if dbName == "" {
				dbName = s.session.dbName
			}
			t := s.session.getTableFromCache(dbName, tblName.Name.O, false)
			if t != nil {
				if tblSource.AsName.L != "" {
					t.AsName = tblSource.AsName.O
					tableInfoList = append(tableInfoList, t.copy())
				} else {
					tableInfoList = append(tableInfoList, t)
				}
			} else {
				tableInfoList = append(tableInfoList, &TableInfo{
					Schema: tblName.Schema.String(),
					Name:   tblName.Name.String(),
				})
			}
		case *ast.SelectStmt:
			// 递归审核子查询
			tmpTables, tmpFields := s.checkSubSelectItem(x, level+1)
			tableInfoList = append(tableInfoList, tmpTables...)
			fields = append(fields, tmpFields...)

			cols := s.session.getSubSelectColumns(x)
			if cols != nil {
				rows := make([]FieldInfo, len(cols))
				for i, colName := range cols {
					rows[i].Field = colName
				}
				t := &TableInfo{
					Schema: "",
					Name:   tblSource.AsName.String(),
					Fields: rows,
				}
				tableInfoList = append(tableInfoList, t)
			}
		default:
			tmpTables, tmpFields := s.checkSelectItem(x, level+1)
			tableInfoList = append(tableInfoList, tmpTables...)
			fields = append(fields, tmpFields...)
		}
	}

	if node.Fields != nil {
		newFields := make([]*ast.SelectField, 0)
		for colIndex, field := range node.Fields.Fields {
			s.colIndex = colIndex
			// if field.WildCard == nil {
			// 	s.checkItem(field.Expr, tableInfoList)
			// }
			var tmpFields []MaskingFieldInfo

			// log.Infof("field.WildCard: %v, %v", field.WildCard, colIndex)
			if field.WildCard == nil {
				// tmpFields = append(tmpFields, s.checkItem(field.Expr, tableInfoList)...)
				// fields = append(fields, tmpFields...)
				s.checkSelectField(field, tableInfoList, level, colIndex)
				newFields = append(newFields, field)
				continue
			}

			db := field.WildCard.Schema.L
			wildTable := field.WildCard.Table.L
			if wildTable == "" {
				for _, tblSource := range tableList {
					tblName, ok := tblSource.Source.(*ast.TableName)
					if ok {
						if tblName.Schema.L == "" {
							tblName.Schema = model.NewCIStr(s.session.dbName)
						}
						t := s.session.getTableFromCache(tblName.Schema.O, tblName.Name.O, false)
						if t != nil {
							tmpFields = append(tmpFields,
								Convert(tblName.Schema.O, tblName.Name.O, t.Fields)...)
							for _index, f := range t.Fields {
								tableName := tblSource.AsName.String()
								if tableName == "" {
									tableName = tblName.Name.O
								}
								newField := &ast.SelectField{
									Expr: &ast.ColumnNameExpr{
										Name: &ast.ColumnName{
											Name:  model.NewCIStr(f.Field),
											Table: model.NewCIStr(tableName),
										},
									},
								}
								s.checkSelectField(newField, tableInfoList,
									level, colIndex+_index)
								newFields = append(newFields, newField)
							}
							s.colIndex += len(t.Fields)
						} else {
							s.appendErrorNo(ER_TABLE_NOT_EXISTED_ERROR,
								fmt.Sprintf("`%s`.`%s`", tblName.Schema.O, tblName.Name.String()))
						}
					} else {
						cols := s.session.getSubSelectColumns(tblSource.Source)
						for _index, f := range cols {
							newField := &ast.SelectField{
								Expr: &ast.ColumnNameExpr{
									Name: &ast.ColumnName{
										Name:  model.NewCIStr(f),
										Table: model.NewCIStr(tblSource.AsName.String()),
									},
								},
							}
							s.checkSelectField(newField, tableInfoList,
								level, colIndex+_index)
							newFields = append(newFields, newField)
						}
						s.colIndex += len(cols)
					}
				}
			} else {
				for _, tblSource := range tableList {
					var tableName string
					tblName, ok := tblSource.Source.(*ast.TableName)
					if tblSource.AsName.L != "" {
						tableName = tblSource.AsName.L
					} else if ok {
						tableName = tblName.Name.L
					}

					if (ok && (db == "" || db == tblName.Schema.L) && wildTable == tableName) || (!ok && wildTable == tableName) {
						if ok {
							dbName := tblName.Schema.O
							if dbName == "" {
								dbName = s.session.dbName
							}
							t := s.session.getTableFromCache(dbName, tblName.Name.O, false)
							if t != nil {
								tmpFields = append(tmpFields,
									Convert(tblName.Schema.O, tblName.Name.O, t.Fields)...)
								for _index, f := range t.Fields {
									tableName := tblSource.AsName.String()
									if tableName == "" {
										tableName = tblName.Name.O
									}
									newField := &ast.SelectField{
										Expr: &ast.ColumnNameExpr{
											Name: &ast.ColumnName{
												Name:  model.NewCIStr(f.Field),
												Table: model.NewCIStr(tableName),
											},
										},
									}
									s.checkSelectField(newField, tableInfoList,
										level, colIndex+_index)
									newFields = append(newFields, newField)
								}
								s.colIndex += len(t.Fields)
							} else {
								s.appendErrorNo(ER_TABLE_NOT_EXISTED_ERROR,
									fmt.Sprintf("`%s`.`%s`", tblName.Schema.O, tblName.Name.String()))
							}
						} else {
							cols := s.session.getSubSelectColumns(tblSource.Source)
							for _index, f := range cols {
								newField := &ast.SelectField{
									Expr: &ast.ColumnNameExpr{
										Name: &ast.ColumnName{
											Name:  model.NewCIStr(f),
											Table: model.NewCIStr(field.WildCard.Table.O),
										},
									},
								}
								s.checkSelectField(newField, tableInfoList,
									level, colIndex+_index)
								newFields = append(newFields, newField)
							}
							s.colIndex += len(cols)
						}
					}
				}
			}
			if tmpFields != nil {
				fields = append(fields, tmpFields...)
			}
		}
		if len(newFields) > len(node.Fields.Fields) {
			node.Fields.Fields = newFields
		}
	}
	return tableInfoList, fields
}

func (s *masking) checkSelectField(field *ast.SelectField,
	tableInfoList []*TableInfo, level, colIndex int) {
	if level == 0 {
		for _, f := range s.checkItem(field.Expr, tableInfoList) {
			f.Index = uint8(colIndex)
			if field.AsName.String() != "" {
				f.Alias = field.AsName.String()
				s.maskingFields = append(s.maskingFields, f)
			} else {
				f.Alias = s.getExprAliasName(field)
				s.maskingFields = append(s.maskingFields, f)
			}
		}
	}
}

func (s *masking) checkItem(expr ast.ExprNode, tables []*TableInfo) (fields []MaskingFieldInfo) {

	if expr == nil {
		return
	}

	// log.Errorf("expr: %#v", expr)
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		f := s.checkFieldItem(e.Name, tables)
		if f == nil {
			s.appendErrorNo(ER_COLUMN_NOT_EXISTED, e.Name.OrigColName())
			db := e.Name.Schema.O
			if db == "" {
				db = s.session.dbName
			}
			fields = append(fields, MaskingFieldInfo{
				Schema: db,
				Table:  e.Name.Table.String(),
				Field:  e.Name.Name.String(),
			})
		} else {
			fields = append(fields, *f)
		}
		if e.Refer != nil {
			fields = append(fields, s.checkItem(e.Refer.Expr, tables)...)
		}

	case *ast.BinaryOperationExpr:
		fields = append(fields, s.checkItem(e.L, tables)...)
		fields = append(fields, s.checkItem(e.R, tables)...)

	case *ast.UnaryOperationExpr:
		fields = append(fields, s.checkItem(e.V, tables)...)

	case *ast.FuncCallExpr:
		fields = append(fields, s.checkFuncItem(e, tables)...)
		// return s.checkFuncItem(e, tables)

	case *ast.FuncCastExpr:
		fields = append(fields, s.checkItem(e.Expr, tables)...)

	case *ast.AggregateFuncExpr:
		return s.checkAggregateFuncItem(e, tables)

	case *ast.PatternInExpr:
		fields = append(fields, s.checkItem(e.Expr, tables)...)
		for _, expr := range e.List {
			fields = append(fields, s.checkItem(expr, tables)...)
		}
		if e.Sel != nil {
			fields = append(fields, s.checkItem(e.Sel, tables)...)
		}
	case *ast.PatternLikeExpr:
		return s.checkItem(e.Expr, tables)
	case *ast.PatternRegexpExpr:
		return s.checkItem(e.Expr, tables)

	case *ast.SubqueryExpr:
		_, fields = s.checkSelectItem(e.Query, 1)
		return fields

	case *ast.CompareSubqueryExpr:
		fields = append(fields, s.checkItem(e.L, tables)...)
		fields = append(fields, s.checkItem(e.R, tables)...)

	case *ast.ExistsSubqueryExpr:
		_, fields = s.checkSelectItem(e.Sel, 1)
		return fields

	case *ast.IsNullExpr:
		return s.checkItem(e.Expr, tables)
	case *ast.IsTruthExpr:
		return s.checkItem(e.Expr, tables)

	case *ast.BetweenExpr:
		fields = append(fields, s.checkItem(e.Expr, tables)...)
		fields = append(fields, s.checkItem(e.Left, tables)...)
		fields = append(fields, s.checkItem(e.Right, tables)...)

	case *ast.CaseExpr:
		fields = append(fields, s.checkItem(e.Value, tables)...)
		for _, when := range e.WhenClauses {
			fields = append(fields, s.checkItem(when.Expr, tables)...)
			fields = append(fields, s.checkItem(when.Result, tables)...)
		}
		fields = append(fields, s.checkItem(e.ElseClause, tables)...)

	case *ast.DefaultExpr:
		// s.checkFieldItem(e.Name, tables)
		// pass

	case *ast.ParenthesesExpr:
		fields = append(fields, s.checkItem(e.Expr, tables)...)

	case *ast.RowExpr:
		for _, expr := range e.Values {
			fields = append(fields, s.checkItem(expr, tables)...)
		}

	case *ast.ValuesExpr:
		fields = append(fields, *s.checkFieldItem(e.Column.Name, tables))

	case *ast.VariableExpr:
		return s.checkItem(e.Value, tables)

	case *ast.ValueExpr, *ast.ParamMarkerExpr, *ast.PositionExpr:
		// pass

	default:
		log.Infof("checkItem: %#v", e)
	}

	return
}

// getExprAliasName 获取列别名
func (s *masking) getExprAliasName(field *ast.SelectField) string {
	expr := field.Expr
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		return e.Name.Name.String()
	case *ast.BinaryOperationExpr, *ast.UnaryOperationExpr, *ast.FuncCallExpr,
		*ast.FuncCastExpr,
		*ast.AggregateFuncExpr, *ast.PatternInExpr, *ast.PatternLikeExpr,
		*ast.PatternRegexpExpr, *ast.SubqueryExpr, *ast.CompareSubqueryExpr,
		*ast.ExistsSubqueryExpr, *ast.IsNullExpr, *ast.IsTruthExpr,
		*ast.BetweenExpr, *ast.CaseExpr, *ast.ParenthesesExpr,
		*ast.RowExpr, *ast.ValuesExpr, *ast.VariableExpr,
		*ast.ValueExpr, *ast.ParamMarkerExpr, *ast.PositionExpr:

		return field.Text()

	default:
		log.Infof("getExprAliasName default: %#v", e)
		return field.Text()
	}
}

func (s *masking) checkFieldItem(name *ast.ColumnName, tables []*TableInfo) *MaskingFieldInfo {
	db := name.Schema.L

	for _, t := range tables {
		if name.Table.L != "" {
			var tName string
			if t.AsName != "" {
				tName = t.AsName
			} else {
				tName = t.Name
			}
			if (db == "" || strings.EqualFold(t.Schema, db)) &&
				(strings.EqualFold(tName, name.Table.L)) {
				if len(t.Fields) == 0 {
					return &MaskingFieldInfo{
						Schema: t.Schema,
						Table:  t.Name,
						Field:  name.Name.String(),
					}
				}
				for _, field := range t.Fields {
					if strings.EqualFold(field.Field, name.Name.L) && !field.IsDeleted {
						return &MaskingFieldInfo{
							Schema: t.Schema,
							Table:  t.Name,
							Field:  name.Name.String(),
							Type:   field.Type,
						}
					}
				}
			}
		} else {
			if len(t.Fields) == 0 {
				return &MaskingFieldInfo{
					Schema: t.Schema,
					Table:  t.Name,
					Field:  name.Name.String(),
				}
			}
			for _, field := range t.Fields {
				if strings.EqualFold(field.Field, name.Name.L) && !field.IsDeleted {
					return &MaskingFieldInfo{
						Schema: t.Schema,
						Table:  t.Name,
						Field:  name.Name.String(),
						Type:   field.Type,
					}
				}
			}
		}
	}
	return nil
}

// checkFuncItem 检查函数的字段
func (s *masking) checkFuncItem(f *ast.FuncCallExpr, tables []*TableInfo) (fields []MaskingFieldInfo) {
	for _, arg := range f.Args {
		fields = append(fields, s.checkItem(arg, tables)...)
		// if c:=s.checkColumnExpr(arg,tables);c!=nil{
		// 	f.Args[index] = c
		// }
	}
	return
}

// checkAggregateFuncItem 检查聚合函数的字段
func (s *masking) checkAggregateFuncItem(f *ast.AggregateFuncExpr, tables []*TableInfo) (fields []MaskingFieldInfo) {
	for _, arg := range f.Args {
		fields = append(fields, s.checkItem(arg, tables)...)
	}
	return
}

func Convert(schema, table string, fs []FieldInfo) []MaskingFieldInfo {
	maskingFields := make([]MaskingFieldInfo, len(fs))
	for index, f := range fs {
		maskingFields[index] = MaskingFieldInfo{
			Field:  f.Field,
			Type:   f.Type,
			Schema: schema,
			Table:  table,
		}
	}
	return maskingFields
}

func (s *masking) appendErrorNo(number ErrorCode, values ...interface{}) {
	// 不检查时退出
	if !s.session.checkInceptionVariables(number) {
		return
	}

	var level uint8
	if v, ok := s.session.incLevel[number.String()]; ok {
		level = v
	} else {
		level = GetErrorLevel(number)
	}

	if level > 0 {
		if len(values) == 0 {
			s.buf.WriteString(s.session.getErrorMessage(number))
		} else {
			s.buf.WriteString(fmt.Sprintf(s.session.getErrorMessage(number), values...))
		}
		s.buf.WriteString("\n")
	}
}

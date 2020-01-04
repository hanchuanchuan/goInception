// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package session

import (
	"fmt"
	"golang.org/x/net/context"

	json "github.com/CorgiMan/json2"
	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/model"
	"github.com/hanchuanchuan/goInception/util/sqlexec"
	log "github.com/sirupsen/logrus"
)

func (s *session) printCommand(ctx context.Context, stmtNode ast.StmtNode,
	currentSql string) ([]sqlexec.RecordSet, error) {
	log.Debug("printCommand")

	// b, err := json.MarshalIndent(stmtNode, "", "  ")
	tree, err := json.Marshal(stmtNode)
	if err != nil {
		log.Error(err)
		s.printSets.Append(2, currentSql, "", err.Error())
	} else {
		s.printSets.Append(0, currentSql, string(tree), "")
	}

	return nil, nil

	// switch node := stmtNode.(type) {
	// case *ast.UseStmt:
	// 	s.checkChangeDB(node, currentSql)

	// case *ast.InsertStmt:
	// 	s.printInsert(node, currentSql)
	// case *ast.DeleteStmt:
	// 	s.checkDelete(node, currentSql)
	// case *ast.UpdateStmt:
	// 	s.checkUpdate(node, currentSql)

	// default:
	// 	log.Infof("无匹配类型:%T\n", stmtNode)
	// 	s.AppendErrorNo(ER_NOT_SUPPORTED_YET)
	// }

	// return nil, nil
}

func (s *session) printInsert(node *ast.InsertStmt, sql string) {

	log.Debug("printInsert")

	t := getSingleTableName(node.Table)

	res := make(map[string]interface{}, 3)
	res["command"] = "insert"

	table_object := make(map[string]interface{}, 2)
	if t.Schema.O == "" {
		table_object["db"] = s.DBName
	} else {
		table_object["db"] = t.Schema.String()
	}
	table_object["table"] = t.Name.String()
	res["table_object"] = table_object

	if len(node.Columns) > 0 {
		fields := make([]interface{}, 0, len(node.Columns))
		for _, c := range node.Columns {
			if c.Schema.O == "" {
				c.Schema = model.NewCIStr(s.DBName)
			}
			if c.Table.O == "" {
				c.Table = model.NewCIStr(t.Name.O)
			}

			field := make(map[string]string, 2)
			field["type"] = "FIELD_ITEM"
			field["field"] = c.Name.String()
			// if c.WildCard == nil {
			field["db"] = c.Schema.String()
			field["table"] = c.Table.String()
			// }

			fields = append(fields, field)
		}
		res["fields"] = fields
	}

	if len(node.Lists) > 0 {
		many_values := make([]interface{}, 0, len(node.Lists))
		for _, list := range node.Lists {
			values := make([]map[string]string, 0, len(list))
			for _, vv := range list {
				f := printItem(vv)
				values = append(values, f.(map[string]string))
			}
			many_values = append(many_values, values)
		}
		res["many_values"] = many_values
	}

	// insert select 语句
	if node.Select != nil {
		select_insert_values := make(map[string]interface{}, 1)

		res["select_insert_values"] = select_insert_values

		s.printSelectItem(node.Select)
		// log.Infof("%#v", node.Select)
		// log.Infof("%#v", node.Select.Fields)
		// log.Infof("%#v", node.Select.From)
	}

	log.Info(res)
}

func printItem(expr ast.ExprNode) interface{} {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	// case *ast.ColumnNameExpr:
	// 	field := make(map[string]string, 2)
	// 	field["type"] = "FIELD_ITEM"
	// 	field["field"] = e.Name.String()
	// 	if e.WildCard == nil {
	// 		field["db"] = e.Schema.String()
	// 		field["table"] = e.Table.String()

	// 	}
	// 	return field

	case *ast.ValueExpr:
		v := e.GetDatum().GetValue()
		value := make(map[string]string, 2)
		value["type"] = fmt.Sprintf("%T", v)
		value["value"] = fmt.Sprint(v)
		return value
		// switch v := e.GetDatum().GetValue().(type) {
		// case nil:
		// 	d.SetNull()
		// case bool:
		// 	if x {
		// 		d.SetInt64(1)
		// 	} else {
		// 		d.SetInt64(0)
		// 	}
		// case int:
		// 	d.SetInt64(int64(x))
		// case int64:
		// 	d.SetInt64(x)
		// case uint64:
		// 	d.SetUint64(x)
		// case float32:
		// 	d.SetFloat32(x)
		// case float64:
		// 	d.SetFloat64(x)
		// case string:
		// 	d.SetString(x)
		// case []byte:
		// 	d.SetBytes(x)
		// case *types.MyDecimal:
		// 	d.SetMysqlDecimal(x)
		// case types.Duration:
		// 	d.SetMysqlDuration(x)
		// case types.Enum:
		// 	d.SetMysqlEnum(x)
		// case types.BinaryLiteral:
		// 	d.SetBinaryLiteral(x)
		// case types.BitLiteral: // Store as BinaryLiteral for Bit and Hex literals
		// 	d.SetBinaryLiteral(BinaryLiteral(x))
		// case types.HexLiteral:
		// 	d.SetBinaryLiteral(BinaryLiteral(x))
		// case types.Set:
		// 	d.SetMysqlSet(x)
		// case json.BinaryJSON:
		// 	d.SetMysqlJSON(x)
		// case Time:
		// 	d.SetMysqlTime(x)
		// default:
		// 	d.SetInterface(x)
		// }
		// case *ast.SelectField:
		//  field := make(map[string]string, 2)
		//  field["type"] = "FIELD_ITEM"
		//  field["field"] = e.AsName.String()
		//  if e.WildCard == nil {
		//      field["db"] = e.Schema.String()
		//      field["table"] = e.Table.String()

		//  }
		//  return field
	}

	return nil
}

func (s *session) printSelectItem(node ast.ResultSetNode) bool {
	log.Debug("printSelectItem")

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
			s.printSubSelectItem(sel)
		}

	case *ast.SelectStmt:
		s.printSubSelectItem(x)
	default:
		log.Info(x)
		log.Infof("%#v", x)
	}
	return !s.hasError()
}

func (s *session) printSubSelectItem(node *ast.SelectStmt) bool {
	log.Debug("printSubSelectItem")

	log.Infof("%#v", node)
	log.Infof("%#v", node.Fields)
	if node.Fields != nil {
		for _, f := range node.Fields.Fields {
			log.Infof("%#v", f)
		}
	}
	log.Infof("%#v", node.From)

	var tableList []*ast.TableSource
	if node.From != nil {
		tableList = extractTableList(node.From.TableRefs, tableList)
	}

	var tableInfoList []*TableInfo
	for _, tblSource := range tableList {

		switch x := tblSource.Source.(type) {
		case *ast.TableName:
			tblName := x
			t := s.getTableFromCache(tblName.Schema.O, tblName.Name.O, false)
			if t != nil {
				if tblSource.AsName.L != "" {
					t.AsName = tblSource.AsName.O
				}
				tableInfoList = append(tableInfoList, t)
			}
		case *ast.SelectStmt:
			s.printSubSelectItem(x)
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

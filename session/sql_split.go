package session

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/util/sqlexec"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

// splitCommand 分隔功能实现
func (s *session) splitCommand(ctx context.Context, stmtNode ast.StmtNode,
	sql string) ([]sqlexec.RecordSet, error) {
	log.Debug("splitCommand")

	if !s.opt.split {
		return nil, nil
	}

	switch node := stmtNode.(type) {

	case *ast.UseStmt:
		s.dbName = node.DBName
		s.addSplitNode(s.dbName, "", true, node, sql)

	case *ast.InsertStmt:
		t := getSingleTableName(node.Table)
		s.addSplitNode(t.Schema.O, t.Name.O, true, node, sql)

	case *ast.DeleteStmt:
		if node.Tables != nil {
			for _, t := range node.Tables.Tables {
				s.addSplitNode(t.Schema.O, t.Name.O, true, node, sql)
				return nil, nil
			}
		} else {
			var tableList []*ast.TableSource
			tableList = extractTableList(node.TableRefs.TableRefs, tableList)

			for _, tblSource := range tableList {
				if t, ok := tblSource.Source.(*ast.TableName); ok {
					s.addSplitNode(t.Schema.O, t.Name.O, true, node, sql)
					return nil, nil
				}
			}
		}
		s.addSplitNode("", "", true, node, sql)
		return nil, nil
	case *ast.UpdateStmt:
		var originTable string
		if node.List != nil {
			for _, l := range node.List {
				originTable = l.Column.Table.L
				break
			}
		}

		var tableList []*ast.TableSource
		tableList = extractTableList(node.TableRefs.TableRefs, tableList)

		for _, tblSource := range tableList {
			tblName, ok := tblSource.Source.(*ast.TableName)
			if ok {
				if originTable == "" {
					s.addSplitNode(tblName.Schema.L, tblName.Name.L, true, node, sql)
					return nil, nil
				} else if originTable == tblName.Name.L || originTable == tblSource.AsName.L {
					s.addSplitNode(tblName.Schema.L, tblName.Name.L, true, node, sql)
					return nil, nil
				}
			}
		}

		s.addSplitNode("", "", true, node, sql)
		return nil, nil

	case *ast.CreateDatabaseStmt:
		s.addSplitNode(node.Name, "", false, node, sql)

	case *ast.DropDatabaseStmt:
		s.addSplitNode(node.Name, "", false, node, sql)

	case *ast.CreateTableStmt:
		s.addSplitNode(node.Table.Schema.O, node.Table.Name.O, false, node, sql)

	case *ast.AlterTableStmt:
		s.addSplitNode(node.Table.Schema.O, node.Table.Name.O, false, node, sql)

	case *ast.DropTableStmt:
		for _, t := range node.Tables {
			s.addSplitNode(t.Schema.O, t.Name.O, false, node, sql)
			return nil, nil
		}
	case *ast.RenameTableStmt:
		s.addSplitNode(node.OldTable.Schema.O, node.OldTable.Name.O, false, node, sql)

	case *ast.TruncateTableStmt:
		s.addSplitNode(node.Table.Schema.O, node.Table.Name.O, true, node, sql)

	case *ast.CreateIndexStmt:

		s.addSplitNode(node.Table.Schema.O, node.Table.Name.O, false, node, sql)

	case *ast.DropIndexStmt:
		s.addSplitNode(node.Table.Schema.O, node.Table.Name.O, false, node, sql)

	case *ast.UnionStmt, *ast.SelectStmt:
		return nil, nil

	case *ast.CreateViewStmt:
		return nil, nil

		// s.appendErrorMessage(fmt.Sprintf("命令禁止! 无法创建视图'%s'.", node.ViewName.Name))

	case *ast.ShowStmt:
		return nil, nil

	case *ast.InceptionSetStmt:
		return nil, nil

	case *ast.ExplainStmt:
		return nil, nil

	case *ast.ShowOscStmt:
		return nil, nil

	case *ast.KillStmt:
		return nil, nil

	default:
		log.Infof("无匹配类型:%T\n", stmtNode)
		return nil, nil
		// s.appendErrorNo(ER_NOT_SUPPORTED_YET)
	}

	return nil, nil
}

// addNewSplitRow 添加新的split分隔节点
func (s *session) addSplitNode(db, tableName string, isDML bool, stmtNode ast.StmtNode, currentSql string) {

	if db == "" {
		db = s.dbName
	}
	key := fmt.Sprintf("%s.%s", db, tableName)
	key = strings.ToLower(key)

	if s.splitSets.id == 0 {
		s.addNewSplitNode()
		if _, ok := stmtNode.(*ast.UseStmt); !ok && s.dbName != "" {
			s.splitSets.sqlBuf.WriteString(fmt.Sprintf("use `%s`;\n", s.dbName))
		}
	} else {
		if isDmlType, ok := s.splitSets.tableList[key]; ok {
			if isDmlType != isDML {
				s.addNewSplitNode()
				if _, ok := stmtNode.(*ast.UseStmt); !ok && s.dbName != "" {
					s.splitSets.sqlBuf.WriteString(fmt.Sprintf("use `%s`;\n", s.dbName))
				}
			}
		}
	}

	s.splitSets.tableList[key] = isDML

	switch stmtNode.(type) {
	case *ast.AlterTableStmt, *ast.DropTableStmt:
		s.splitSets.ddlflag = 1
	}

	s.splitSets.sqlBuf.WriteString(currentSql)
	s.splitSets.sqlBuf.WriteString(";\n")
}

// addNewSplitRow 添加新的split分隔节点
func (s *session) addNewSplitNode() {
	if s.splitSets == nil {
		s.splitSets = NewSplitSets()
	}

	sql := s.splitSets.sqlBuf.String()

	// if len(sql) == 0{
	// 	return
	// }

	if s.splitSets.id > 0 && len(sql) > 0 {
		s.splitSets.Append(sql, "")
	}

	s.splitSets.id += 1
	s.splitSets.tableList = make(map[string]bool)
	s.splitSets.ddlflag = 0
	s.splitSets.sqlBuf = new(bytes.Buffer)
}

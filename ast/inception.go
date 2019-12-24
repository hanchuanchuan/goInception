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

package ast

import (
	. "github.com/hanchuanchuan/goInception/format"
	"github.com/hanchuanchuan/goInception/util/auth"
)

var (
	_ StmtNode = &InceptionStartStmt{}
	_ StmtNode = &InceptionCommitStmt{}
	_ StmtNode = &InceptionStmt{}
	_ StmtNode = &InceptionSetStmt{}

	_ StmtNode = &ShowOscStmt{}
)

type OscOptionType int

const (
	// 显示osc进程信息
	OscOptionNone OscOptionType = iota
	// kill进程
	OscOptionKill
	// 暂停进程(仅gh-ost支持)
	OscOptionPause
	// 恢复进程(仅gh-ost支持)
	OscOptionResume
)

const (
	// Show statement types.
	ShowLevels ShowStmtType = 1001
)

// ShowOscStmt pt-osc和gh-ost的语法解析
type ShowOscStmt struct {
	stmtNode

	Sqlsha1 string

	// Kill bool
	Tp OscOptionType
}

func (n *ShowOscStmt) Restore(ctx *RestoreCtx) error {
	return nil
}

// Accept implements Node Accept interface.
func (n *ShowOscStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ShowOscStmt)
	return v.Leave(n)
}

// InceptionSetStmt is the statement to set variables.
type InceptionSetStmt struct {
	stmtNode
	// Variables is the list of variable assignment.
	Variables []*VariableAssignment
}

func (n *InceptionSetStmt) Restore(ctx *RestoreCtx) error {
	return nil
}

// Accept implements Node Accept interface.
func (n *InceptionSetStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*InceptionSetStmt)
	for i, val := range n.Variables {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Variables[i] = node.(*VariableAssignment)
	}
	return v.Leave(n)
}

// CommitStmt is a statement to commit the current transaction.
// See https://dev.mysql.com/doc/refman/5.7/en/commit.html
type InceptionStmt struct {
	stmtNode

	resultSetNode

	Tp     ShowStmtType // Databases/Tables/Columns/....
	DBName string
	Table  *TableName  // Used for showing columns.
	Column *ColumnName // Used for `desc table column`.
	Flag   int         // Some flag parsed from sql, such as FULL.
	Full   bool
	User   *auth.UserIdentity // Used for show grants.

	// GlobalScope is used by show variables
	GlobalScope bool
	Pattern     *PatternLikeExpr
	Where       ExprNode
}

func (n *InceptionStmt) Restore(ctx *RestoreCtx) error {
	return nil
}

// Accept implements Node Accept interface.
func (n *InceptionStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*InceptionStmt)
	if n.Table != nil {
		node, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		n.Table = node.(*TableName)
	}
	if n.Column != nil {
		node, ok := n.Column.Accept(v)
		if !ok {
			return n, false
		}
		n.Column = node.(*ColumnName)
	}
	if n.Pattern != nil {
		node, ok := n.Pattern.Accept(v)
		if !ok {
			return n, false
		}
		n.Pattern = node.(*PatternLikeExpr)
	}

	switch n.Tp {
	case ShowTriggers, ShowProcedureStatus, ShowProcessList, ShowEvents:
		// We don't have any data to return for those types,
		// but visiting Where may cause resolving error, so return here to avoid error.
		return v.Leave(n)
	}

	if n.Where != nil {
		node, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		n.Where = node.(ExprNode)
	}
	return v.Leave(n)
}

// CommitStmt is a statement to commit the current transaction.
// See https://dev.mysql.com/doc/refman/5.7/en/commit.html
type InceptionStartStmt struct {
	stmtNode
}

// Accept implements Node Accept interface.
func (n *InceptionStartStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*InceptionStartStmt)
	return v.Leave(n)
}

func (n *InceptionStartStmt) Restore(ctx *RestoreCtx) error {
	return nil
}

// CommitStmt is a statement to commit the current transaction.
// See https://dev.mysql.com/doc/refman/5.7/en/commit.html
type InceptionCommitStmt struct {
	stmtNode
}

// Accept implements Node Accept interface.
func (n *InceptionCommitStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*InceptionCommitStmt)
	return v.Leave(n)
}

func (n *InceptionCommitStmt) Restore(ctx *RestoreCtx) error {
	return nil
}

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
	"testing"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/parser"
	"github.com/hanchuanchuan/goInception/sessionctx/variable"
)

func Test_checkDDLInstantMySQL(t *testing.T) {
	parser := parser.New()
	sessionVars := variable.NewSessionVars()
	parser.SetSQLMode(sessionVars.SQLMode)
	sql := "alter table t1 add column c1 int;"
	stmts, _, err := parser.Parse(sql, "", "")
	if err != nil {
		t.Error(err)
	}

	table := &TableInfo{Name: "t1",
		Fields: []FieldInfo{
			{Field: "c1", Extra: ""},
			{Field: "c2", Extra: "VIRTUAL"},
			{Field: "c3", Extra: "STORED"}}}

	for _, stmtNode := range stmts {
		switch node := stmtNode.(type) {
		case *ast.AlterTableStmt:
			canInstant := checkDDLInstantMySQL57(node)
			if canInstant {
				t.Errorf("canInstant is %v, but excepted %v", canInstant, !canInstant)
			}

			canInstant = checkDDLInstantMySQL80(node, table, 80000)
			if !canInstant {
				t.Errorf("canInstant is %v, but excepted %v", canInstant, !canInstant)
			}

		}
	}

	sql = "ALTER TABLE t1 ADD COLUMN c2 INT GENERATED ALWAYS AS (c1 + 1) VIRTUAL;"
	stmts, _, err = parser.Parse(sql, "", "")
	if err != nil {
		t.Error(err)
	}

	for _, stmtNode := range stmts {
		switch node := stmtNode.(type) {
		case *ast.AlterTableStmt:
			canInstant := checkDDLInstantMySQL57(node)
			if !canInstant {
				t.Errorf("canInstant is %v, but excepted %v", canInstant, !canInstant)
			}

			canInstant = checkDDLInstantMySQL80(node, table, 80000)
			if !canInstant {
				t.Errorf("canInstant is %v, but excepted %v", canInstant, !canInstant)
			}
		}
	}

	sql = "ALTER TABLE t1 ADD COLUMN c3 INT GENERATED ALWAYS AS (c1 + 1) STORED;"
	stmts, _, err = parser.Parse(sql, "", "")
	if err != nil {
		t.Error(err)
	}

	for _, stmtNode := range stmts {
		switch node := stmtNode.(type) {
		case *ast.AlterTableStmt:
			canInstant := checkDDLInstantMySQL57(node)
			if canInstant {
				t.Errorf("canInstant is %v, but excepted %v", canInstant, !canInstant)
			}

			canInstant = checkDDLInstantMySQL80(node, table, 80000)
			if canInstant {
				t.Errorf("canInstant is %v, but excepted %v", canInstant, !canInstant)
			}
		}
	}

	sql = "ALTER TABLE t1 ADD COLUMN c4 INT AFTER c1;"
	stmts, _, err = parser.Parse(sql, "", "")
	if err != nil {
		t.Error(err)
	}
	for _, stmtNode := range stmts {
		switch node := stmtNode.(type) {
		case *ast.AlterTableStmt:
			canInstant := checkDDLInstantMySQL80(node, table, 80000)
			if canInstant {
				t.Errorf("canInstant is %v, but excepted %v", canInstant, !canInstant)
			}
		}
	}
}

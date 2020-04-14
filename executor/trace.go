// Copyright 2018 PingCAP, Inc.
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

package executor

import (
	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/planner"
	plannercore "github.com/hanchuanchuan/goInception/planner/core"
	"github.com/hanchuanchuan/goInception/util/chunk"
	"github.com/pingcap/errors"
	"golang.org/x/net/context"
)

// TraceExec represents a root executor of trace query.
type TraceExec struct {
	baseExecutor
	// exhausted being true means there is no more result.
	exhausted bool
	// stmtNode is the real query ast tree and it is used for building real query's plan.
	stmtNode ast.StmtNode

	builder *executorBuilder
}

// Next executes real query and collects span later.
func (e *TraceExec) Next(ctx context.Context, chk *chunk.Chunk) error {
	chk.Reset()
	if e.exhausted {
		return nil
	}

	// record how much time was spent for optimizeing plan
	stmtPlan, err := planner.Optimize(e.builder.ctx, e.stmtNode, e.builder.is)
	if err != nil {
		return err
	}

	pp, ok := stmtPlan.(plannercore.PhysicalPlan)
	if !ok {
		return errors.New("cannot cast logical plan to physical plan")
	}

	// append select executor to trace executor
	stmtExec := e.builder.build(pp)

	err = stmtExec.Open(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	stmtExecChk := stmtExec.newFirstChunk()

	for {
		if err := stmtExec.Next(ctx, stmtExecChk); err != nil {
			return errors.Trace(err)
		}
		if stmtExecChk.NumRows() == 0 {
			break
		}
	}

	e.exhausted = true
	return nil
}

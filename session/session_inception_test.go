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

package session_test

import (
	"fmt"
	// "os"
	// "sync"
	// "sync/atomic"
	"testing"
	// "time"

	// "github.com/hanchuanchuan/tidb/config"
	"github.com/hanchuanchuan/tidb/domain"
	// "github.com/hanchuanchuan/tidb/executor"
	"github.com/hanchuanchuan/tidb/kv"
	// "github.com/hanchuanchuan/tidb/model"
	// "github.com/hanchuanchuan/tidb/mysql"
	// "github.com/hanchuanchuan/tidb/parser"
	// plannercore "github.com/hanchuanchuan/tidb/planner/core"
	// "github.com/hanchuanchuan/tidb/privilege/privileges"
	"github.com/hanchuanchuan/tidb/session"
	// "github.com/hanchuanchuan/tidb/sessionctx"
	// "github.com/hanchuanchuan/tidb/sessionctx/variable"
	"github.com/hanchuanchuan/tidb/store/mockstore"
	"github.com/hanchuanchuan/tidb/store/mockstore/mocktikv"
	// "github.com/hanchuanchuan/tidb/table/tables"
	// "github.com/hanchuanchuan/tidb/terror"
	// "github.com/hanchuanchuan/tidb/types"
	// "github.com/hanchuanchuan/tidb/util/auth"
	// "github.com/hanchuanchuan/tidb/util/logutil"
	// "github.com/hanchuanchuan/tidb/util/sqlexec"
	"github.com/hanchuanchuan/tidb/util/testkit"
	"github.com/hanchuanchuan/tidb/util/testleak"
	// "github.com/hanchuanchuan/tidb/util/testutil"
	. "github.com/pingcap/check"
	// "github.com/pingcap/tipb/go-binlog"
	// "golang.org/x/net/context"
	// "google.golang.org/grpc"
)

var _ = Suite(&testSessionIncSuite{})

func Test(t *testing.T) { TestingT(t) }

// func TestT(t *testing.T) {
// 	logutil.InitLogger(&logutil.LogConfig{
// 		Level: "warn",
// 	})
// 	CustomVerboseFlag = true
// 	TestingT(t)
// }

type testSessionIncSuite struct {
	cluster   *mocktikv.Cluster
	mvccStore mocktikv.MVCCStore
	store     kv.Storage
	dom       *domain.Domain
}

func (s *testSessionIncSuite) SetUpSuite(c *C) {
	testleak.BeforeTest()
	s.cluster = mocktikv.NewCluster()
	mocktikv.BootstrapWithSingleStore(s.cluster)
	s.mvccStore = mocktikv.MustNewMVCCStore()
	store, err := mockstore.NewMockTikvStore(
		mockstore.WithCluster(s.cluster),
		mockstore.WithMVCCStore(s.mvccStore),
	)
	c.Assert(err, IsNil)
	s.store = store
	session.SetSchemaLease(0)
	session.SetStatsLease(0)
	s.dom, err = session.BootstrapSession(s.store)
	c.Assert(err, IsNil)
}

func (s *testSessionIncSuite) TearDownSuite(c *C) {
	s.dom.Close()
	s.store.Close()
	testleak.AfterTest(c)()
}

func (s *testSessionIncSuite) TearDownTest(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	r := tk.MustQuery("show tables")
	for _, tb := range r.Rows() {
		tableName := tb[0]
		tk.MustExec(fmt.Sprintf("drop table %v", tableName))
	}
}

func makeSql(tk *testkit.TestKit, sql string) *testkit.Result {
	a := `/*--user=admin;--password=han123;--host=127.0.0.1;--check=1;--backup=1;--port=3306;--enable-ignore-warnings;*/
inception_magic_start;
use test;
%s;
inception_magic_commit;`
	return tk.MustQueryInc(fmt.Sprintf(a, sql))
}

func (s *testSessionIncSuite) TestBegin(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	res := tk.MustQueryInc("create table t1(id int);")

	c.Assert(int(tk.Se.AffectedRows()), Equals, 1)

	for _, row := range res.Rows() {
		c.Assert(row[2], Equals, "2")
		c.Assert(row[4], Equals, "Must start as begin statement.")
	}
}

func (s *testSessionIncSuite) TestNoSourceInfo(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	res := tk.MustQueryInc("inception_magic_start;\ncreate table t1(id int);")

	c.Assert(int(tk.Se.AffectedRows()), Equals, 1)

	for _, row := range res.Rows() {
		c.Assert(row[2], Equals, "2")
		c.Assert(row[4], Equals, "Invalid source infomation.")
	}
}

func (s *testSessionIncSuite) TestWrongDBName(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	res := tk.MustQueryInc(`/*--user=admin;--password=han123;--host=127.0.0.1;--check=1;--backup=1;--port=3306;--enable-ignore-warnings;*/
inception_magic_start;create table t1(id int);inception_magic_commit;`)

	c.Assert(int(tk.Se.AffectedRows()), Equals, 1)

	for _, row := range res.Rows() {
		c.Assert(row[2], Equals, "2")
		c.Assert(row[4], Equals, "Incorrect database name ''.")
	}
}

func (s *testSessionIncSuite) TestEnd(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	res := tk.MustQueryInc(`/*--user=admin;--password=han123;--host=127.0.0.1;--check=1;--backup=1;--port=3306;--enable-ignore-warnings;*/
inception_magic_start;use test;create table t1(id int);`)

	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)

	row := res.Rows()[2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Must end with commit.")
}

func (s *testSessionIncSuite) TestCreateTable(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	res := makeSql(tk, "create table t1(id int);create table t1(id int);")

	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)

	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 't1' already exists.")
}

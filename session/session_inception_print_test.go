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
	"strconv"
	"strings"
	"testing"

	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/session"
	"github.com/hanchuanchuan/goInception/util/testkit"
	. "github.com/pingcap/check"
)

var _ = Suite(&testSessionPrintSuite{})

func TestPrint(t *testing.T) {
	TestingT(t)
}

type testSessionPrintSuite struct {
	testCommon
}

func (s *testSessionPrintSuite) SetUpSuite(c *C) {

	s.initSetUp(c)

	config.GetGlobalConfig().Inc.Lang = "en-US"
	config.GetGlobalConfig().Inc.EnableFingerprint = true
	config.GetGlobalConfig().Inc.SqlSafeUpdates = 0

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}
}

func (s *testSessionPrintSuite) TearDownSuite(c *C) {
	s.tearDownSuite(c)
}

func (s *testSessionPrintSuite) makeSQL(sql string) *testkit.Result {
	a := `/*--user=test;--password=test;--host=127.0.0.1;--query-print=1;--backup=0;--port=3306;--enable-ignore-warnings;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	return s.tk.MustQueryInc(fmt.Sprintf(a, sql))
}

func (s *testSessionPrintSuite) testErrorCode(c *C, sql string, errors ...*session.SQLError) {
	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	res := s.makeSQL(sql)
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]

	errCode := 0
	if len(errors) > 0 {
		for _, e := range errors {
			level := session.GetErrorLevel(e.Code)
			if int(level) > errCode {
				errCode = int(level)
			}
		}
	}

	if errCode > 0 {
		errMsgs := []string{}
		for _, e := range errors {
			errMsgs = append(errMsgs, e.Error())
		}
		c.Assert(row[4], Equals, strings.Join(errMsgs, "\n"), Commentf("%v", row))
	}

	c.Assert(row[2], Equals, strconv.Itoa(errCode), Commentf("%v", row))

	s.rows = res.Rows()
}

func (s *testSessionPrintSuite) TestInsert(c *C) {

	res := s.makeSQL("insert into t1 values(1);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0", Commentf("%v", row))

	res = s.makeSQL("insert into t1 values;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2", Commentf("%v", row))
	// c.Assert(row[4], Equals, "line 1 column 21 near \"\" (total length 21)", Commentf("%v", row))
	c.Assert(row[4], Equals, "You have an error in your SQL syntax, near '' at line 1", Commentf("%v", row))

}

func (s *testSessionPrintSuite) TestUpdate(c *C) {

	res := s.makeSQL(`update t1 set c1=1 where a=1;`)
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0", Commentf("%v", row))

	res = s.makeSQL("update t1 inner join t2 on t1.id=t2.id set c2=1 where t1.c1=1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0", Commentf("%v", row))

}

func (s *testSessionPrintSuite) TestDelete(c *C) {

	res := s.makeSQL(`delete from t1 where id=1;`)
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0", Commentf("%v", row))

	// res = s.makeSQL("update t1 inner join t2 on t1.id=t2.id set c2=1 where t1.c1=1;")
	// row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0", Commentf("%v", row))

}

func (s *testSessionPrintSuite) TestAlterTable(c *C) {

	res := s.makeSQL(`alter table t1 add column c1 int;`)
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0", Commentf("%v", row))

}

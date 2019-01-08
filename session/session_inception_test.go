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
	// log "github.com/sirupsen/logrus"

	"github.com/hanchuanchuan/tidb/config"
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
use test_inc;
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
inception_magic_start;use test_inc;create table t1(id int);`)

	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)

	row := res.Rows()[2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Must end with commit.")
}

func (s *testSessionIncSuite) TestCreateTable(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	res := makeSql(tk, "create table t1(id int);create table t1(id int);")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)

	// 表存在
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 't1' already exists.")

	// 重复列
	res = makeSql(tk, "create table t1(c1 int,c1 int);")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Duplicate column name 'c1'.")

	// 主键
	config.GetGlobalConfig().Inc.CheckPrimaryKey = true
	res = makeSql(tk, "create table t1(id int);")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set a primary key for table 't1'.")
	config.GetGlobalConfig().Inc.CheckPrimaryKey = false

	// 数据类型 警告
	res = makeSql(tk, "create table t1(id int,c1 bit);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c1'.")

	res = makeSql(tk, "create table t1(id int,c1 enum('red', 'blue', 'black'));")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c1'.")

	res = makeSql(tk, "create table t1(id int,c1 set('red', 'blue', 'black'));")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c1'.")

	// char列建议
	config.GetGlobalConfig().Inc.MaxCharLength = 100
	res = makeSql(tk, `create table t1(id int,c1 char(200));`)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set column 'c1' to VARCHAR type.")

	// 字符集
	res = makeSql(tk, `create table t1(id int,c1 varchar(20) character set utf8,
		c2 varchar(20) COLLATE utf8_bin);`)
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "表 't1' 列 'c1' 禁止设置字符集!\n表 't1' 列 'c2' 禁止设置字符集!")

	res = makeSql(tk, `create table t1(id int,c1 varchar(20)) character set utf8;`)
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "表 't1' 禁止设置字符集!")

	res = makeSql(tk, `create table t1(id int,c1 varchar(20)) COLLATE utf8_bin;`)
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "表 't1' 禁止设置字符集!")

	// 关键字
	config.GetGlobalConfig().Inc.EnableIdentiferKeyword = false
	config.GetGlobalConfig().Inc.CheckIdentifier = true

	res = makeSql(tk, "create table t1(id int, TABLES varchar(20),`c1$` varchar(20),c1234567890123456789012345678901234567890123456789012345678901234567890 varchar(20));")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Identifier 'TABLES' is keyword in MySQL.\nIdentifier 'c1$' is invalid, valid options: [a-z|A-Z|0-9|_].\nIdentifier name 'c1234567890123456789012345678901234567890123456789012345678901234567890' is too long.")

	// 列注释
	config.GetGlobalConfig().Inc.CheckColumnComment = true
	res = makeSql(tk, "create table t1(c1 varchar(20));")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Column 'c1' in table 't1' have no comments.")

	config.GetGlobalConfig().Inc.CheckColumnComment = false

	// 表注释
	config.GetGlobalConfig().Inc.CheckTableComment = true
	res = makeSql(tk, "create table t1(c1 varchar(20));")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set comments for table 't1'.")

	config.GetGlobalConfig().Inc.CheckTableComment = false
	res = makeSql(tk, "create table t1(c1 varchar(20));")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "0")

	// 无效默认值
	res = makeSql(tk, "create table t1(id int,c1 int default '');")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Invalid default value for column 'c1'.")

	// blob/text字段
	config.GetGlobalConfig().Inc.EnableBlobType = false
	res = makeSql(tk, "create table t1(id int,c1 blob, c2 text);")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.\nType blob/text is used in column 'c2'.")

	config.GetGlobalConfig().Inc.EnableBlobType = true
	res = makeSql(tk, "create table t1(id int,c1 blob not null);")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "TEXT/BLOB Column 'c1' in table 't1' can't  been not null.")

	// 检查默认值
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = true
	res = makeSql(tk, "create table t1(c1 varchar(10));")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set Default value for column 'c1' in table 't1'")
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = false

}

func (s *testSessionIncSuite) TestDropTable(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableDropTable = false

	res := makeSql(tk, "create table t1(id int);drop table t1;")

	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)

	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "禁用【DROP】|【TRUNCATE】删除/清空表 't1', 请改用RENAME重写.")

	config.GetGlobalConfig().Inc.EnableDropTable = true

	res = makeSql(tk, "create table t1(id int);drop table t1;")

	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)

	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
}

func (s *testSessionIncSuite) TestAlterTableAddColumn(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	res := makeSql(tk, "create table t1(id int);alter table t1 add column c1 int;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	res = makeSql(tk, "create table t1(id int);alter table t1 add column c1 int;alter table t1 add column c1 int;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 4)

	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c1' have existed.")

	res = makeSql(tk, "create table t1(id int);alter table t1 add column c1 int first;alter table t1 add column c2 int after c1;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 4)
	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2")
	}

	// after 不存在的列
	res = makeSql(tk, "create table t1(id int);alter table t1 add column c2 int after c1;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c1' not existed.")

	// 数据类型 警告
	res = makeSql(tk, "create table t1(id int);alter table t1 add column c2 bit;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

	res = makeSql(tk, "create table t1(id int);alter table t1 add column c2 enum('red', 'blue', 'black');")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

	res = makeSql(tk, "create table t1(id int);alter table t1 add column c2 set('red', 'blue', 'black');")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

	// char列建议
	config.GetGlobalConfig().Inc.MaxCharLength = 100
	res = makeSql(tk, `create table t1(id int);
		alter table t1 add column c1 char(200);
		alter table t1 add column c2 varchar(200);`)
	c.Assert(int(tk.Se.AffectedRows()), Equals, 4)
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set column 'c1' to VARCHAR type.")

	row = res.Rows()[3]
	c.Assert(row[2], Equals, "0")

	// 字符集
	res = makeSql(tk, `create table t1(id int);
		alter table t1 add column c1 varchar(20) character set utf8;
		alter table t1 add column c2 varchar(20) COLLATE utf8_bin;`)
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "表 't1' 列 'c1' 禁止设置字符集!")

	row = res.Rows()[3]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "表 't1' 列 'c2' 禁止设置字符集!")

	// 关键字
	config.GetGlobalConfig().Inc.EnableIdentiferKeyword = false
	config.GetGlobalConfig().Inc.CheckIdentifier = true

	res = makeSql(tk, "create table t1(id int);alter table t1 add column TABLES varchar(20);alter table t1 add column `c1$` varchar(20);alter table t1 add column c1234567890123456789012345678901234567890123456789012345678901234567890 varchar(20);")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Identifier 'TABLES' is keyword in MySQL.")
	row = res.Rows()[3]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Identifier 'c1$' is invalid, valid options: [a-z|A-Z|0-9|_].")
	row = res.Rows()[4]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Identifier name 'c1234567890123456789012345678901234567890123456789012345678901234567890' is too long.")

	// 列注释
	config.GetGlobalConfig().Inc.CheckColumnComment = true
	res = makeSql(tk, "create table t1(id int);alter table t1 add column c1 varchar(20);")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Column 'c1' in table 't1' have no comments.")

	config.GetGlobalConfig().Inc.CheckColumnComment = false

	// 无效默认值
	res = makeSql(tk, "create table t1(id int);alter table t1 add column c1 int default '';")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Invalid default value for column 'c1'.")

	// blob/text字段
	config.GetGlobalConfig().Inc.EnableBlobType = false
	res = makeSql(tk, "create table t1(id int);alter table t1 add column c1 blob;alter table t1 add column c2 text;")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

	row = res.Rows()[3]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c2'.")

	config.GetGlobalConfig().Inc.EnableBlobType = true
	res = makeSql(tk, "create table t1(id int);alter table t1 add column c1 blob not null;")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "TEXT/BLOB Column 'c1' in table 't1' can't  been not null.")

	// 检查默认值
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = true
	res = makeSql(tk, "create table t1(id int);alter table t1 add column c1 varchar(10);")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set Default value for column 'c1' in table 't1'")
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = false

}

func (s *testSessionIncSuite) TestAlterTableAlterColumn(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	res := makeSql(tk, "create table t1(id int);alter table t1 alter column id set default '';")
	row := res.Rows()[2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Invalid default value for column 'id'.")

	res = makeSql(tk, "create table t1(id int);alter table t1 alter column id set default '1';")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "0")

	res = makeSql(tk, "create table t1(id int);alter table t1 alter column id drop default ;alter table t1 alter column id set default '1';")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "0")
	row = res.Rows()[3]
	c.Assert(row[2], Equals, "0")
}

func (s *testSessionIncSuite) TestAlterTableModifyColumn(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	res := makeSql(tk, "create table t1(id int,c1 int);alter table t1 modify column c1 int first;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2")
	}

	res = makeSql(tk, "create table t1(id int,c1 int);alter table t1 modify column id int after c1;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2")
	}

	// after 不存在的列
	res = makeSql(tk, "create table t1(id int);alter table t1 modify column c1 int after id;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c1' not existed.")

	res = makeSql(tk, "create table t1(id int,c1 int);alter table t1 modify column c1 int after id1;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.id1' not existed.")

	// 数据类型 警告
	res = makeSql(tk, "create table t1(id bit);alter table t1 modify column id bit;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

	res = makeSql(tk, "create table t1(id enum('red', 'blue'));alter table t1 modify column id enum('red', 'blue', 'black');")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

	res = makeSql(tk, "create table t1(id set('red'));alter table t1 modify column id set('red', 'blue', 'black');")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

	// char列建议
	config.GetGlobalConfig().Inc.MaxCharLength = 100
	res = makeSql(tk, `create table t1(id int,c1 char(10));
		alter table t1 modify column c1 char(200);`)
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set column 'c1' to VARCHAR type.")

	// 字符集
	res = makeSql(tk, `create table t1(id int,c1 varchar(20));
		alter table t1 modify column c1 varchar(20) character set utf8;
		alter table t1 modify column c1 varchar(20) COLLATE utf8_bin;`)
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "表 't1' 列 'c1' 禁止设置字符集!")

	row = res.Rows()[3]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "表 't1' 列 'c1' 禁止设置字符集!")

	// 列注释
	config.GetGlobalConfig().Inc.CheckColumnComment = true
	res = makeSql(tk, "create table t1(id int,c1 varchar(10));alter table t1 modify column c1 varchar(20);")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Column 'c1' in table 't1' have no comments.")

	config.GetGlobalConfig().Inc.CheckColumnComment = false

	// 无效默认值
	res = makeSql(tk, "create table t1(id int,c1 int);alter table t1 modify column c1 int default '';")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Invalid default value for column 'c1'.")

	// blob/text字段
	config.GetGlobalConfig().Inc.EnableBlobType = false
	res = makeSql(tk, "create table t1(id int,c1 varchar(10));alter table t1 modify column c1 blob;alter table t1 modify column c1 text;")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

	row = res.Rows()[3]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

	config.GetGlobalConfig().Inc.EnableBlobType = true
	res = makeSql(tk, "create table t1(id int,c1 blob);alter table t1 modify column c1 blob not null;")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "TEXT/BLOB Column 'c1' in table 't1' can't  been not null.")

	// 检查默认值
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = true
	res = makeSql(tk, "create table t1(id int,c1 varchar(5));alter table t1 modify column c1 varchar(10);")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set Default value for column 'c1' in table 't1'")
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = false

	// 变更类型
	res = makeSql(tk, "create table t1(c1 int,c1 int);alter table t1 modify column c1 varchar(10);")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "类型转换警告: 列 't1.c1' int(11) -> varchar(10).")

	res = makeSql(tk, "create table t1(c1 char(100));alter table t1 modify column c1 char(20);")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "0")

	res = makeSql(tk, "create table t1(c1 varchar(100));alter table t1 modify column c1 varchar(10);")
	row = res.Rows()[2]
	c.Assert(row[2], Equals, "0")
}

func (s *testSessionIncSuite) TestAlterTableDropColumn(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	res := makeSql(tk, "create table t1(id int,c1 int);alter table t1 drop column c2;")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c2' not existed.")

	res = makeSql(tk, "create table t1(id int,c1 int);alter table t1 drop column c1;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
}

func (s *testSessionIncSuite) TestInsert(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false

	// 表不存在
	res := makeSql(tk, "insert into t1 values(1,1);")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.")

	// 列数不匹配
	res = makeSql(tk, "create table t1(id int,c1 int);insert into t1(id) values(1,1);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column count doesn't match value count at row 1.")

	res = makeSql(tk, "create table t1(id int,c1 int);insert into t1(id) values(1),(2,1);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column count doesn't match value count at row 2.")

	res = makeSql(tk, "create table t1(id int,c1 int not null);insert into t1(id,c1) select 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column count doesn't match value count at row 1.")

	// 列重复
	res = makeSql(tk, "create table t1(id int,c1 int);insert into t1(id,id) values(1,1);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'id' specified twice in table 't1'.")

	res = makeSql(tk, "create table t1(id int,c1 int);insert into t1(id,id) select 1,1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'id' specified twice in table 't1'.")

	// 字段警告
	config.GetGlobalConfig().Inc.CheckInsertField = true
	res = makeSql(tk, "create table t1(id int,c1 int);insert into t1 values(1,1);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "insert语句需要指定字段列表.")
	config.GetGlobalConfig().Inc.CheckInsertField = false

	res = makeSql(tk, "create table t1(id int,c1 int);insert into t1(id) values();")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "insert语句需要指定值列表.")

	// 列不允许为空
	res = makeSql(tk, "create table t1(id int,c1 int not null);insert into t1(id,c1) values(1,null);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'test_inc.t1.c1' cannot be null in 1 row.")

	res = makeSql(tk, "create table t1(id int,c1 int not null default 1);insert into t1(id,c1) values(1,null);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'test_inc.t1.c1' cannot be null in 1 row.")

	// insert select 表不存在
	res = makeSql(tk, "create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t2;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t2' doesn't exist.")

	// select where
	config.GetGlobalConfig().Inc.CheckDMLWhere = true
	res = makeSql(tk, "create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "selete语句请指定where条件.")
	config.GetGlobalConfig().Inc.CheckDMLWhere = false

	// limit
	config.GetGlobalConfig().Inc.CheckDMLLimit = true
	res = makeSql(tk, "create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1 limit 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLLimit = false

	// order by rand()
	// config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
	res = makeSql(tk, "create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1 order by rand();")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Order by rand is not allowed in select statement.")
	// config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

	// 受影响行数
	res = makeSql(tk, "create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "2")

	res = makeSql(tk, "create table t1(id int,c1 int );insert into t1(id,c1) select 1,null;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "1")
}

func (s *testSessionIncSuite) TestUpdate(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false

	// 表不存在
	res := makeSql(tk, "update t1 set c1 = 1;")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.")

	res = makeSql(tk, "create table t1(id int);update t1 set c1 = 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'c1' not existed.")

	res = makeSql(tk, "create table t1(id int,c1 int);update t1 set c1 = 1,c2 = 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c2' not existed.")

	// where
	config.GetGlobalConfig().Inc.CheckDMLWhere = true
	res = makeSql(tk, "create table t1(id int,c1 int);update t1 set c1 = 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "selete语句请指定where条件.")
	config.GetGlobalConfig().Inc.CheckDMLWhere = false

	// limit
	config.GetGlobalConfig().Inc.CheckDMLLimit = true
	res = makeSql(tk, "create table t1(id int,c1 int);update t1 set c1 = 1 limit 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLLimit = false

	// order by rand()
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
	res = makeSql(tk, "create table t1(id int,c1 int);update t1 set c1 = 1 order by rand();")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Order by is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

	// 受影响行数
	res = makeSql(tk, "create table t1(id int,c1 int);update t1 set c1 = 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "0")
}

func (s *testSessionIncSuite) TestDelete(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false

	// 表不存在
	res := makeSql(tk, "delete from t1 where c1 = 1;")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.\nColumn 'c1' not existed.")

	// res = makeSql(tk, "create table t1(id int);delete from t1 where c1 = 1;")
	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "2")
	// c.Assert(row[4], Equals, "Column 'c1' not existed.")

	// res = makeSql(tk, "create table t1(id int,c1 int);delete from t1 where c1 = 1 and c2 = 1;")
	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "2")
	// c.Assert(row[4], Equals, "Column 't1.c2' not existed.")

	// where
	config.GetGlobalConfig().Inc.CheckDMLWhere = true
	res = makeSql(tk, "create table t1(id int,c1 int);delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "selete语句请指定where条件.")
	config.GetGlobalConfig().Inc.CheckDMLWhere = false

	// limit
	config.GetGlobalConfig().Inc.CheckDMLLimit = true
	res = makeSql(tk, "create table t1(id int,c1 int);delete from t1 where id = 1 limit 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLLimit = false

	// order by rand()
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
	res = makeSql(tk, "create table t1(id int,c1 int);delete from t1 where id = 1 order by rand();")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Order by is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

	// 表不存在
	res = makeSql(tk, `create table t1(id int primary key,c1 int);
		create table t2(id int primary key,c1 int,c2 int);
		delete from t3 where id1 =1;`)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t3' doesn't exist.\nColumn 'id1' not existed.")

	res = makeSql(tk, `create table t1(id int primary key,c1 int);
		create table t2(id int primary key,c1 int,c2 int);
		delete from t1 where id1 =1;`)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'id1' not existed.")

	res = makeSql(tk, `create table t1(id int primary key,c1 int);
		create table t2(id int primary key,c1 int,c2 int);
		delete t2 from t1 inner join t2 on t1.id=t2.id2 where c11=1;`)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't2.id2' not existed.\nColumn 'c11' not existed.")

	// 受影响行数
	res = makeSql(tk, "create table t1(id int,c1 int);delete from t1 where id = 1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "0")
}

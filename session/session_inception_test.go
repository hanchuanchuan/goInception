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
	// log "github.com/sirupsen/logrus"

	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/domain"
	// "github.com/hanchuanchuan/goInception/executor"
	"github.com/hanchuanchuan/goInception/kv"
	// "github.com/hanchuanchuan/goInception/model"
	// "github.com/hanchuanchuan/goInception/mysql"
	// "github.com/hanchuanchuan/goInception/parser"
	// plannercore "github.com/hanchuanchuan/goInception/planner/core"
	// "github.com/hanchuanchuan/goInception/privilege/privileges"
	"github.com/hanchuanchuan/goInception/session"
	// "github.com/hanchuanchuan/goInception/sessionctx"
	// "github.com/hanchuanchuan/goInception/sessionctx/variable"
	"github.com/hanchuanchuan/goInception/store/mockstore"
	"github.com/hanchuanchuan/goInception/store/mockstore/mocktikv"
	// "github.com/hanchuanchuan/goInception/table/tables"
	// "github.com/hanchuanchuan/goInception/terror"
	// "github.com/hanchuanchuan/goInception/types"
	// "github.com/hanchuanchuan/goInception/util/auth"
	// "github.com/hanchuanchuan/goInception/util/logutil"
	// "github.com/hanchuanchuan/goInception/util/sqlexec"
	"github.com/hanchuanchuan/goInception/util/testkit"
	"github.com/hanchuanchuan/goInception/util/testleak"
	// "github.com/hanchuanchuan/goInception/util/testutil"
	. "github.com/pingcap/check"
	// "github.com/pingcap/tipb/go-binlog"
	// "golang.org/x/net/context"
	// "google.golang.org/grpc"
)

var _ = Suite(&testSessionIncSuite{})

func Test1(t *testing.T) {
	TestingT(t)
}

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
	tk        *testkit.TestKit
}

func (s *testSessionIncSuite) SetUpSuite(c *C) {

	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

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
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	} else {
		s.dom.Close()
		s.store.Close()
		testleak.AfterTest(c)()
	}
}

func (s *testSessionIncSuite) TearDownTest(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

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

func (s *testSessionIncSuite) testErrorCode(c *C, sql string, errors ...*session.SQLError) {
	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	res := makeSql(s.tk, sql)
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
		c.Assert(row[4], Equals, strings.Join(errMsgs, "\n"))
	}

	c.Assert(row[2], Equals, strconv.Itoa(errCode))
}

func (s *testSessionIncSuite) TestBegin(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

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

	sql := ""

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

	sql = "create table test_error_code1 (c1 int, c2 int, c2 int)"
	s.testErrorCode(c, sql, session.NewErr(session.ER_DUP_FIELDNAME, "c2"))

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

	config.GetGlobalConfig().Inc.EnableSetCharset = false
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

	// 支持innodb引擎
	res = makeSql(tk, "create table t1(c1 varchar(10))engine = innodb;")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "0")

	res = makeSql(tk, "create table t1(c1 varchar(10))engine = myisam;")
	row = res.Rows()[1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Set engine to innodb for table 't1'.")

	// 时间戳 timestamp默认值
	sql = "create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 timestamp default CURRENT_TIMESTAMP);"
	s.testErrorCode(c, sql,
		session.NewErrf("Incorrect table definition; there can be only one TIMESTAMP column with CURRENT_TIMESTAMP in DEFAULT or ON UPDATE clause"))

	sql = "create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 timestamp ON UPDATE CURRENT_TIMESTAMP);"
	s.testErrorCode(c, sql)

	sql = "create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 date default CURRENT_TIMESTAMP);"
	s.testErrorCode(c, sql,
		session.NewErrf("Invalid default value for column '%s'.", "t2"))

	// create table
	sql = "create table test_error_code1 (c1 int, c2 int, c2 int)"
	s.testErrorCode(c, sql, session.NewErr(session.ER_DUP_FIELDNAME, "c2"))

	sql = "create table test_error_code1 (c1 int, aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa int)"
	s.testErrorCode(c, sql, session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	sql = "create table aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa(a int)"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	sql = "create table test_error_code1 (c1 int, c2 int, key aa (c1, c2), key aa (c1))"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_DUP_INDEX, "aa", "test_inc", "test_error_code1"),
		session.NewErr(session.ER_DUP_KEYNAME, "aa"))

	sql = "create table test_error_code1 (c1 int, c2 int, c3 int, key(c_not_exist))"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_NAME_FOR_INDEX, "NULL", "test_error_code1"),
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "test_error_code1.c_not_exist"))

	sql = "create table test_error_code1 (c1 int, c2 int, c3 int, primary key(c_not_exist))"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "test_error_code1.c_not_exist"))

	sql = "create table test_error_code1 (c1 int not null default '')"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_DEFAULT, "c1"))

	sql = "CREATE TABLE `t` (`a` double DEFAULT 1.0 DEFAULT 2.0 DEFAULT now());"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_DEFAULT, "a"))

	sql = "CREATE TABLE `t` (`a` double DEFAULT now());"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_DEFAULT, "a"))

	// 字符集
	config.GetGlobalConfig().Inc.EnableSetCharset = false
	config.GetGlobalConfig().Inc.SupportCharset = ""
	sql = "create table t1(a int) character set utf8;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TABLE_CHARSET_MUST_NULL, "t1"))

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8mb4"
	sql = "create table t1(a int) character set utf8;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_NAMES_MUST_UTF8, "utf8mb4"))

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	sql = "create table t1(a int) character set utf8;"
	s.testErrorCode(c, sql)

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	sql = "create table t1(a int) character set laitn1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_NAMES_MUST_UTF8, "utf8,utf8mb4"))

	// 外键
	sql = "create table test_error_code (a int not null ,b int not null,c int not null, d int not null, foreign key (b, c) references product(id));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_NAME_FOR_INDEX, "NULL", "test_error_code"),
		session.NewErr(session.ER_FOREIGN_KEY, "test_error_code"))

	sql = "create table test_error_code (a int not null ,b int not null,c int not null, d int not null, foreign key fk_1(b, c) references product(id));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_FOREIGN_KEY, "test_error_code"))

	sql = "create table test_error_code_2;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_MUST_HAVE_COLUMNS))

	sql = "create table test_error_code_2 (unique(c1));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_MUST_HAVE_COLUMNS))

	sql = "create table test_error_code_2(c1 int, c2 int, c3 int, primary key(c1), primary key(c2));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_MULTIPLE_PRI_KEY))

	config.GetGlobalConfig().Inc.EnableBlobType = false
	sql = "create table test_error_code_3(pt blob ,primary key (pt));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_USE_TEXT_OR_BLOB, "pt"),
		session.NewErr(session.ER_BLOB_USED_AS_KEY, "pt"))

	config.GetGlobalConfig().Inc.EnableBlobType = true
	sql = "create table test_error_code_3(pt blob ,primary key (pt));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_BLOB_USED_AS_KEY, "pt"))

	// 索引长度
	sql = "create table test_error_code_3(a text, unique (a(3073)));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_NAME_FOR_INDEX, "NULL", "test_error_code_3"),
		session.NewErr(session.ER_TOO_LONG_KEY, "", 3072))

	sql = "create table test_error_code_3(c1 int,c2 text, unique uq_1(c1,c2(3069)));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TOO_LONG_KEY, "uq_1", 3072))

	sql = "create table test_error_code_3(c1 int,c2 text, unique uq_1(c1,c2(3068)));"
	s.testErrorCode(c, sql)

	sql = "create table test_error_code_3(`id` int, key `primary`(`id`));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_NAME_FOR_INDEX, "primary", "test_error_code_3"))

	sql = "create table t2(c1.c2 blob default null);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_TABLE_NAME, "c1"))

	// sql = "create table t2 (id int default null primary key , age int);"
	// s.testErrorCode(c, sql, tmysql.ErrInvalidDefault)
	// sql = "create table t2 (id int null primary key , age int);"
	// s.testErrorCode(c, sql, tmysql.ErrPrimaryCantHaveNull)
	// sql = "create table t2 (id int default null, age int, primary key(id));"
	// s.testErrorCode(c, sql, tmysql.ErrPrimaryCantHaveNull)
	// sql = "create table t2 (id int null, age int, primary key(id));"
	// s.testErrorCode(c, sql, tmysql.ErrPrimaryCantHaveNull)

	// sql = "create table t2 (id int primary key , age int);"
	// s.tk.MustExec(sql)

	// // add column
	// sql = "alter table test_error_code_succ add column c1 int"
	// s.testErrorCode(c, sql, tmysql.ErrDupFieldName)
	// sql = "alter table test_error_code_succ add column aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa int"
	// s.testErrorCode(c, sql, tmysql.ErrTooLongIdent)
	// sql = "alter table test_comment comment 'test comment'"
	// s.testErrorCode(c, sql, tmysql.ErrNoSuchTable)
	// sql = "alter table test_error_code_succ add column `a ` int ;"
	// s.testErrorCode(c, sql, tmysql.ErrWrongColumnName)
	// s.tk.MustExec("create table test_on_update (c1 int, c2 int);")
	// sql = "alter table test_on_update add column c3 int on update current_timestamp;"
	// s.testErrorCode(c, sql, tmysql.ErrInvalidOnUpdate)
	// sql = "create table test_on_update_2(c int on update current_timestamp);"
	// s.testErrorCode(c, sql, tmysql.ErrInvalidOnUpdate)

	// // drop column
	// sql = "alter table test_error_code_succ drop c_not_exist"
	// s.testErrorCode(c, sql, tmysql.ErrCantDropFieldOrKey)
	// s.tk.MustExec("create table test_drop_column (c1 int );")
	// sql = "alter table test_drop_column drop column c1;"
	// s.testErrorCode(c, sql, tmysql.ErrCantRemoveAllFields)
	// // add index
	// sql = "alter table test_error_code_succ add index idx (c_not_exist)"
	// s.testErrorCode(c, sql, tmysql.ErrKeyColumnDoesNotExits)
	// s.tk.Exec("alter table test_error_code_succ add index idx (c1)")
	// sql = "alter table test_error_code_succ add index idx (c1)"
	// s.testErrorCode(c, sql, tmysql.ErrDupKeyName)
	// // drop index
	// sql = "alter table test_error_code_succ drop index idx_not_exist"
	// s.testErrorCode(c, sql, tmysql.ErrCantDropFieldOrKey)
	// sql = "alter table test_error_code_succ drop column c3"
	// s.testErrorCode(c, sql, int(tmysql.ErrUnknown))
	// // modify column
	// sql = "alter table test_error_code_succ modify testx.test_error_code_succ.c1 bigint"
	// s.testErrorCode(c, sql, tmysql.ErrWrongDBName)
	// sql = "alter table test_error_code_succ modify t.c1 bigint"
	// s.testErrorCode(c, sql, tmysql.ErrWrongTableName)
	// // insert value
	// s.tk.MustExec("create table test_error_code_null(c1 char(100) not null);")
	// sql = "insert into test_error_code_null (c1) values(null);"
	// s.testErrorCode(c, sql, tmysql.ErrBadNull)
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

	res = makeSql(tk, `create table t1(id int primary key,c1 int);
		create table t2(id int primary key,c1 int,c2 int);
		update t1 inner join t2 on t1.id=t2.id2  set t1.c1=t2.c1 where c11=1;`)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't2.id2' not existed.\nColumn 'c11' not existed.")

	res = makeSql(tk, `create table t1(id int primary key,c1 int);
		create table t2(id int primary key,c1 int,c2 int);
		update t1,t2 t3 set t1.c1=t2.c3 where t1.id=t3.id;`)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't2.c3' not existed.")

	res = makeSql(tk, `create table t1(id int primary key,c1 int);
		create table t2(id int primary key,c1 int,c2 int);
		update t1,t2 t3 set t1.c1=t2.c3 where t1.id=t3.id;`)
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't2.c3' not existed.")

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

func (s *testSessionIncSuite) TestCreateDataBase(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)

	res := makeSql(tk, "drop database if exists test1111111111111111111;create database test1111111111111111111;")
	c.Assert(int(tk.Se.AffectedRows()), Equals, 3)

	// 不存在
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// 存在
	res = makeSql(tk, "drop database if exists test1111111111111111111;create database test1111111111111111111;create database test1111111111111111111;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "数据库'test1111111111111111111'已存在.")

	// 不存在
	res = makeSql(tk, "drop database if exists test1111111111111111111;drop database test1111111111111111111;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "命令禁止! 无法删除数据库'test1111111111111111111'.")

	// if not exists 创建
	res = makeSql(tk, "create database if not exists test1111111111111111111;create database if not exists test1111111111111111111;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// if not exists 删除
	res = makeSql(tk, "drop database if exists test1111111111111111111;drop database if exists test1111111111111111111;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "命令禁止! 无法删除数据库'test1111111111111111111'.")

	// create database
	sql := "create database aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	sql = "create database mysql"
	s.testErrorCode(c, sql,
		session.NewErrf("数据库'%s'已存在.", "mysql"))

	// 字符集
	config.GetGlobalConfig().Inc.EnableSetCharset = false
	config.GetGlobalConfig().Inc.SupportCharset = ""
	sql = "create database test1 character set utf8;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_SET_CHARSET, "utf8"))

	config.GetGlobalConfig().Inc.SupportCharset = "utf8mb4"
	sql = "create database test1 character set utf8;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_SET_CHARSET, "utf8"),
		session.NewErr(session.ER_NAMES_MUST_UTF8, "utf8mb4"))

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	sql = "create database test1 character set utf8;"
	s.testErrorCode(c, sql)

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	sql = "create database test1 character set laitn1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_NAMES_MUST_UTF8, "utf8,utf8mb4"))
}

func (s *testSessionIncSuite) TestRenameTable(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)

	// 不存在
	res := makeSql(tk, "drop table if exists t1;create table t1(id int primary key);alter table t1 rename t2;")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	res = makeSql(tk, "drop table if exists t1;create table t1(id int primary key);rename table t1 to t2;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// 存在
	res = makeSql(tk, "drop table if exists t1;create table t1(id int primary key);alter table t1 rename t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 't1' already exists.")

	res = makeSql(tk, "drop table if exists t1;create table t1(id int primary key);rename table t1 to t1;")
	fmt.Println(tk.Se.AffectedRows())
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 't1' already exists.")
}

func (s *testSessionIncSuite) TestCreateView(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)

	res := makeSql(tk, "create table t1(id int primary key);create view v1 as select * from t1;")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "命令禁止! 无法创建视图'v1'.")
}

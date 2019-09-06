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
	"path"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/domain"
	"github.com/hanchuanchuan/goInception/kv"
	"github.com/hanchuanchuan/goInception/session"
	"github.com/hanchuanchuan/goInception/store/mockstore"
	"github.com/hanchuanchuan/goInception/store/mockstore/mocktikv"
	"github.com/hanchuanchuan/goInception/util/testkit"
	"github.com/hanchuanchuan/goInception/util/testleak"
	. "github.com/pingcap/check"
)

var _ = Suite(&testSessionIncExecSuite{})

func TestExec(t *testing.T) {
	TestingT(t)
}

type testSessionIncExecSuite struct {
	cluster   *mocktikv.Cluster
	mvccStore mocktikv.MVCCStore
	store     kv.Storage
	dom       *domain.Domain
	tk        *testkit.TestKit

	sqlMode string
	// 时间戳类型是否需要明确指定默认值
	explicitDefaultsForTimestamp bool

	realRowCount bool
}

func (s *testSessionIncExecSuite) SetUpSuite(c *C) {

	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

	s.realRowCount = true
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

	cfg := config.GetGlobalConfig()
	_, localFile, _, _ := runtime.Caller(0)
	localFile = path.Dir(localFile)
	configFile := path.Join(localFile[0:len(localFile)-len("session")], "config/config.toml.example")
	c.Assert(cfg.Load(configFile), IsNil)

	// 启用自定义审核级别
	// config.GetGlobalConfig().Inc.EnableLevel = true

	inc := &config.GetGlobalConfig().Inc

	inc.BackupHost = "127.0.0.1"
	inc.BackupPort = 3306
	inc.BackupUser = "inception"
	inc.BackupPassword = "inception"

	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Inc.EnableFingerprint = true
	config.GetGlobalConfig().Inc.SqlSafeUpdates = 0

	inc.EnableDropTable = true

	config.GetGlobalConfig().Inc.Lang = "en-US"
	session.SetLanguage("en-US")

	fmt.Println("ExplicitDefaultsForTimestamp: ", s.getExplicitDefaultsForTimestamp(c))
	fmt.Println("SQLMode: ", s.getSQLMode(c))
}

func (s *testSessionIncExecSuite) TearDownSuite(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	} else {
		s.dom.Close()
		s.store.Close()
		testleak.AfterTest(c)()
	}
}

func (s *testSessionIncExecSuite) TearDownTest(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}
}

func (s *testSessionIncExecSuite) makeExecSQL(tk *testkit.TestKit, sql string) *testkit.Result {

	session.CheckAuditSetting(config.GetGlobalConfig())

	a := `/*--user=test;--password=test;--host=127.0.0.1;--execute=1;--backup=0;--port=3306;--enable-ignore-warnings;real_row_count=%v;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	return tk.MustQueryInc(fmt.Sprintf(a, s.realRowCount, sql))
}

func (s *testSessionIncExecSuite) testErrorCode(c *C, sql string, errors ...*session.SQLError) {
	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	session.CheckAuditSetting(config.GetGlobalConfig())

	res := s.makeExecSQL(s.tk, sql)
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
	// 无错误时需要校验结果是否标记为已执行
	if errCode == 0 {
		if !strings.Contains(row[3].(string), "Execute Successfully") {
			fmt.Println(res.Rows())
		}
		c.Assert(strings.Contains(row[3].(string), "Execute Successfully"), Equals, true)
		// c.Assert(row[4].(string), IsNil)
	}
}

func (s *testSessionIncExecSuite) TestBegin(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

	res := s.tk.MustQueryInc("drop table if exists t1;create table t1(id int);")

	c.Assert(len(res.Rows()), Equals, 1, Commentf("%v", res.Rows()))

	for _, row := range res.Rows() {
		c.Assert(row[2], Equals, "2")
		c.Assert(row[4], Equals, "Must start as begin statement.")
	}
}

func (s *testSessionIncExecSuite) TestNoSourceInfo(c *C) {
	res := s.tk.MustQueryInc("inception_magic_start;\ncreate table t1(id int);")

	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 1)

	for _, row := range res.Rows() {
		c.Assert(row[2], Equals, "2")
		c.Assert(row[4], Equals, "Invalid source infomation.")
	}
}

func (s *testSessionIncExecSuite) TestWrongDBName(c *C) {
	res := s.tk.MustQueryInc(`/*--user=test;--password=test;--host=127.0.0.1;--check=1;--backup=0;--port=3306;--enable-ignore-warnings;*/
inception_magic_start;create table t1(id int);inception_magic_commit;`)

	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 1)

	for _, row := range res.Rows() {
		c.Assert(row[2], Equals, "2")
		c.Assert(row[4], Equals, "Incorrect database name ''.")
	}
}

func (s *testSessionIncExecSuite) TestEnd(c *C) {
	res := s.tk.MustQueryInc(`/*--user=test;--password=test;--host=127.0.0.1;--check=1;--backup=0;--port=3306;--enable-ignore-warnings;*/
inception_magic_start;use test_inc;create table t1(id int);`)

	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 3)

	row := res.Rows()[2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Must end with commit.")
}

func (s *testSessionIncExecSuite) TestCreateTable(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	sql := ""

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	sql = "drop table if exists nullkeytest1;create table nullkeytest1(c1 int, c2 int, c3 int, primary key(c1), key ix_1(c2));"
	s.testErrorCode(c, sql)

	// 表存在
	res := s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);create table t1(id int);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 't1' already exists.")

	// 重复列
	sql = "create table test_error_code1 (c1 int, c2 int, c2 int)"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_DUP_FIELDNAME, "c2"))

	// 主键
	config.GetGlobalConfig().Inc.CheckPrimaryKey = true
	res = s.makeExecSQL(s.tk, "drop table t1;create table t1(id int);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set a primary key for table 't1'.")
	config.GetGlobalConfig().Inc.CheckPrimaryKey = false

	// 数据类型 警告
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 bit);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c1'.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 enum('red', 'blue', 'black'));")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c1'.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 set('red', 'blue', 'black'));")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c1'.")

	// char列建议
	config.GetGlobalConfig().Inc.MaxCharLength = 100
	res = s.makeExecSQL(s.tk, `drop table if exists t1;create table t1(id int,c1 char(200));`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set column 'c1' to VARCHAR type.")

	// 字符集
	sql = `drop table if exists t1;create table t1(id int,c1 varchar(20) character set utf8,
        c2 varchar(20) COLLATE utf8_bin);`
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c1"),
		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c2"))

	config.GetGlobalConfig().Inc.EnableSetCharset = false
	sql = `drop table if exists t1;create table t1(id int,c1 varchar(20)) character set utf8;`
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TABLE_CHARSET_MUST_NULL, "t1"))

	sql = `drop table if exists t1;create table t1(id int,c1 varchar(20)) COLLATE utf8_bin;`
	s.testErrorCode(c, sql,
		session.NewErr(session.ErrTableCollationNotSupport, "t1"))

	// 关键字
	config.GetGlobalConfig().Inc.EnableIdentiferKeyword = false
	config.GetGlobalConfig().Inc.CheckIdentifier = true

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int, TABLES varchar(20),`c1$` varchar(20),c1234567890123456789012345678901234567890123456789012345678901234567890 varchar(20));")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Identifier 'TABLES' is keyword in MySQL.\nIdentifier 'c1$' is invalid, valid options: [a-z|A-Z|0-9|_].\nIdentifier name 'c1234567890123456789012345678901234567890123456789012345678901234567890' is too long.")

	// 列注释
	config.GetGlobalConfig().Inc.CheckColumnComment = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(20));")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Column 'c1' in table 't1' have no comments.")

	config.GetGlobalConfig().Inc.CheckColumnComment = false

	// 表注释
	config.GetGlobalConfig().Inc.CheckTableComment = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(20));")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set comments for table 't1'.")

	config.GetGlobalConfig().Inc.CheckTableComment = false
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(20));")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// 无效默认值
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int default '');")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Invalid default value for column 'c1'.")

	// blob/text字段
	config.GetGlobalConfig().Inc.EnableBlobType = false
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 blob, c2 text);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.\nType blob/text is used in column 'c2'.")

	config.GetGlobalConfig().Inc.EnableBlobType = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 blob not null);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "TEXT/BLOB Column 'c1' in table 't1' can't  been not null.")

	// 检查默认值
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(10));")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set Default value for column 'c1' in table 't1'")
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = false

	// 支持innodb引擎
	config.GetGlobalConfig().Inc.EnableSetEngine = true
	config.GetGlobalConfig().Inc.SupportEngine = "innodb"
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(10))engine = innodb;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(10))engine = myisam;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Set engine to one of 'innodb'")

	// 禁止设置存储引擎
	config.GetGlobalConfig().Inc.EnableSetEngine = false
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(10))engine = innodb;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Cannot set engine 't1'")

	// 允许设置存储引擎
	config.GetGlobalConfig().Inc.EnableSetEngine = true
	config.GetGlobalConfig().Inc.SupportEngine = "innodb"
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(10))engine = innodb;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// 时间戳 timestamp默认值
	sql = "drop table if exists t1;create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 timestamp default CURRENT_TIMESTAMP);"
	s.testErrorCode(c, sql,
		session.NewErrf("Incorrect table definition; there can be only one TIMESTAMP column with CURRENT_TIMESTAMP in DEFAULT or ON UPDATE clause"))

	if s.getDBVersion(c) >= 50600 {
		sql = "drop table if exists t1;create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 timestamp ON UPDATE CURRENT_TIMESTAMP);"
		if s.getExplicitDefaultsForTimestamp(c) || !(strings.Contains(s.getSQLMode(c), "TRADITIONAL") ||
			(strings.Contains(s.getSQLMode(c), "STRICT_") && strings.Contains(s.getSQLMode(c), "NO_ZERO_DATE"))) {
			s.testErrorCode(c, sql)
		} else {
			s.testErrorCode(c, sql, session.NewErr(session.ER_INVALID_DEFAULT, "t2"))
		}
	}

	sql = "drop table if exists t1;create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 date default CURRENT_TIMESTAMP);"
	s.testErrorCode(c, sql,
		session.NewErrf("Invalid default value for column '%s'.", "t2"))

	sql = "drop table if exists t1;create table test_error_code1 (c1 int, aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa int)"
	s.testErrorCode(c, sql, session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	sql = "drop table if exists t1;create table aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa(a int)"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	sql = "drop table if exists t1;create table test_error_code1 (c1 int, c2 int, key aa (c1, c2), key aa (c1))"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_DUP_INDEX, "aa", "test_inc", "test_error_code1"),
		session.NewErr(session.ER_DUP_KEYNAME, "aa"))

	sql = "drop table if exists t1;create table test_error_code1 (c1 int, c2 int, c3 int, key(c_not_exist))"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_NAME_FOR_INDEX, "NULL", "test_error_code1"),
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "test_error_code1.c_not_exist"))

	sql = "drop table if exists t1;create table test_error_code1 (c1 int, c2 int, c3 int, primary key(c_not_exist))"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "test_error_code1.c_not_exist"))

	sql = "drop table if exists t1;create table test_error_code1 (c1 int not null default '')"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_DEFAULT, "c1"))

	sql = "drop table if exists t1;CREATE TABLE `t` (`a` double DEFAULT 1.0 DEFAULT 2.0 DEFAULT now());"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_DEFAULT, "a"))

	sql = "drop table if exists t1;CREATE TABLE `t` (`a` double DEFAULT now());"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_DEFAULT, "a"))

	// 字符集
	config.GetGlobalConfig().Inc.EnableSetCharset = false
	config.GetGlobalConfig().Inc.SupportCharset = ""
	sql = "drop table if exists t1;create table t1(a int) character set utf8;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TABLE_CHARSET_MUST_NULL, "t1"))

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8mb4"
	sql = "drop table if exists t1;create table t1(a int) character set utf8;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ErrCharsetNotSupport, "utf8mb4"))

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	sql = "drop table if exists t1;create table t1(a int) character set utf8;"
	s.testErrorCode(c, sql)

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	sql = "drop table if exists t1;create table t1(a int) character set laitn1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ErrCharsetNotSupport, "utf8,utf8mb4"))

	// 外键
	sql = "drop table if exists t1;create table test_error_code (a int not null ,b int not null,c int not null, d int not null, foreign key (b, c) references product(id));"
	s.testErrorCode(c, sql,
		// session.NewErr(session.ER_WRONG_NAME_FOR_INDEX, "NULL", "test_error_code"),
		session.NewErr(session.ER_FOREIGN_KEY, "test_error_code"))

	sql = "drop table if exists t1;create table test_error_code (a int not null ,b int not null,c int not null, d int not null, foreign key fk_1(b, c) references product(id));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_FOREIGN_KEY, "test_error_code"))

	sql = "drop table if exists test_error_code_2;create table test_error_code_2;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_MUST_AT_LEAST_ONE_COLUMN))

	sql = "drop table if exists test_error_code_2;create table test_error_code_2 (unique(c1));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_MUST_AT_LEAST_ONE_COLUMN))

	sql = "drop table if exists t1;create table test_error_code_2(c1 int, c2 int, c3 int, primary key(c1), primary key(c2));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_MULTIPLE_PRI_KEY))

	fmt.Println("数据库版本: ", s.getDBVersion(c))

	indexMaxLength := 767
	if s.getDBVersion(c) >= 50700 {
		indexMaxLength = 3072
	}

	config.GetGlobalConfig().Inc.EnableBlobType = false
	sql = "drop table if exists t1;create table t1(pt text ,primary key (pt));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_USE_TEXT_OR_BLOB, "pt"),
		session.NewErr(session.ER_TOO_LONG_KEY, "PRIMARY", indexMaxLength))

	config.GetGlobalConfig().Inc.EnableBlobType = true
	// 索引长度
	sql = "drop table if exists t1;create table t1(a text, unique (a(3073)));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_NAME_FOR_INDEX, "NULL", "t1"),
		session.NewErr(session.ER_TOO_LONG_KEY, "", indexMaxLength))

	sql = "drop table if exists t1;create table t1(c1 int,c2 text, unique uq_1(c1,c2(3069)));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TOO_LONG_KEY, "uq_1", indexMaxLength))

	config.GetGlobalConfig().Inc.EnableBlobType = true
	// sql = "drop table if exists t1;create table t1(c1 int,c2 text, unique uq_1(c1,c2(3068)));"
	// if indexMaxLength == 3072 {
	// 	s.testErrorCode(c, sql)
	// } else {
	// 	s.testErrorCode(c, sql,
	// 		session.NewErr(session.ER_TOO_LONG_KEY, "", indexMaxLength))
	// }

	config.GetGlobalConfig().Inc.EnableBlobType = false

	config.GetGlobalConfig().Inc.EnableBlobType = true
	sql = "drop table if exists t1;create table t1(pt blob ,primary key (pt));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_BLOB_USED_AS_KEY, "pt"))

	sql = "drop table if exists t1;create table t1(`id` int, key `primary`(`id`));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_NAME_FOR_INDEX, "primary", "t1"))

	sql = "drop table if exists t1;create table t1(c1.c2 varchar(10));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_TABLE_NAME, "c1"))

	sql = "drop table if exists t1;create table t1(c1 int default null primary key , age int);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_PRIMARY_CANT_HAVE_NULL))

	sql = "drop table if exists t1;create table t1(id int null primary key , age int);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_PRIMARY_CANT_HAVE_NULL))

	sql = "drop table if exists t1;create table t1(id int default null, age int, primary key(id));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_PRIMARY_CANT_HAVE_NULL))

	sql = "drop table if exists t1;create table t1(id int null, age int, primary key(id));"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_PRIMARY_CANT_HAVE_NULL))

	if s.getDBVersion(c) >= 50600 {
		sql = `drop table if exists t1;create table t1(
id int auto_increment comment 'test',
crtTime datetime not null DEFAULT CURRENT_TIMESTAMP comment 'test',
uptTime datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'test',
primary key(id)) comment 'test';`
		s.testErrorCode(c, sql)
	}

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 int);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// 测试表名大小写
	sql = "insert into T1 values(1);"
	s.testErrorCode(c, sql)

}

func (s *testSessionIncExecSuite) TestDropTable(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableDropTable = false
	sql := ""
	sql = "drop table if exists t1;create table t1(id int);drop table t1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_DROP_TABLE, "t1"))

	config.GetGlobalConfig().Inc.EnableDropTable = true

	res := s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);drop table t1;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
}

func (s *testSessionIncExecSuite) TestAlterTableAddColumn(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableDropTable = true

	sql := ""
	sql = "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int;"
	s.testErrorCode(c, sql)

	res := s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int;alter table t1 add column c1 int;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c1' have existed.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int first;alter table t1 add column c2 int after c1;")
	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2")
	}

	// after 不存在的列
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c2 int after c1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c1' not existed.")

	// 数据类型 警告
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c2 bit;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c2 enum('red', 'blue', 'black');")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c2 set('red', 'blue', 'black');")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

	// char列建议
	config.GetGlobalConfig().Inc.MaxCharLength = 100
	res = s.makeExecSQL(s.tk, `drop table if exists t1;create table t1(id int);
        alter table t1 add column c1 char(200);
        alter table t1 add column c2 varchar(200);`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set column 'c1' to VARCHAR type.")

	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// 字符集
	sql = `drop table if exists t1;create table t1(id int);
        alter table t1 add column c1 varchar(20) character set utf8;`
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c1"))

	sql = `drop table if exists t1;create table t1(id int);
        alter table t1 add column c2 varchar(20) COLLATE utf8_bin;`
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c2"))

	// 关键字
	config.GetGlobalConfig().Inc.EnableIdentiferKeyword = false
	config.GetGlobalConfig().Inc.CheckIdentifier = true

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column TABLES varchar(20);alter table t1 add column `c1$` varchar(20);alter table t1 add column c1234567890123456789012345678901234567890123456789012345678901234567890 varchar(20);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-3]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Identifier 'TABLES' is keyword in MySQL.")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-2]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Identifier 'c1$' is invalid, valid options: [a-z|A-Z|0-9|_].")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Identifier name 'c1234567890123456789012345678901234567890123456789012345678901234567890' is too long.")

	// 列注释
	config.GetGlobalConfig().Inc.CheckColumnComment = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 varchar(20);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Column 'c1' in table 't1' have no comments.")

	config.GetGlobalConfig().Inc.CheckColumnComment = false

	// 无效默认值
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int default '';")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Invalid default value for column 'c1'.")

	// blob/text字段
	config.GetGlobalConfig().Inc.EnableBlobType = false
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 blob;alter table t1 add column c2 text;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c2'.")

	config.GetGlobalConfig().Inc.EnableBlobType = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 blob not null;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "TEXT/BLOB Column 'c1' in table 't1' can't  been not null.")

	// 检查默认值
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 varchar(10);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set Default value for column 'c1' in table 't1'")
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = false

	sql = "drop table if exists t1;create table t1 (id int primary key , age int);"
	s.testErrorCode(c, sql)

	// // add column
	sql = "drop table if exists t1;create table t1 (c1 int primary key);alter table t1 add column c1 int"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_EXISTED, "t1.c1"))

	sql = "drop table if exists t1;create table t1 (c1 int primary key);alter table t1 add column aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa int"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	sql = "drop table if exists t1;alter table t1 comment 'test comment'"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TABLE_NOT_EXISTED_ERROR, "test_inc.t1"))

	sql = "drop table if exists t1;create table t1 (c1 int primary key);alter table t1 add column `a ` int ;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_IDENT, "a "),
		session.NewErr(session.ER_WRONG_COLUMN_NAME, "a "))

	sql = "drop table if exists t1;create table t1 (c1 int primary key);alter table t1 add column c2 int on update current_timestamp;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_ON_UPDATE, "c2"))

	sql = "drop table if exists t1;create table t1(c2 int on update current_timestamp);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_INVALID_ON_UPDATE, "c2"))

	sql = "drop table if exists t1;create table t1 (id int primary key);alter table t1 add column (c1 int,c2 varchar(20));"
	s.testErrorCode(c, sql)

	if s.getDBVersion(c) >= 50600 {
		// 指定特殊选项
		sql = "drop table if exists t1;create table t1 (id int primary key);alter table t1 add column c1 int,ALGORITHM=INPLACE, LOCK=NONE;"
		s.testErrorCode(c, sql)
	}

	// 特殊字符
	config.GetGlobalConfig().Inc.CheckIdentifier = false
	sql = "drop table if exists `t3!@#$^&*()`;create table `t3!@#$^&*()`(id int primary key);alter table `t3!@#$^&*()` add column `c3!@#$^&*()2` int comment '123';"
	s.testErrorCode(c, sql)

	// pt-osc
	config.GetGlobalConfig().Osc.OscOn = true
	s.testErrorCode(c, sql)

	// gh-ost
	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = true
	s.testErrorCode(c, sql)

}

func (s *testSessionIncExecSuite) TestAlterTableAlterColumn(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	res := s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '';")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Invalid default value for column 'id'.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '1';")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id drop default ;alter table t1 alter column id set default '1';")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-2]
	c.Assert(row[2], Equals, "0")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
}

func (s *testSessionIncExecSuite) TestAlterTableModifyColumn(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	sql := ""

	res := s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column c1 int first;")
	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2")
	}

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column id int after c1;")
	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2")
	}

	// after 不存在的列
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);alter table t1 modify column c1 int after id;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c1' not existed.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column c1 int after id1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.id1' not existed.")

	// 数据类型 警告
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id bit);alter table t1 modify column id bit;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id enum('red', 'blue'));alter table t1 modify column id enum('red', 'blue', 'black');")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id set('red'));alter table t1 modify column id set('red', 'blue', 'black');")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

	// char列建议
	config.GetGlobalConfig().Inc.MaxCharLength = 100
	res = s.makeExecSQL(s.tk, `drop table if exists t1;create table t1(id int,c1 char(10));
        alter table t1 modify column c1 char(200);`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set column 'c1' to VARCHAR type.")

	// 字符集
	sql = `drop table if exists t1;create table t1(id int,c1 varchar(20));
        alter table t1 modify column c1 varchar(20) character set utf8;`
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c1"))

	sql = `drop table if exists t1;create table t1(id int,c1 varchar(20));
        alter table t1 modify column c1 varchar(20) COLLATE utf8_bin;`
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c1"))

	// 列注释
	config.GetGlobalConfig().Inc.CheckColumnComment = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 varchar(10));alter table t1 modify column c1 varchar(20);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Column 'c1' in table 't1' have no comments.")

	config.GetGlobalConfig().Inc.CheckColumnComment = false

	// 无效默认值
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column c1 int default '';")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Invalid default value for column 'c1'.")

	// blob/text字段
	config.GetGlobalConfig().Inc.EnableBlobType = false
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 varchar(10));alter table t1 modify column c1 blob;alter table t1 modify column c1 text;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-2]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

	config.GetGlobalConfig().Inc.EnableBlobType = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 blob);alter table t1 modify column c1 blob not null;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "TEXT/BLOB Column 'c1' in table 't1' can't  been not null.")

	// 检查默认值
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 varchar(5));alter table t1 modify column c1 varchar(10);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Set Default value for column 'c1' in table 't1'")
	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = false

	// 变更类型
	sql = "drop table if exists t1;create table t1(c1 int,c1 int);alter table t1 modify column c1 varchar(10);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CHANGE_COLUMN_TYPE, "t1.c1", "int(11)", "varchar(10)"))

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 char(100));alter table t1 modify column c1 char(20);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(c1 varchar(100));alter table t1 modify column c1 varchar(10);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	if s.getDBVersion(c) >= 50600 {
		sql = "drop table if exists t1;create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 timestamp ON UPDATE CURRENT_TIMESTAMP);"
		if s.getExplicitDefaultsForTimestamp(c) || !(strings.Contains(s.getSQLMode(c), "TRADITIONAL") ||
			(strings.Contains(s.getSQLMode(c), "STRICT_") && strings.Contains(s.getSQLMode(c), "NO_ZERO_DATE"))) {
			s.testErrorCode(c, sql)
		} else {
			s.testErrorCode(c, sql, session.NewErr(session.ER_INVALID_DEFAULT, "t2"))
		}
	}

	// modify column
	sql = "drop table if exists t1;create table t1(id int primary key,c1 int);alter table t1 modify testx.t1.c1 int"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_DB_NAME, "testx"))

	sql = "drop table if exists t1;create table t1(id int primary key,c1 int);alter table t1 modify t.c1 int"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WRONG_TABLE_NAME, "t"))
}

func (s *testSessionIncExecSuite) TestAlterTableDropColumn(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()
	sql := ""

	res := s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c2;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't1.c2' not existed.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// // drop column
	sql = "drop table if exists t1;create table t1(id int null);alter table t1 drop c1"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t1.c1"))

	sql = "drop table if exists t1;create table t1(id int null);alter table t1 drop id;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ErrCantRemoveAllFields))
}

func (s *testSessionIncExecSuite) TestInsert(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false
	config.GetGlobalConfig().IncLevel.ER_WITH_INSERT_FIELD = 0

	sql := ""

	// 表不存在
	res := s.makeExecSQL(s.tk, "insert into t1 values(1,1);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.")

	// 列数不匹配
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id) values(1,1);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column count doesn't match value count at row 1.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id) values(1),(2,1);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column count doesn't match value count at row 2.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int not null);insert into t1(id,c1) select 1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column count doesn't match value count at row 1.")

	// 列重复
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id,id) values(1,1);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'id' specified twice in table 't1'.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id,id) select 1,1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'id' specified twice in table 't1'.")

	// 字段警告
	config.GetGlobalConfig().Inc.CheckInsertField = true
	config.GetGlobalConfig().IncLevel.ER_WITH_INSERT_FIELD = 1
	sql = "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WITH_INSERT_FIELD))

	config.GetGlobalConfig().Inc.CheckInsertField = false
	config.GetGlobalConfig().IncLevel.ER_WITH_INSERT_FIELD = 0

	sql = "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id) values();"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WITH_INSERT_VALUES))

	// 列不允许为空
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int not null);insert into t1(id,c1) values(1,null);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'test_inc.t1.c1' cannot be null in 1 row.")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int not null default 1);insert into t1(id,c1) values(1,null);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'test_inc.t1.c1' cannot be null in 1 row.")

	// insert select 表不存在
	res = s.makeExecSQL(s.tk, "drop table if exists t1;drop table if exists t2;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t2;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t2' doesn't exist.")

	// select where
	config.GetGlobalConfig().Inc.CheckDMLWhere = true
	sql = "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_NO_WHERE_CONDITION))
	config.GetGlobalConfig().Inc.CheckDMLWhere = false

	// limit
	config.GetGlobalConfig().Inc.CheckDMLLimit = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1 limit 1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLLimit = false

	// order by rand()
	// config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1 order by rand();")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Order by rand is not allowed in select statement.")
	// config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

	// 受影响行数
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "2")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "1")

	sql = "drop table if exists t1;create table t1(c1 char(100) not null);insert into t1(c1) values(null);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_BAD_NULL_ERROR, "test_inc.t1.c1", 1))
}

func (s *testSessionIncExecSuite) TestUpdate(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false
	config.GetGlobalConfig().IncLevel.ER_WITH_INSERT_FIELD = 0
	sql := ""

	// 表不存在
	sql = "drop table if exists t1;update t1 set c1 = 1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TABLE_NOT_EXISTED_ERROR, "test_inc.t1"))

	sql = "drop table if exists t1;create table t1(id int);update t1 set c1 = 1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "c1"))

	sql = "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1,c2 = 1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t1.c2"))

	res := s.makeExecSQL(s.tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        update t1 inner join t2 on t1.id=t2.id2  set t1.c1=t2.c1 where c11=1;`)
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't2.id2' not existed.\nColumn 'c11' not existed.")

	res = s.makeExecSQL(s.tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        update t1,t2 t3 set t1.c1=t2.c3 where t1.id=t3.id;`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't2.c3' not existed.")

	res = s.makeExecSQL(s.tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        update t1,t2 t3 set t1.c1=t2.c3 where t1.id=t3.id;`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't2.c3' not existed.")

	// where
	config.GetGlobalConfig().Inc.CheckDMLWhere = true
	sql = "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_NO_WHERE_CONDITION))

	config.GetGlobalConfig().Inc.CheckDMLWhere = false

	// limit
	config.GetGlobalConfig().Inc.CheckDMLLimit = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1 limit 1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLLimit = false

	// order by rand()
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1 order by rand();")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Order by is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

	// 受影响行数
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "0")

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int primary key,c1 int);insert into t1 values(1,1),(2,2);update t1 set c1 = 1 where id = 1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "0", Commentf("%v", res.Rows()))

	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int primary key,c1 int);insert into t1 values(1,1),(2,2);update t1 set c1 = 10 where id = 1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "1", Commentf("%v", res.Rows()))
}

func (s *testSessionIncExecSuite) TestDelete(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
		config.GetGlobalConfig().IncLevel.ER_WITH_INSERT_FIELD = 1
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false
	config.GetGlobalConfig().IncLevel.ER_WITH_INSERT_FIELD = 0
	sql := ""

	sql = "drop table if exists t1"
	s.testErrorCode(c, sql)

	// 表不存在
	res := s.makeExecSQL(s.tk, "delete from t1 where c1 = 1;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.")

	// res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int);delete from t1 where c1 = 1;")
	// row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "2")
	// c.Assert(row[4], Equals, "Column 'c1' not existed.")

	// res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where c1 = 1 and c2 = 1;")
	// row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "2")
	// c.Assert(row[4], Equals, "Column 't1.c2' not existed.")

	// where
	config.GetGlobalConfig().Inc.CheckDMLWhere = true
	sql = "drop table if exists t1;create table t1(id int,c1 int);delete from t1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_NO_WHERE_CONDITION))

	config.GetGlobalConfig().Inc.CheckDMLWhere = false

	// limit
	config.GetGlobalConfig().Inc.CheckDMLLimit = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where id = 1 limit 1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLLimit = false

	// order by rand()
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where id = 1 order by rand();")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "1")
	c.Assert(row[4], Equals, "Order by is not allowed in update/delete statement.")
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

	// 表不存在
	res = s.makeExecSQL(s.tk, `drop table if exists t1;
        delete from t1 where id1 =1;`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.")

	res = s.makeExecSQL(s.tk, `drop table if exists t1;create table t1(id int,c1 int);
        delete from t1 where id1 =1;`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 'id1' not existed.")

	res = s.makeExecSQL(s.tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        delete t2 from t1 inner join t2 on t1.id=t2.id2 where c11=1;`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2")
	c.Assert(row[4], Equals, "Column 't2.id2' not existed.")

	res = s.makeExecSQL(s.tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        delete t2 from t1 inner join t2 on t1.id=t2.id where t1.c1=1;`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")

	// 受影响行数
	res = s.makeExecSQL(s.tk, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where id = 1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "0")
	c.Assert(row[6], Equals, "0")
}

func (s *testSessionIncExecSuite) TestCreateDataBase(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableDropDatabase = true

	sql := ""

	sql = "drop database if exists test1111111111111111111;create database if not exists test1111111111111111111;"
	s.testErrorCode(c, sql)

	// 存在
	sql = "create database test1111111111111111111;create database test1111111111111111111;"
	s.testErrorCode(c, sql,
		session.NewErrf("数据库'test1111111111111111111'已存在."))

	config.GetGlobalConfig().Inc.EnableDropDatabase = false
	// 不存在
	sql = "drop database if exists test1111111111111111111;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_DROP_DATABASE, "test1111111111111111111"))

	sql = "drop database test1111111111111111111;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_DROP_DATABASE, "test1111111111111111111"))

	config.GetGlobalConfig().Inc.EnableDropDatabase = true

	// if not exists 创建
	sql = "create database if not exists test1111111111111111111;create database if not exists test1111111111111111111;"
	s.testErrorCode(c, sql)

	// create database
	sql = "create database aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	sql = "create database mysql"
	s.testErrorCode(c, sql,
		session.NewErrf("数据库'%s'已存在.", "mysql"))

	// 字符集
	config.GetGlobalConfig().Inc.EnableSetCharset = false
	config.GetGlobalConfig().Inc.SupportCharset = ""
	sql = "drop database if exists test123456;create database test123456 character set utf8;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_SET_CHARSET, "utf8"))

	config.GetGlobalConfig().Inc.SupportCharset = "utf8mb4"
	sql = "drop database if exists test123456;create database test123456 character set utf8;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_SET_CHARSET, "utf8"))

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	sql = "drop database if exists test123456;create database test123456 character set utf8;"
	s.testErrorCode(c, sql)

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	sql = "drop database if exists test123456;create database test123456 character set laitn1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ErrCharsetNotSupport, "utf8,utf8mb4"))
}

func (s *testSessionIncExecSuite) TestRenameTable(c *C) {
	sql := ""
	// 不存在
	sql = "drop table if exists t1;drop table if exists t2;create table t1(id int primary key);alter table t1 rename t2;"
	s.testErrorCode(c, sql)

	sql = "drop table if exists t1;drop table if exists t2;create table t1(id int primary key);rename table t1 to t2;"
	s.testErrorCode(c, sql)

	// 存在
	sql = "drop table if exists t1;create table t1(id int primary key);rename table t1 to t1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TABLE_EXISTS_ERROR, "t1"))
}

func (s *testSessionIncExecSuite) TestCreateView(c *C) {
	sql := ""
	sql = "drop table if exists t1;create table t1(id int primary key);create view v1 as select * from t1;"
	s.testErrorCode(c, sql,
		session.NewErrf("命令禁止! 无法创建视图'v1'."))
}

func (s *testSessionIncExecSuite) TestAlterTableAddIndex(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	sql := ""
	// add index
	sql = "drop table if exists t1;create table t1(id int);alter table t1 add index idx (c1)"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t1.c1"))

	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 add index idx (c1);"
	s.testErrorCode(c, sql)

	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 add index idx (c1);alter table t1 add index idx (c1);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_DUP_INDEX, "idx", "test_inc", "t1"))
}

func (s *testSessionIncExecSuite) TestAlterTableDropIndex(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	sql := ""
	// drop index
	sql = "drop table if exists t1;create table t1(id int);alter table t1 drop index idx"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_DROP_FIELD_OR_KEY, "t1.idx"))

	sql = "drop table if exists t1;create table t1(c1 int);alter table t1 add index idx (c1);alter table t1 drop index idx;"
	s.testErrorCode(c, sql)
}

func (s *testSessionIncExecSuite) TestShowVariables(c *C) {
	sql := ""
	sql = "inception show variables;"
	s.tk.MustQueryInc(sql)
	c.Assert(s.tk.Se.AffectedRows(), GreaterEqual, uint64(102))

	sql = "inception get variables;"
	s.tk.MustQueryInc(sql)
	c.Assert(s.tk.Se.AffectedRows(), GreaterEqual, uint64(102))

	sql = "inception show variables like 'backup_password';"
	res := s.tk.MustQueryInc(sql)
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	if row[1].(string) != "" {
		c.Assert(row[1].(string)[:1], Equals, "*")
	}
}

// 无法审核，原因是需要自己做SetSessionManager
// func (s *testSessionIncExecSuite) TestShowProcesslist(c *C) {
// 	sql := ""
// 	sql = "inception show processlist;"
// 	tk.MustQueryInc(sql)
// 	c.Assert(tk.Se.AffectedRows(), GreaterEqual, 1)

// 	sql = "inception get processlist;"
// 	tk.MustQueryInc(sql)
// 	c.Assert(tk.Se.AffectedRows(), GreaterEqual, 1)

// 	sql = "inception show variables like 'backup_password';"
// 	res := s.tk.MustQueryInc(sql)
// 	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
// 	if row[1].(string) != "" {
// 		c.Assert(row[1].(string)[:1], Equals, "*")
// 	}
// }

func (s *testSessionIncExecSuite) TestSetVariables(c *C) {
	sql := ""
	sql = "inception show variables;"
	s.tk.MustQueryInc(sql)
	c.Assert(s.tk.Se.AffectedRows(), GreaterEqual, uint64(102))

	// 不区分session和global.所有会话全都global级别
	s.tk.MustExecInc("inception set global max_keys = 20;")
	s.tk.MustExecInc("inception set session max_keys = 10;")
	result := s.tk.MustQueryInc("inception show variables like 'max_keys';")
	result.Check(testkit.Rows("max_keys 10"))

	s.tk.MustExecInc("inception set ghost_default_retries = 70;")
	result = s.tk.MustQueryInc("inception show variables like 'ghost_default_retries';")
	result.Check(testkit.Rows("ghost_default_retries 70"))

	s.tk.MustExecInc("inception set osc_max_thread_running = 100;")
	result = s.tk.MustQueryInc("inception show variables like 'osc_max_thread_running';")
	result.Check(testkit.Rows("osc_max_thread_running 100"))

	// 无效参数
	res, err := s.tk.ExecInc("inception set osc_max_thread_running1 = 100;")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "无效参数")
	if res != nil {
		c.Assert(res.Close(), IsNil)
	}

	// 无效参数
	res, err = s.tk.ExecInc("inception set osc_max_thread_running = 'abc';")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "[variable:1232]Incorrect argument type to variable 'osc_max_thread_running'")
	if res != nil {
		c.Assert(res.Close(), IsNil)
	}
}

func (s *testSessionIncExecSuite) TestAlterTable(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableDropTable = true
	sql := ""
	// 删除后添加列
	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1;alter table t1 add column c1 varchar(20);"
	s.testErrorCode(c, sql)

	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1,add column c1 varchar(20);"
	s.testErrorCode(c, sql)

	// 删除后添加索引
	sql = "drop table if exists t1;create table t1(id int,c1 int,key ix(c1));alter table t1 drop index ix;alter table t1 add index ix(c1);"
	s.testErrorCode(c, sql)

	sql = "drop table if exists t1;create table t1(id int,c1 int,key ix(c1));alter table t1 drop index ix,add index ix(c1);"
	s.testErrorCode(c, sql)

}

func (s *testSessionIncExecSuite) getDBVersion(c *C) int {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	sql := "show variables like 'version'"

	res := s.makeExecSQL(s.tk, sql)
	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2)

	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	versionStr := row[5].(string)

	versionStr = strings.SplitN(versionStr, "|", 2)[1]
	value := strings.Replace(versionStr, "'", "", -1)
	value = strings.TrimSpace(value)

	// if strings.Contains(strings.ToLower(value), "mariadb") {
	// 	return DBTypeMariaDB
	// }

	versionStr = strings.Split(value, "-")[0]
	versionSeg := strings.Split(versionStr, ".")
	if len(versionSeg) == 3 {
		versionStr = fmt.Sprintf("%s%02s%02s", versionSeg[0], versionSeg[1], versionSeg[2])
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			fmt.Println(err)
		}
		return version
	}

	return 50700
}

func (s *testSessionIncExecSuite) getSQLMode(c *C) string {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

	if s.sqlMode != "" {
		return s.sqlMode
	}

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	sql := "show variables like 'sql_mode'"

	res := s.makeExecSQL(s.tk, sql)
	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2)

	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	versionStr := row[5].(string)

	versionStr = strings.SplitN(versionStr, "|", 2)[1]
	value := strings.Replace(versionStr, "'", "", -1)
	value = strings.TrimSpace(value)

	s.sqlMode = value
	return value
}

func (s *testSessionIncExecSuite) getExplicitDefaultsForTimestamp(c *C) bool {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

	if s.sqlMode != "" {
		return s.explicitDefaultsForTimestamp
	}

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	sql := "show variables where Variable_name='explicit_defaults_for_timestamp';"

	res := s.makeExecSQL(s.tk, sql)
	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2)

	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	versionStr := row[5].(string)

	if strings.Contains(versionStr, "|") {
		versionStr = strings.SplitN(versionStr, "|", 2)[1]
		value := strings.Replace(versionStr, "'", "", -1)
		value = strings.TrimSpace(value)
		if value == "ON" {
			s.explicitDefaultsForTimestamp = true
		}
	}
	return s.explicitDefaultsForTimestamp
}

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

	_ "github.com/go-sql-driver/mysql"
	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/domain"
	"github.com/hanchuanchuan/goInception/kv"
	"github.com/hanchuanchuan/goInception/session"
	"github.com/hanchuanchuan/goInception/store/mockstore"
	"github.com/hanchuanchuan/goInception/store/mockstore/mocktikv"
	"github.com/hanchuanchuan/goInception/util/testkit"
	"github.com/hanchuanchuan/goInception/util/testleak"
	"github.com/jinzhu/gorm"
	. "github.com/pingcap/check"
)

var _ = Suite(&testSessionIncBackupSuite{})

func TestBackup(t *testing.T) {
	TestingT(t)
}

type testSessionIncBackupSuite struct {
	cluster   *mocktikv.Cluster
	mvccStore mocktikv.MVCCStore
	store     kv.Storage
	dom       *domain.Domain
	tk        *testkit.TestKit
	db        *gorm.DB
}

func (s *testSessionIncBackupSuite) SetUpSuite(c *C) {

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

	inc := &config.GetGlobalConfig().Inc

	inc.BackupHost = "127.0.0.1"
	inc.BackupPort = 3306
	inc.BackupUser = "test"
	inc.BackupPassword = "test"

	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = false

	inc.EnableDropTable = true

	config.GetGlobalConfig().Inc.Lang = "en-US"
	session.SetLanguage("en-US")

	if s.db == nil {
		dbName := "127_0_0_1_3306_test_inc"
		addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&maxAllowedPacket=4194304",
			inc.BackupUser, inc.BackupPassword, inc.BackupHost, inc.BackupPort, dbName)
		db, err := gorm.Open("mysql", addr)
		if err != nil {
			fmt.Println(err)
		}
		// 禁用日志记录器，不显示任何日志
		db.LogMode(false)
		s.db = db
		fmt.Println("创建数据库连接")
	}
}

func (s *testSessionIncBackupSuite) TearDownSuite(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	} else {
		s.dom.Close()
		s.store.Close()
		testleak.AfterTest(c)()

		// if s.db != nil {
		// 	fmt.Println("关闭数据库连接~~~")
		// 	s.db.Close()
		// }
	}
}

func (s *testSessionIncBackupSuite) TearDownTest(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}
}

func (s *testSessionIncBackupSuite) makeSQL(tk *testkit.TestKit, sql string) *testkit.Result {
	a := `/*--user=test;--password=test;--host=127.0.0.1;--execute=1;--backup=1;--port=3306;--enable-ignore-warnings;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	return tk.MustQueryInc(fmt.Sprintf(a, sql))
}

func (s *testSessionIncBackupSuite) testErrorCode(c *C, sql string, errors ...*session.SQLError) {
	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	res := s.makeSQL(s.tk, sql)
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
	// 无错误时需要校验结果是否标记为已执行
	if errCode == 0 {
		if !strings.Contains(row[3].(string), "Execute Successfully") {
			fmt.Println(res.Rows())
		}
		c.Assert(strings.Contains(row[3].(string), "Execute Successfully"), Equals, true)
		// c.Assert(row[4].(string), IsNil)
	}
}

func (s *testSessionIncBackupSuite) TestCreateTable(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;")
}

func (s *testSessionIncBackupSuite) TestDropTable(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableDropTable = true
	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;")

	res = s.makeSQL(tk, "drop table t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8;")

	s.makeSQL(tk, "create table t1(id int) default charset utf8mb4;")
	res = s.makeSQL(tk, "drop table t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")

	s.makeSQL(tk, "create table t1(id int not null default 0);")
	res = s.makeSQL(tk, "drop table t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) NOT NULL DEFAULT '0'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8;")

}

// func (s *testSessionIncBackupSuite) TestAlterTableAddColumn(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	config.GetGlobalConfig().Inc.CheckColumnComment = false
// 	config.GetGlobalConfig().Inc.CheckTableComment = false
// 	config.GetGlobalConfig().Inc.EnableDropTable = true

// 	sql := ""
// 	sql = "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int;"
// 	s.testErrorCode(c, sql)

// 	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int;alter table t1 add column c1 int;")
// 	row := res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't1.c1' have existed.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int first;alter table t1 add column c2 int after c1;")
// 	for _, row := range res.Rows() {
// 		c.Assert(row[2], Not(Equals), "2")
// 	}

// 	// after 不存在的列
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c2 int after c1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't1.c1' not existed.")

// 	// 数据类型 警告
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c2 bit;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c2 enum('red', 'blue', 'black');")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c2 set('red', 'blue', 'black');")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Not supported data type on field: 'c2'.")

// 	// char列建议
// 	config.GetGlobalConfig().Inc.MaxCharLength = 100
// 	res = s.makeSQL(tk, `drop table if exists t1;create table t1(id int);
//         alter table t1 add column c1 char(200);
//         alter table t1 add column c2 varchar(200);`)
// 	row = res.Rows()[int(tk.Se.AffectedRows())-2]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Set column 'c1' to VARCHAR type.")

// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")

// 	// 字符集
// 	sql = `drop table if exists t1;create table t1(id int);
//         alter table t1 add column c1 varchar(20) character set utf8;`
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c1"))

// 	sql = `drop table if exists t1;create table t1(id int);
//         alter table t1 add column c2 varchar(20) COLLATE utf8_bin;`
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c2"))

// 	// 关键字
// 	config.GetGlobalConfig().Inc.EnableIdentiferKeyword = false
// 	config.GetGlobalConfig().Inc.CheckIdentifier = true

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column TABLES varchar(20);alter table t1 add column `c1$` varchar(20);alter table t1 add column c1234567890123456789012345678901234567890123456789012345678901234567890 varchar(20);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-3]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Identifier 'TABLES' is keyword in MySQL.")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-2]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Identifier 'c1$' is invalid, valid options: [a-z|A-Z|0-9|_].")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Identifier name 'c1234567890123456789012345678901234567890123456789012345678901234567890' is too long.")

// 	// 列注释
// 	config.GetGlobalConfig().Inc.CheckColumnComment = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 varchar(20);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Column 'c1' in table 't1' have no comments.")

// 	config.GetGlobalConfig().Inc.CheckColumnComment = false

// 	// 无效默认值
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int default '';")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Invalid default value for column 'c1'.")

// 	// blob/text字段
// 	config.GetGlobalConfig().Inc.EnableBlobType = false
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 blob;alter table t1 add column c2 text;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-2]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Type blob/text is used in column 'c2'.")

// 	config.GetGlobalConfig().Inc.EnableBlobType = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 blob not null;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "TEXT/BLOB Column 'c1' in table 't1' can't  been not null.")

// 	// 检查默认值
// 	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 varchar(10);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Set Default value for column 'c1' in table 't1'")
// 	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = false

// 	sql = "drop table if exists t1;create table t1 (id int primary key , age int);"
// 	s.testErrorCode(c, sql)

// 	// // add column
// 	sql = "drop table if exists t1;create table t1 (c1 int primary key);alter table t1 add column c1 int"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_COLUMN_EXISTED, "t1.c1"))

// 	sql = "drop table if exists t1;create table t1 (c1 int primary key);alter table t1 add column aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa int"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

// 	sql = "drop table if exists t1;alter table t1 comment 'test comment'"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_TABLE_NOT_EXISTED_ERROR, "test_inc.t1"))

// 	sql = "drop table if exists t1;create table t1 (c1 int primary key);alter table t1 add column `a ` int ;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_INVALID_IDENT, "a "),
// 		session.NewErr(session.ER_WRONG_COLUMN_NAME, "a "))

// 	sql = "drop table if exists t1;create table t1 (c1 int primary key);alter table t1 add column c2 int on update current_timestamp;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_INVALID_ON_UPDATE, "c2"))

// 	sql = "drop table if exists t1;create table t1(c2 int on update current_timestamp);"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_INVALID_ON_UPDATE, "c2"))
// }

// func (s *testSessionIncBackupSuite) TestAlterTableAlterColumn(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '';")
// 	row := res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Invalid default value for column 'id'.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '1';")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id drop default ;alter table t1 alter column id set default '1';")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-2]
// 	c.Assert(row[2], Equals, "0")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")
// }

// func (s *testSessionIncBackupSuite) TestAlterTableModifyColumn(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	config.GetGlobalConfig().Inc.CheckColumnComment = false
// 	config.GetGlobalConfig().Inc.CheckTableComment = false
// 	sql := ""

// 	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column c1 int first;")
// 	for _, row := range res.Rows() {
// 		c.Assert(row[2], Not(Equals), "2")
// 	}

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column id int after c1;")
// 	for _, row := range res.Rows() {
// 		c.Assert(row[2], Not(Equals), "2")
// 	}

// 	// after 不存在的列
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 modify column c1 int after id;")
// 	row := res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't1.c1' not existed.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column c1 int after id1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't1.id1' not existed.")

// 	// 数据类型 警告
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id bit);alter table t1 modify column id bit;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id enum('red', 'blue'));alter table t1 modify column id enum('red', 'blue', 'black');")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id set('red'));alter table t1 modify column id set('red', 'blue', 'black');")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Not supported data type on field: 'id'.")

// 	// char列建议
// 	config.GetGlobalConfig().Inc.MaxCharLength = 100
// 	res = s.makeSQL(tk, `drop table if exists t1;create table t1(id int,c1 char(10));
//         alter table t1 modify column c1 char(200);`)
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Set column 'c1' to VARCHAR type.")

// 	// 字符集
// 	sql = `drop table if exists t1;create table t1(id int,c1 varchar(20));
//         alter table t1 modify column c1 varchar(20) character set utf8;`
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c1"))

// 	sql = `drop table if exists t1;create table t1(id int,c1 varchar(20));
//         alter table t1 modify column c1 varchar(20) COLLATE utf8_bin;`
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CHARSET_ON_COLUMN, "t1", "c1"))

// 	// 列注释
// 	config.GetGlobalConfig().Inc.CheckColumnComment = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 varchar(10));alter table t1 modify column c1 varchar(20);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Column 'c1' in table 't1' have no comments.")

// 	config.GetGlobalConfig().Inc.CheckColumnComment = false

// 	// 无效默认值
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column c1 int default '';")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Invalid default value for column 'c1'.")

// 	// blob/text字段
// 	config.GetGlobalConfig().Inc.EnableBlobType = false
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 varchar(10));alter table t1 modify column c1 blob;alter table t1 modify column c1 text;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-2]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Type blob/text is used in column 'c1'.")

// 	config.GetGlobalConfig().Inc.EnableBlobType = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 blob);alter table t1 modify column c1 blob not null;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "TEXT/BLOB Column 'c1' in table 't1' can't  been not null.")

// 	// 检查默认值
// 	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 varchar(5));alter table t1 modify column c1 varchar(10);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Set Default value for column 'c1' in table 't1'")
// 	config.GetGlobalConfig().Inc.CheckColumnDefaultValue = false

// 	// 变更类型
// 	sql = "drop table if exists t1;create table t1(c1 int,c1 int);alter table t1 modify column c1 varchar(10);"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CHANGE_COLUMN_TYPE, "t1.c1", "int(11)", "varchar(10)"))

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(c1 char(100));alter table t1 modify column c1 char(20);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(c1 varchar(100));alter table t1 modify column c1 varchar(10);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")

// 	sql = "drop table if exists t1;create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 timestamp ON UPDATE CURRENT_TIMESTAMP);"
// 	s.testErrorCode(c, sql)

// 	// modify column
// 	sql = "drop table if exists t1;create table t1(id int primary key,c1 int);alter table t1 modify testx.t1.c1 int"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_WRONG_DB_NAME, "testx"))

// 	sql = "drop table if exists t1;create table t1(id int primary key,c1 int);alter table t1 modify t.c1 int"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_WRONG_TABLE_NAME, "t"))
// }

// func (s *testSessionIncBackupSuite) TestAlterTableDropColumn(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()
// 	sql := ""

// 	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c2;")
// 	row := res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't1.c2' not existed.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")

// 	// // drop column
// 	sql = "drop table if exists t1;create table t1(id int null);alter table t1 drop c1"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t1.c1"))

// 	sql = "drop table if exists t1;create table t1(id int null);alter table t1 drop id;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ErrCantRemoveAllFields))
// }

// func (s *testSessionIncBackupSuite) TestInsert(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	config.GetGlobalConfig().Inc.CheckInsertField = false

// 	sql := ""

// 	// 表不存在
// 	res := s.makeSQL(tk, "insert into t1 values(1,1);")
// 	row := res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.")

// 	// 列数不匹配
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id) values(1,1);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column count doesn't match value count at row 1.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id) values(1),(2,1);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column count doesn't match value count at row 2.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int not null);insert into t1(id,c1) select 1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column count doesn't match value count at row 1.")

// 	// 列重复
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id,id) values(1,1);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 'id' specified twice in table 't1'.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id,id) select 1,1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 'id' specified twice in table 't1'.")

// 	// 字段警告
// 	config.GetGlobalConfig().Inc.CheckInsertField = true
// 	sql = "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1);"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_WITH_INSERT_FIELD))

// 	config.GetGlobalConfig().Inc.CheckInsertField = false

// 	sql = "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id) values();"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_WITH_INSERT_VALUES))

// 	// 列不允许为空
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int not null);insert into t1(id,c1) values(1,null);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 'test_inc.t1.c1' cannot be null in 1 row.")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int not null default 1);insert into t1(id,c1) values(1,null);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 'test_inc.t1.c1' cannot be null in 1 row.")

// 	// insert select 表不存在
// 	res = s.makeSQL(tk, "drop table if exists t1;drop table if exists t2;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t2;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Table 'test_inc.t2' doesn't exist.")

// 	// select where
// 	config.GetGlobalConfig().Inc.CheckDMLWhere = true
// 	sql = "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_NO_WHERE_CONDITION))
// 	config.GetGlobalConfig().Inc.CheckDMLWhere = false

// 	// limit
// 	config.GetGlobalConfig().Inc.CheckDMLLimit = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1 limit 1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
// 	config.GetGlobalConfig().Inc.CheckDMLLimit = false

// 	// order by rand()
// 	// config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1 order by rand();")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Order by rand is not allowed in select statement.")
// 	// config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

// 	// 受影响行数
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")
// 	c.Assert(row[6], Equals, "2")

// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")
// 	c.Assert(row[6], Equals, "1")

// 	sql = "drop table if exists t1;create table t1(c1 char(100) not null);insert into t1(c1) values(null);"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_BAD_NULL_ERROR, "test_inc.t1.c1", 1))
// }

// func (s *testSessionIncBackupSuite) TestUpdate(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	config.GetGlobalConfig().Inc.CheckInsertField = false
// 	sql := ""

// 	// 表不存在
// 	sql = "drop table if exists t1;update t1 set c1 = 1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_TABLE_NOT_EXISTED_ERROR, "test_inc.t1"))

// 	sql = "drop table if exists t1;create table t1(id int);update t1 set c1 = 1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "c1"))

// 	sql = "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1,c2 = 1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t1.c2"))

// 	res := s.makeSQL(tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
//         create table t2(id int primary key,c1 int,c2 int);
//         update t1 inner join t2 on t1.id=t2.id2  set t1.c1=t2.c1 where c11=1;`)
// 	row := res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't2.id2' not existed.\nColumn 'c11' not existed.")

// 	res = s.makeSQL(tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
//         create table t2(id int primary key,c1 int,c2 int);
//         update t1,t2 t3 set t1.c1=t2.c3 where t1.id=t3.id;`)
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't2.c3' not existed.")

// 	res = s.makeSQL(tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
//         create table t2(id int primary key,c1 int,c2 int);
//         update t1,t2 t3 set t1.c1=t2.c3 where t1.id=t3.id;`)
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't2.c3' not existed.")

// 	// where
// 	config.GetGlobalConfig().Inc.CheckDMLWhere = true
// 	sql = "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_NO_WHERE_CONDITION))

// 	config.GetGlobalConfig().Inc.CheckDMLWhere = false

// 	// limit
// 	config.GetGlobalConfig().Inc.CheckDMLLimit = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1 limit 1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
// 	config.GetGlobalConfig().Inc.CheckDMLLimit = false

// 	// order by rand()
// 	config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1 order by rand();")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Order by is not allowed in update/delete statement.")
// 	config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

// 	// 受影响行数
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")
// 	c.Assert(row[6], Equals, "0")

// 	res = s.makeSQL(tk, "create table t1(id int primary key,c1 int);insert into t1 values(1,1),(2,2);update t1 set c1 = 1 where id = 1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")
// 	c.Assert(row[6], Equals, "1")
// }

// func (s *testSessionIncBackupSuite) TestDelete(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	config.GetGlobalConfig().Inc.CheckInsertField = false
// 	sql := ""

// 	sql = "drop table if exists t1"
// 	s.testErrorCode(c, sql)

// 	// 表不存在
// 	res := s.makeSQL(tk, "delete from t1 where c1 = 1;")
// 	row := res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.")

// 	// res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);delete from t1 where c1 = 1;")
// 	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	// c.Assert(row[2], Equals, "2")
// 	// c.Assert(row[4], Equals, "Column 'c1' not existed.")

// 	// res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where c1 = 1 and c2 = 1;")
// 	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	// c.Assert(row[2], Equals, "2")
// 	// c.Assert(row[4], Equals, "Column 't1.c2' not existed.")

// 	// where
// 	config.GetGlobalConfig().Inc.CheckDMLWhere = true
// 	sql = "drop table if exists t1;create table t1(id int,c1 int);delete from t1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_NO_WHERE_CONDITION))

// 	config.GetGlobalConfig().Inc.CheckDMLWhere = false

// 	// limit
// 	config.GetGlobalConfig().Inc.CheckDMLLimit = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where id = 1 limit 1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Limit is not allowed in update/delete statement.")
// 	config.GetGlobalConfig().Inc.CheckDMLLimit = false

// 	// order by rand()
// 	config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where id = 1 order by rand();")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "1")
// 	c.Assert(row[4], Equals, "Order by is not allowed in update/delete statement.")
// 	config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

// 	// 表不存在
// 	res = s.makeSQL(tk, `drop table if exists t1;
//         delete from t1 where id1 =1;`)
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Table 'test_inc.t1' doesn't exist.")

// 	res = s.makeSQL(tk, `drop table if exists t1;create table t1(id int,c1 int);
//         delete from t1 where id1 =1;`)
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 'id1' not existed.")

// 	res = s.makeSQL(tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
//         create table t2(id int primary key,c1 int,c2 int);
//         delete t2 from t1 inner join t2 on t1.id=t2.id2 where c11=1;`)
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "2")
// 	c.Assert(row[4], Equals, "Column 't2.id2' not existed.")

// 	res = s.makeSQL(tk, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
//         create table t2(id int primary key,c1 int,c2 int);
//         delete t2 from t1 inner join t2 on t1.id=t2.id where t1.c1=1;`)
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")

// 	// 受影响行数
// 	res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where id = 1;")
// 	row = res.Rows()[int(tk.Se.AffectedRows())-1]
// 	c.Assert(row[2], Equals, "0")
// 	c.Assert(row[6], Equals, "0")
// }

// func (s *testSessionIncBackupSuite) TestCreateDataBase(c *C) {
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	config.GetGlobalConfig().Inc.EnableDropDatabase = true

// 	sql := ""

// 	sql = "drop database if exists test1111111111111111111;create database if not exists test1111111111111111111;"
// 	s.testErrorCode(c, sql)

// 	// 存在
// 	sql = "create database test1111111111111111111;create database test1111111111111111111;"
// 	s.testErrorCode(c, sql,
// 		session.NewErrf("数据库'test1111111111111111111'已存在."))

// 	config.GetGlobalConfig().Inc.EnableDropDatabase = false
// 	// 不存在
// 	sql = "drop database if exists test1111111111111111111;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CANT_DROP_DATABASE, "test1111111111111111111"))

// 	sql = "drop database test1111111111111111111;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CANT_DROP_DATABASE, "test1111111111111111111"))

// 	config.GetGlobalConfig().Inc.EnableDropDatabase = true

// 	// if not exists 创建
// 	sql = "create database if not exists test1111111111111111111;create database if not exists test1111111111111111111;"
// 	s.testErrorCode(c, sql)

// 	// create database
// 	sql = "create database aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_TOO_LONG_IDENT, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

// 	sql = "create database mysql"
// 	s.testErrorCode(c, sql,
// 		session.NewErrf("数据库'%s'已存在.", "mysql"))

// 	// 字符集
// 	config.GetGlobalConfig().Inc.EnableSetCharset = false
// 	config.GetGlobalConfig().Inc.SupportCharset = ""
// 	sql = "drop database if exists test123456;create database test123456 character set utf8;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CANT_SET_CHARSET, "utf8"))

// 	config.GetGlobalConfig().Inc.SupportCharset = "utf8mb4"
// 	sql = "drop database if exists test123456;create database test123456 character set utf8;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CANT_SET_CHARSET, "utf8"),
// 		session.NewErr(session.ER_NAMES_MUST_UTF8, "utf8mb4"))

// 	config.GetGlobalConfig().Inc.EnableSetCharset = true
// 	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
// 	sql = "drop database if exists test123456;create database test123456 character set utf8;"
// 	s.testErrorCode(c, sql)

// 	config.GetGlobalConfig().Inc.EnableSetCharset = true
// 	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
// 	sql = "drop database if exists test123456;create database test123456 character set laitn1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_NAMES_MUST_UTF8, "utf8,utf8mb4"))
// }

// func (s *testSessionIncBackupSuite) TestRenameTable(c *C) {
// 	sql := ""
// 	// 不存在
// 	sql = "drop table if exists t1;drop table if exists t2;create table t1(id int primary key);alter table t1 rename t2;"
// 	s.testErrorCode(c, sql)

// 	sql = "drop table if exists t1;drop table if exists t2;create table t1(id int primary key);rename table t1 to t2;"
// 	s.testErrorCode(c, sql)

// 	// 存在
// 	sql = "drop table if exists t1;create table t1(id int primary key);rename table t1 to t1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_TABLE_EXISTS_ERROR, "t1"))
// }

// func (s *testSessionIncBackupSuite) TestCreateView(c *C) {
// 	sql := ""
// 	sql = "drop table if exists t1;create table t1(id int primary key);create view v1 as select * from t1;"
// 	s.testErrorCode(c, sql,
// 		session.NewErrf("命令禁止! 无法创建视图'v1'."))
// }

// func (s *testSessionIncBackupSuite) TestAlterTableAddIndex(c *C) {
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	config.GetGlobalConfig().Inc.CheckColumnComment = false
// 	config.GetGlobalConfig().Inc.CheckTableComment = false
// 	sql := ""
// 	// add index
// 	sql = "drop table if exists t1;create table t1(id int);alter table t1 add index idx (c1)"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t1.c1"))

// 	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 add index idx (c1);"
// 	s.testErrorCode(c, sql)

// 	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 add index idx (c1);alter table t1 add index idx (c1);"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_DUP_INDEX, "idx", "test_inc", "t1"))
// }

// func (s *testSessionIncBackupSuite) TestAlterTableDropIndex(c *C) {
// 	saved := config.GetGlobalConfig().Inc
// 	defer func() {
// 		config.GetGlobalConfig().Inc = saved
// 	}()

// 	config.GetGlobalConfig().Inc.CheckColumnComment = false
// 	config.GetGlobalConfig().Inc.CheckTableComment = false
// 	sql := ""
// 	// drop index
// 	sql = "drop table if exists t1;create table t1(id int);alter table t1 drop index idx"
// 	s.testErrorCode(c, sql,
// 		session.NewErr(session.ER_CANT_DROP_FIELD_OR_KEY, "t1.idx"))

// 	sql = "drop table if exists t1;create table t1(c1 int);alter table t1 add index idx (c1);alter table t1 drop index idx;"
// 	s.testErrorCode(c, sql)
// }

// func (s *testSessionIncBackupSuite) TestShowVariables(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)

// 	sql := ""
// 	sql = "inception show variables;"
// 	tk.MustQueryInc(sql)
// 	c.Assert(tk.Se.AffectedRows(), GreaterEqual, uint64(102))

// 	sql = "inception get variables;"
// 	tk.MustQueryInc(sql)
// 	c.Assert(tk.Se.AffectedRows(), GreaterEqual, uint64(102))

// 	sql = "inception show variables like 'backup_password';"
// 	res := tk.MustQueryInc(sql)
// 	row := res.Rows()[int(tk.Se.AffectedRows())-1]
// 	if row[1].(string) != "" {
// 		c.Assert(row[1].(string)[:1], Equals, "*")
// 	}
// }

// func (s *testSessionIncBackupSuite) TestSetVariables(c *C) {
// 	tk := testkit.NewTestKitWithInit(c, s.store)

// 	sql := ""
// 	sql = "inception show variables;"
// 	tk.MustQueryInc(sql)
// 	c.Assert(tk.Se.AffectedRows(), GreaterEqual, uint64(102))

// 	// 不区分session和global.所有会话全都global级别
// 	tk.MustExecInc("inception set global max_keys = 20;")
// 	tk.MustExecInc("inception set session max_keys = 10;")
// 	result := tk.MustQueryInc("inception show variables like 'max_keys';")
// 	result.Check(testkit.Rows("max_keys 10"))

// 	tk.MustExecInc("inception set ghost_default_retries = 70;")
// 	result = tk.MustQueryInc("inception show variables like 'ghost_default_retries';")
// 	result.Check(testkit.Rows("ghost_default_retries 70"))

// 	tk.MustExecInc("inception set osc_max_thread_running = 100;")
// 	result = tk.MustQueryInc("inception show variables like 'osc_max_thread_running';")
// 	result.Check(testkit.Rows("osc_max_thread_running 100"))

// 	// 无效参数
// 	res, err := tk.ExecInc("inception set osc_max_thread_running1 = 100;")
// 	c.Assert(err, NotNil)
// 	c.Assert(err.Error(), Equals, "无效参数")
// 	if res != nil {
// 		c.Assert(res.Close(), IsNil)
// 	}

// 	// 无效参数
// 	res, err = tk.ExecInc("inception set osc_max_thread_running = 'abc';")
// 	c.Assert(err, NotNil)
// 	c.Assert(err.Error(), Equals, "[variable:1232]Incorrect argument type to variable 'osc_max_thread_running'")
// 	if res != nil {
// 		c.Assert(res.Close(), IsNil)
// 	}
// }

// func (s *testSessionIncBackupSuite) getDBVersion(c *C) int {
// 	if testing.Short() {
// 		c.Skip("skipping test; in TRAVIS mode")
// 	}

// 	if s.tk == nil {
// 		s.tk = testkit.NewTestKitWithInit(c, s.store)
// 	}

// 	sql := "show variables like 'version'"

// 	res := s.makeSQL(s.tk, sql)
// 	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2)

// 	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
// 	versionStr := row[5].(string)

// 	versionStr = strings.SplitN(versionStr, "|", 2)[1]
// 	value := strings.Replace(versionStr, "'", "", -1)
// 	value = strings.TrimSpace(value)

// 	// if strings.Contains(strings.ToLower(value), "mariadb") {
// 	//  return DBTypeMariaDB
// 	// }

// 	versionStr = strings.Split(value, "-")[0]
// 	versionSeg := strings.Split(versionStr, ".")
// 	if len(versionSeg) == 3 {
// 		versionStr = fmt.Sprintf("%s%02s%02s", versionSeg[0], versionSeg[1], versionSeg[2])
// 		version, err := strconv.Atoi(versionStr)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 		return version
// 	}

// 	return 50700
// }

func (s *testSessionIncBackupSuite) query(table, opid string) string {

	inc := config.GetGlobalConfig().Inc

	inc.BackupHost = "127.0.0.1"
	inc.BackupPort = 3306
	inc.BackupUser = "test"
	inc.BackupPassword = "test"

	result := []string{}

	sql := "select rollback_statement from %s where opid_time = ?;"
	sql = fmt.Sprintf(sql, table)

	rows, err := s.db.Raw(sql, opid).Rows()
	if err != nil {
		fmt.Println(err)
		panic(err)
	} else {
		defer rows.Close()
		for rows.Next() {
			str := ""
			rows.Scan(&str)
			result = append(result, trim(str))
		}
	}

	return strings.Join(result, "\n")
	// return result
}

func trim(s string) string {
	if strings.Contains(s, "  ") {
		return trim(strings.Replace(s, "  ", " ", -1))
	}
	return s
}

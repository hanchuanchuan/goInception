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

	config.GetGlobalConfig().Inc.EnableDropTable = true
	config.GetGlobalConfig().Inc.EnableBlobType = true

	inc.EnableDropTable = true

	config.GetGlobalConfig().Inc.Lang = "en-US"
	session.SetLanguage("en-US")

}

func (s *testSessionIncBackupSuite) TearDownSuite(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	} else {
		s.dom.Close()
		s.store.Close()
		testleak.AfterTest(c)()

		if s.db != nil {
			fmt.Println("关闭数据库连接~~~")
			s.db.Close()
		}
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

func (s *testSessionIncBackupSuite) getDBVersion(c *C) int {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	sql := "show variables like 'version'"

	res := s.makeSQL(s.tk, sql)
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

func (s *testSessionIncBackupSuite) TestCreateTable(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))
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

func (s *testSessionIncBackupSuite) TestAlterTableAddColumn(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(tk, "alter table t1 add column c1 int;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;")

	res = s.makeSQL(tk, "alter table t1 add column c2 bit first;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c2`;")
}

func (s *testSessionIncBackupSuite) TestAlterTableAlterColumn(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	// res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '';")
	// row := res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "2")
	// c.Assert(row[4], Equals, "Invalid default value for column 'id'.")

	// res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '1';")
	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")

	// res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id drop default ;alter table t1 alter column id set default '1';")
	// row = res.Rows()[int(tk.Se.AffectedRows())-2]
	// c.Assert(row[2], Equals, "0")
	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")

	s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);")
	res := s.makeSQL(tk, "alter table t1 alter column c1 set default '1';")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP DEFAULT;", Commentf("%v", res.Rows()))

	res = s.makeSQL(tk, "alter table t1 alter column c1 drop default;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` SET DEFAULT '1';")
}

func (s *testSessionIncBackupSuite) TestAlterTableModifyColumn(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(tk, "alter table t1 modify column c1 varchar(10);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	res = s.makeSQL(tk, "alter table t1 modify column c1 varchar(100) not null;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(10);")

	res = s.makeSQL(tk, "alter table t1 modify column c1 int null;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(100) NOT NULL;")

	res = s.makeSQL(tk, "alter table t1 modify column c1 int first;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	res = s.makeSQL(tk, "alter table t1 modify column c1 varchar(200) not null comment '测试列';")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	res = s.makeSQL(tk, "alter table t1 modify column c1 int;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(200) NOT NULL COMMENT '测试列';")

	res = s.makeSQL(tk, "alter table t1 modify column c1 varchar(20) character set utf8;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	res = s.makeSQL(tk, "alter table t1 modify column c1 varchar(20) COLLATE utf8_bin;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(20);")
}

func (s *testSessionIncBackupSuite) TestAlterTableDropColumn(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(tk, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c1` int(11);")

	s.makeSQL(tk, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c1` int(11) not null default 0 comment '测试列';")
	res = s.makeSQL(tk, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c1` int(11) NOT NULL DEFAULT '0' COMMENT '测试列';")
}

func (s *testSessionIncBackupSuite) TestInsert(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false

	res := s.makeSQL(tk, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(tk, "insert into t1 values(1);")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `id`=1;", Commentf("%v", res.Rows()))

	// // 受影响行数
	// res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")
	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")
	// c.Assert(row[6], Equals, "2")

	// res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null;")
	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")
	// c.Assert(row[6], Equals, "1")

	// sql = "drop table if exists t1;create table t1(c1 char(100) not null);insert into t1(c1) values(null);"
	// s.testErrorCode(c, sql,
	// 	session.NewErr(session.ER_BAD_NULL_ERROR, "test_inc.t1.c1", 1))
}

func (s *testSessionIncBackupSuite) TestUpdate(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")

	res := s.makeSQL(tk, "update t1 set c1=10 where id = 1;")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "UPDATE `test_inc`.`t1` SET `id`=1, `c1`=1 WHERE `id`=1 AND `c1`=10;", Commentf("%v", res.Rows()))

	// // 受影响行数
	// res = s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1;")
	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")
	// c.Assert(row[6], Equals, "0")

	// res = s.makeSQL(tk, "create table t1(id int primary key,c1 int);insert into t1 values(1,1),(2,2);update t1 set c1 = 1 where id = 1;")
	// row = res.Rows()[int(tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")
	// c.Assert(row[6], Equals, "1")
}

func (s *testSessionIncBackupSuite) TestDelete(c *C) {
	tk := testkit.NewTestKitWithInit(c, s.store)
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.makeSQL(tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")

	res := s.makeSQL(tk, "delete from t1 where id <= 2;")
	row := res.Rows()[int(tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);",
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(2,2);",
		}, "\n"), Commentf("%v", res.Rows()))

	s.makeSQL(tk, "drop table if exists t1;create table t1(id int primary key,c1 blob);insert into t1 values(1,X'010203');")
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'\x01\x02\x03');", Commentf("%v", res.Rows()))

	if s.getDBVersion(c) >= 50708 {
		s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 json);
	insert into t1 values(1,'{"time":"2015-01-01 13:00:00","result":"fail"}');`)
		res = s.makeSQL(tk, "delete from t1;")
		row = res.Rows()[int(tk.Se.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');", Commentf("%v", res.Rows()))

		s.makeSQL(tk, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');")
		res = s.makeSQL(tk, "delete from t1;")
		row = res.Rows()[int(tk.Se.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');", Commentf("%v", res.Rows()))
	}

	s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 enum('type1','type2','type3','type4'));
	insert into t1 values(1,'type2');`)
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);", Commentf("%v", res.Rows()))

	s.makeSQL(tk, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);")
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);")

	s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 bit);
	insert into t1 values(1,1);`)
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);", Commentf("%v", res.Rows()))

	s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 decimal(10,2));
	insert into t1 values(1,1.11);`)
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11);", Commentf("%v", res.Rows()))

	s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 double);
	insert into t1 values(1,1.11e100);`)
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);", Commentf("%v", res.Rows()))

	s.makeSQL(tk, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);")
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);")

	s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 date);
	insert into t1 values(1,'2019-1-1');`)
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'2019-01-01');", Commentf("%v", res.Rows()))

	s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 timestamp);
	insert into t1(id) values(1);`)
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,NULL);", Commentf("%v", res.Rows()))

	s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 time);
	insert into t1 values(1,'00:01:01');`)
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'00:01:01');", Commentf("%v", res.Rows()))

	s.makeSQL(tk, `drop table if exists t1;create table t1(id int primary key,c1 year);
	insert into t1 values(1,2019);`)
	res = s.makeSQL(tk, "delete from t1;")
	row = res.Rows()[int(tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2019);", Commentf("%v", res.Rows()))

}

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

func (s *testSessionIncBackupSuite) query(table, opid string) string {
	inc := config.GetGlobalConfig().Inc
	if s.db == nil || s.db.DB().Ping() != nil {
		// dbName := "127_0_0_1_3306_test_inc"
		addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local&maxAllowedPacket=4194304",
			inc.BackupUser, inc.BackupPassword, inc.BackupHost, inc.BackupPort)
		db, err := gorm.Open("mysql", addr)
		if err != nil {
			fmt.Println(err)
		}
		// 禁用日志记录器，不显示任何日志
		db.LogMode(false)
		s.db = db
	}

	result := []string{}
	sql := "select rollback_statement from 127_0_0_1_3306_test_inc.%s where opid_time = ?;"
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
}

func trim(s string) string {
	if strings.Contains(s, "  ") {
		return trim(strings.Replace(s, "  ", " ", -1))
	}
	return s
}

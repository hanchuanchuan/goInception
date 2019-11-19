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

	version int
	sqlMode string
	// 时间戳类型是否需要明确指定默认值
	explicitDefaultsForTimestamp bool

	realRowCount bool
}

func (s *testSessionIncBackupSuite) SetUpSuite(c *C) {

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

	inc := &config.GetGlobalConfig().Inc

	inc.BackupHost = "127.0.0.1"
	inc.BackupPort = 3306
	inc.BackupUser = "test"
	inc.BackupPassword = "test"

	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = false
	config.GetGlobalConfig().Inc.EnableFingerprint = true
	config.GetGlobalConfig().Inc.SqlSafeUpdates = 0

	config.GetGlobalConfig().Inc.EnableDropTable = true
	config.GetGlobalConfig().Inc.EnableBlobType = true
	config.GetGlobalConfig().Inc.EnableJsonType = true

	inc.EnableDropTable = true

	config.GetGlobalConfig().Inc.Lang = "en-US"
	session.SetLanguage("en-US")

	fmt.Println("ExplicitDefaultsForTimestamp: ", s.getExplicitDefaultsForTimestamp(c))
	fmt.Println("SQLMode: ", s.getSQLMode(c))
	fmt.Println("version: ", s.getDBVersion(c))
}

func (s *testSessionIncBackupSuite) TearDownSuite(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	} else {
		s.dom.Close()
		s.store.Close()
		testleak.AfterTest(c)()

		if s.db != nil {
			s.db.Close()
		}
	}
}

func (s *testSessionIncBackupSuite) TearDownTest(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}
}

func (s *testSessionIncBackupSuite) makeSQL(c *C, sql string) *testkit.Result {
	a := `/*--user=test;--password=test;--host=127.0.0.1;--execute=1;--backup=1;--port=3306;--enable-ignore-warnings;real_row_count=%v;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	res := s.tk.MustQueryInc(fmt.Sprintf(a, s.realRowCount, sql))

	// 需要成功执行
	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2", Commentf("%v", row))
	}

	return res
}

func (s *testSessionIncBackupSuite) makeExecSQL(sql string) *testkit.Result {
	a := `/*--user=test;--password=test;--host=127.0.0.1;--execute=1;--backup=1;--port=3306;--enable-ignore-warnings;real_row_count=%v;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	return s.tk.MustQueryInc(fmt.Sprintf(a, s.realRowCount, sql))
}

func (s *testSessionIncBackupSuite) getDBVersion(c *C) int {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	if s.version > 0 {
		return s.version
	}
	sql := "show variables like 'version'"

	res := s.makeSQL(c, sql)
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
		s.version = version
		return version
	}

	return 50700
}

func (s *testSessionIncBackupSuite) TestCreateTable(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	res := s.makeSQL(c, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))
}

func (s *testSessionIncBackupSuite) TestDropTable(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableDropTable = true
	res := s.makeSQL(c, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;")

	res = s.makeSQL(c, "drop table t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	// mysql 8.0版本默认字符集改成了utf8mb4
	if s.getDBVersion(c) < 80000 {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8;")
	} else {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;")
	}

	s.makeSQL(c, "create table t1(id int) default charset utf8mb4;")
	res = s.makeSQL(c, "drop table t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	if s.getDBVersion(c) < 80000 {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")
	} else {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;")
	}
	s.makeSQL(c, "create table t1(id int not null default 0);")
	res = s.makeSQL(c, "drop table t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	if s.getDBVersion(c) < 80000 {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) NOT NULL DEFAULT '0'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8;")
	} else {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) NOT NULL DEFAULT '0'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;")
	}
}

func (s *testSessionIncBackupSuite) TestAlterTableAddColumn(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	res := s.makeSQL(c, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, "alter table t1 add column c1 int;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;")

	res = s.makeSQL(c, "alter table t1 add column c2 bit first;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c2`;")

	res = s.makeSQL(c, "alter table t1 add column (c3 int,c4 varchar(20));")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c4`,DROP COLUMN `c3`;")

	// 特殊字符
	config.GetGlobalConfig().Inc.CheckIdentifier = false
	res = s.makeSQL(c, "drop table if exists `t3!@#$^&*()`;create table `t3!@#$^&*()`(id int primary key);alter table `t3!@#$^&*()` add column `c3!@#$^&*()2` int comment '123';")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t3!@#$^&*()", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t3!@#$^&*()` DROP COLUMN `c3!@#$^&*()2`;", Commentf("%v", res.Rows()))

	// pt-osc
	config.GetGlobalConfig().Osc.OscOn = true
	res = s.makeSQL(c, "drop table if exists `t3!@#$^&*()`;create table `t3!@#$^&*()`(id int primary key);alter table `t3!@#$^&*()` add column `c3!@#$^&*()2` int comment '123';")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t3!@#$^&*()", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t3!@#$^&*()` DROP COLUMN `c3!@#$^&*()2`;", Commentf("%v", res.Rows()))

	// gh-ost
	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = true
	res = s.makeSQL(c, "drop table if exists `t3!@#$^&*()`;create table `t3!@#$^&*()`(id int primary key);alter table `t3!@#$^&*()` add column `c3!@#$^&*()2` int comment '123';")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t3!@#$^&*()", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t3!@#$^&*()` DROP COLUMN `c3!@#$^&*()2`;", Commentf("%v", res.Rows()))

}

func (s *testSessionIncBackupSuite) TestAlterTableAlterColumn(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	// res := s.makeSQL(c,tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '';")
	// row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "2")
	// c.Assert(row[4], Equals, "Invalid default value for column 'id'.")

	// res = s.makeSQL(c,tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '1';")
	// row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")

	// res = s.makeSQL(c,tk, "drop table if exists t1;create table t1(id int);alter table t1 alter column id drop default ;alter table t1 alter column id set default '1';")
	// row = res.Rows()[int(s.tk.Se.AffectedRows())-2]
	// c.Assert(row[2], Equals, "0")
	// row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")

	s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);")
	res := s.makeSQL(c, "alter table t1 alter column c1 set default '1';")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP DEFAULT;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, "alter table t1 alter column c1 drop default;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` SET DEFAULT '1';")
}

func (s *testSessionIncBackupSuite) TestAlterTableModifyColumn(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	res := s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, "alter table t1 modify column c1 varchar(10);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	res = s.makeSQL(c, "alter table t1 modify column c1 varchar(100) not null;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(10);")

	res = s.makeSQL(c, "alter table t1 modify column c1 int null;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(100) NOT NULL;")

	res = s.makeSQL(c, "alter table t1 modify column c1 int first;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	res = s.makeSQL(c, "alter table t1 modify column c1 varchar(200) not null comment '测试列';")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	res = s.makeSQL(c, "alter table t1 modify column c1 int;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(200) NOT NULL COMMENT '测试列';")

	res = s.makeSQL(c, "alter table t1 modify column c1 varchar(20) character set utf8;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	res = s.makeSQL(c, "alter table t1 modify column c1 varchar(20) COLLATE utf8_bin;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(20);")
}

func (s *testSessionIncBackupSuite) TestAlterTableDropColumn(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	res := s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c1` int(11);")

	s.makeSQL(c, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c1` int(11) not null default 0 comment '测试列';")
	res = s.makeSQL(c, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c1` int(11) NOT NULL DEFAULT '0' COMMENT '测试列';")
}

func (s *testSessionIncBackupSuite) TestInsert(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false

	res := s.makeSQL(c, "drop table if exists t1;create table t1(id int);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, "insert into t1 values(1);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `id`=1;", Commentf("%v", res.Rows()))

	// 值溢出时的回滚解析
	res = s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,
			c1 tinyint unsigned,
			c2 smallint unsigned,
			c3 mediumint unsigned,
			c4 int unsigned,
			c5 bigint unsigned
		);`)
	res = s.makeSQL(c, "insert into t1 values(1,128,32768,8388608,2147483648,9223372036854775808);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `id`=1;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, `drop table if exists t1;
		create table t1(id int,
			c1 tinyint unsigned,
			c2 smallint unsigned,
			c3 mediumint unsigned,
			c4 int unsigned,
			c5 bigint unsigned,
			c6 decimal unsigned,
			c7 decimal(4,2) unsigned,
			c8 float unsigned,
			c9 double unsigned
		);`)
	res = s.makeSQL(c, `insert into t1 values(1,128,32768,8388608,2147483648,9223372036854775808,
		9999999999,99.99,3.402823466e+38,1.7976931348623157e+308);`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `id`=1 AND `c1`=128 AND `c2`=32768 AND "+
		"`c3`=8388608 AND `c4`=2147483648 AND `c5`=9223372036854775808 AND "+
		"`c6`=9999999999 AND `c7`=99.99 AND `c8`=3.4028235e+38 AND `c9`=1.7976931348623157e+308;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, `update t1 set c1 = 129,
		c2 = 32769,
		c3 = 8388609,
		c4 = 2147483649,
		c5 = 9223372036854775809,
		c6 = 9999999990,
		c7 = 88.88,
		c8 = 3e+38,
		c9 = 1e+308
		where id = 1;`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "UPDATE `test_inc`.`t1` SET `id`=1, `c1`=128, `c2`=32768, "+
		"`c3`=8388608, `c4`=2147483648, `c5`=9223372036854775808, `c6`=9999999999, "+
		"`c7`=99.99, `c8`=3.4028235e+38, `c9`=1.7976931348623157e+308 "+
		"WHERE `id`=1 AND `c1`=129 AND `c2`=32769 AND `c3`=8388609 AND "+
		"`c4`=2147483649 AND `c5`=9223372036854775809 AND `c6`=9999999990 "+
		"AND `c7`=88.88 AND `c8`=3e+38 AND `c9`=1e+308;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, `delete from t1 where id = 1;`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`,`c2`,`c3`,`c4`,`c5`,`c6`,`c7`,`c8`,`c9`)"+
		" VALUES(1,129,32769,8388609,2147483649,9223372036854775809,9999999990,88.88,3e+38,1e+308);", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, `drop table if exists t1;
		create table t1(c1 bigint unsigned);`)
	res = s.makeSQL(c, `insert into t1 values(9223372036854775808);`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `c1`=9223372036854775808;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, `insert into t1 values(18446744073709551615);`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `c1`=18446744073709551615;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, `drop table if exists t1;
create table t1(id int primary key,c1 varchar(100))default character set utf8mb4;
insert into t1(id,c1)values(1,'😁😄🙂👩');`)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `id`=1;", Commentf("%v", res.Rows()))

	// // 受影响行数
	// res = s.makeSQL(c,tk, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")
	// row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")
	// c.Assert(row[6], Equals, "2")

	// res = s.makeSQL(c,tk, "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null;")
	// row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	// c.Assert(row[2], Equals, "0")
	// c.Assert(row[6], Equals, "1")

	// sql = "drop table if exists t1;create table t1(c1 char(100) not null);insert into t1(c1) values(null);"
	// s.testErrorCode(c, sql,
	// 	session.NewErr(session.ER_BAD_NULL_ERROR, "test_inc.t1.c1", 1))
}

func (s *testSessionIncBackupSuite) TestUpdate(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")

	res := s.makeSQL(c, "update t1 set c1=10 where id = 1;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		"UPDATE `test_inc`.`t1` SET `id`=1, `c1`=1 WHERE `id`=1 AND `c1`=10;", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 int);
		insert into t1 values(1,1),(2,2);`)
	res = s.makeSQL(c, "update t1 set id=id+2 where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `id`=1, `c1`=1 WHERE `id`=3;",
			"UPDATE `test_inc`.`t1` SET `id`=2, `c1`=2 WHERE `id`=4;",
		}, "\n"), Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 decimal(18,4),c2 decimal(18,4));
	insert into t1 values(1,123456789012.1234,123456789012.1235);`)
	res = s.makeSQL(c, "update t1 set c1 = 123456789012 where id>0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "UPDATE `test_inc`.`t1` SET `id`=1, `c1`=123456789012.1234, `c2`=123456789012.1235 WHERE `id`=1;", Commentf("%v", res.Rows()))

	config.GetGlobalConfig().Inc.EnableMinimalRollback = true
	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 decimal(18,4),c2 decimal(18,4));
	insert into t1 values(1,123456789012.1234,123456789012.1235);`)
	res = s.makeSQL(c, "update t1 set c1 = 123456789012 where id>0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "UPDATE `test_inc`.`t1` SET `c1`=123456789012.1234 WHERE `id`=1;", Commentf("%v", res.Rows()))

}

func (s *testSessionIncBackupSuite) TestMinimalUpdate(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableMinimalRollback = true

	s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")

	res := s.makeSQL(c, "update t1 set c1=10 where id = 1;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "UPDATE `test_inc`.`t1` SET `c1`=1 WHERE `id`=1 AND `c1`=10;", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 int);
		insert into t1 values(1,1),(2,2);`)

	res = s.makeSQL(c, "update t1 set c1=10 where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `c1`=1 WHERE `id`=1;",
			"UPDATE `test_inc`.`t1` SET `c1`=2 WHERE `id`=2;",
		}, "\n"), Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 int);
		insert into t1 values(1,1),(2,2);`)

	res = s.makeSQL(c, "update t1 set c1=2 where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		"UPDATE `test_inc`.`t1` SET `c1`=1 WHERE `id`=1;", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 tinyint unsigned,c2 varchar(100));
		insert into t1 values(1,127,'t1'),(2,130,'t2');`)

	res = s.makeSQL(c, "update t1 set c1=130,c2='aa' where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `c1`=127, `c2`='t1' WHERE `id`=1;",
			"UPDATE `test_inc`.`t1` SET `c2`='t2' WHERE `id`=2;",
		}, "\n"), Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int,c1 tinyint unsigned,c2 varchar(100));
		insert into t1 values(1,127,'t1'),(2,130,'t2');`)

	res = s.makeSQL(c, "update t1 set c1=130,c2='aa' where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `c1`=127, `c2`='t1' WHERE `id`=1 AND `c1`=130 AND `c2`='aa';",
			"UPDATE `test_inc`.`t1` SET `c2`='t2' WHERE `id`=2 AND `c1`=130 AND `c2`='aa';",
		}, "\n"), Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 int);
		insert into t1 values(1,1),(2,2);`)
	res = s.makeSQL(c, "update t1 set id=id+2 where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `id`=1 WHERE `id`=3;",
			"UPDATE `test_inc`.`t1` SET `id`=2 WHERE `id`=4;",
		}, "\n"), Commentf("%v", res.Rows()))
}
func (s *testSessionIncBackupSuite) TestDelete(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")

	res := s.makeSQL(c, "delete from t1 where id <= 2;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);",
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(2,2);",
		}, "\n"), Commentf("%v", res.Rows()))

	// ------------- 二进制类型: binary,varbinary,blob ----------------
	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 blob);
		insert into t1 values(1,X'010203');`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'\x01\x02\x03');", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 binary(100));
		insert into t1 values(1,X'010203');`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'\x01\x02\x03');", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 varbinary(100));
		insert into t1 values(1,X'010203');`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'\x01\x02\x03');", Commentf("%v", res.Rows()))

	// 二进制在解析时会误存为string,此时会报错
	s.makeExecSQL(`drop table if exists t1;
		create table t1(id int primary key,c1 varbinary(100));
		insert into t1 values(1,X'71E6D5A383BB447C');`)
	res = s.makeExecSQL("delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	c.Assert(row[2], Equals, "2", Commentf("%v", row))

	// 开启二进制自动转换为十六进制字符串
	config.GetGlobalConfig().Inc.HexBlob = true
	s.makeSQL(c, `drop table if exists t1;
		create table t1(id int primary key,c1 varbinary(100));
		insert into t1 values(1,X'71E6D5A383BB447C');`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,X'71e6d5a383bb447c');", Commentf("%v", res.Rows()))

	s.makeSQL(c, `insert into t1 values(1,unhex('71E6D5A383BB447C'));`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,X'71e6d5a383bb447c');", Commentf("%v", res.Rows()))

	// ------------- json类型 ----------------
	if s.getDBVersion(c) >= 50708 {
		s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 json);
	insert into t1 values(1,'{"time":"2015-01-01 13:00:00","result":"fail"}');`)
		res = s.makeSQL(c, "delete from t1;")
		row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');", Commentf("%v", res.Rows()))

		s.makeSQL(c, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');")
		res = s.makeSQL(c, "delete from t1;")
		row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');", Commentf("%v", res.Rows()))
	}

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 enum('type1','type2','type3','type4'));
	insert into t1 values(1,'type2');`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);", Commentf("%v", res.Rows()))

	s.makeSQL(c, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);")
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);")

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 bit);
	insert into t1 values(1,1);`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 decimal(10,2));
	insert into t1 values(1,1.11);`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11);", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 decimal(18,4));
	insert into t1 values(1,123456789012.1234);`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,123456789012.1234);", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 double);
	insert into t1 values(1,1.11e100);`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);", Commentf("%v", res.Rows()))

	s.makeSQL(c, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);")
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);")

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 date);
	insert into t1 values(1,'2019-1-1');`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'2019-01-01');", Commentf("%v", res.Rows()))

	if s.getDBVersion(c) >= 50700 {
		s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 timestamp);
	insert into t1(id) values(1);`)
		res = s.makeSQL(c, "delete from t1;")
		row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		if s.getExplicitDefaultsForTimestamp(c) {
			c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,NULL);", Commentf("%v", res.Rows()))
		} else {
			v := strings.HasPrefix(backup, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'20")
			c.Assert(v, Equals, true, Commentf("%v", res.Rows()))
		}
	}

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 time);
	insert into t1 values(1,'00:01:01');`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'00:01:01');", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;create table t1(id int primary key,c1 year);
	insert into t1 values(1,2019);`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2019);", Commentf("%v", res.Rows()))

	s.makeSQL(c, `drop table if exists t1;
	create table t1(id int primary key,c1 varchar(100))default character set utf8mb4;
	insert into t1(id,c1)values(1,'😁😄🙂👩');`)
	res = s.makeSQL(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'😁😄🙂👩');", Commentf("%v", res.Rows()))

	s.makeSQL(c, `DROP TABLE IF EXISTS t1;
			CREATE TABLE t1 (
			    id     int(11) not null auto_increment,
			    v4_2 decimal(4,2),
			    v5_0 decimal(5,0),
			    v7_3 decimal(7,3),
			    v10_2 decimal(10,2),
			    v10_3 decimal(10,3),
			    v13_2 decimal(13,2),
			    v15_14 decimal(15,14),
			    v20_10 decimal(20,10),
			    v30_5 decimal(30,5),
			    v30_20 decimal(30,20),
			    v30_25 decimal(30,25),
			    prec   int(11),
			    scale  int(11),
			    PRIMARY KEY(id)
			);
 INSERT INTO t1 (v4_2,v5_0,v7_3,v10_2,v10_3,v13_2,v15_14,v20_10,v30_5,v30_20,v30_25,prec,scale) VALUES
(-10.55 ,   -11 ,   -10.550 ,      -10.55 ,     -10.550 ,         -10.55 , -9.99999999999999 ,        -10.5500000000 ,           -10.55000 ,        -10.55000000000000000000 ,   -10.5500000000000000000000000 ,    4 ,     2),
(  0.01 ,     0 ,     0.012 ,        0.01 ,       0.012 ,           0.01 ,  0.01234567890123 ,          0.0123456789 ,             0.01235 ,          0.01234567890123456789 ,     0.0123456789012345678912345 ,   30 ,    25 ),
( 99.99 , 12345 ,  9999.999 ,    12345.00 ,   12345.000 ,       12345.00 ,  9.99999999999999 ,      12345.0000000000 ,         12345.00000 ,      12345.00000000000000000000 , 12345.0000000000000000000000000 ,    5 ,     0 ),
( 99.99 , 12345 ,  9999.999 ,    12345.00 ,   12345.000 ,       12345.00 ,  9.99999999999999 ,      12345.0000000000 ,         12345.00000 ,      12345.00000000000000000000 , 12345.0000000000000000000000000 ,   10 ,     3 ),
( 99.99 ,   123 ,   123.450 ,      123.45 ,     123.450 ,         123.45 ,  9.99999999999999 ,        123.4500000000 ,           123.45000 ,        123.45000000000000000000 ,   123.4500000000000000000000000 ,   10 ,     3 ),
(-99.99 ,  -123 ,  -123.450 ,     -123.45 ,    -123.450 ,        -123.45 , -9.99999999999999 ,       -123.4500000000 ,          -123.45000 ,       -123.45000000000000000000 ,  -123.4500000000000000000000000 ,   20 ,    10 ),
(  0.00 ,     0 ,     0.000 ,        0.00 ,       0.000 ,           0.00 ,  0.00012345000099 ,          0.0001234500 ,             0.00012 ,          0.00012345000098765000 ,     0.0001234500009876500000000 ,   15 ,    14 ),
(  0.00 ,     0 ,     0.000 ,        0.00 ,       0.000 ,           0.00 ,  0.00012345000099 ,          0.0001234500 ,             0.00012 ,          0.00012345000098765000 ,     0.0001234500009876500000000 ,   22 ,    20 ),
(  0.12 ,     0 ,     0.123 ,        0.12 ,       0.123 ,           0.12 ,  0.12345000098765 ,          0.1234500010 ,             0.12345 ,          0.12345000098765000000 ,     0.1234500009876500000000000 ,   30 ,    20 ),
(  0.00 ,     0 ,     0.000 ,        0.00 ,       0.000 ,           0.00 , -0.00000001234500 ,         -0.0000000123 ,             0.00000 ,         -0.00000001234500009877 ,    -0.0000000123450000987650000 ,   30 ,    20 ),
( 99.99 , 99999 ,  9999.999 , 99999999.99 , 9999999.999 , 99999999999.99 ,  9.99999999999999 , 9999999999.9999999999 , 1234500009876.50000 , 9999999999.99999999999999999999 , 99999.9999999999999999999999999 ,   30 ,     5 ),
( 99.99 , 99999 ,  9999.999 , 99999999.99 , 9999999.999 ,   111111111.11 ,  9.99999999999999 ,  111111111.1100000000 ,     111111111.11000 ,  111111111.11000000000000000000 , 99999.9999999999999999999999999 ,   10 ,     2 ),
(  0.01 ,     0 ,     0.010 ,        0.01 ,       0.010 ,           0.01 ,  0.01000000000000 ,          0.0100000000 ,             0.01000 ,          0.01000000000000000000 ,     0.0100000000000000000000000 ,    7 ,     3 ),
( 99.99 ,   123 ,   123.400 ,      123.40 ,     123.400 ,         123.40 ,  9.99999999999999 ,        123.4000000000 ,           123.40000 ,        123.40000000000000000000 ,   123.4000000000000000000000000 ,   10 ,     2 ),
(-99.99 ,  -563 ,  -562.580 ,     -562.58 ,    -562.580 ,        -562.58 , -9.99999999999999 ,       -562.5800000000 ,          -562.58000 ,       -562.58000000000000000000 ,  -562.5800000000000000000000000 ,   13 ,     2 ),
(-99.99 , -3699 , -3699.010 ,    -3699.01 ,   -3699.010 ,       -3699.01 , -9.99999999999999 ,      -3699.0100000000 ,         -3699.01000 ,      -3699.01000000000000000000 , -3699.0100000000000000000000000 ,   13 ,     2 ),
(-99.99 , -1948 , -1948.140 ,    -1948.14 ,   -1948.140 ,       -1948.14 , -9.99999999999999 ,      -1948.1400000000 ,         -1948.14000 ,      -1948.14000000000000000000 , -1948.1400000000000000000000000 ,   13 ,     2 )`)

	res = s.makeSQL(c, "delete from t1 where id>0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(1,-10.55,-11,-10.55,-10.55,-10.55,-10.55,-9.99999999999999,-10.55,-10.55,-10.55,-10.55,4,2);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(2,0.01,0,0.012,0.01,0.012,0.01,0.01234567890123,0.0123456789,0.01235,0.01234567890123456789,0.0123456789012345678912345,30,25);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(3,99.99,12345,9999.999,12345,12345,12345,9.99999999999999,12345,12345,12345,12345,5,0);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(4,99.99,12345,9999.999,12345,12345,12345,9.99999999999999,12345,12345,12345,12345,10,3);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(5,99.99,123,123.45,123.45,123.45,123.45,9.99999999999999,123.45,123.45,123.45,123.45,10,3);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(6,-99.99,-123,-123.45,-123.45,-123.45,-123.45,-9.99999999999999,-123.45,-123.45,-123.45,-123.45,20,10);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(7,0,0,0,0,0,0,0.00012345000099,0.00012345,0.00012,0.00012345000098765,0.00012345000098765,15,14);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(8,0,0,0,0,0,0,0.00012345000099,0.00012345,0.00012,0.00012345000098765,0.00012345000098765,22,20);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(9,0.12,0,0.123,0.12,0.123,0.12,0.12345000098765,0.123450001,0.12345,0.12345000098765,0.12345000098765,30,20);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(10,0,0,0,0,0,0,-0.000000012345,-0.0000000123,0,-0.00000001234500009877,-0.000000012345000098765,30,20);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(11,99.99,99999,9999.999,99999999.99,9999999.999,99999999999.99,9.99999999999999,9999999999.9999999999,1234500009876.5,9999999999.99999999999999999999,99999.9999999999999999999999999,30,5);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(12,99.99,99999,9999.999,99999999.99,9999999.999,111111111.11,9.99999999999999,111111111.11,111111111.11,111111111.11,99999.9999999999999999999999999,10,2);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(13,0.01,0,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,0.01,7,3);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(14,99.99,123,123.4,123.4,123.4,123.4,9.99999999999999,123.4,123.4,123.4,123.4,10,2);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(15,-99.99,-563,-562.58,-562.58,-562.58,-562.58,-9.99999999999999,-562.58,-562.58,-562.58,-562.58,13,2);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(16,-99.99,-3699,-3699.01,-3699.01,-3699.01,-3699.01,-9.99999999999999,-3699.01,-3699.01,-3699.01,-3699.01,13,2);\n"+
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(17,-99.99,-1948,-1948.14,-1948.14,-1948.14,-1948.14,-9.99999999999999,-1948.14,-1948.14,-1948.14,-1948.14,13,2);", Commentf("%v", res.Rows()))

}

func (s *testSessionIncBackupSuite) TestCreateDataBase(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableDropDatabase = true

	s.makeSQL(c, "drop database if exists test123456;")
	res := s.makeSQL(c, "create database test123456;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "", Commentf("%v", res.Rows()))

	s.makeSQL(c, "drop database if exists test123456;")
}

func (s *testSessionIncBackupSuite) TestRenameTable(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.makeSQL(c, "drop table if exists t1;drop table if exists t2;create table t1(id int primary key);")
	res := s.makeSQL(c, "rename table t1 to t2;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t2", row[7].(string))
	c.Assert(backup, Equals, "RENAME TABLE `test_inc`.`t2` TO `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.makeSQL(c, "alter table t2 rename to t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` RENAME TO `test_inc`.`t2`;", Commentf("%v", res.Rows()))

}

func (s *testSessionIncBackupSuite) TestAlterTableCreateIndex(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);")
	res := s.makeSQL(c, "alter table t1 add index idx (c1);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `idx`;", Commentf("%v", res.Rows()))

	s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);")
	res = s.makeSQL(c, "create index idx on t1(c1);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP INDEX `idx` ON `test_inc`.`t1`;", Commentf("%v", res.Rows()))

}

func (s *testSessionIncBackupSuite) TestAlterTableDropIndex(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()
	sql := ""

	s.makeSQL(c, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 add index idx (c1);")
	res := s.makeSQL(c, "alter table t1 drop index idx;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD INDEX `idx`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
	create table t1(id int primary key,c1 int,unique index ix_1(c1));
	alter table t1 drop index ix_1;`
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD UNIQUE INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
	create table t1(id int primary key,c1 int);
	alter table t1 add unique index ix_1(c1);
	alter table t1 drop index ix_1;`
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD UNIQUE INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
	create table t1(id int primary key,c1 GEOMETRY not null ,SPATIAL index ix_1(c1));
	alter table t1 drop index ix_1;`
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
	create table t1(id int primary key,c1 GEOMETRY not null);
	alter table t1 add SPATIAL index ix_1(c1);
	alter table t1 drop index ix_1;`
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
	create table t1(id int primary key,c1 GEOMETRY not null);
	alter table t1 add SPATIAL index ix_1(c1);`
	s.makeExecSQL(sql)
	sql = "alter table t1 drop index ix_1;"
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

}

func (s *testSessionIncBackupSuite) TestAlterTable(c *C) {
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
	res := s.makeSQL(c, sql)
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;", Commentf("%v", res.Rows()))

	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1,add column c1 varchar(20);"
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`,ADD COLUMN `c1` int(11);", Commentf("%v", res.Rows()))

	// 删除后添加索引
	sql = "drop table if exists t1;create table t1(id int ,c1 int,key ix(c1));alter table t1 drop index ix;alter table t1 add index ix(c1);"
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `ix`;", Commentf("%v", res.Rows()))

	sql = "drop table if exists t1;create table t1(id int,c1 int,c2 int,key ix(c2));alter table t1 drop index ix,add index ix(c1);"
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `ix`,ADD INDEX `ix`(`c2`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
	create table t1(id int,c1 int,c2 datetime null default current_timestamp on update current_timestamp comment '123');
	alter table t1 modify c2 datetime;`
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c2` datetime ON UPDATE CURRENT_TIMESTAMP COMMENT '123';", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
	create table t1(id int,c1 int,c2 datetime null default current_timestamp on update current_timestamp comment '123');
	alter table t1 modify c2 datetime;`
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c2` datetime ON UPDATE CURRENT_TIMESTAMP COMMENT '123';", Commentf("%v", res.Rows()))

	// 空间类型使用的是别名,逆向SQL还有问题,待修复
	sql = `drop table if exists t1;
	create table t1(id int primary key);
	alter table t1 add column c1 geometry;
	alter table t1 add column c2 point;
	alter table t1 add column c3 linestring;
	alter table t1 add column c4 polygon;
	alter table t1 drop column c1,drop column c2,drop column c3,drop column c4;`
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c4` geometry,ADD COLUMN `c3` geometry,ADD COLUMN `c2` geometry,ADD COLUMN `c1` geometry;", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
	create table t1(id int primary key);
	alter table t1 add column c1 geometry;
	alter table t1 add column c2 point;
	alter table t1 add column c3 linestring;
	alter table t1 add column c4 polygon;`
	s.makeExecSQL(sql)

	sql = `alter table t1 drop column c1,drop column c2,drop column c3,drop column c4; `
	res = s.makeSQL(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c4` polygon,ADD COLUMN `c3` linestring,ADD COLUMN `c2` point,ADD COLUMN `c1` geometry;", Commentf("%v", res.Rows()))

}

func (s *testSessionIncBackupSuite) query(table, opid string) string {
	inc := config.GetGlobalConfig().Inc
	if s.db == nil || s.db.DB().Ping() != nil {
		addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local&maxAllowedPacket=4194304",
			inc.BackupUser, inc.BackupPassword, inc.BackupHost, inc.BackupPort)

		db, err := gorm.Open("mysql", addr)
		if err != nil {
			fmt.Println(err)
			return err.Error()
		}
		// 禁用日志记录器，不显示任何日志
		db.LogMode(false)
		s.db = db
	}

	result := []string{}
	sql := "select rollback_statement from 127_0_0_1_%d_test_inc.`%s` where opid_time = ?;"
	sql = fmt.Sprintf(sql, inc.BackupPort, table)

	rows, err := s.db.Raw(sql, opid).Rows()
	if err != nil {
		fmt.Println(err)
		return err.Error()
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

func (s *testSessionIncBackupSuite) queryStatistics() []int {
	inc := config.GetGlobalConfig().Inc
	if s.db == nil || s.db.DB().Ping() != nil {

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

	sql := `select usedb, deleting, inserting, updating,
		selecting, altertable, renaming, createindex, dropindex, addcolumn,
		dropcolumn, changecolumn, alteroption, alterconvert,
		createtable, droptable, CREATEDB, truncating from inception.statistic order by id desc limit 1;`
	values := make([]int, 18)

	rows, err := s.db.Raw(sql).Rows()
	if err != nil {
		fmt.Println(err)
		panic(err)
	} else {
		defer rows.Close()
		for rows.Next() {
			rows.Scan(&values[0],
				&values[1],
				&values[2],
				&values[3],
				&values[4],
				&values[5],
				&values[6],
				&values[7],
				&values[8],
				&values[9],
				&values[10],
				&values[11],
				&values[12],
				&values[13],
				&values[14],
				&values[15],
				&values[16],
				&values[17])
		}
	}
	return values
}

func trim(s string) string {
	if strings.Contains(s, "  ") {
		return trim(strings.Replace(s, "  ", " ", -1))
	}
	return s
}

func (s *testSessionIncBackupSuite) getSQLMode(c *C) string {
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

	res := s.makeSQL(c, sql)
	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2, Commentf("%v", res.Rows()))

	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	versionStr := row[5].(string)

	versionStr = strings.SplitN(versionStr, "|", 2)[1]
	value := strings.Replace(versionStr, "'", "", -1)
	value = strings.TrimSpace(value)

	s.sqlMode = value
	return value
}

func (s *testSessionIncBackupSuite) getExplicitDefaultsForTimestamp(c *C) bool {
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

	res := s.makeSQL(c, sql)
	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2, Commentf("%v", res.Rows()))

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

func (s *testSessionIncBackupSuite) TestStatistics(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableSqlStatistic = true

	sql := ""

	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1;alter table t1 add column c1 varchar(20);"
	res := s.makeSQL(c, sql)
	statistics := s.queryStatistics()
	result := []int{
		1, // usedb,
		0, // deleting,
		0, // inserting,
		0, // updating,
		0, // selecting,
		2, // altertable,
		0, // renaming,
		0, // createindex,
		0, // dropindex,
		1, // addcolumn,
		1, // dropcolumn,
		0, // changecolumn,
		0, // alteroption,
		0, // alterconvert,
		1, // createtable,
		1, // droptable,
		0, // CREATEDB,
		0, // truncating
	}

	c.Assert(len(statistics), Equals, len(result), Commentf("%v", res.Rows()))
	for i, v := range statistics {
		c.Assert(v, Equals, result[i], Commentf("%v", res.Rows()))
	}

	sql = `
	DROP TABLE IF EXISTS t1;

	CREATE TABLE t1(id int,c1 int);
	ALTER TABLE t1 add COLUMN c2 int;
	ALTER TABLE t1 modify COLUMN c2 varchar(100);
	alter table t1 alter column c1 set default 100;

	insert into t1(id) values(1);
	update t1 set c1=1 where id=1;
	delete from t1 where id=1;

	truncate table t1;
	`
	res = s.makeSQL(c, sql)
	statistics = s.queryStatistics()
	result = []int{
		1, // usedb,
		1, // deleting,
		1, // inserting,
		1, // updating,
		0, // selecting,
		3, // altertable,
		0, // renaming,
		0, // createindex,
		0, // dropindex,
		1, // addcolumn,
		0, // dropcolumn,
		1, // changecolumn,
		0, // alteroption,
		0, // alterconvert,
		1, // createtable,
		1, // droptable,
		0, // CREATEDB,
		1, // truncating
	}

	c.Assert(len(statistics), Equals, len(result), Commentf("%v", res.Rows()))
	for i, v := range statistics {
		c.Assert(v, Equals, result[i], Commentf("%v", statistics))
	}
}

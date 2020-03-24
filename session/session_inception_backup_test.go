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
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hanchuanchuan/goInception/config"
	. "github.com/pingcap/check"
	log "github.com/sirupsen/logrus"
)

var _ = Suite(&testSessionIncBackupSuite{})

func TestBackup(t *testing.T) {
	TestingT(t)
}

type testSessionIncBackupSuite struct {
	testCommon
}

func (s *testSessionIncBackupSuite) SetUpSuite(c *C) {

	s.initSetUp(c)

	inc := config.GetGlobalConfig().Inc

	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = false
	inc.EnableFingerprint = true
	inc.SqlSafeUpdates = 0

	inc.EnableDropTable = true
	inc.EnableBlobType = true
	inc.EnableJsonType = true

	s.defaultInc = inc
}

func (s *testSessionIncBackupSuite) TearDownSuite(c *C) {
	s.tearDownSuite(c)
}

func (s *testSessionIncBackupSuite) TearDownTest(c *C) {
	s.reset()
	// s.tearDownTest(c)
}

func (s *testSessionIncBackupSuite) TestCreateTable(c *C) {
	s.mustRunBackup(c, "drop table if exists t1;create table t1(id int);")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", s.rows))
}

func (s *testSessionIncBackupSuite) TestDropTable(c *C) {
	config.GetGlobalConfig().Inc.EnableDropTable = true
	s.mustRunBackup(c, "drop table if exists t1;create table t1(id int);")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;")

	s.mustRunBackup(c, "drop table t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	// mysql 8.0版本默认字符集改成了utf8mb4
	if s.DBVersion < 80000 {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8;")
	} else {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;")
	}

	s.mustRunBackup(c, "create table t1(id int) default charset utf8mb4;")
	s.mustRunBackup(c, "drop table t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	if s.DBVersion < 80000 {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;")
	} else {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) DEFAULT NULL\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;")
	}

	s.mustRunExec(c, "create table t1(id int not null default 0);")
	s.mustRunBackup(c, "drop table t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	if s.DBVersion < 80000 {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) NOT NULL DEFAULT '0'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8;")
	} else {
		c.Assert(backup, Equals, "CREATE TABLE `t1` (\n `id` int(11) NOT NULL DEFAULT '0'\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;")
	}
}

func (s *testSessionIncBackupSuite) TestAlterTableAddColumn(c *C) {
	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.CheckIdentifier = false // 特殊字符

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int);")
	var (
		sql    string
		row    []interface{}
		backup string
	)

	sql = `alter table t1 add column c1 int;
	alter table t1 add column c2 bit first;
	alter table t1 add column (c3 int,c4 varchar(20));
	`

	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;",
		"ALTER TABLE `test_inc`.`t1` DROP COLUMN `c2`;",
		"ALTER TABLE `test_inc`.`t1` DROP COLUMN `c4`,DROP COLUMN `c3`;",
	)

	// 特殊字符
	s.mustRunExec(c, "drop table if exists `t3!@#$^&*()`;create table `t3!@#$^&*()`(id int primary key);")

	sqls := []string{
		"alter table `t3!@#$^&*()` add column `c3!@#$^&*()2` int comment '123';",
	}

	s.mustRunBackup(c, strings.Join(sqls, "\n"))
	s.assertRows(c, s.rows[1:],
		"ALTER TABLE `test_inc`.`t3!@#$^&*()` DROP COLUMN `c3!@#$^&*()2`;",
	)

	// pt-osc
	config.GetGlobalConfig().Osc.OscOn = true
	s.mustRunBackup(c, "alter table `t3!@#$^&*()` add column `c3!@#$^&*()3` int comment '123';")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t3!@#$^&*()", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t3!@#$^&*()` DROP COLUMN `c3!@#$^&*()3`;", Commentf("%v", s.rows))

	// gh-ost
	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = true
	s.mustRunBackup(c, "alter table `t3!@#$^&*()` add column `c3!@#$^&*()4` int comment '123';")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t3!@#$^&*()", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t3!@#$^&*()` DROP COLUMN `c3!@#$^&*()4`;", Commentf("%v", s.rows))

}

func (s *testSessionIncBackupSuite) TestAlterTableAlterColumn(c *C) {

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);")

	sql := strings.Join([]string{
		"alter table t1 alter column c1 set default '1';",
		"alter table t1 alter column c1 drop default;",
	}, "\n")
	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"ALTER TABLE `test_inc`.`t1` DROP DEFAULT;",
		"ALTER TABLE `test_inc`.`t1` SET DEFAULT '1';",
	)
}

func (s *testSessionIncBackupSuite) TestAlterTableModifyColumn(c *C) {
	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	var sql string
	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);")

	sql = strings.Join([]string{
		"alter table t1 modify column c1 varchar(10);",
		"alter table t1 modify column c1 varchar(100) not null;",
		"alter table t1 modify column c1 int null;",
		"alter table t1 modify column c1 int first;",
		"alter table t1 modify column c1 varchar(200) not null comment '测试列';",
		"alter table t1 modify column c1 int;",
		"alter table t1 modify column c1 varchar(20) character set utf8;",
		"alter table t1 modify column c1 varchar(20) COLLATE utf8_bin;",
	}, "\n")
	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);",
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(10);",
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(100) NOT NULL;",
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);",
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);",
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(200) NOT NULL COMMENT '测试列';",
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);",
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` varchar(20);",
	)
}

func (s *testSessionIncBackupSuite) TestAlterTableChangeColumn(c *C) {
	var sql string
	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableChangeColumn = true

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);")

	s.mustRunBackup(c, "alter table t1 change column c1 c1 varchar(10);")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c1` int(11);")

	sql = `alter table t1 change column c1 c2 varchar(100) not null;
	alter table t1 add column c3 int,add column c4 int;
	alter table t1 change column c3 c1 int first;
	alter table t1 change column c2 c5 int after c1,modify column c4 int after c5;
	`

	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"ALTER TABLE `test_inc`.`t1` CHANGE COLUMN `c2` `c1` varchar(10);",
		"ALTER TABLE `test_inc`.`t1` DROP COLUMN `c4`,DROP COLUMN `c3`;",
		"ALTER TABLE `test_inc`.`t1` CHANGE COLUMN `c1` `c3` int(11);",
		"ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c4` int(11),CHANGE COLUMN `c5` `c2` varchar(100) NOT NULL;",
	)
}

func (s *testSessionIncBackupSuite) TestAlterTableDropColumn(c *C) {
	var sql string
	sql = `drop table if exists t1;
	create table t1(id int primary key,
		c1 int,
		c2 int(11) not null default 0 comment '测试列');`
	s.mustRunExec(c, sql)

	sql = strings.Join([]string{
		"ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;",
		"ALTER TABLE `test_inc`.`t1` DROP COLUMN `c2`;",
	}, "\n")

	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"ALTER TABLE `test_inc`.`t1` ADD COLUMN `c1` int(11);",
		"ALTER TABLE `test_inc`.`t1` ADD COLUMN `c2` int(11) NOT NULL DEFAULT '0' COMMENT '测试列';",
	)
}

func (s *testSessionIncBackupSuite) TestInsert(c *C) {
	config.GetGlobalConfig().Inc.CheckInsertField = false

	s.mustRunBackup(c, "drop table if exists t1;create table t1(id int);")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP TABLE `test_inc`.`t1`;", Commentf("%v", s.rows))

	s.mustRunBackup(c, "insert into t1 values(1);")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `id`=1;", Commentf("%v", s.rows))

	// 值溢出时的回滚解析
	s.mustRunBackup(c, `drop table if exists t1;
		create table t1(id int primary key,
			c1 tinyint unsigned,
			c2 smallint unsigned,
			c3 mediumint unsigned,
			c4 int unsigned,
			c5 bigint unsigned
		);`)
	s.mustRunBackup(c, "insert into t1 values(1,128,32768,8388608,2147483648,9223372036854775808);")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `id`=1;", Commentf("%v", s.rows))

	s.mustRunExec(c, `drop table if exists t1;
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

	sql := strings.Join([]string{
		`insert into t1 values(1,128,32768,8388608,2147483648,9223372036854775808,
		9999999999,99.99,3.402823466e+38,1.7976931348623157e+308);`,
		`UPDATE t1
			SET c1 = 129,
			    c2 = 32769,
			    c3 = 8388609,
			    c4 = 2147483649,
			    c5 = 9223372036854775809,
			    c6 = 9999999990,
			    c7 = 88.88, c8 = 3e+38,
			                      c9 = 1e+308
			WHERE id = 1;`,
		`delete from t1 where id = 1;`,
	}, "\n")
	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"DELETE FROM `test_inc`.`t1` WHERE `id`=1 AND `c1`=128 AND `c2`=32768 AND "+
			"`c3`=8388608 AND `c4`=2147483648 AND `c5`=9223372036854775808 AND "+
			"`c6`=9999999999 AND `c7`=99.99 AND `c8`=3.4028235e+38 AND `c9`=1.7976931348623157e+308;",
		"UPDATE `test_inc`.`t1` SET `id`=1, `c1`=128, `c2`=32768, "+
			"`c3`=8388608, `c4`=2147483648, `c5`=9223372036854775808, `c6`=9999999999, "+
			"`c7`=99.99, `c8`=3.4028235e+38, `c9`=1.7976931348623157e+308 "+
			"WHERE `id`=1 AND `c1`=129 AND `c2`=32769 AND `c3`=8388609 AND "+
			"`c4`=2147483649 AND `c5`=9223372036854775809 AND `c6`=9999999990 "+
			"AND `c7`=88.88 AND `c8`=3e+38 AND `c9`=1e+308;",
		"INSERT INTO `test_inc`.`t1`(`id`,`c1`,`c2`,`c3`,`c4`,`c5`,`c6`,`c7`,`c8`,`c9`)"+
			" VALUES(1,129,32769,8388609,2147483649,9223372036854775809,9999999990,88.88,3e+38,1e+308);",
	)

	s.mustRunExec(c, `drop table if exists t1;
		create table t1(c1 bigint unsigned);`)

	sql = strings.Join([]string{
		"insert into t1 values(9223372036854775808);",
		"insert into t1 values(18446744073709551615);",
	}, "\n")
	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"DELETE FROM `test_inc`.`t1` WHERE `c1`=9223372036854775808;",
		"DELETE FROM `test_inc`.`t1` WHERE `c1`=18446744073709551615;",
	)

	s.mustRunBackup(c, `drop table if exists t1;
create table t1(id int primary key,c1 varchar(100))default character set utf8mb4;
insert into t1(id,c1)values(1,'😁😄🙂👩');`)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DELETE FROM `test_inc`.`t1` WHERE `id`=1;", Commentf("%v", s.rows))
}

func (s *testSessionIncBackupSuite) TestUpdate(c *C) {

	// 无主键update
	s.mustRunExec(c, `drop table if exists t1;
	create table t1(id int,c1 int);
	insert into t1 values(1,1),(2,2);`)

	sql := strings.Join([]string{
		"update t1 set c1=10 where id = 1;",
		"update t1 set id=id+2 where id > 0;",
	}, "\n")
	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"UPDATE `test_inc`.`t1` SET `id`=1, `c1`=1 WHERE `id`=1 AND `c1`=10;",
		"UPDATE `test_inc`.`t1` SET `id`=1, `c1`=10 WHERE `id`=3 AND `c1`=10;",
		"UPDATE `test_inc`.`t1` SET `id`=2, `c1`=2 WHERE `id`=4 AND `c1`=2;",
	)

	s.mustRunExec(c, `drop table if exists t1;
	create table t1(id int primary key,c1 decimal(18,4),c2 decimal(18,4));
	insert into t1 values(1,123456789012.1234,123456789012.1235);`)

	s.mustRunBackup(c, "update t1 set c1 = 123456789012 where id>0;")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "UPDATE `test_inc`.`t1` SET `id`=1, `c1`=123456789012.1234, `c2`=123456789012.1235 WHERE `id`=1;", Commentf("%v", s.rows))

	// -------------------- 多表update -------------------
	sql = `drop table if exists table1;drop table if exists table2;
		create table table1(id1 int primary key,c1 int,c2 int);
		create table table2(id2 int primary key,c1 int,c2 int,c22 int);
		insert into table1 values(1,1,1),(2,1,1);
		insert into table2 values(1,1,1,null),(2,1,null,null);`
	s.mustRunExec(c, sql)

	sql = `update table1 t1,table2 t2 set t1.c1=10,t2.c22=20 where t1.id1=t2.id2 and t2.c1=1;`
	s.mustRunBackup(c, sql)

	s.assertRows(c, s.rows[1:],
		"UPDATE `test_inc`.`table2` SET `id2`=1, `c1`=1, `c2`=1, `c22`=NULL WHERE `id2`=1;",
		"UPDATE `test_inc`.`table2` SET `id2`=2, `c1`=1, `c2`=NULL, `c22`=NULL WHERE `id2`=2;",
		"UPDATE `test_inc`.`table1` SET `id1`=1, `c1`=1, `c2`=1 WHERE `id1`=1;",
		"UPDATE `test_inc`.`table1` SET `id1`=2, `c1`=1, `c2`=1 WHERE `id1`=2;",
	)

	sql = `drop table if exists table1;drop table if exists table2;
		create table table1(id1 int primary key,c1 int,c2 int);
		create table table2(id2 int primary key,c1 int,c2 int,c22 int);
		insert into table1 values(1,1,1),(2,1,1);
		insert into table2 values(1,1,1,null),(2,1,null,null);`
	s.mustRunExec(c, sql)

	sql = `update table1 t1,table2 t2 set c22=20,t1.c1=10 where t1.id1=t2.id2 and t2.c1=1;`

	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"UPDATE `test_inc`.`table2` SET `id2`=1, `c1`=1, `c2`=1, `c22`=NULL WHERE `id2`=1;",
		"UPDATE `test_inc`.`table2` SET `id2`=2, `c1`=1, `c2`=NULL, `c22`=NULL WHERE `id2`=2;",
		"UPDATE `test_inc`.`table1` SET `id1`=1, `c1`=1, `c2`=1 WHERE `id1`=1;",
		"UPDATE `test_inc`.`table1` SET `id1`=2, `c1`=1, `c2`=1 WHERE `id1`=2;",
	)

}

func (s *testSessionIncBackupSuite) TestMinimalUpdate(c *C) {
	config.GetGlobalConfig().Inc.EnableMinimalRollback = true

	s.mustRunExec(c, `drop table if exists t1;
	create table t1(id int primary key,c1 int);
	insert into t1 values(1,1),(2,2);`)

	sql = `update t1 set c1=10 where id = 1;
		update t1 set c1=10 where id > 0;
		update t1 set c1=2 where id > 0;
		`

	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"UPDATE `test_inc`.`t1` SET `c1`=1 WHERE `id`=1;",
		"UPDATE `test_inc`.`t1` SET `c1`=2 WHERE `id`=2;",
		"UPDATE `test_inc`.`t1` SET `c1`=10 WHERE `id`=1;",
		"UPDATE `test_inc`.`t1` SET `c1`=10 WHERE `id`=2;",
	)

	s.mustRunExec(c, `drop table if exists t1;
		create table t1(id int primary key,c1 tinyint unsigned,c2 varchar(100));
		insert into t1 values(1,127,'t1'),(2,130,'t2');`)

	s.mustRunBackup(c, "update t1 set c1=130,c2='aa' where id > 0;")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `c1`=127, `c2`='t1' WHERE `id`=1;",
			"UPDATE `test_inc`.`t1` SET `c2`='t2' WHERE `id`=2;",
		}, "\n"), Commentf("%v", s.rows))

	s.mustRunBackup(c, `drop table if exists t1;
		create table t1(id int,c1 tinyint unsigned,c2 varchar(100));
		insert into t1 values(1,127,'t1'),(2,130,'t2');`)

	s.mustRunBackup(c, "update t1 set c1=130,c2='aa' where id > 0;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `c1`=127, `c2`='t1' WHERE `id`=1 AND `c1`=130 AND `c2`='aa';",
			"UPDATE `test_inc`.`t1` SET `c2`='t2' WHERE `id`=2 AND `c1`=130 AND `c2`='aa';",
		}, "\n"), Commentf("%v", s.rows))

	s.mustRunBackup(c, `drop table if exists t1;
		create table t1(id int primary key,c1 int);
		insert into t1 values(1,1),(2,2);`)
	s.mustRunBackup(c, "update t1 set id=id+2 where id > 0;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `id`=1 WHERE `id`=3;",
			"UPDATE `test_inc`.`t1` SET `id`=2 WHERE `id`=4;",
		}, "\n"), Commentf("%v", s.rows))

	sql = `drop table if exists table1;drop table if exists table2;
		create table table1(id1 int primary key,c1 int,c2 int);
		create table table2(id2 int primary key,c1 int,c2 int,c22 int);
		insert into table1 values(1,1,1),(2,1,1);
		insert into table2 values(1,1,1,null),(2,1,null,null);`
	s.mustRunExec(c, sql)

	sql = `update table1 t1,table2 t2 set t1.c1=10,t2.c22=20 where t1.id1=t2.id2 and t2.c1=1;`

	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"UPDATE `test_inc`.`table2` SET `c22`=NULL WHERE `id2`=1;",
		"UPDATE `test_inc`.`table2` SET `c22`=NULL WHERE `id2`=2;",
		"UPDATE `test_inc`.`table1` SET `c1`=1 WHERE `id1`=1;",
		"UPDATE `test_inc`.`table1` SET `c1`=1 WHERE `id1`=2;",
	)

	sql = `drop table if exists table1;drop table if exists table2;
	create table table1(id1 int primary key,c1 int,c2 int);
	create table table2(id2 int primary key,c1 int,c2 int,c22 int);
	insert into table1 values(1,1,1),(2,1,1);
	insert into table2 values(1,1,1,null),(2,1,null,null);`
	s.mustRunExec(c, sql)

	sql = `update table1 t1,table2 t2 set c22=20,t1.c1=10 where t1.id1=t2.id2 and t2.c1=1;`

	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"UPDATE `test_inc`.`table2` SET `c22`=NULL WHERE `id2`=1;",
		"UPDATE `test_inc`.`table2` SET `c22`=NULL WHERE `id2`=2;",
		"UPDATE `test_inc`.`table1` SET `c1`=1 WHERE `id1`=1;",
		"UPDATE `test_inc`.`table1` SET `c1`=1 WHERE `id1`=2;",
	)
}

func (s *testSessionIncBackupSuite) TestDelete(c *C) {

	s.mustRunExec(c, `drop table if exists t1;
	create table t1(id int,c1 int);
	insert into t1 values(1,1),(2,2);`)

	s.mustRunBackup(c, "delete from t1 where id <= 2;")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);",
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(2,2);",
		}, "\n"), Commentf("%v", s.rows))

	// ------------- 二进制类型: binary,varbinary,blob ----------------
	s.mustRunBackup(c, `drop table if exists t1;
		create table t1(id int primary key,c1 blob);
		insert into t1 values(1,X'010203');`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'\x01\x02\x03');", Commentf("%v", s.rows))

	s.mustRunBackup(c, `drop table if exists t1;
		create table t1(id int primary key,c1 binary(100));
		insert into t1 values(1,X'010203');`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'\x01\x02\x03');", Commentf("%v", s.rows))

	s.mustRunBackup(c, `drop table if exists t1;
		create table t1(id int primary key,c1 varbinary(100));
		insert into t1 values(1,X'010203');`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'\x01\x02\x03');", Commentf("%v", s.rows))

	// 开启二进制自动转换为十六进制字符串
	config.GetGlobalConfig().Inc.HexBlob = true
	s.mustRunBackup(c, `drop table if exists t1;
		create table t1(id int primary key,c1 varbinary(100));
		insert into t1 values(1,X'71E6D5A383BB447C');`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,X'71e6d5a383bb447c');", Commentf("%v", s.rows))

	s.mustRunBackup(c, `insert into t1 values(1,unhex('71E6D5A383BB447C'));`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,X'71e6d5a383bb447c');", Commentf("%v", s.rows))
	config.GetGlobalConfig().Inc.HexBlob = false

	// ------------- json类型 ----------------
	if s.DBVersion >= 50708 {
		sql = `drop table if exists t1;
		create table t1(id int primary key,c1 json);
		insert into t1 values(1,'{"time":"2015-01-01 13:00:00","result":"fail"}');
		INSERT INTO t1(id,c1) VALUES(2,'{"result":"fail","time":"2015-01-01 13:00:00"}');
		`
		s.mustRunExec(c, sql)

		sql = `delete from t1 where id > 0;`
		s.mustRunBackup(c, sql)
		s.assertRows(c, s.rows[1:],
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');",
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(2,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');",
		)
	}

	sql = `drop table if exists t1;
	create table t1(id int primary key,c1 enum('type1','type2','type3','type4'));
	insert into t1 values(1,'type2');
	INSERT INTO t1(id,c1) VALUES(2,2);`
	s.mustRunExec(c, sql)

	sql = `delete from t1 where id > 0;`
	s.mustRunBackup(c, sql)
	s.assertRows(c, s.rows[1:],
		"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);",
		"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(2,2);",
	)

	s.mustRunBackup(c, `drop table if exists t1;create table t1(id int primary key,c1 bit);
	insert into t1 values(1,1);`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);", Commentf("%v", s.rows))

	s.mustRunBackup(c, `drop table if exists t1;create table t1(id int primary key,c1 double);
	insert into t1 values(1,1.11e100);`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);", Commentf("%v", s.rows))

	s.mustRunBackup(c, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);")
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);")

	s.mustRunBackup(c, `drop table if exists t1;create table t1(id int primary key,c1 date);
	insert into t1 values(1,'2019-1-1');`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'2019-01-01');", Commentf("%v", s.rows))

	if s.DBVersion >= 50700 {
		s.mustRunBackup(c, `drop table if exists t1;create table t1(id int primary key,c1 timestamp);
	insert into t1(id) values(1);`)
		s.mustRunBackup(c, "delete from t1;")
		row = s.rows[int(s.session.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		if s.explicitDefaultsForTimestamp {
			c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,NULL);", Commentf("%v", s.rows))
		} else {
			v := strings.HasPrefix(backup, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'20")
			c.Assert(v, Equals, true, Commentf("%v", s.rows))
		}
	}

	s.mustRunBackup(c, `drop table if exists t1;create table t1(id int primary key,c1 time);
	insert into t1 values(1,'00:01:01');`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'00:01:01');", Commentf("%v", s.rows))

	s.mustRunBackup(c, `drop table if exists t1;create table t1(id int primary key,c1 year);
	insert into t1 values(1,2019);`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2019);", Commentf("%v", s.rows))

	s.mustRunBackup(c, `drop table if exists t1;
	create table t1(id int primary key,c1 varchar(100))default character set utf8mb4;
	insert into t1(id,c1)values(1,'😁😄🙂👩');`)
	s.mustRunBackup(c, "delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'😁😄🙂👩');", Commentf("%v", s.rows))

	s.mustRunBackup(c, `DROP TABLE IF EXISTS t1;
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

	s.mustRunBackup(c, "delete from t1 where id>0;")
	row = s.rows[int(s.session.AffectedRows())-1]
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
		"INSERT INTO `test_inc`.`t1`(`id`,`v4_2`,`v5_0`,`v7_3`,`v10_2`,`v10_3`,`v13_2`,`v15_14`,`v20_10`,`v30_5`,`v30_20`,`v30_25`,`prec`,`scale`) VALUES(17,-99.99,-1948,-1948.14,-1948.14,-1948.14,-1948.14,-9.99999999999999,-1948.14,-1948.14,-1948.14,-1948.14,13,2);", Commentf("%v", s.rows))

	log.SetLevel(log.FatalLevel)
	// 二进制在解析时会误存为string,此时会报错
	s.runBackup(`drop table if exists t1;
		create table t1(id int primary key,c1 varbinary(100));
		insert into t1 values(1,X'71E6D5A383BB447C');`)
	s.runBackup("delete from t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	if s.DBVersion >= 50700 {
		c.Assert(row[2], Equals, "2", Commentf("%v", row))
	} else {
		c.Assert(row[2], Equals, "0", Commentf("%v", row))
	}
}

func (s *testSessionIncBackupSuite) TestCreateDataBase(c *C) {
	inc := &config.GetGlobalConfig().Inc
	inc.EnableDropDatabase = true

	dbname := fmt.Sprintf("%s_%d_%s", strings.ReplaceAll(inc.BackupHost, ".", "_"), inc.BackupPort, "test_inc")
	s.mustRunExec(c, fmt.Sprintf("drop database if exists %s;", dbname))

	s.mustRunExec(c, "drop database if exists test123456;")

	s.mustRunBackup(c, "create database test123456;")
	// row := s.rows[int(s.session.AffectedRows())-1]
	// backup := s.query("t1", row[7].(string))
	// c.Assert(backup, Equals, "", Commentf("%v", s.rows))

	s.mustRunExec(c, "drop database if exists test123456;")
}

func (s *testSessionIncBackupSuite) TestRenameTable(c *C) {
	s.mustRunExec(c, "drop table if exists t1;drop table if exists t2;create table t1(id int primary key);")
	s.mustRunBackup(c, "rename table t1 to t2;")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t2", row[7].(string))
	c.Assert(backup, Equals, "RENAME TABLE `test_inc`.`t2` TO `test_inc`.`t1`;", Commentf("%v", s.rows))

	s.mustRunBackup(c, "alter table t2 rename to t1;")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` RENAME TO `test_inc`.`t2`;", Commentf("%v", s.rows))

}

func (s *testSessionIncBackupSuite) TestAlterTableCreateIndex(c *C) {
	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);")
	s.mustRunBackup(c, "alter table t1 add index idx (c1);")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `idx`;", Commentf("%v", s.rows))

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);")
	s.mustRunBackup(c, "create index idx on t1(c1);")
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP INDEX `idx` ON `test_inc`.`t1`;", Commentf("%v", s.rows))

}

func (s *testSessionIncBackupSuite) TestAlterTableDropIndex(c *C) {
	sql := ""

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 add index idx (c1);")
	s.mustRunBackup(c, "alter table t1 drop index idx;")
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD INDEX `idx`(`c1`);", Commentf("%v", s.rows))

	sql = `drop table if exists t1;
	create table t1(id int primary key,c1 int,unique index ix_1(c1));
	alter table t1 drop index ix_1;`
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD UNIQUE INDEX `ix_1`(`c1`);", Commentf("%v", s.rows))

	sql = `drop table if exists t1;
	create table t1(id int primary key,c1 int);
	alter table t1 add unique index ix_1(c1);
	alter table t1 drop index ix_1;`
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD UNIQUE INDEX `ix_1`(`c1`);", Commentf("%v", s.rows))

	// 几何类型字段低版本不支持
	if s.DBVersion >= 50700 {
		sql = `drop table if exists t1;
	create table t1(id int primary key,c1 GEOMETRY not null ,SPATIAL index ix_1(c1));
	alter table t1 drop index ix_1;`
		s.mustRunBackup(c, sql)
		row = s.rows[int(s.session.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", s.rows))

		sql = `drop table if exists t1;
	create table t1(id int primary key,c1 GEOMETRY not null);
	alter table t1 add SPATIAL index ix_1(c1);
	alter table t1 drop index ix_1;`
		s.mustRunBackup(c, sql)
		row = s.rows[int(s.session.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", s.rows))

		sql = `drop table if exists t1;
	create table t1(id int primary key,c1 GEOMETRY not null);
	alter table t1 add SPATIAL index ix_1(c1);`
		s.runBackup(sql)
		sql = "alter table t1 drop index ix_1;"
		s.mustRunBackup(c, sql)
		row = s.rows[int(s.session.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", s.rows))
	}
}

func (s *testSessionIncBackupSuite) TestAlterTable(c *C) {
	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableDropTable = true

	sql := ""

	// 删除后添加列
	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1;alter table t1 add column c1 varchar(20);"
	s.mustRunBackup(c, sql)
	row := s.rows[int(s.session.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;", Commentf("%v", s.rows))

	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1,add column c1 varchar(20);"
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`,ADD COLUMN `c1` int(11);", Commentf("%v", s.rows))

	// 删除后添加索引
	sql = "drop table if exists t1;create table t1(id int ,c1 int,key ix(c1));alter table t1 drop index ix;alter table t1 add index ix(c1);"
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `ix`;", Commentf("%v", s.rows))

	sql = "drop table if exists t1;create table t1(id int,c1 int,c2 int,key ix(c2));alter table t1 drop index ix,add index ix(c1);"
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `ix`,ADD INDEX `ix`(`c2`);", Commentf("%v", s.rows))

	sql = `drop table if exists t1;
	create table t1(id int,c1 int,c2 datetime null default current_timestamp on update current_timestamp comment '123');
	alter table t1 modify c2 datetime;`
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c2` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '123';", Commentf("%v", s.rows))

	sql = `drop table if exists t1;
	create table t1(id int,c1 int,c2 datetime null default current_timestamp
		on update current_timestamp comment '123',
		c3 int default 10);`
	s.mustRunExec(c, sql)
	sql = `alter table t1 modify c2 datetime;`
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c2` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '123';", Commentf("%v", s.rows))

	sql = `alter table t1 modify c3 bigint;`
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c3` int(11) DEFAULT '10';", Commentf("%v", s.rows))

	sql = `drop table if exists t1;
	create table t1(id int,c1 int,
		c2 datetime null default current_timestamp
		on update current_timestamp comment '123');
	alter table t1 modify c2 datetime;`
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c2` datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '123';", Commentf("%v", s.rows))

	// 空间类型使用的是别名,逆向SQL还有问题,待修复
	sql = `drop table if exists t1;
	create table t1(id int primary key);
	alter table t1 add column c1 geometry;
	alter table t1 add column c2 point;
	alter table t1 add column c3 linestring;
	alter table t1 add column c4 polygon;
	alter table t1 drop column c1,drop column c2,drop column c3,drop column c4;`
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c4` geometry,ADD COLUMN `c3` geometry,ADD COLUMN `c2` geometry,ADD COLUMN `c1` geometry;", Commentf("%v", s.rows))

	sql = `drop table if exists t1;
	create table t1(id int primary key);
	alter table t1 add column c1 geometry;
	alter table t1 add column c2 point;
	alter table t1 add column c3 linestring;
	alter table t1 add column c4 polygon;`
	s.runBackup(sql)

	sql = `alter table t1 drop column c1,drop column c2,drop column c3,drop column c4; `
	s.mustRunBackup(c, sql)
	row = s.rows[int(s.session.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c4` polygon,ADD COLUMN `c3` linestring,ADD COLUMN `c2` point,ADD COLUMN `c1` geometry;", Commentf("%v", s.rows))

}

func (s *testSessionIncBackupSuite) TestStatistics(c *C) {
	config.GetGlobalConfig().Inc.EnableSqlStatistic = true

	sql := ""

	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1;alter table t1 add column c1 varchar(20);"
	s.mustRunBackup(c, sql)
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

	c.Assert(len(statistics), Equals, len(result), Commentf("%v", s.rows))
	for i, v := range statistics {
		c.Assert(v, Equals, result[i], Commentf("%v", s.rows))
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
	s.mustRunBackup(c, sql)
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

	c.Assert(len(statistics), Equals, len(result), Commentf("%v", s.rows))
	for i, v := range statistics {
		c.Assert(v, Equals, result[i], Commentf("%v", statistics))
	}
}

func (s *testSessionIncBackupSuite) TestEmptyUseDB(c *C) {
	old := s.useDB
	s.useDB = ""
	go func() {
		s.useDB = old
	}()

	s.mustRunBackup(c, `drop table if exists test_inc.t_no_db;
	create table test_inc.t_no_db(id int);
	insert into test_inc.t_no_db values(1);`)
	s.assertRows(c, s.rows[1:],
		"DROP TABLE `test_inc`.`t_no_db`;",
		"DELETE FROM `test_inc`.`t_no_db` WHERE `id`=1;",
	)

	s.mustRunExec(c, `drop table if exists test_inc.t1;`)
}

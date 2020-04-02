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
	"github.com/hanchuanchuan/goInception/util/testkit"
	. "github.com/pingcap/check"
)

var _ = Suite(&testSessionIncTranSuite{})

func TestTranBackup(t *testing.T) {
	TestingT(t)
}

type testSessionIncTranSuite struct {
	testCommon
}

func (s *testSessionIncTranSuite) SetUpSuite(c *C) {
	s.initSetUp(c)

	inc := &config.GetGlobalConfig().Inc
	inc.EnableFingerprint = true
	inc.SqlSafeUpdates = 0

	inc.EnableDropTable = true
	inc.EnableBlobType = true
	inc.EnableJsonType = true
}

func (s *testSessionIncTranSuite) TearDownSuite(c *C) {
	s.tearDownSuite(c)
}

func (s *testSessionIncTranSuite) TearDownTest(c *C) {
	s.tearDownTest(c)
}

func (s *testSessionIncTranSuite) TestInsert(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false
	var (
		res *testkit.Result
		// row    []interface{}
		// backup string
	)
	res = s.mustRunBackupTran(c, "drop table if exists t1;create table t1(id int);")
	s.assertRows(c, res.Rows()[2:], "DROP TABLE `test_inc`.`t1`;")

	res = s.mustRunBackupTran(c, "insert into t1 values(1);")
	s.assertRows(c, res.Rows()[1:], "DELETE FROM `test_inc`.`t1` WHERE `id`=1;")

	res = s.mustRunBackupTran(c, `drop table if exists t1;
create table t1(id int primary key,c1 varchar(100))default character set utf8mb4;
insert into t1(id,c1)values(1,'😁😄🙂👩');
delete from t1 where id=1;`)
	s.assertRows(c, res.Rows()[3:],
		"DELETE FROM `test_inc`.`t1` WHERE `id`=1;",
		"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'😁😄🙂👩');")

	res = s.mustRunBackupTran(c, `drop table if exists t1;
create table t1(id int primary key,c1 varchar(100),c2 int);

delete from t1 where id>0;
insert into t1(id,c1) values(1,"1");
insert into t1(id,c1) values(2,"2");
insert into t1(id,c1) values(3,"3"),(4,"4");
update t1 set c1='10' where id>0;`)
	s.assertRows(c, res.Rows()[3:],
		"DELETE FROM `test_inc`.`t1` WHERE `id`=1;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=2;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=3;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=4;",
		"UPDATE `test_inc`.`t1` SET `id`=1, `c1`='1', `c2`=NULL WHERE `id`=1;",
		"UPDATE `test_inc`.`t1` SET `id`=2, `c1`='2', `c2`=NULL WHERE `id`=2;",
		"UPDATE `test_inc`.`t1` SET `id`=3, `c1`='3', `c2`=NULL WHERE `id`=3;",
		"UPDATE `test_inc`.`t1` SET `id`=4, `c1`='4', `c2`=NULL WHERE `id`=4;")

	s.runTranSQL(`create database if not exists test;`, 10)

	res = s.mustRunBackupTran(c, `drop table if exists t1;
create table t1(id int primary key,c1 varchar(100),c2 int);

delete from t1 where id>0;
insert into t1(id,c1) values(1,"1");
insert into t1(id,c1) values(2,"2");
insert into t1(id,c1) values(3,"3"),(4,"4");
update t1 set c1='10' where id>0;

use test;
drop table if exists t22;
create table t22(id int primary key,c1 varchar(100),c2 int);

insert into t22(id,c1) values(1,"1");
insert into t22(id,c1) values(2,"2");
insert into t22(id,c1) values(3,"3");
insert into t22(id,c1) values(4,"4");
insert into t22(id,c1) values(5,"5");
insert into t22(id,c1) values(6,"6");`)
	s.assertRows(c, res.Rows()[3:8],
		"DELETE FROM `test_inc`.`t1` WHERE `id`=1;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=2;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=3;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=4;",
		"UPDATE `test_inc`.`t1` SET `id`=1, `c1`='1', `c2`=NULL WHERE `id`=1;",
		"UPDATE `test_inc`.`t1` SET `id`=2, `c1`='2', `c2`=NULL WHERE `id`=2;",
		"UPDATE `test_inc`.`t1` SET `id`=3, `c1`='3', `c2`=NULL WHERE `id`=3;",
		"UPDATE `test_inc`.`t1` SET `id`=4, `c1`='4', `c2`=NULL WHERE `id`=4;",
	)
	s.assertRows(c, res.Rows()[10:],
		"DROP TABLE `test`.`t22`;",
		"DELETE FROM `test`.`t22` WHERE `id`=1;",
		"DELETE FROM `test`.`t22` WHERE `id`=2;",
		"DELETE FROM `test`.`t22` WHERE `id`=3;",
		"DELETE FROM `test`.`t22` WHERE `id`=4;",
		"DELETE FROM `test`.`t22` WHERE `id`=5;",
		"DELETE FROM `test`.`t22` WHERE `id`=6;")

	// 主键冲突时
	res = s.runTranSQL(`drop table if exists t1;
create table t1(id int primary key,c1 varchar(100),c2 int);
insert into t1(id,c1) values(1,"1");
insert into t1(id,c1) values(2,"2");
insert into t1(id,c1) values(3,"3");
insert into t1(id,c1) values(3,"4");
insert into t1(id,c1) values(5,"5");
update t1 set c1='10' where id>0;`, 3)

	s.assertRows(c, res.Rows()[3:],
		"DELETE FROM `test_inc`.`t1` WHERE `id`=1;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=2;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=3;",
	)

	res = s.runTranSQL(`drop table if exists t1;
create table t1(id int primary key,c1 varchar(100),c2 int);
insert into t1(id,c1) values(1,"1");
insert into t1(id,c1) values(2,"2");
insert into t1(id,c1) values(3,"3");
insert into t1(id,c1) values(3,"4");
insert into t1(id,c1) values(5,"5");
insert into t1(id,c1) values(6,"6");
update t1 set c1='10' where id>0;`, 5)

	s.assertRows(c, res.Rows()[3:])

	res = s.runTranSQL(`drop table if exists t1;
create table t1(id int primary key,c1 varchar(100),c2 int);
insert into t1(id,c1) values(1,"1");
insert into t1(id,c1) values(2,"2");
insert into t1(id,c1) values(3,"3");
insert into t1(id,c1) values(4,"4");
insert into t1(id,c1) values(4,"5");
insert into t1(id,c1) values(6,"6");
update t1 set c1='10' where id>0;`, 2)

	s.assertRows(c, res.Rows()[3:],
		"DELETE FROM `test_inc`.`t1` WHERE `id`=1;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=2;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=3;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=4;",
	)

	// 测试不同分批下的大量数据
	s.insertMulti(c, 2)
	s.insertMulti(c, 79)

	// for i := 41; i <= 60; i++ {
	// 	s.insertMulti(c, i)
	// }
}

func (s *testSessionIncTranSuite) insertMulti(c *C, batch int) {
	// 插入大量数据
	insert_sql := []string{
		"drop table if exists t1;",
		"create table t1(id int primary key,c1 varchar(100),c2 int);",
	}

	var rollback_sql []string
	for i := 1; i <= 1000; i++ {
		insert_sql = append(insert_sql,
			fmt.Sprintf("insert into t1(id,c1) values(%d,'%d');", i, i))

		rollback_sql = append(rollback_sql,
			fmt.Sprintf("DELETE FROM `test_inc`.`t1` WHERE `id`=%d;", i))
	}

	res := s.runTranSQL(strings.Join(insert_sql, "\n"), batch)

	s.assertRows(c, res.Rows()[3:],
		rollback_sql...,
	)
}

func (s *testSessionIncTranSuite) TestUpdate(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.mustRunBackupTran(c, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")

	res := s.mustRunBackupTran(c, "update t1 set c1=10 where id = 1;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		"UPDATE `test_inc`.`t1` SET `id`=1, `c1`=1 WHERE `id`=1 AND `c1`=10;", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;
        create table t1(id int primary key,c1 int);
        insert into t1 values(1,1),(2,2);`)
	res = s.mustRunBackupTran(c, "update t1 set id=id+2 where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `id`=1, `c1`=1 WHERE `id`=3;",
			"UPDATE `test_inc`.`t1` SET `id`=2, `c1`=2 WHERE `id`=4;",
		}, "\n"), Commentf("%v", res.Rows()))
}

func (s *testSessionIncTranSuite) TestMinimalUpdate(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableMinimalRollback = true

	s.mustRunBackupTran(c, `drop table if exists t1;
	create table t1(id int,c1 int);
	insert into t1 values(1,1),(2,2);`)

	res := s.mustRunBackupTran(c, "update t1 set c1=10 where id = 1;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "UPDATE `test_inc`.`t1` SET `c1`=1 WHERE `id`=1 AND `c1`=10;", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;
        create table t1(id int primary key,c1 int);
        insert into t1 values(1,1),(2,2);`)

	res = s.mustRunBackupTran(c, "update t1 set c1=10 where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `c1`=1 WHERE `id`=1;",
			"UPDATE `test_inc`.`t1` SET `c1`=2 WHERE `id`=2;",
		}, "\n"), Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;
        create table t1(id int primary key,c1 int);
        insert into t1 values(1,1),(2,2);`)

	res = s.mustRunBackupTran(c, "update t1 set c1=2 where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		"UPDATE `test_inc`.`t1` SET `c1`=1 WHERE `id`=1;", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;
        create table t1(id int primary key,c1 tinyint unsigned,c2 varchar(100));
        insert into t1 values(1,127,'t1'),(2,130,'t2');`)

	res = s.mustRunBackupTran(c, "update t1 set c1=130,c2='aa' where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `c1`=127, `c2`='t1' WHERE `id`=1;",
			"UPDATE `test_inc`.`t1` SET `c2`='t2' WHERE `id`=2;",
		}, "\n"), Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;
        create table t1(id int,c1 tinyint unsigned,c2 varchar(100));
        insert into t1 values(1,127,'t1'),(2,130,'t2');`)

	res = s.mustRunBackupTran(c, "update t1 set c1=130,c2='aa' where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `c1`=127, `c2`='t1' WHERE `id`=1 AND `c1`=130 AND `c2`='aa';",
			"UPDATE `test_inc`.`t1` SET `c2`='t2' WHERE `id`=2 AND `c1`=130 AND `c2`='aa';",
		}, "\n"), Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;
        create table t1(id int primary key,c1 int);
        insert into t1 values(1,1),(2,2);`)
	res = s.mustRunBackupTran(c, "update t1 set id=id+2 where id > 0;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"UPDATE `test_inc`.`t1` SET `id`=1 WHERE `id`=3;",
			"UPDATE `test_inc`.`t1` SET `id`=2 WHERE `id`=4;",
		}, "\n"), Commentf("%v", res.Rows()))
}
func (s *testSessionIncTranSuite) TestDelete(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.mustRunBackupTran(c, `drop table if exists t1;
	create table t1(id int,c1 int);
	insert into t1 values(1,1),(2,2);`)

	res := s.mustRunBackupTran(c, "delete from t1 where id <= 2;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals,
		strings.Join([]string{
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);",
			"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(2,2);",
		}, "\n"), Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, "drop table if exists t1;create table t1(id int primary key,c1 blob);insert into t1 values(1,X'010203');")
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'\x01\x02\x03');", Commentf("%v", res.Rows()))

	if s.DBVersion >= 50708 {
		s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 json);
    insert into t1 values(1,'{"time":"2015-01-01 13:00:00","result":"fail"}');`)
		res = s.mustRunBackupTran(c, "delete from t1;")
		row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');", Commentf("%v", res.Rows()))

		s.mustRunBackupTran(c, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');")
		res = s.mustRunBackupTran(c, "delete from t1;")
		row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'{\\\"result\\\":\\\"fail\\\",\\\"time\\\":\\\"2015-01-01 13:00:00\\\"}');", Commentf("%v", res.Rows()))
	}

	s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 enum('type1','type2','type3','type4'));
    insert into t1 values(1,'type2');`)
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);")
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2);")

	s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 bit);
    insert into t1 values(1,1);`)
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 decimal(10,2));
    insert into t1 values(1,1.11);`)
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11);", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 double);
    insert into t1 values(1,1.11e100);`)
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);")
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1.11e+100);")

	s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 date);
    insert into t1 values(1,'2019-1-1');`)
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'2019-01-01');", Commentf("%v", res.Rows()))

	if s.DBVersion >= 50700 {
		s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 timestamp);
    insert into t1(id) values(1);`)
		res = s.mustRunBackupTran(c, "delete from t1;")
		row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
		backup = s.query("t1", row[7].(string))
		if s.explicitDefaultsForTimestamp {
			c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,NULL);", Commentf("%v", res.Rows()))
		} else {
			v := strings.HasPrefix(backup, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'20")
			c.Assert(v, Equals, true, Commentf("%v", res.Rows()))
		}
	}

	s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 time);
    insert into t1 values(1,'00:01:01');`)
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'00:01:01');", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;create table t1(id int primary key,c1 year);
    insert into t1 values(1,2019);`)
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,2019);", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, `drop table if exists t1;
    create table t1(id int primary key,c1 varchar(100))default character set utf8mb4;
    insert into t1(id,c1)values(1,'😁😄🙂👩');`)
	res = s.mustRunBackupTran(c, "delete from t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,'😁😄🙂👩');", Commentf("%v", res.Rows()))

}

func (s *testSessionIncTranSuite) TestCreateTable(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	var (
		res *testkit.Result
		// row    []interface{}
		// backup string
	)

	res = s.mustRunBackupTran(c, `DROP TABLE IF EXISTS t1,t2;

	CREATE TABLE t1 (id int(11) NOT NULL,
		c1 int(11) DEFAULT NULL,
		c2 int(11) DEFAULT NULL,
		PRIMARY KEY (id));

	INSERT INTO t1 VALUES (1, 1, 1);

	CREATE TABLE t2 (id int(11) NOT NULL,
		c1 int(11) DEFAULT NULL,
		c2 int(11) DEFAULT NULL,
		PRIMARY KEY (id))`)
	s.assertRows(c, res.Rows()[2:],
		"DROP TABLE `test_inc`.`t1`;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=1;",
		"DROP TABLE `test_inc`.`t2`;")

	res = s.mustRunBackupTran(c, `DROP TABLE IF EXISTS t1,t2;
		create table t1(id int primary key,c1 int);
		insert into t1 values(1,1),(2,2);
		delete from t1 where id=1;
		alter table t1 add column c2 int;
		insert into t1 values(3,3,3);
		delete from t1 where id>0;
		create table t2(id int primary key,c1 int);
		insert into t2 values(3,3);`)
	s.assertRows(c, res.Rows()[2:],
		"DROP TABLE `test_inc`.`t1`;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=1;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=2;",
		"INSERT INTO `test_inc`.`t1`(`id`,`c1`) VALUES(1,1);",
		"ALTER TABLE `test_inc`.`t1` DROP COLUMN `c2`;",
		"DELETE FROM `test_inc`.`t1` WHERE `id`=3;",
		"INSERT INTO `test_inc`.`t1`(`id`,`c1`,`c2`) VALUES(2,2,NULL);",
		"INSERT INTO `test_inc`.`t1`(`id`,`c1`,`c2`) VALUES(3,3,3);",
		"DROP TABLE `test_inc`.`t2`;",
		"DELETE FROM `test_inc`.`t2` WHERE `id`=3;")

}

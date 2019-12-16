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
	"github.com/jinzhu/gorm"
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
	s.insertMulti(c, 3)
	s.insertMulti(c, 13)
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

	s.mustRunBackupTran(c, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")

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

	s.mustRunBackupTran(c, "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1),(2,2);")

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
		if s.getExplicitDefaultsForTimestamp(c) {
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

func (s *testSessionIncTranSuite) TestCreateDataBase(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableDropDatabase = true

	s.mustRunBackupTran(c, "drop database if exists test123456;")
	res := s.mustRunBackupTran(c, "create database test123456;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, "drop database if exists test123456;")
}

func (s *testSessionIncTranSuite) TestRenameTable(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.mustRunBackupTran(c, "drop table if exists t1;drop table if exists t2;create table t1(id int primary key);")
	res := s.mustRunBackupTran(c, "rename table t1 to t2;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t2", row[7].(string))
	c.Assert(backup, Equals, "RENAME TABLE `test_inc`.`t2` TO `test_inc`.`t1`;", Commentf("%v", res.Rows()))

	res = s.mustRunBackupTran(c, "alter table t2 rename to t1;")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` RENAME TO `test_inc`.`t2`;", Commentf("%v", res.Rows()))

}

func (s *testSessionIncTranSuite) TestAlterTableCreateIndex(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	s.mustRunBackupTran(c, "drop table if exists t1;create table t1(id int,c1 int);")
	res := s.mustRunBackupTran(c, "alter table t1 add index idx (c1);")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `idx`;", Commentf("%v", res.Rows()))

	s.mustRunBackupTran(c, "drop table if exists t1;create table t1(id int,c1 int);")
	res = s.mustRunBackupTran(c, "create index idx on t1(c1);")
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "DROP INDEX `idx` ON `test_inc`.`t1`;", Commentf("%v", res.Rows()))

}

func (s *testSessionIncTranSuite) TestAlterTableDropIndex(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()
	sql := ""

	s.mustRunBackupTran(c, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 add index idx (c1);")
	res := s.mustRunBackupTran(c, "alter table t1 drop index idx;")
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD INDEX `idx`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
    create table t1(id int primary key,c1 int,unique index ix_1(c1));
    alter table t1 drop index ix_1;`
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD UNIQUE INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
    create table t1(id int primary key,c1 int);
    alter table t1 add unique index ix_1(c1);
    alter table t1 drop index ix_1;`
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD UNIQUE INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
    create table t1(id int primary key,c1 GEOMETRY not null ,SPATIAL index ix_1(c1));
    alter table t1 drop index ix_1;`
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
    create table t1(id int primary key,c1 GEOMETRY not null);
    alter table t1 add SPATIAL index ix_1(c1);
    alter table t1 drop index ix_1;`
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
    create table t1(id int primary key,c1 GEOMETRY not null);
    alter table t1 add SPATIAL index ix_1(c1);`
	s.runTranSQL(sql, 10)
	sql = "alter table t1 drop index ix_1;"
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD SPATIAL INDEX `ix_1`(`c1`);", Commentf("%v", res.Rows()))

}

func (s *testSessionIncTranSuite) TestAlterTable(c *C) {
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
	res := s.mustRunBackupTran(c, sql)
	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup := s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`;", Commentf("%v", res.Rows()))

	sql = "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1,add column c1 varchar(20);"
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP COLUMN `c1`,ADD COLUMN `c1` int(11);", Commentf("%v", res.Rows()))

	// 删除后添加索引
	sql = "drop table if exists t1;create table t1(id int ,c1 int,key ix(c1));alter table t1 drop index ix;alter table t1 add index ix(c1);"
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `ix`;", Commentf("%v", res.Rows()))

	sql = "drop table if exists t1;create table t1(id int,c1 int,c2 int,key ix(c2));alter table t1 drop index ix,add index ix(c1);"
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` DROP INDEX `ix`,ADD INDEX `ix`(`c2`);", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
    create table t1(id int,c1 int,c2 datetime null default current_timestamp on update current_timestamp comment '123');
    alter table t1 modify c2 datetime;`
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` MODIFY COLUMN `c2` datetime ON UPDATE CURRENT_TIMESTAMP COMMENT '123';", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
    create table t1(id int,c1 int,c2 datetime null default current_timestamp on update current_timestamp comment '123');
    alter table t1 modify c2 datetime;`
	res = s.mustRunBackupTran(c, sql)
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
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c4` geometry,ADD COLUMN `c3` geometry,ADD COLUMN `c2` geometry,ADD COLUMN `c1` geometry;", Commentf("%v", res.Rows()))

	sql = `drop table if exists t1;
    create table t1(id int primary key);
    alter table t1 add column c1 geometry;
    alter table t1 add column c2 point;
    alter table t1 add column c3 linestring;
    alter table t1 add column c4 polygon;`
	s.runTranSQL(sql, 10)

	sql = `alter table t1 drop column c1,drop column c2,drop column c3,drop column c4; `
	res = s.mustRunBackupTran(c, sql)
	row = res.Rows()[int(s.tk.Se.AffectedRows())-1]
	backup = s.query("t1", row[7].(string))
	c.Assert(backup, Equals, "ALTER TABLE `test_inc`.`t1` ADD COLUMN `c4` polygon,ADD COLUMN `c3` linestring,ADD COLUMN `c2` point,ADD COLUMN `c1` geometry;", Commentf("%v", res.Rows()))

}

func (s *testSessionIncTranSuite) query(table, opid string) string {
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
			result = append(result, s.trim(str))
		}
	}
	return strings.Join(result, "\n")
}

func (s *testSessionIncTranSuite) queryStatistics() []int {
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

func (s *testSessionIncTranSuite) trim(str string) string {
	if strings.Contains(str, "  ") {
		return s.trim(strings.Replace(str, "  ", " ", -1))
	}
	return str
}

func (s *testSessionIncTranSuite) getSQLMode(c *C) string {
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

	res := s.mustRunBackupTran(c, sql)
	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2, Commentf("%v", res.Rows()))

	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	versionStr := row[5].(string)

	versionStr = strings.SplitN(versionStr, "|", 2)[1]
	value := strings.Replace(versionStr, "'", "", -1)
	value = strings.TrimSpace(value)

	s.sqlMode = value
	return value
}

func (s *testSessionIncTranSuite) getExplicitDefaultsForTimestamp(c *C) bool {
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

	res := s.mustRunBackupTran(c, sql)
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

func (s *testSessionIncTranSuite) TestStatistics(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableSqlStatistic = true

	sql := ""

	sql = `drop table if exists t1;
	create table t1(id int,c1 int);
	alter table t1 drop column c1;
	alter table t1 add column c1 varchar(20);`
	res := s.mustRunBackupTran(c, sql)
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
	res = s.mustRunBackupTran(c, sql)
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

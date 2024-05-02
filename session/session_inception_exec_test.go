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
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/session"
	"github.com/hanchuanchuan/goInception/util"
	"github.com/hanchuanchuan/goInception/util/testkit"
	. "github.com/pingcap/check"
	"golang.org/x/net/context"
)

var _ = Suite(&testSessionIncExecSuite{})

func TestExec(t *testing.T) {
	TestingT(t)
}

type testSessionIncExecSuite struct {
	testCommon
}

func (s *testSessionIncExecSuite) SetUpSuite(c *C) {

	s.initSetUp(c)

	inc := &s.defaultInc

	inc.EnableFingerprint = true
	inc.SqlSafeUpdates = 0
	inc.EnableDropTable = true

	config.GetGlobalConfig().Osc.OscMaxFlowCtl = 1
}

func (s *testSessionIncExecSuite) TearDownSuite(c *C) {
	s.tearDownSuite(c)
}

func (s *testSessionIncExecSuite) TearDownTest(c *C) {
	s.reset()
	s.tearDownTest(c)
}

func (s *testSessionIncExecSuite) testErrorCode(c *C, sql string, errors ...*session.SQLError) [][]interface{} {
	if s.isAPI {
		s.sessionService.LoadOptions(session.SourceOptions{
			Host:           s.defaultInc.BackupHost,
			Port:           int(s.defaultInc.BackupPort),
			User:           s.defaultInc.BackupUser,
			Password:       s.defaultInc.BackupPassword,
			RealRowCount:   s.realRowCount,
			IgnoreWarnings: true,
		})
		s.testExecuteResult(c, sql, errors...)
		return nil
	}

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	res := s.runExec(sql)
	row := res.Rows()[s.getAffectedRows()-1]

	errCode := 0
	if len(errors) > 0 {
		for _, e := range errors {
			if e.Level > 0 {
				if int(e.Level) > errCode {
					errCode = int(e.Level)
				}
			} else {
				level := session.GetErrorLevel(e.Code)
				if int(level) > errCode {
					errCode = int(level)
				}
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
	// 没有错误时允许有警告
	// if errCode == 0 {
	// 	c.Assert(row[2], Not(Equals), "2", Commentf("%v", res.Rows()))
	// } else {
	// 	c.Assert(row[2], Equals, strconv.Itoa(errCode), Commentf("%v", res.Rows()))
	// }
	// 无错误时需要校验结果是否标记为已执行
	if errCode == 0 {
		c.Assert(strings.Contains(row[3].(string), "Execute Successfully"), Equals, true, Commentf("%v", res.Rows()))
		for _, row := range res.Rows() {
			c.Assert(row[2], Not(Equals), "2", Commentf("%v", res.Rows()))
		}
	} else {
		c.Assert(row[2], Equals, strconv.Itoa(errCode), Commentf("%v", res.Rows()))
	}

	return res.Rows()
}

func (s *testSessionIncExecSuite) testExecuteResult(c *C, sql string, errors ...*session.SQLError) {

	result, err := s.sessionService.RunExecute(context.Background(), s.useDB+sql)
	c.Assert(err, IsNil)
	// for _, row := range result {
	// 	if row.ErrLevel == 2 {
	// 		fmt.Println(fmt.Sprintf("sql: %v, err: %v", row.Sql, row.ErrorMessage))
	// 	} else {
	// 		fmt.Println(fmt.Sprintf("[%v] sql: %v", session.StatusList[row.StageStatus], row.Sql))
	// 	}
	// }

	s.records = result

	row := result[len(result)-1]

	errCode := uint8(0)
	if len(errors) > 0 {
		for _, e := range errors {
			level := session.GetErrorLevel(e.Code)
			if level > errCode {
				errCode = level
			}
		}
	}

	if errCode > 0 {
		errMsgs := []string{}
		for _, e := range errors {
			errMsgs = append(errMsgs, e.Error())
		}
		c.Assert(row.ErrorMessage, Equals, strings.Join(errMsgs, "\n"), Commentf("%v", result))
	}

	if errCode == 0 {
		// Execute Successfully
		c.Assert(row.StageStatus, Equals, uint8(2), Commentf("%v", s.records))
		for _, row := range s.records {
			c.Assert(row.ErrLevel, Not(Equals), uint8(2), Commentf("%v", s.records))
		}
	} else {
		c.Assert(row.ErrLevel, Equals, errCode, Commentf("%#v", row))
	}
}

func (s *testSessionIncExecSuite) TestCreateTable(c *C) {

	sql := ""

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	sql = `drop table if exists nullkeytest1;
	create table nullkeytest1(c1 int, c2 int, c3 int, primary key(c1), key ix_1(c2));`
	s.testErrorCode(c, sql)

	// 支持innodb引擎
	config.GetGlobalConfig().Inc.EnableSetEngine = true
	config.GetGlobalConfig().Inc.SupportEngine = "innodb"
	s.mustRunExec(c, "drop table if exists t1;create table t1(c1 varchar(10))engine = innodb;")

	// 允许设置存储引擎
	config.GetGlobalConfig().Inc.EnableSetEngine = true
	config.GetGlobalConfig().Inc.SupportEngine = "innodb"
	s.mustRunExec(c, "drop table if exists t1;create table t1(c1 varchar(10))engine = innodb;")

	if s.DBVersion >= 50600 {
		sql = "drop table if exists t1;create table t1(id int primary key,t1 timestamp default CURRENT_TIMESTAMP,t2 timestamp ON UPDATE CURRENT_TIMESTAMP);"
		if s.explicitDefaultsForTimestamp || !(strings.Contains(s.sqlMode, "TRADITIONAL") ||
			(strings.Contains(s.sqlMode, "STRICT_") && strings.Contains(s.sqlMode, "NO_ZERO_DATE"))) {
			s.testErrorCode(c, sql)
		} else {
			s.testErrorCode(c, sql, session.NewErr(session.ER_INVALID_DEFAULT, "t2"))
		}
	}

	indexMaxLength := 767
	if s.innodbLargePrefix {
		indexMaxLength = 3072
	}

	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"

	s.runExec("drop table if exists t1")
	if s.innodbLargePrefix {
		sql = "create table t1(c1 int,c2 varchar(400),c3 varchar(400),index idx_1(c1,c2))charset utf8;"
		s.testErrorCode(c, sql)
	} else {
		sql = "create table t1(c1 int,c2 varchar(400),c3 varchar(400),index idx_1(c1,c2))charset utf8;"
		s.testErrorCode(c, sql,
			session.NewErr(session.ER_TOO_LONG_KEY, "idx_1", indexMaxLength))
	}

	sql = "drop table if exists t1;create table t1(c1 int,c2 varchar(200),c3 varchar(200),index idx_1(c1,c2))charset utf8;;"
	s.testErrorCode(c, sql)

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

	if s.DBVersion >= 50600 {
		sql = `drop table if exists t1;create table t1(
id int auto_increment comment 'test',
crtTime datetime not null DEFAULT CURRENT_TIMESTAMP comment 'test',
uptTime datetime DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP comment 'test',
primary key(id)) comment 'test';`
		s.testErrorCode(c, sql)
	}

	s.mustRunExec(c, "drop table if exists t1;create table t1(c1 int);")

	// 测试表名大小写
	sql = "insert into T1 values(1);"
	if s.ignoreCase {
		s.testErrorCode(c, sql)
	} else {
		s.testErrorCode(c, sql,
			session.NewErr(session.ER_TABLE_NOT_EXISTED_ERROR, "test_inc.T1"))
	}
}

func (s *testSessionIncExecSuite) TestCreateTableAsSelect(c *C) {
	if !s.enforeGtidConsistency {
		s.mustRunExec(c, "drop table if exists t1,t2;create table t1(c1 int);")
		sql = "create table t2 as select * from t1;"
		s.testErrorCode(c, sql)
	}
}

func (s *testSessionIncExecSuite) TestDropTable(c *C) {

	config.GetGlobalConfig().Inc.EnableDropTable = true

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int);drop table t1;")
}

func (s *testSessionIncExecSuite) TestAlterTableAddColumn(c *C) {

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableDropTable = true

	sql := ""
	sql = "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int;"
	s.testErrorCode(c, sql)

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int);alter table t1 add column c1 int first;alter table t1 add column c2 int after c1;")

	// char列建议
	config.GetGlobalConfig().Inc.MaxCharLength = 100
	s.mustRunExec(c, `drop table if exists t1;create table t1(id int);
        alter table t1 add column c1 char(200);
        alter table t1 add column c2 varchar(200);`)

	config.GetGlobalConfig().Inc.EnableColumnCharset = true
	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb3,utf8mb4"
	sql = "drop table if exists t1;create table t1(id int primary key);alter table t1 add column c1 varchar(10) character set utf8mb3;"
	if s.DBVersion > 50700 {
		s.testErrorCode(c, sql)
	} else {
		s.testErrorCode(c, sql,
			session.NewErr(session.ErrUnknownCharset, "utf8mb3"))
	}

	sql = "drop table if exists t1;create table t1(id int primary key);alter table t1 add column c1 bit default b'0';"
	// pt-osc
	config.GetGlobalConfig().Osc.OscOn = true
	s.mustRunExec(c, sql)

	// gh-ost
	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = true
	s.mustRunExec(c, sql)

	sql = "drop table if exists t1;create table t1 (id int primary key , age int);"
	s.testErrorCode(c, sql)

	sql = "drop table if exists t1;create table t1 (id int primary key);alter table t1 add column (c1 int,c2 varchar(20));"
	s.testErrorCode(c, sql)

	if s.DBVersion >= 50600 {
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

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int);alter table t1 alter column id set default '1';")

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int);alter table t1 alter column id drop default ;alter table t1 alter column id set default '1';")
}

func (s *testSessionIncExecSuite) TestAlterTableModifyColumn(c *C) {

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column c1 int first;")

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 modify column id int after c1;")

}

func (s *testSessionIncExecSuite) TestAlterTableDropColumn(c *C) {

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);alter table t1 drop column c1;")

}

func (s *testSessionIncExecSuite) TestInsert(c *C) {

	config.GetGlobalConfig().Inc.CheckInsertField = false
	config.GetGlobalConfig().IncLevel.ErWithInsertField = 0

	sql := ""

	// 字段警告
	config.GetGlobalConfig().Inc.CheckInsertField = true
	config.GetGlobalConfig().IncLevel.ErWithInsertField = 1
	sql = "drop table if exists t1;create table t1(id int,c1 int);insert into t1 values(1,1);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WITH_INSERT_FIELD))

	config.GetGlobalConfig().Inc.CheckInsertField = false
	config.GetGlobalConfig().IncLevel.ErWithInsertField = 0

	sql = "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id) values();"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WITH_INSERT_VALUES))

	// insert select 表不存在
	sql = ("drop table if exists t1,t2;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t2;")
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_TABLE_NOT_EXISTED_ERROR, "test_inc.t2"))

	// select where
	config.GetGlobalConfig().Inc.CheckDMLWhere = true
	sql = "drop table if exists t1;create table t1(id int,c1 int );insert into t1(id,c1) select 1,null from t1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_NO_WHERE_CONDITION))
	config.GetGlobalConfig().Inc.CheckDMLWhere = false

	// limit
	config.GetGlobalConfig().Inc.CheckDMLLimit = true
	sql = "insert into t1(id,c1) select 1,null from t1 limit 1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WITH_LIMIT_CONDITION))
	config.GetGlobalConfig().Inc.CheckDMLLimit = false

	// order by rand()
	// config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
	sql = "insert into t1(id,c1) select 1,null from t1 order by rand();"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_ORDERY_BY_RAND))

	// 受影响行数
	s.testErrorCode(c, "insert into t1 values(1,1),(2,2);")
	s.testAffectedRows(c, 2)

	s.testErrorCode(c, "insert into t1(id,c1) select 1,null;")
	s.testAffectedRows(c, 1)

	sql = "drop table if exists t1;create table t1(c1 char(100) not null);insert into t1(c1) values(null);"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_BAD_NULL_ERROR, "test_inc.t1.c1", 1))
}

func (s *testSessionIncExecSuite) TestUpdate(c *C) {

	config.GetGlobalConfig().Inc.CheckInsertField = false
	config.GetGlobalConfig().IncLevel.ErWithInsertField = 0
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

	sql = (`drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        update t1 inner join t2 on t1.id=t2.id2  set t1.c1=t2.c1 where c11=1;`)
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t2.id2"),
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "c11"),
	)

	sql = (`drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        update t1,t2 t3 set t1.c1=t2.c3 where t1.id=t3.id;`)
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t2.c3"),
	)

	sql = (`drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        update t1,t2 t3 set t1.c1=t2.c3 where t1.id=t3.id;`)
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_COLUMN_NOT_EXISTED, "t2.c3"),
	)

	// where
	config.GetGlobalConfig().Inc.CheckDMLWhere = true
	sql = "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_NO_WHERE_CONDITION))

	config.GetGlobalConfig().Inc.CheckDMLWhere = false

	// limit
	config.GetGlobalConfig().Inc.CheckDMLLimit = true
	sql = "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1 limit 1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WITH_LIMIT_CONDITION),
	)

	config.GetGlobalConfig().Inc.CheckDMLLimit = false

	// order by rand()
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = true
	sql = ("drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1 order by rand();")
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_WITH_ORDERBY_CONDITION),
	)
	config.GetGlobalConfig().Inc.CheckDMLOrderBy = false

	// 受影响行数
	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);update t1 set c1 = 1;")
	s.testAffectedRows(c, 0)

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int primary key,c1 int);insert into t1 values(1,1),(2,2);update t1 set c1 = 1 where id = 1;")
	s.testAffectedRows(c, 0)

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int primary key,c1 int);insert into t1 values(1,1),(2,2);update t1 set c1 = 10 where id = 1;")
	s.testAffectedRows(c, 1)

	// -------------------- 多表update -------------------
	sql = `drop table if exists table1;drop table if exists table2;
		create table table1(id1 int primary key,c1 int,c2 int);
		create table table2(id2 int primary key,c1 int,c2 int,c22 int);
		insert into table1 values(1,1,1),(2,1,1);
		insert into table2 values(1,1,1,null),(2,null,null,null);
		update table1 t1,table2 t2 set t1.c1=10,t2.c22=20 where t1.id1=t2.id2 and t2.c1=1;`
	s.mustRunExec(c, sql)
	s.testAffectedRows(c, 2)

}

func (s *testSessionIncExecSuite) TestDelete(c *C) {
	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
		config.GetGlobalConfig().IncLevel.ErWithInsertField = 1
	}()

	config.GetGlobalConfig().Inc.CheckInsertField = false
	config.GetGlobalConfig().IncLevel.ErWithInsertField = 0
	sql := ""

	sql = "drop table if exists t1"
	s.testErrorCode(c, sql)

	s.mustRunExec(c, `drop table if exists t1;drop table if exists t2;create table t1(id int primary key,c1 int);
        create table t2(id int primary key,c1 int,c2 int);
        delete t2 from t1 inner join t2 on t1.id=t2.id where t1.c1=1;`)

	// 受影响行数
	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);delete from t1 where id = 1;")
	s.testAffectedRows(c, 0)

	s.mustRunExec(c, "drop table if exists t1;create table t1(id int,c1 int);insert into t1(id) values(1);")
	sql = `delete from t1 where id =1;`
	s.testErrorCode(c, sql)
	s.testAffectedRows(c, 1)

	sql = `drop table if exists t1;create table t1(id int primary key,c1 int);
		delete tmp from t1 as tmp where tmp.id=1;`
	s.testErrorCode(c, sql)

	sql = `drop table if exists t1;create table t1(id int primary key,c1 int);
		delete s1 from t1 as s1 inner join t1 as s2 on s1.c1 = s2.c1 where s1.id > s2.id;`
	s.testErrorCode(c, sql)

	s.mustRunExec(c, "drop table if exists t1,t2;create table t1(id int primary key,c1 int);")
	sql = `create table t2(id int primary key,c1 int);
			delete t1 from t1 inner join t2 where t2.c1 = 1;`
	s.testErrorCode(c, sql)

	s.mustRunExec(c, `drop table if exists t1;CREATE TABLE t1 (
		id bigint(20) AUTO_INCREMENT primary key,
		goods_id bigint(20) unsigned NOT NULL DEFAULT '0' ,
		sku_id bigint(20) unsigned NOT NULL DEFAULT '0'  ,
		bar_code varchar(30) NOT NULL DEFAULT ''
	  );`)
	sql = "delete from t1 where id in (select id from (select ANY_VALUE(id) as id,count(*) as num from t1 group by `goods_id`, `sku_id`, `bar_code`,id having num > 1) as t)"
	s.testErrorCode(c, sql)
}

func (s *testSessionIncExecSuite) TestCreateDataBase(c *C) {
	inc := &config.GetGlobalConfig().Inc
	inc.EnableDropDatabase = true

	sql := ""

	sql = "drop database if exists test1111111111111111111;create database if not exists test1111111111111111111;"
	s.testErrorCode(c, sql)

	dbname := fmt.Sprintf("%s_%d_%s", strings.ReplaceAll(inc.BackupHost, ".", "_"), inc.BackupPort, "test_inc")
	s.testErrorCode(c, fmt.Sprintf("drop database if exists %s;", dbname))

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

	sql = "drop database if exists test1111111111111111111;"
	s.mustRunExec(c, sql)

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
	sql = "drop database if exists test123456;create database test123456 character set latin1;"
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

	s.mustRunExec(c, "drop table if exists t1;drop view if exists v_1;")
	sql = "create table t1(id int primary key);create view v_1 as select * from t1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ErrViewSupport, "v_1"))

	config.GetGlobalConfig().Inc.EnableUseView = true
	sql = "create table t1(id int primary key);create view v_1 as select * from t1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_SELECT_ONLY_STAR))

	s.mustRunExec(c, "drop table if exists t1;drop view if exists v_1;")
	sql = "create table t1(id int primary key);create view v_1 as select id from t1;"
	s.testErrorCode(c, sql)

	s.mustRunExec(c, "drop table if exists t1;drop view if exists v_1;")
	sql = `create table t1(id int primary key,c1 int);
	create view v_1 as select id,c1 from t1 where id = 1 union all select id,c1 from t1 where id = 2;`
	s.testErrorCode(c, sql)
}

func (s *testSessionIncExecSuite) TestAlterTableAddIndex(c *C) {

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
	c.Assert(s.getAffectedRows(), GreaterEqual, 102)

	sql = "inception get variables;"
	s.tk.MustQueryInc(sql)
	c.Assert(s.getAffectedRows(), GreaterEqual, 102)

	sql = "inception show variables like 'backup_password';"
	res := s.tk.MustQueryInc(sql)
	row := res.Rows()[s.getAffectedRows()-1]
	if row[1].(string) != "" {
		c.Assert(row[1].(string)[:1], Equals, "*")
	}
}

func (s *testSessionIncExecSuite) TestShowProcesslist(c *C) {
	sql := ""
	sql = "inception show processlist;"
	s.tk.MustQueryInc(sql)
	// c.Assert(s.getAffectedRows(), GreaterEqual, 1)

	sql = "inception get processlist;"
	s.tk.MustQueryInc(sql)
	// c.Assert(s.getAffectedRows(), GreaterEqual, 1)
}

func (s *testSessionIncExecSuite) TestShowOscProcesslist1(c *C) {
	sm := s.tk.Se.GetSessionManager()

	sqlsha1 := "*EF80A7086FC120D173E95699A9DDB828FCA51111"
	p := &util.OscProcessInfo{
		ID:         1,
		ConnID:     1024,
		Schema:     "test",
		Table:      "test",
		Command:    "select 1",
		Percent:    0,
		RemainTime: "",
		Sqlsha1:    sqlsha1,
		Info:       "",
		IsGhost:    true,
		PanicAbort: make(chan util.ProcessOperation),
		RW:         &sync.RWMutex{},
	}
	sm.AddOscProcess(p)

	sql := ""
	sql = "inc get osc processlist;"
	result := s.tk.MustQueryInc(sql)
	c.Assert(s.getAffectedRows(), GreaterEqual, 1)

	resBuff := bytes.NewBufferString("")
	for _, row := range result.Rows() {
		fmt.Fprintf(resBuff, "%s\n", row)
	}
	c.Assert(resBuff.String(), Equals, "[test test select 1 *EF80A7086FC120D173E95699A9DDB828FCA51111 0  ]\n")

	return

	// --- kill ---
	defer func() {
		if p.PanicAbort != nil {
			close(p.PanicAbort)
		}
	}()

	go func() {
		for oper := range p.PanicAbort {
			sm.OscLock()
			p.RW.Lock()
			switch oper {
			case util.ProcessOperationKill:
				p.Killed = true
			case util.ProcessOperationPause:
				p.Pause = true
			case util.ProcessOperationResume:
				p.Pause = false
			}
			p.RW.Unlock()
			sm.OscUnLock()
		}
	}()

	var err error
	sql = `inc pause osc '%s'`
	_, err = s.tk.ExecInc(fmt.Sprintf(sql, sqlsha1))
	c.Assert(err, IsNil)

	sql = `inc pause osc '123'`
	_, err = s.tk.ExecInc(sql)
	c.Assert(err, NotNil)

	sql = `inc resume osc '%s'`
	_, err = s.tk.ExecInc(fmt.Sprintf(sql, sqlsha1))
	c.Assert(err, IsNil)

	sql = `inc kill osc '%s'`
	_, err = s.tk.ExecInc(fmt.Sprintf(sql, sqlsha1))
	c.Assert(err, IsNil)

	sql = `inc kill osc '%s'`
	_, err = s.tk.ExecInc(fmt.Sprintf(sql, sqlsha1))
	c.Assert(err, NotNil)

	sql = `inc kill osc '123'`
	_, err = s.tk.ExecInc(sql)
	c.Assert(err, NotNil)
}

func (s *testSessionIncExecSuite) TestSetVariables(c *C) {
	sql := ""
	sql = "inception show variables;"
	s.tk.MustQueryInc(sql)
	c.Assert(s.getAffectedRows(), GreaterEqual, 102)

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

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableDropTable = true
	sql := ""

	sql = "drop table if exists t1;create table t1(id int auto_increment primary key,c1 int);"
	s.mustRunExec(c, sql)

	// 删除后添加列
	sql = "alter table t1 drop column c1;alter table t1 add column c1 varchar(20);"
	s.testErrorCode(c, sql)

	sql = "alter table t1 drop column c1,add column c1 varchar(20);"
	s.testErrorCode(c, sql)

	// 删除后添加索引
	sql = "drop table if exists t1;create table t1(id int primary key,c1 int,key ix(c1));"
	s.mustRunExec(c, sql)

	sql = "alter table t1 drop index ix;alter table t1 add index ix(c1);"
	s.testErrorCode(c, sql)

	sql = "alter table t1 drop index ix,add index ix(c1);"
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column c2 varchar(20) comment '!@#$%^&*()_+[]{}\\|;:\",.<>/?';"
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column c3 varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";"
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column `c4` varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";"
	s.testErrorCode(c, sql)

}

func (s *testSessionIncExecSuite) TestTableCharsetCollation(c *C) {
	sql := ""

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableSetCollation = true
	config.GetGlobalConfig().Inc.EnableSetCharset = true
	config.GetGlobalConfig().Inc.EnableColumnCharset = true

	config.GetGlobalConfig().Inc.SupportCharset = "utf8,utf8mb4"
	config.GetGlobalConfig().Inc.SupportCollation = "utf8_bin,utf8mb4_bin"
	if s.DBVersion >= 50700 {
		sql = `create table t1(id int primary key,c1 varchar(20),white_list VARCHAR (16000)) character set utf8mb4 COLLATE utf8mb4_bin;`
		s.testErrorCode(c, sql)
	} else {
		sql = `create table t1(id int primary key,c1 varchar(20),white_list VARCHAR (20000)) character set utf8 COLLATE utf8_bin;`
		s.testErrorCode(c, sql)
	}
}

func (s *testSessionIncExecSuite) TestAlterTablePtOSC(c *C) {
	saved := config.GetGlobalConfig().Inc
	savedOsc := config.GetGlobalConfig().Osc
	defer func() {
		config.GetGlobalConfig().Inc = saved
		config.GetGlobalConfig().Osc = savedOsc
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableDropTable = true
	config.GetGlobalConfig().Osc.OscOn = true
	config.GetGlobalConfig().Ghost.GhostOn = false
	config.GetGlobalConfig().Osc.OscMinTableSize = 0

	sql := "drop table if exists t1;create table t1(id int auto_increment primary key,c1 int);"
	s.mustRunExec(c, sql)

	// 删除后添加列
	sql = `# 这是一条注释
		alter table t1 drop column c1;alter table t1 add column c1 varchar(20);`
	s.testErrorCode(c, sql)

	sql = `/* 这是一条注释 */
		alter table t1 drop column c1,add column c1 varchar(20);`
	s.testErrorCode(c, sql)

	sql = `-- 这是一条注释
	alter table t1 add column c2 varchar(20) comment '!@#$%^&*()_+[]{}\\|;:",.<>/?';`
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column c3 varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";"
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column `c4` varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";"
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column `c5` varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";  -- 测试注释"
	s.testErrorCode(c, sql)
}

func (s *testSessionIncExecSuite) TestAlterTableGhost(c *C) {
	saved := config.GetGlobalConfig().Inc
	savedOsc := config.GetGlobalConfig().Osc
	defer func() {
		config.GetGlobalConfig().Inc = saved
		config.GetGlobalConfig().Osc = savedOsc
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableDropTable = true
	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = true
	config.GetGlobalConfig().Osc.OscMinTableSize = 0

	sql := "drop table if exists t1;create table t1(id int auto_increment primary key,c1 int);"
	s.mustRunExec(c, sql)

	// 删除后添加列
	sql = `# 这是一条注释
	alter table t1 drop column c1;alter table t1 add column c1 varchar(20);`
	s.testErrorCode(c, sql)

	sql = `/* 这是一条注释 */
	alter table t1 drop column c1,add column c1 varchar(20);`
	s.testErrorCode(c, sql)

	sql = `-- 这是一条注释
	alter table t1 add column c2 varchar(20) comment '!@#$%^&*()_+[]{}\\|;:",.<>/?';`
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column c3 varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";"
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column `c4` varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";"
	s.testErrorCode(c, sql)

	sql = "alter table t1 add column `c5` varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";  -- 测试注释"
	s.testErrorCode(c, sql)

	config.GetGlobalConfig().Ghost.GhostBinDir = "/usr/local/bin"

	sql = "alter table t1 add column `c6` varchar(20) comment \"!@#$%^&*()_+[]{}\\|;:',.<>/?\";  -- 测试注释"
	s.testErrorCode(c, sql)
}

// 测试忽略指定的Alter语句
func (s *testSessionIncExecSuite) TestIgnoreOscAlterStmt(c *C) {
	saved := config.GetGlobalConfig().Inc
	savedOsc := config.GetGlobalConfig().Osc
	defer func() {
		config.GetGlobalConfig().Inc = saved
		config.GetGlobalConfig().Osc = savedOsc
	}()

	config.GetGlobalConfig().Inc.IgnoreOscAlterStmt = "drop index,drop partition"
	config.GetGlobalConfig().Osc.OscMinTableSize = 0
	config.GetGlobalConfig().Osc.OscOn = false
	config.GetGlobalConfig().Ghost.GhostOn = true

	sql := "drop table if exists t1;create table t1(id int auto_increment primary key, c1 int, key ix(c1));"
	s.mustRunExec(c, sql)

	// drop index
	sql = "alter table t1 drop index ix;"
	s.testErrorCode(c, sql)

	sql = "drop table if exists t1;create table t1(id int auto_increment primary key, c1 int, key ix(c1));"
	s.mustRunExec(c, sql)

	// drop index,add index
	sql = "alter table t1 drop index ix,add index ix2(c1);"
	s.testErrorCode(c, sql)
}

// TestDisplayWidth 测试列指定长度参数
func (s *testSessionIncExecSuite) TestDisplayWidth(c *C) {
	sql := ""
	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableEnumSetBit = true

	s.mustRunExec(c, "drop table if exists t1;")
	// 数据类型 警告
	sql = `create table t1(c1 bit(64),
	c2 tinyint(255),
	c3 smallint(255),
	c4 mediumint(255),
	c5 int(255),
	c6 bigint(255) );`
	s.testErrorCode(c, sql)

	// 数据类型 警告
	sql = `alter table t1 add column c11 tinyint(255);`
	s.testErrorCode(c, sql)

}

func (s *testSessionIncExecSuite) TestWhereCondition(c *C) {
	sql := ""
	s.mustRunExec(c, "drop table if exists t1,t2;create table t1(id int);")

	sql = "update t1 set id = 1 where 123;"
	s.mustRunExec(c, sql)

	sql = "update t1 set id = 1 where null;"
	s.mustRunExec(c, sql)

	sql = `
	delete from t1 where 1+2;
	delete from t1 where 1-2;
	delete from t1 where 1*2;
	delete from t1 where 1/2;
	update t1 set id = 1 where 1&2;
	update t1 set id = 1 where 1|2;
	update t1 set id = 1 where 1^2;
	update t1 set id = 1 where 1 div 2;
	`
	s.mustRunExec(c, sql)

	sql = `
	update t1 set id = 1 where 1+2=3;
	update t1 set id = 1 where id is null;
	update t1 set id = 1 where id is not null;
	update t1 set id = 1 where 1=1;
	update t1 set id = 1 where id in (1,2);
	update t1 set id = 1 where id is true;
	`
	s.mustRunExec(c, sql)
}

func (s *testSessionIncExecSuite) TestExecAnyStatement(c *C) {
	sql := ""
	config.GetGlobalConfig().Inc.EnableAnyStatement = true
	config.GetGlobalConfig().Inc.EnableDropTable = false

	sql = "drop table if exists t1;create table t1(id int);drop table t1;"
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_CANT_DROP_TABLE, "t1"))

	s.mustRunExec(c, "set global binlog_format = ROW;")
	s.mustRunExec(c, "create user test1@'127.0.0.1' identified by '123';")
	s.mustRunExec(c, "drop user test1@'127.0.0.1';")
}

func (s *testSessionIncExecSuite) TestMaxExecutionTime(c *C) {
	saved := config.GetGlobalConfig().Inc
	savedOsc := config.GetGlobalConfig().Osc
	defer func() {
		config.GetGlobalConfig().Inc = saved
		config.GetGlobalConfig().Osc = savedOsc
	}()

	config.GetGlobalConfig().Inc.CheckColumnComment = false
	config.GetGlobalConfig().Inc.CheckTableComment = false
	config.GetGlobalConfig().Inc.EnableDropTable = true
	config.GetGlobalConfig().Osc.OscOn = true
	config.GetGlobalConfig().Ghost.GhostOn = false
	config.GetGlobalConfig().Osc.OscMinTableSize = 0
	config.GetGlobalConfig().Inc.MaxExecutionTime = 1

	sql := "drop table if exists t1;create table t1(id int auto_increment primary key,c1 int);"
	s.mustRunExec(c, sql)

	// 删除后添加列
	sql = `# 这是一条注释
		alter table t1 drop column c1;alter table t1 add column c1 varchar(20);`
	s.testErrorCode(c, sql)
}

func (s *testSessionIncExecSuite) TestPartition(c *C) {
	config.GetGlobalConfig().Inc.EnablePartitionTable = false
	config.GetGlobalConfig().IncLevel.ErPartitionNotAllowed = 2

	s.mustRunExec(c, "drop table if exists test_partition;")
	sql = `create table test_partition(
		id int unsigned not null auto_increment comment 'id',
		a varchar(10) comment 'a',
		buss_day varchar(8) comment 'day',
		primary key (id,buss_day)
	) comment '分区表'
	partition by list columns (buss_day) (
		partition p1 values in('20200101'),partition p2 values in('20200102'))`
	s.testErrorCode(c, sql,
		session.NewErr(session.ER_PARTITION_NOT_ALLOWED).SetLevel(2))

	//	----------- enable partition -----------
	config.GetGlobalConfig().Inc.EnablePartitionTable = true
	s.mustRunExec(c, "drop table if exists test_partition;")
	s.testErrorCode(c, sql)

	sql = `alter table test_partition add partition (
			partition p4 values in('20200104'),
			partition p5 values in('20200105'),
			partition p6 values in('20200106')
			);`
	s.testErrorCode(c, sql)

	sql = `alter table test_partition drop partition p5,p6;`
	s.testErrorCode(c, sql)

	sql = `alter table test_partition drop partition p5,p6;`
	s.testErrorCode(c, sql,
		session.NewErr(session.ErrPartitionNotExisted, "p5"),
		session.NewErr(session.ErrPartitionNotExisted, "p6"))

	sql = `CREATE TABLE my_range_datetime(
    id INT,
    hiredate DATETIME
)
PARTITION BY RANGE (TO_DAYS(hiredate) ) (
    PARTITION p1 VALUES LESS THAN ( TO_DAYS('20171202') ),
    PARTITION p2 VALUES LESS THAN ( TO_DAYS('20171203') ),
    PARTITION p3 VALUES LESS THAN ( TO_DAYS('20171204') ),
    PARTITION p4 VALUES LESS THAN ( TO_DAYS('20171205') ),
    PARTITION p5 VALUES LESS THAN ( TO_DAYS('20171206') ),
    PARTITION p6 VALUES LESS THAN ( TO_DAYS('20171207') ),
    PARTITION p7 VALUES LESS THAN ( TO_DAYS('20171208') ),
    PARTITION p8 VALUES LESS THAN ( TO_DAYS('20171209') ),
    PARTITION p9 VALUES LESS THAN ( TO_DAYS('20171210') ),
    PARTITION p10 VALUES LESS THAN ( TO_DAYS('20171211') ),
    PARTITION p11 VALUES LESS THAN (MAXVALUE)
);`
	s.testErrorCode(c, sql)

	sql = `create table t_list(a int(11),b int(11))
		partition by list (b)(
		partition p0 values in (1,3,5,7,9),
		partition p1 values in (2,4,6,8,0));`
	s.testErrorCode(c, sql)

	sql = `CREATE TABLE my_member (
		id INT NOT NULL,
		fname VARCHAR(30),
		lname VARCHAR(30),
		created DATE NOT NULL DEFAULT '1970-01-01',
		separated DATE NOT NULL DEFAULT '9999-12-31',
		job_code INT,
		store_id INT
	)
	PARTITION BY HASH(id)
	PARTITIONS 4;`
	s.testErrorCode(c, sql)

	sql = `CREATE TABLE my_members (
		id INT NOT NULL,
		fname VARCHAR(30),
		lname VARCHAR(30),
		hired DATE NOT NULL DEFAULT '1970-01-01',
		separated DATE NOT NULL DEFAULT '9999-12-31',
		job_code INT,
		store_id INT
	)
	PARTITION BY LINEAR HASH( id )
	PARTITIONS 4;`
	s.mustRunExec(c, sql)

	sql = `CREATE TABLE k1 (
		id INT NOT NULL PRIMARY KEY,
		name VARCHAR(20)
	)
	PARTITION BY KEY()
	PARTITIONS 2;`
	s.mustRunExec(c, sql)

	sql = `CREATE TABLE tm1 (
		s1 CHAR(32)
	)
	PARTITION BY KEY(s1)
	PARTITIONS 10;`
	s.mustRunExec(c, sql)

	sql = `CREATE TABLE customer_login_log (
		customer_id int(10) unsigned NOT NULL COMMENT '登录用户ID',
		login_time DATETIME NOT NULL COMMENT '用户登录时间',
		login_ip int(10) unsigned NOT NULL COMMENT '登录IP',
		login_type tinyint(4) NOT NULL COMMENT '登录类型:0未成功 1成功'
	  ) ENGINE=InnoDB
	  PARTITION BY RANGE (YEAR(login_time))(
	  PARTITION p0 VALUES LESS THAN (2017),
	  PARTITION p1 VALUES LESS THAN (2018),
	  PARTITION p2 VALUES LESS THAN (2019)
	  );
	  CREATE TABLE arch_customer_login_log (
        customer_id INT unsigned NOT NULL COMMENT '登录用户ID',
        login_time DATETIME NOT NULL COMMENT '用户登录时间',
        login_ip INT unsigned NOT NULL COMMENT '登录IP',
        login_type TINYINT NOT NULL COMMENT '登录类型:0未成功 1成功'
      ) ENGINE=InnoDB ;`
	s.mustRunExec(c, sql)
	sql = `ALTER TABLE customer_login_log
		  exchange PARTITION p1 WITH TABLE arch_customer_login_log;`
	s.mustRunExec(c, sql)

	s.mustRunExec(c, `drop table if exists t1;`)
	sql = `create table t1(id int,c1 datetime,bill_date datetime,primary key(id,bill_date));`
	s.mustRunExec(c, sql)

	// sql = `alter table t1 partition by range (to_days(c1)) (
	// 	partition p202201 values less than (to_days('2022-02-01')) ENGINE = InnoDB,
	// 	partition p202202 values less than (to_days('2022-03-01')) ENGINE = InnoDB
	// 	);`
	// s.testErrorCode(c, sql,
	// 	session.NewErrf("Table '%s' is already a partitioned table.", "t1"))

	sql = `alter table t1 partition by range (to_days(bill_date)) (
		partition p202201 values less than (to_days('2022-02-01')) ENGINE = InnoDB,
		partition p202202 values less than (to_days('2022-03-01')) ENGINE = InnoDB
		);`
	s.testErrorCode(c, sql)

	sql = `alter table t1 remove partitioning;`
	s.testErrorCode(c, sql)

	sql = `drop table if exists t1;CREATE TABLE t1 (
			customer_id int(10) unsigned NOT NULL COMMENT '登录用户ID',
			bill_date DATETIME NOT NULL COMMENT '用户登录时间',
			login_ip int(10) unsigned NOT NULL COMMENT '登录IP',
			login_type tinyint(4) NOT NULL COMMENT '登录类型:0未成功 1成功'
		  ) ENGINE=InnoDB
		  PARTITION BY RANGE (YEAR(bill_date))(
		  PARTITION p0 VALUES LESS THAN (2017),
		  PARTITION p1 VALUES LESS THAN (2018));		  `
	s.mustRunExec(c, sql)

	sql = `alter table t1 partition by range (to_days(bill_date)) (
			partition p202201 values less than (to_days('2022-02-01')) ENGINE = InnoDB,
			partition p202202 values less than (to_days('2022-03-01')) ENGINE = InnoDB
			);`
	s.testErrorCode(c, sql)

	sql = `alter table t1 remove partitioning;`
	s.testErrorCode(c, sql)
}

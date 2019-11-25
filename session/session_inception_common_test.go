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
	"errors"
	"fmt"
	"path"
	"runtime"
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
	log "github.com/sirupsen/logrus"
)

var _ = Suite(&testCommon{})
var sql string

func TestCommonTest(t *testing.T) {
	TestingT(t)
}

type testCommon struct {
	cluster   *mocktikv.Cluster
	mvccStore mocktikv.MVCCStore
	store     kv.Storage
	dom       *domain.Domain
	tk        *testkit.TestKit
	db        *gorm.DB
	dbAddr    string

	DBVersion         int
	sqlMode           string
	innodbLargePrefix bool
	// 时间戳类型是否需要明确指定默认值
	explicitDefaultsForTimestamp bool
	// 是否忽略大小写(lower_case_table_names为1和2时忽略,否则不忽略)
	ignoreCase bool

	realRowCount bool
}

func (s *testCommon) initSetUp(c *C) {

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

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	cfg := config.GetGlobalConfig()
	_, localFile, _, _ := runtime.Caller(0)
	localFile = path.Dir(localFile)
	configFile := path.Join(localFile[0:len(localFile)-len("session")], "config/config.toml.example")
	c.Assert(cfg.Load(configFile), IsNil)

	inc := &config.GetGlobalConfig().Inc

	inc.BackupHost = "127.0.0.1"
	inc.BackupPort = 3306
	inc.BackupUser = "test"
	inc.BackupPassword = "test"

	inc.Lang = "en-US"
	inc.EnableFingerprint = true
	inc.SqlSafeUpdates = 0
	inc.EnableDropTable = true

	session.SetLanguage("en-US")

	c.Assert(s.mysqlServerVersion(), IsNil)
	c.Assert(s.sqlMode, Not(Equals), "")
	// log.Infof("%#v", s)
	fmt.Println("数据库版本: ", s.DBVersion)
}

func (s *testCommon) tearDownSuite(c *C) {
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

func (s *testCommon) tearDownTest(c *C) {
	if testing.Short() {
		c.Skip("skipping test; in TRAVIS mode")
	}

	if s.tk == nil {
		s.tk = testkit.NewTestKitWithInit(c, s.store)
	}

	saved := config.GetGlobalConfig().Inc
	defer func() {
		config.GetGlobalConfig().Inc = saved
	}()

	config.GetGlobalConfig().Inc.EnableDropTable = true
	session.CheckAuditSetting(config.GetGlobalConfig())

	res := s.runCheck("show tables")
	c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2)

	row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
	sql := row[5]

	exec := `/*%s;--execute=1;--backup=0;--enable-ignore-warnings;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	for _, name := range strings.Split(sql.(string), "\n") {
		if strings.HasPrefix(name, "show tables") {
			continue
		}
		n := strings.Replace(name, "'", "", -1)
		res := s.tk.MustQueryInc(fmt.Sprintf(exec, s.getAddr(), "drop table `"+n+"`"))
		// fmt.Println(res.Rows())
		c.Assert(int(s.tk.Se.AffectedRows()), Equals, 2)
		row := res.Rows()[int(s.tk.Se.AffectedRows())-1]
		c.Assert(row[2], Equals, "0", Commentf("%v", row))
		c.Assert(row[3], Equals, "Execute Successfully", Commentf("%v", row))
		// c.Assert(err, check.IsNil, check.Commentf("sql:%s, %v, error stack %v", sql, args, errors.ErrorStack(err)))
		// fmt.Println(row[4])
		// c.Assert(row[4].(string), IsNil)
	}

}

func (s *testCommon) runCheck(sql string) *testkit.Result {

	tk := s.tk
	session.CheckAuditSetting(config.GetGlobalConfig())

	a := `/*%s;--check=1;--backup=0;--enable-ignore-warnings;real_row_count=%v;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	return tk.MustQueryInc(fmt.Sprintf(a, s.getAddr(), s.realRowCount, sql))
}

func (s *testCommon) runExec(sql string) *testkit.Result {
	tk := s.tk
	session.CheckAuditSetting(config.GetGlobalConfig())

	a := `/*%s;--execute=1;--backup=0;--enable-ignore-warnings;real_row_count=%v;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	return tk.MustQueryInc(fmt.Sprintf(a, s.getAddr(), s.realRowCount, sql))
}

func (s *testCommon) runBackup(sql string) *testkit.Result {
	a := `/*%s;--execute=1;--backup=1;--enable-ignore-warnings;real_row_count=%v;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	return s.tk.MustQueryInc(fmt.Sprintf(a, s.getAddr(), s.realRowCount, sql))
}

func (s *testCommon) mustRunBackup(c *C, sql string) *testkit.Result {
	a := `/*%s;--execute=1;--backup=1;--enable-ignore-warnings;real_row_count=%v;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	res := s.tk.MustQueryInc(fmt.Sprintf(a, s.getAddr(), s.realRowCount, sql))

	// 需要成功执行
	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2", Commentf("%v", row))
	}

	return res
}

func (s *testCommon) mustRunExec(c *C, sql string) *testkit.Result {
	config.GetGlobalConfig().Inc.EnableDropTable = true
	session.CheckAuditSetting(config.GetGlobalConfig())
	a := `/*%s;--execute=1;--backup=0;--enable-ignore-warnings;real_row_count=%v;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	res := s.tk.MustQueryInc(fmt.Sprintf(a, s.getAddr(), s.realRowCount, sql))

	for _, row := range res.Rows() {
		c.Assert(row[2], Not(Equals), "2", Commentf("%v", row))
	}

	return res
}

func (s *testCommon) getAddr() string {
	if s.dbAddr != "" {
		return s.dbAddr
	}
	inc := config.GetGlobalConfig().Inc
	s.dbAddr = fmt.Sprintf(`--host=%s;--port=%d;--user=%s;--password=%s;`,
		inc.BackupHost, inc.BackupPort, inc.BackupUser, inc.BackupPassword)
	return s.dbAddr
}

func (s *testCommon) mysqlServerVersion() error {
	inc := config.GetGlobalConfig().Inc
	if s.db == nil || s.db.DB().Ping() != nil {
		addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql?charset=utf8mb4&parseTime=True&loc=Local&maxAllowedPacket=4194304",
			inc.BackupUser, inc.BackupPassword, inc.BackupHost, inc.BackupPort)

		db, err := gorm.Open("mysql", addr)
		if err != nil {
			return err
		}
		// 禁用日志记录器，不显示任何日志
		db.LogMode(false)
		s.db = db
	}

	var name, value string
	sql := `show variables where Variable_name in 
		('explicit_defaults_for_timestamp','innodb_large_prefix','version','sql_mode','lower_case_table_names');`
	rows, err := s.db.Raw(sql).Rows()
	if err != nil {
		return err
	}
	emptyInnodbLargePrefix := true
	for rows.Next() {
		rows.Scan(&name, &value)

		switch name {
		case "version":
			versionStr := strings.Split(value, "-")[0]
			versionSeg := strings.Split(versionStr, ".")
			if len(versionSeg) == 3 {
				versionStr = fmt.Sprintf("%s%02s%02s", versionSeg[0], versionSeg[1], versionSeg[2])
				version, err := strconv.Atoi(versionStr)
				if err != nil {
					return err
				}
				s.DBVersion = version
			} else {
				return errors.New(fmt.Sprintf("无法解析版本号:%s", value))
			}
			log.Debug("db version: ", s.DBVersion)
		// case "innodb_large_prefix":
		// 	emptyInnodbLargePrefix = false
		// 	s.innodbLargePrefix = value == "ON"
		case "sql_mode":
			s.sqlMode = value
		case "lower_case_table_names":
			if v, err := strconv.Atoi(value); err != nil {
				return err
			} else {
				s.ignoreCase = v > 0
			}
		case "explicit_defaults_for_timestamp":
			if value == "ON" {
				s.explicitDefaultsForTimestamp = true
			}
		}
	}

	// 如果没有innodb_large_prefix系统变量
	if emptyInnodbLargePrefix {
		if s.DBVersion > 50700 {
			s.innodbLargePrefix = true
		} else {
			s.innodbLargePrefix = false
		}
	}
	return nil
}

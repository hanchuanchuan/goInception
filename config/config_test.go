// Copyright 2017 PingCAP, Inc.
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

package config

import (
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/hanchuanchuan/goInception/mysql"
	. "github.com/pingcap/check"
)

var _ = Suite(&testConfigSuite{})

type testConfigSuite struct{}

func TestT(t *testing.T) {
	CustomVerboseFlag = true
	TestingT(t)
}

func (s *testConfigSuite) TestConfig(c *C) {
	conf := new(Config)
	conf.Binlog.BinlogSocket = "/tmp/socket"
	conf.Binlog.IgnoreError = true
	conf.TiKVClient.CommitTimeout = "10s"

	configFile := "config.test.toml"
	_, localFile, _, _ := runtime.Caller(0)
	configFile = path.Join(path.Dir(localFile), configFile)

	f, err := os.Create(configFile)
	c.Assert(err, IsNil)
	_, err = f.WriteString(`
skip_grant_table = true
store = "mocktikv"
run-ddl = true
lease = "0s"
split-table = true
token-limit = 1000
oom-action = "log"
mem-quota-query = 34359738368
enable-streaming = false
compatible-kill-query = false
lower-case-table-names = 2

[log]
slow-query-file = ""
slow-threshold = 300
expensive-threshold = 10000
query-log-max-len = 2048

[security]
skip_grant_table = true
ssl-ca = ""
ssl-cert = ""
ssl-key = ""
cluster-ssl-ca = ""
cluster-ssl-cert = ""
cluster-ssl-key = ""

[status]
report-status = false
status-port = 10080
metrics-addr = ""
metrics-interval = 15

[performance]
max-procs = 0
stmt-count-limit = 5000
tcp-keep-alive = false
cross-join = true
stats-lease = "0s"
run-auto-analyze = false
feedback-probability = 0.0
query-feedback-limit = 0
pseudo-estimate-ratio = 0.8
force-priority = "NO_PRIORITY"

[proxy-protocol]
networks = ""
header-timeout = 5

[prepared-plan-cache]
enabled = false
capacity = 100

[tikv-client]
grpc-connection-count = 16
grpc-keepalive-time = 10
grpc-keepalive-timeout = 3
commit-timeout = "41s"

[txn-local-latches]
enabled = true
capacity = 2048000

[binlog]
write-timeout = "15s"
ignore-error = false

`)
	c.Assert(err, IsNil)
	c.Assert(f.Sync(), IsNil)

	c.Assert(conf.Load(configFile), IsNil)

	// 加载不包含的项时,原始值保持不变
	c.Assert(conf.Binlog.BinlogSocket, Equals, "/tmp/socket")

	c.Assert(f.Close(), IsNil)
	c.Assert(os.Remove(configFile), IsNil)

	f, err = os.Create(configFile)
	c.Assert(err, IsNil)
	_, err = f.WriteString(`
[binlog]
binlog-socket = ""
`)
	c.Assert(err, IsNil)
	c.Assert(f.Sync(), IsNil)
	c.Assert(conf.Load(configFile), IsNil)

	// 值在config file指定时,会替换掉内存值
	c.Assert(conf.Binlog.BinlogSocket, Equals, "")

	c.Assert(conf.TiKVClient.CommitTimeout, Equals, "41s")
	c.Assert(f.Close(), IsNil)
	c.Assert(os.Remove(configFile), IsNil)

	configFile = path.Join(path.Dir(localFile), "config.toml.default")
	c.Assert(conf.Load(configFile), IsNil)

	conf.Inc.Version = strings.TrimRight(mysql.TiDBReleaseVersion, "-dirty")

	// fmt.Println(conf)
	// fmt.Println(GetGlobalConfig())
	// Make sure the example config is the same as default config.
	c.Assert(conf, DeepEquals, GetGlobalConfig())
}

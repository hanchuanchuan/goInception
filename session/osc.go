// Copyright 2013 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

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

// +build !windows,!nacl,!plan9

package session

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/format"
	"github.com/hanchuanchuan/goInception/util"
	"github.com/hanchuanchuan/goInception/util/auth"

	"github.com/hanchuanchuan/gh-ost/go/base"
	"github.com/hanchuanchuan/gh-ost/go/logic"
	log "github.com/sirupsen/logrus"
)

var (
	oscConnID uint32
)

type ChanOscData struct {
	out string
	p   *util.OscProcessInfo
}

// Copying `test`.`t1`:  99% 00:00 remain
// 匹配osc执行进度
var regOscPercent *regexp.Regexp = regexp.MustCompile(`^Copying .*? (\d+)% (\d+:\d+) remain`)
var regGhostPercent *regexp.Regexp = regexp.MustCompile(`^Copy:.*?(\d+).\d+%;.*?ETA: (.*)?`)

func (s *session) checkAlterUseOsc(t *TableInfo) {
	if (s.Osc.OscOn || s.Ghost.GhostOn) && (s.Osc.OscMinTableSize == 0 || t.TableSize >= s.Osc.OscMinTableSize) {
		s.myRecord.useOsc = true
	} else {
		s.myRecord.useOsc = false
	}
}

func (s *session) mysqlComputeSqlSha1(r *Record) {
	if !r.useOsc || r.Sqlsha1 != "" {
		return
	}

	buf := bytes.NewBufferString(r.DBName)

	buf.WriteString(s.opt.password)
	buf.WriteString(s.opt.host)
	buf.WriteString(s.opt.user)
	buf.WriteString(strconv.Itoa(s.opt.port))
	buf.WriteString(strconv.Itoa(r.SeqNo))
	buf.WriteString(r.Sql)

	r.Sqlsha1 = auth.EncodePassword(buf.String())
}

func (s *session) mysqlExecuteAlterTableOsc(r *Record) {

	err := os.Setenv("PATH", fmt.Sprintf("%s%s%s",
		s.Osc.OscBinDir, string(os.PathListSeparator), os.Getenv("PATH")))
	if err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
		return
	}

	if _, err := exec.LookPath("pt-online-schema-change"); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
		return
	}

	buf := bytes.NewBufferString("pt-online-schema-change")

	buf.WriteString(" --alter \"")
	buf.WriteString(s.getAlterTablePostPart(r.Sql, true))

	if s.hasError() {
		return
	}

	buf.WriteString("\" ")
	if s.Osc.OscPrintSql {
		buf.WriteString(" --print ")
	}
	buf.WriteString(" --charset=utf8 ")
	buf.WriteString(" --chunk-time ")
	buf.WriteString(fmt.Sprintf("%g ", s.Osc.OscChunkTime))

	buf.WriteString(" --critical-load ")
	buf.WriteString("Threads_connected:")
	buf.WriteString(strconv.Itoa(s.Osc.OscCriticalThreadConnected))
	buf.WriteString(",Threads_running:")
	buf.WriteString(strconv.Itoa(s.Osc.OscCriticalThreadRunning))
	buf.WriteString(" ")

	buf.WriteString(" --max-load ")
	buf.WriteString("Threads_connected:")
	buf.WriteString(strconv.Itoa(s.Osc.OscMaxThreadConnected))
	buf.WriteString(",Threads_running:")
	buf.WriteString(strconv.Itoa(s.Osc.OscMaxThreadRunning))
	buf.WriteString(" ")

	buf.WriteString(" --recurse=1 ")

	buf.WriteString(" --check-interval ")
	buf.WriteString(fmt.Sprintf("%d ", s.Osc.OscCheckInterval))

	if !s.Osc.OscDropNewTable {
		buf.WriteString(" --no-drop-new-table ")
	}

	if !s.Osc.OscDropOldTable {
		buf.WriteString(" --no-drop-old-table ")
	}

	if !s.Osc.OscCheckReplicationFilters {
		buf.WriteString(" --no-check-replication-filters ")
	}

	if !s.Osc.OscCheckUniqueKeyChange {
		buf.WriteString(" --no-check-unique-key-change ")
	}

	if !s.Osc.OscCheckAlter {
		buf.WriteString(" --no-check-alter ")
	}

	buf.WriteString(" --alter-foreign-keys-method=")
	buf.WriteString(s.Osc.OscAlterForeignKeysMethod)

	if s.Osc.OscAlterForeignKeysMethod == "none" {
		buf.WriteString(" --force ")
	}

	buf.WriteString(" --execute ")
	buf.WriteString(" --statistics ")
	buf.WriteString(" --max-lag=")
	buf.WriteString(fmt.Sprintf("%d", s.Osc.OscMaxLag))

	if s.IsClusterNode && s.DBVersion > 50600 && s.Osc.OscMaxFlowCtl >= 0 {
		buf.WriteString(" --max-flow-ctl=")
		buf.WriteString(fmt.Sprintf("%d", s.Osc.OscMaxFlowCtl))
	}

	buf.WriteString(" --no-version-check ")
	buf.WriteString(" --recursion-method=")
	buf.WriteString(s.Osc.OscRecursionMethod)

	buf.WriteString(" --progress ")
	buf.WriteString("percentage,1 ")

	buf.WriteString(" --user=\"")
	buf.WriteString(s.opt.user)
	buf.WriteString("\" --password='")
	buf.WriteString(strings.Replace(s.opt.password, "'", "'\"'\"'", -1))
	buf.WriteString("' --host=")
	buf.WriteString(s.opt.host)
	buf.WriteString(" --port=")
	buf.WriteString(strconv.Itoa(s.opt.port))

	buf.WriteString(" D=")
	buf.WriteString(r.TableInfo.Schema)
	buf.WriteString(",t='")
	buf.WriteString(r.TableInfo.Name)
	buf.WriteString("'")

	str := buf.String()

	s.execCommand(r, "sh", []string{"-c", str})

}

func (s *session) mysqlExecuteAlterTableGhost(r *Record) {
	migrationContext := base.NewMigrationContext()
	// flag.StringVar(&migrationContext.InspectorConnectionConfig.Key.Hostname, "host", "127.0.0.1", "MySQL hostname (preferably a replica, not the master)")
	migrationContext.InspectorConnectionConfig.Key.Hostname = s.opt.host
	// flag.StringVar(&migrationContext.AssumeMasterHostname, "assume-master-host", "", "(optional) explicitly tell gh-ost the identity of the master. Format: some.host.com[:port] This is useful in master-master setups where you wish to pick an explicit master, or in a tungsten-replicator where gh-ost is unable to determine the master")

	// RDS数据库需要做特殊处理
	if s.Ghost.GhostAliyunRds && s.Ghost.GhostAssumeMasterHost == "" {
		migrationContext.AssumeMasterHostname = fmt.Sprintf("%s:%d", s.opt.host, s.opt.port)
	} else {
		migrationContext.AssumeMasterHostname = s.Ghost.GhostAssumeMasterHost
	}

	log.Debug("assume_master_host: ", migrationContext.AssumeMasterHostname)
	// flag.IntVar(&migrationContext.InspectorConnectionConfig.Key.Port, "port", 3306, "MySQL port (preferably a replica, not the master)")
	migrationContext.InspectorConnectionConfig.Key.Port = s.opt.port
	// flag.StringVar(&migrationContext.CliUser, "user", "", "MySQL user")
	migrationContext.CliUser = s.opt.user
	// flag.StringVar(&migrationContext.CliPassword, "password", "", "MySQL password")
	migrationContext.CliPassword = s.opt.password
	// flag.StringVar(&migrationContext.CliMasterUser, "master-user", "", "MySQL user on master, if different from that on replica. Requires --assume-master-host")
	// flag.StringVar(&migrationContext.CliMasterPassword, "master-password", "", "MySQL password on master, if different from that on replica. Requires --assume-master-host")
	// flag.StringVar(&migrationContext.ConfigFile, "conf", "", "Config file")

	migrationContext.UseTLS = false
	// migrationContext.TLSAllowInsecure = s.GhostSslAllowInsecure
	// flag.BoolVar(&migrationContext.UseTLS, "ssl", false, "Enable SSL encrypted connections to MySQL hosts")
	// flag.StringVar(&migrationContext.TLSCACertificate, "ssl-ca", "", "CA certificate in PEM format for TLS connections to MySQL hosts. Requires --ssl")
	// flag.BoolVar(&migrationContext.TLSAllowInsecure, "ssl-allow-insecure", false, "Skips verification of MySQL hosts' certificate chain and host name. Requires --ssl")

	migrationContext.DatabaseName = r.TableInfo.Schema
	// flag.StringVar(&migrationContext.DatabaseName, "database", "", "database name (mandatory)")
	migrationContext.OriginalTableName = r.TableInfo.Name

	migrationContext.AlterStatement = s.getAlterTablePostPart(r.Sql, false)

	// flag.StringVar(&migrationContext.OriginalTableName, "table", "", "table name (mandatory)")
	// flag.StringVar(&migrationContext.AlterStatement, "alter", "", "alter statement (mandatory)")
	migrationContext.CountTableRows = s.Ghost.GhostExactRowcount
	// flag.BoolVar(&migrationContext.CountTableRows, "exact-rowcount", false, "actually count table rows as opposed to estimate them (results in more accurate progress estimation)")
	migrationContext.ConcurrentCountTableRows = s.Ghost.GhostConcurrentRowcount
	// flag.BoolVar(&migrationContext.ConcurrentCountTableRows, "concurrent-rowcount", true, "(with --exact-rowcount), when true (default): count rows after row-copy begins, concurrently, and adjust row estimate later on; when false: first count rows, then start row copy")
	migrationContext.AllowedRunningOnMaster = s.Ghost.GhostAllowOnMaster
	// flag.BoolVar(&migrationContext.AllowedRunningOnMaster, "allow-on-master", false, "allow this migration to run directly on master. Preferably it would run on a replica")
	migrationContext.AllowedMasterMaster = s.Ghost.GhostAllowMasterMaster
	// flag.BoolVar(&migrationContext.AllowedMasterMaster, "allow-master-master", false, "explicitly allow running in a master-master setup")
	migrationContext.NullableUniqueKeyAllowed = s.Ghost.GhostAllowNullableUniqueKey
	// flag.BoolVar(&migrationContext.NullableUniqueKeyAllowed, "allow-nullable-unique-key", false, "allow gh-ost to migrate based on a unique key with nullable columns. As long as no NULL values exist, this should be OK. If NULL values exist in chosen key, data may be corrupted. Use at your own risk!")
	migrationContext.ApproveRenamedColumns = s.Ghost.GhostApproveRenamedColumns
	// flag.BoolVar(&migrationContext.ApproveRenamedColumns, "approve-renamed-columns", false, "in case your `ALTER` statement renames columns, gh-ost will note that and offer its interpretation of the rename. By default gh-ost does not proceed to execute. This flag approves that gh-ost's interpretation is correct")
	migrationContext.SkipRenamedColumns = false
	// flag.BoolVar(&migrationContext.SkipRenamedColumns, "skip-renamed-columns", false, "in case your `ALTER` statement renames columns, gh-ost will note that and offer its interpretation of the rename. By default gh-ost does not proceed to execute. This flag tells gh-ost to skip the renamed columns, i.e. to treat what gh-ost thinks are renamed columns as unrelated columns. NOTE: you may lose column data")
	migrationContext.IsTungsten = s.Ghost.GhostTungsten
	// flag.BoolVar(&migrationContext.IsTungsten, "tungsten", false, "explicitly let gh-ost know that you are running on a tungsten-replication based topology (you are likely to also provide --assume-master-host)")
	migrationContext.DiscardForeignKeys = s.Ghost.GhostDiscardForeignKeys
	// flag.BoolVar(&migrationContext.DiscardForeignKeys, "discard-foreign-keys", false, "DANGER! This flag will migrate a table that has foreign keys and will NOT create foreign keys on the ghost table, thus your altered table will have NO foreign keys. This is useful for intentional dropping of foreign keys")
	migrationContext.SkipForeignKeyChecks = s.Ghost.GhostSkipForeignKeyChecks
	// flag.BoolVar(&migrationContext.SkipForeignKeyChecks, "skip-foreign-key-checks", false, "set to 'true' when you know for certain there are no foreign keys on your table, and wish to skip the time it takes for gh-ost to verify that")
	migrationContext.AliyunRDS = s.Ghost.GhostAliyunRds
	// flag.BoolVar(&migrationContext.AliyunRDS, "aliyun-rds", false, "set to 'true' when you execute on Aliyun RDS.")
	migrationContext.GoogleCloudPlatform = s.Ghost.GhostGcp
	// flag.BoolVar(&migrationContext.GoogleCloudPlatform, "gcp", false, "set to 'true' when you execute on a 1st generation Google Cloud Platform (GCP).")

	// executeFlag := flag.Bool("execute", false, "actually execute the alter & migrate the table. Default is noop: do some tests and exit")
	migrationContext.TestOnReplica = false
	// flag.BoolVar(&migrationContext.TestOnReplica, "test-on-replica", false, "Have the migration run on a replica, not on the master. At the end of migration replication is stopped, and tables are swapped and immediately swap-revert. Replication remains stopped and you can compare the two tables for building trust")
	migrationContext.TestOnReplicaSkipReplicaStop = false
	// flag.BoolVar(&migrationContext.TestOnReplicaSkipReplicaStop, "test-on-replica-skip-replica-stop", false, "When --test-on-replica is enabled, do not issue commands stop replication (requires --test-on-replica)")
	migrationContext.MigrateOnReplica = false
	// flag.BoolVar(&migrationContext.MigrateOnReplica, "migrate-on-replica", false, "Have the migration run on a replica, not on the master. This will do the full migration on the replica including cut-over (as opposed to --test-on-replica)")

	migrationContext.OkToDropTable = s.Ghost.GhostOkToDropTable
	// flag.BoolVar(&migrationContext.OkToDropTable, "ok-to-drop-table", false, "Shall the tool drop the old table at end of operation. DROPping tables can be a long locking operation, which is why I'm not doing it by default. I'm an online tool, yes?")
	migrationContext.InitiallyDropOldTable = s.Ghost.GhostInitiallyDropOldTable
	// flag.BoolVar(&migrationContext.InitiallyDropOldTable, "initially-drop-old-table", false, "Drop a possibly existing OLD table (remains from a previous run?) before beginning operation. Default is to panic and abort if such table exists")
	migrationContext.InitiallyDropGhostTable = s.Ghost.GhostInitiallyDropGhostTable
	// flag.BoolVar(&migrationContext.InitiallyDropGhostTable, "initially-drop-ghost-table", false, "Drop a possibly existing Ghost table (remains from a previous run?) before beginning operation. Default is to panic and abort if such table exists")
	migrationContext.TimestampOldTable = false
	// flag.BoolVar(&migrationContext.TimestampOldTable, "timestamp-old-table", false, "Use a timestamp in old table name. This makes old table names unique and non conflicting cross migrations")
	cutOver := s.Ghost.GhostCutOver
	// cutOver := flag.String("cut-over", "atomic", "choose cut-over type (default|atomic, two-step)")
	migrationContext.ForceNamedCutOverCommand = s.Ghost.GhostForceNamedCutOver
	// flag.BoolVar(&migrationContext.ForceNamedCutOverCommand, "force-named-cut-over", false, "When true, the 'unpostpone|cut-over' interactive command must name the migrated table")
	migrationContext.SwitchToRowBinlogFormat = false
	// flag.BoolVar(&migrationContext.SwitchToRowBinlogFormat, "switch-to-rbr", false, "let this tool automatically switch binary log format to 'ROW' on the replica, if needed. The format will NOT be switched back. I'm too scared to do that, and wish to protect you if you happen to execute another migration while this one is running")
	migrationContext.AssumeRBR = s.Ghost.GhostAssumeRbr
	// flag.BoolVar(&migrationContext.AssumeRBR, "assume-rbr", false, "set to 'true' when you know for certain your server uses 'ROW' binlog_format. gh-ost is unable to tell, event after reading binlog_format, whether the replication process does indeed use 'ROW', and restarts replication to be certain RBR setting is applied. Such operation requires SUPER privileges which you might not have. Setting this flag avoids restarting replication and you can proceed to use gh-ost without SUPER privileges")
	migrationContext.CutOverExponentialBackoff = s.Ghost.GhostCutOverExponentialBackoff
	// flag.BoolVar(&migrationContext.CutOverExponentialBackoff, "cut-over-exponential-backoff", false, "Wait exponentially longer intervals between failed cut-over attempts. Wait intervals obey a maximum configurable with 'exponential-backoff-max-interval').")
	exponentialBackoffMaxInterval := s.Ghost.GhostExponentialBackoffMaxInterval
	// exponentialBackoffMaxInterval := flag.Int64("exponential-backoff-max-interval", 64, "Maximum number of seconds to wait between attempts when performing various operations with exponential backoff.")
	chunkSize := s.Ghost.GhostChunkSize
	// chunkSize := flag.Int64("chunk-size", 1000, "amount of rows to handle in each iteration (allowed range: 100-100,000)")
	dmlBatchSize := s.Ghost.GhostDmlBatchSize
	// dmlBatchSize := flag.Int64("dml-batch-size", 10, "batch size for DML events to apply in a single transaction (range 1-100)")
	defaultRetries := s.Ghost.GhostDefaultRetries
	// defaultRetries := flag.Int64("default-retries", 60, "Default number of retries for various operations before panicking")
	cutOverLockTimeoutSeconds := s.Ghost.GhostCutOverLockTimeoutSeconds
	// cutOverLockTimeoutSeconds := flag.Int64("cut-over-lock-timeout-seconds", 3, "Max number of seconds to hold locks on tables while attempting to cut-over (retry attempted when lock exceeds timeout)")
	niceRatio := s.Ghost.GhostNiceRatio
	// niceRatio := flag.Float64("nice-ratio", 0, "force being 'nice', imply sleep time per chunk time; range: [0.0..100.0]. Example values: 0 is aggressive. 1: for every 1ms spent copying rows, sleep additional 1ms (effectively doubling runtime); 0.7: for every 10ms spend in a rowcopy chunk, spend 7ms sleeping immediately after")
	maxLagMillis := s.Ghost.GhostMaxLagMillis
	// maxLagMillis := flag.Int64("max-lag-millis", 1500, "replication lag at which to throttle operation")
	replicationLagQuery := s.Ghost.GhostReplicationLagQuery
	// replicationLagQuery := flag.String("replication-lag-query", "", "Deprecated. gh-ost uses an internal, subsecond resolution query")
	throttleControlReplicas := s.Ghost.GhostThrottleControlReplicas
	// throttleControlReplicas := flag.String("throttle-control-replicas", "", "List of replicas on which to check for lag; comma delimited. Example: myhost1.com:3306,myhost2.com,myhost3.com:3307")
	throttleQuery := s.Ghost.GhostThrottleQuery
	// throttleQuery := flag.String("throttle-query", "", "when given, issued (every second) to check if operation should throttle. Expecting to return zero for no-throttle, >0 for throttle. Query is issued on the migrated server. Make sure this query is lightweight")
	throttleHTTP := s.Ghost.GhostThrottleHTTP
	// throttleHTTP := flag.String("throttle-http", "", "when given, gh-ost checks given URL via HEAD request; any response code other than 200 (OK) causes throttling; make sure it has low latency response")
	heartbeatIntervalMillis := s.Ghost.GhostHeartbeatIntervalMillis
	// heartbeatIntervalMillis := flag.Int64("heartbeat-interval-millis", 100, "how frequently would gh-ost inject a heartbeat value")
	migrationContext.ThrottleFlagFile = s.Ghost.GhostThrottleFlagFile
	// flag.StringVar(&migrationContext.ThrottleFlagFile, "throttle-flag-file", "", "operation pauses when this file exists; hint: use a file that is specific to the table being altered")
	migrationContext.ThrottleAdditionalFlagFile = s.Ghost.GhostThrottleAdditionalFlagFile
	// flag.StringVar(&migrationContext.ThrottleAdditionalFlagFile, "throttle-additional-flag-file", "/tmp/gh-ost.throttle", "operation pauses when this file exists; hint: keep default, use for throttling multiple gh-ost operations")
	migrationContext.PostponeCutOverFlagFile = s.Ghost.GhostPostponeCutOverFlagFile
	// flag.StringVar(&migrationContext.PostponeCutOverFlagFile, "postpone-cut-over-flag-file", "", "while this file exists, migration will postpone the final stage of swapping tables, and will keep on syncing the ghost table. Cut-over/swapping would be ready to perform the moment the file is deleted.")
	// migrationContext.PanicFlagFile = s.Ghost.GhostPanicFlagFile
	// flag.StringVar(&migrationContext.PanicFlagFile, "panic-flag-file", "", "when this file is created, gh-ost will immediately terminate, without cleanup")
	migrationContext.DropServeSocket = s.Ghost.GhostInitiallyDropSocketFile
	// flag.BoolVar(&migrationContext.DropServeSocket, "initially-drop-socket-file", false, "Should gh-ost forcibly delete an existing socket file. Be careful: this might drop the socket file of a running migration!")
	// migrationContext.ServeSocketFile = s.Ghost.GhostServeSocketFile
	migrationContext.ServeSocketFile = ""
	// flag.StringVar(&migrationContext.ServeSocketFile, "serve-socket-file", "", "Unix socket file to serve on. Default: auto-determined and advertised upon startup")
	// migrationContext.DropServeSocket = s.Ghost.GhostServeTcpPort
	// flag.Int64Var(&migrationContext.ServeTCPPort, "serve-tcp-port", 0, "TCP port to serve on. Default: disabled")

	// flag.StringVar(&migrationContext.HooksPath, "hooks-path", "", "directory where hook files are found (default: empty, ie. hooks disabled). Hook files found on this path, and conforming to hook naming conventions will be executed")
	// flag.StringVar(&migrationContext.HooksHintMessage, "hooks-hint", "", "arbitrary message to be injected to hooks via GH_OST_HOOKS_HINT, for your convenience")
	// flag.StringVar(&migrationContext.HooksHintOwner, "hooks-hint-owner", "", "arbitrary name of owner to be injected to hooks via GH_OST_HOOKS_HINT_OWNER, for your convenience")
	// flag.StringVar(&migrationContext.HooksHintToken, "hooks-hint-token", "", "arbitrary token to be injected to hooks via GH_OST_HOOKS_HINT_TOKEN, for your convenience")

	migrationContext.ReplicaServerId = 2000100000 + uint(s.sessionVars.ConnectionID%10000)
	// flag.UintVar(&migrationContext.ReplicaServerId, "replica-server-id", 99999, "server id used by gh-ost process. Default: 99999")
	// maxLoad := s.Ghost.GhostMaxLoad
	// maxLoad := flag.String("max-load", "", "Comma delimited status-name=threshold. e.g: 'Threads_running=100,Threads_connected=500'. When status exceeds threshold, app throttles writes")
	// criticalLoad := s.Ghost.GhostCriticalLoad
	// criticalLoad := flag.String("critical-load", "", "Comma delimited status-name=threshold, same format as --max-load. When status exceeds threshold, app panics and quits")
	criticalLoad := fmt.Sprintf("Threads_running=%d,Threads_connected=%d",
		s.Osc.OscCriticalThreadRunning, s.Osc.OscCriticalThreadConnected)

	maxLoad := fmt.Sprintf("Threads_running=%d,Threads_connected=%d",
		s.Osc.OscMaxThreadRunning, s.Osc.OscMaxThreadConnected)
	// buf.WriteString(" --critical-load ")
	// buf.WriteString("Threads_connected:")
	// buf.WriteString(strconv.Itoa(s.Osc.OscCriticalThreadConnected))
	// buf.WriteString(",Threads_running:")
	// buf.WriteString(strconv.Itoa(s.Osc.OscCriticalThreadRunning))
	// buf.WriteString(" ")

	// buf.WriteString(" --max-load ")
	// buf.WriteString("Threads_connected:")
	// buf.WriteString(strconv.Itoa(s.Osc.OscMaxThreadConnected))
	// buf.WriteString(",Threads_running:")
	// buf.WriteString(strconv.Itoa(s.Osc.OscMaxThreadRunning))
	// buf.WriteString(" ")

	migrationContext.CriticalLoadIntervalMilliseconds = s.Ghost.GhostCriticalLoadIntervalMillis
	// flag.Int64Var(&migrationContext.CriticalLoadIntervalMilliseconds, "critical-load-interval-millis", 0, "When 0, migration immediately bails out upon meeting critical-load. When non-zero, a second check is done after given interval, and migration only bails out if 2nd check still meets critical load")
	migrationContext.CriticalLoadHibernateSeconds = s.Ghost.GhostCriticalLoadHibernateSeconds
	// flag.Int64Var(&migrationContext.CriticalLoadHibernateSeconds, "critical-load-hibernate-seconds", 0, "When nonzero, critical-load does not panic and bail out; instead, gh-ost goes into hibernate for the specified duration. It will not read/write anything to from/to any server")
	// quiet := false
	// verbose := false
	// debug := false
	// stack := false
	// help := false
	// version := false
	migrationContext.ForceTmpTableName = s.Ghost.GhostForceTableNames
	// flag.StringVar(&migrationContext.ForceTmpTableName, "force-table-names", "", "table name prefix to be used on the temporary tables")
	// flag.CommandLine.SetOutput(os.Stdout)

	// if *verbose {
	// 	log.SetLevel(log.INFO)
	// }
	// if *stack {
	// 	log.SetPrintStackTrace(stack)
	// }

	// if migrationContext.DatabaseName == "" {
	// 	log.Fatalf("--database must be provided and database name must not be empty")
	// }
	// if migrationContext.OriginalTableName == "" {
	// 	log.Fatalf("--table must be provided and table name must not be empty")
	// }
	// if migrationContext.AlterStatement == "" {
	// 	log.Fatalf("--alter must be provided and statement must not be empty")
	// }
	migrationContext.Noop = false
	if migrationContext.AllowedRunningOnMaster && migrationContext.TestOnReplica {
		log.Error("--allow-on-master and --test-on-replica are mutually exclusive")
		s.AppendErrorMessage("--allow-on-master and --test-on-replica are mutually exclusive")
	}
	if migrationContext.AllowedRunningOnMaster && migrationContext.MigrateOnReplica {
		log.Error("--allow-on-master and --migrate-on-replica are mutually exclusive")
		s.AppendErrorMessage("--allow-on-master and --migrate-on-replica are mutually exclusive")
	}
	if migrationContext.MigrateOnReplica && migrationContext.TestOnReplica {
		log.Error("--migrate-on-replica and --test-on-replica are mutually exclusive")
		s.AppendErrorMessage("--migrate-on-replica and --test-on-replica are mutually exclusive")
	}
	if migrationContext.SwitchToRowBinlogFormat && migrationContext.AssumeRBR {
		log.Error("--switch-to-rbr and --assume-rbr are mutually exclusive")
		s.AppendErrorMessage("--switch-to-rbr and --assume-rbr are mutually exclusive")
	}
	if migrationContext.TestOnReplicaSkipReplicaStop {
		if !migrationContext.TestOnReplica {
			log.Error("--test-on-replica-skip-replica-stop requires --test-on-replica to be enabled")
			s.AppendErrorMessage("--test-on-replica-skip-replica-stop requires --test-on-replica to be enabled")
		}
		log.Warning("--test-on-replica-skip-replica-stop enabled. We will not stop replication before cut-over. Ensure you have a plugin that does this.")
	}
	if migrationContext.CliMasterUser != "" && migrationContext.AssumeMasterHostname == "" {
		log.Error("--master-user requires --assume-master-host")
		s.AppendErrorMessage("--master-user requires --assume-master-host")
	}
	if migrationContext.CliMasterPassword != "" && migrationContext.AssumeMasterHostname == "" {
		log.Error("--master-password requires --assume-master-host")
		s.AppendErrorMessage("--master-password requires --assume-master-host")
	}
	if migrationContext.TLSCACertificate != "" && !migrationContext.UseTLS {
		log.Error("--ssl-ca requires --ssl")
		s.AppendErrorMessage("--ssl-ca requires --ssl")
	}
	if migrationContext.TLSAllowInsecure && !migrationContext.UseTLS {
		log.Error("--ssl-allow-insecure requires --ssl")
		s.AppendErrorMessage("--ssl-allow-insecure requires --ssl")
	}
	if replicationLagQuery != "" {
		log.Warningf("--replication-lag-query is deprecated")
	}

	switch cutOver {
	case "atomic", "default", "":
		migrationContext.CutOverType = base.CutOverAtomic
	case "two-step":
		migrationContext.CutOverType = base.CutOverTwoStep
	default:
		log.Errorf("Unknown cut-over: %s", cutOver)
		s.AppendErrorMessage(fmt.Sprintf("Unknown cut-over: %s", cutOver))
	}
	if err := migrationContext.ReadConfigFile(); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
	}
	if err := migrationContext.ReadThrottleControlReplicaKeys(throttleControlReplicas); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
	}
	if err := migrationContext.ReadMaxLoad(maxLoad); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
	}
	if err := migrationContext.ReadCriticalLoad(criticalLoad); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
	}
	if migrationContext.ServeSocketFile == "" {
		// unix socket file max 104 characters (or 107)
		socketFile := fmt.Sprintf("/tmp/gh-ost.%s.%d.%s.%s.sock", s.opt.host, s.opt.port,
			migrationContext.DatabaseName, migrationContext.OriginalTableName)
		if len(socketFile) > 100 {
			// 字符串过长时转换为hash值
			host := truncateString(s.opt.host, 30)
			dbName := truncateString(migrationContext.DatabaseName, 30)
			tableName := truncateString(migrationContext.OriginalTableName, 30)

			socketFile = fmt.Sprintf("/tmp/gh-ost.%s.%d.%s.%s.sock", host, s.opt.port,
				dbName, tableName)

			if len(socketFile) > 100 {
				socketFile = fmt.Sprintf("/tmp/gh%s%d%s%s.sock", host, s.opt.port,
					dbName, tableName)
			}

		}
		migrationContext.ServeSocketFile = socketFile
	}

	migrationContext.SetHeartbeatIntervalMilliseconds(heartbeatIntervalMillis)
	migrationContext.SetNiceRatio(niceRatio)
	migrationContext.SetChunkSize(chunkSize)
	migrationContext.SetDMLBatchSize(dmlBatchSize)
	migrationContext.SetMaxLagMillisecondsThrottleThreshold(maxLagMillis)
	migrationContext.SetThrottleQuery(throttleQuery)
	migrationContext.SetThrottleHTTP(throttleHTTP)
	migrationContext.SetDefaultNumRetries(defaultRetries)
	migrationContext.ApplyCredentials()
	if err := migrationContext.SetupTLS(); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
	}
	if err := migrationContext.SetCutOverLockTimeoutSeconds(cutOverLockTimeoutSeconds); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
	}
	if err := migrationContext.SetExponentialBackoffMaxInterval(exponentialBackoffMaxInterval); err != nil {
		log.Error(err)
		s.AppendErrorMessage(err.Error())
	}

	if s.hasError() {
		return
	}

	p := &util.OscProcessInfo{
		ID:         uint64(atomic.AddUint32(&oscConnID, 1)),
		ConnID:     s.sessionVars.ConnectionID,
		Schema:     r.TableInfo.Schema,
		Table:      r.TableInfo.Name,
		Command:    r.Sql,
		Sqlsha1:    r.Sqlsha1,
		Percent:    0,
		RemainTime: "",
		Info:       "",
		IsGhost:    true,
	}
	s.sessionManager.AddOscProcess(p)

	// defer func() {
	// 	// 执行完成或中止后清理osc进程信息
	// 	pl := s.sessionManager.ShowOscProcessList()
	// 	delete(pl, p.Sqlsha1)
	// }()

	done := false
	buf := bytes.NewBufferString("")
	migrator := logic.NewMigrator(migrationContext)

	//实时循环读取输出流中的一行内容
	f := func(reader *bufio.Reader) {
		statusTick := time.Tick(100 * time.Millisecond)
		for range statusTick {
			for {
				line, err2 := reader.ReadString('\n')
				if err2 != nil || io.EOF == err2 || line == "" {
					break
				}
				buf.WriteString(line)
				buf.WriteString("\n")

				s.mysqlAnalyzeGhostOutput(line, p)
				if p.Killed {
					migrationContext.PanicAbort <- fmt.Errorf("Execute has been abort in percent: %d, remain time: %s",
						p.Percent, p.RemainTime)

					done = true
				} else if p.Pause && migrationContext.ThrottleCommandedByUser == 0 {

					atomic.StoreInt64(&migrationContext.ThrottleCommandedByUser, 1)
				} else if !p.Pause && migrationContext.ThrottleCommandedByUser == 1 {

					atomic.StoreInt64(&migrationContext.ThrottleCommandedByUser, 0)
				}
			}
			if done {
				break
			}
		}
	}

	// log.Debugf("%#v", migrationContext)

	migrator.Log = bytes.NewBufferString("")

	go f(bufio.NewReader(migrator.Log))

	// if config.GetGlobalConfig().Log.Level == "debug" ||
	// 	config.GetGlobalConfig().Log.Level == "info" {
	// 	ghostlog.SetLevel(ghostlog.INFO)
	// } else {
	// 	ghostlog.SetLevel(ghostlog.ERROR)
	// }

	// ghostlog.SetOutput(log.StandardLogger().Out)
	// ghostlog.SetLevel(ghostlog.DEBUG)

	if err := migrator.Migrate(); err != nil {
		log.Error(err)
		done = true
		s.AppendErrorMessage(err.Error())
	}

	done = true

	if s.hasError() {
		r.StageStatus = StatusExecFail
	} else {
		r.StageStatus = StatusExecOK
		r.ExecComplete = true
	}

	if s.hasError() || s.Osc.OscPrintNone {
		r.Buf.WriteString(buf.String())
		r.Buf.WriteString("\n")
	}
}

func (s *session) execCommand(r *Record, commandName string, params []string) bool {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command(commandName, params...)

	// log.Infof("%s %s", commandName, params)

	//StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.AppendErrorMessage(err.Error())
		log.Error(err)
		return false
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.AppendErrorMessage(err.Error())
		log.Error(err)
		return false
	}

	// 保证关闭输出流
	defer stdout.Close()
	defer stderr.Close()

	// 运行命令
	if err := cmd.Start(); err != nil {
		s.AppendErrorMessage(err.Error())
		log.Error(err)
		return false
	}

	p := &util.OscProcessInfo{
		ID:         uint64(atomic.AddUint32(&oscConnID, 1)),
		ConnID:     s.sessionVars.ConnectionID,
		Schema:     r.TableInfo.Schema,
		Table:      r.TableInfo.Name,
		Command:    r.Sql,
		Sqlsha1:    r.Sqlsha1,
		Percent:    0,
		RemainTime: "",
		Info:       "",
	}
	s.sessionManager.AddOscProcess(p)

	var wg sync.WaitGroup
	wg.Add(2)

	// 消息
	reader := bufio.NewReader(stdout)
	// 进度
	reader2 := bufio.NewReader(stderr)

	buf := bytes.NewBufferString("")

	//实时循环读取输出流中的一行内容
	f := func(reader *bufio.Reader) {
		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				wg.Done()
				break
			}
			buf.WriteString(line)
			buf.WriteString("\n")
			s.mysqlAnalyzeOscOutput(line, p)
			if p.Killed {
				if err := cmd.Process.Kill(); err != nil {
					s.AppendErrorMessage(err.Error())
				} else {
					s.AppendErrorMessage(fmt.Sprintf("Execute has been abort in percent: %d, remain time: %s",
						p.Percent, p.RemainTime))
				}
			}
		}
	}

	go f(reader)
	go f(reader2)

	wg.Wait()

	//阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	err = cmd.Wait()
	if err != nil {
		s.AppendErrorMessage(err.Error())
		log.Errorf("%s %s", commandName, params)
		log.Error(err)
	}
	if p.Percent < 100 || s.hasError() {
		s.recordSets.MaxLevel = 2
		r.ErrLevel = 2
		r.StageStatus = StatusExecFail
	} else {
		r.StageStatus = StatusExecOK
		r.ExecComplete = true
	}

	if p.Percent < 100 || s.Osc.OscPrintNone {
		r.Buf.WriteString(buf.String())
		r.Buf.WriteString("\n")
	}

	// 执行完成或中止后清理osc进程信息
	// pl := s.sessionManager.ShowOscProcessList()
	// delete(pl, p.Sqlsha1)

	return true
}

func (s *session) mysqlAnalyzeOscOutput(out string, p *util.OscProcessInfo) {
	firsts := regOscPercent.FindStringSubmatch(out)
	// log.Info(p.Killed)
	if len(firsts) < 3 {
		if strings.HasPrefix(out, "Successfully altered") {

			p.Percent = 100
			p.RemainTime = ""
			p.Info = strings.TrimSpace(out)
		}
		return
	}

	pct, _ := strconv.Atoi(firsts[1])
	remain := firsts[2]
	p.Info = strings.TrimSpace(out)

	p.Percent = pct
	p.RemainTime = remain

}

func (s *session) mysqlAnalyzeGhostOutput(out string, p *util.OscProcessInfo) {
	firsts := regGhostPercent.FindStringSubmatch(out)
	// log.Info(p.Killed)
	if len(firsts) < 3 {
		// if strings.HasPrefix(out, "Successfully altered") {
		// 	p.Percent = 100
		// 	p.RemainTime = ""
		// 	p.Info = strings.TrimSpace(out)
		// }
		return
	}

	pct, _ := strconv.Atoi(firsts[1])
	remain := firsts[2]
	if remain == "due" {
		remain = ""
	}
	p.Percent = pct
	p.RemainTime = remain
	p.Info = strings.TrimSpace(out)

}

func (s *session) getAlterTablePostPart(sql string, isPtOSC bool) string {

	sql = strings.Replace(sql, "\\", "\\\\", -1)
	part, ok := s.getAlterPartSql(sql)
	if !ok {
		return ""
	}

	// 解析后的语句长度不能小于解析前!
	// sqlParts := strings.Fields(sql)
	// if len(sqlParts) >= 4 {
	// 	newSql := strings.Join(sqlParts[3:], " ")
	// 	if len(part) < len(newSql) &&
	// 		!strings.Contains(part, "UNIQUE") {
	// 		log.Errorf("origin sql: %s", sql)
	// 		log.Errorf("parsed after: %s", part)
	// 		s.AppendErrorMessage("alter子句解析失败,请联系作者或自行调整!")
	// 		return ""
	// 	}
	// }

	// gh-ost不需要处理`,pt-osc需要处理
	if isPtOSC {
		part = strings.Replace(part, "\"", "\\\"", -1)
		part = strings.Replace(part, "`", "\\`", -1)
		part = strings.Replace(part, "$", "\\$", -1)
	}

	return part

	// var buf []string
	// for _, line := range strings.Split(sql, "\n") {
	// 	line = strings.TrimSpace(line)
	// 	if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-- ") || strings.HasPrefix(line, "/*") {
	// 		continue
	// 	}
	// 	buf = append(buf, line)
	// }
	// sql = strings.Join(buf, "\n")
	// index := strings.Index(strings.ToUpper(sql), "ALTER")
	// if index == -1 {
	// 	s.AppendErrorMessage("无效alter语句!")
	// 	return sql
	// }

	// sql = sql[index:]

	// parts := strings.Fields(sql)
	// if len(parts) < 4 {
	// 	s.AppendErrorMessage("无效alter语句!")
	// 	return sql
	// }

	// supportOper := []string{
	// 	"add",
	// 	"algorithm",
	// 	"alter",
	// 	"auto_increment",
	// 	"avg_row_length",
	// 	"change",
	// 	"character",
	// 	"checksum",
	// 	"comment",
	// 	"convert",
	// 	"collate",
	// 	"default",
	// 	"disable",
	// 	"discard",
	// 	"drop",
	// 	"enable",
	// 	"engine",
	// 	"force",
	// 	"import",
	// 	"modify",
	// 	"rename", // rename index
	// }

	// support := false
	// for _, p := range supportOper {
	// 	// if strings.ToLower(parts[3]) == p {
	// 	if strings.HasPrefix(strings.ToLower(parts[3]), p) {
	// 		support = true
	// 		break
	// 	}
	// }
	// if !support {
	// 	s.AppendErrorMessage(fmt.Sprintf("不支持的osc操作!(%s)", sql))
	// 	return sql
	// }

	// sql = strings.Join(parts[3:], " ")

	// sql = strings.Replace(sql, "\"", "\\\"", -1)

	// // gh-ost不需要处理`,pt-osc需要处理
	// if isPtOSC {
	// 	sql = strings.Replace(sql, "`", "\\`", -1)
	// }

	// return sql
}

// getAlterPartSql 获取alter子句部分
func (s *session) getAlterPartSql(sql string) (string, bool) {
	// sql = strings.Replace(sql, "\n", " ", -1)
	// sql = strings.Replace(sql, "\r", " ", -1)

	charsetInfo, collation := s.sessionVars.GetCharsetInfo()
	stmtNodes, _, err := s.parser.Parse(sql, charsetInfo, collation)
	if err != nil {
		s.AppendErrorMessage(err.Error())
		return "", false
	}
	var builder strings.Builder
	var columns []string

	if len(stmtNodes) == 0 {
		s.AppendErrorMessage(fmt.Sprintf("未正确解析ALTER语句: %s", sql))
		log.Error(fmt.Sprintf("未正确解析ALTER语句: %s", sql))
		return "", false
	}

	for _, stmtNode := range stmtNodes {

		switch node := stmtNode.(type) {
		case *ast.AlterTableStmt:
			for _, alter := range node.Specs {
				builder.Reset()
				err = alter.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &builder))
				if err != nil {
					s.AppendErrorMessage(err.Error())
					return "", false
				}
				restoreSQL := builder.String()
				columns = append(columns, restoreSQL)
			}
		default:
			s.AppendErrorMessage(fmt.Sprintf("无效类型: %v", stmtNode))
			return "", false
		}
	}

	if len(columns) > 0 {
		return strings.Join(columns, ", "), true
	}

	s.AppendErrorMessage(fmt.Sprintf("未正确解析SQL: %s", sql))
	return "", false
}

// truncateString: 根据指定长度做字符串截断,长度溢出时转换为hash值以避免重复
func truncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}
	v := md5.Sum([]byte(str))
	if length < 32 {
		return hex.EncodeToString(v[:])[:length]
	}
	return hex.EncodeToString(v[:])
}

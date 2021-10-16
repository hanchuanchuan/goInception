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
var regOscPercent = regexp.MustCompile(`^Copying .*? (\d+)% (\d+:\d+|\d+:\d+:\d+) remain`)
var regGhostPercent *regexp.Regexp = regexp.MustCompile(`^Copy:.*?(\d+).\d+%;.*?ETA: (.*)?`)

func (s *session) checkAlterUseOsc(t *TableInfo) {
	if (s.osc.OscOn || s.ghost.GhostOn) && (s.osc.OscMinTableSize == 0 || t.TableSize >= s.osc.OscMinTableSize) {
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

	buf.WriteString(s.opt.Password)
	buf.WriteString(s.opt.Host)
	buf.WriteString(s.opt.User)
	buf.WriteString(strconv.Itoa(s.opt.Port))
	buf.WriteString(strconv.Itoa(r.SeqNo))
	buf.WriteString(r.Sql)

	r.Sqlsha1 = auth.EncodePassword(buf.String())
}

func (s *session) mysqlExecuteAlterTableOsc(r *Record) {

	err := os.Setenv("PATH", fmt.Sprintf("%s%s%s",
		s.osc.OscBinDir, string(os.PathListSeparator), os.Getenv("PATH")))
	if err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
		return
	}

	if _, err := exec.LookPath("pt-online-schema-change"); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
		return
	}

	buf := bytes.NewBufferString("pt-online-schema-change")

	buf.WriteString(" --alter \"")
	buf.WriteString(s.getAlterTablePostPart(r.Sql, true))

	if s.hasError() {
		return
	}

	buf.WriteString("\" ")
	if s.osc.OscPrintSql {
		buf.WriteString(" --print ")
	}
	buf.WriteString(fmt.Sprintf("  --set-vars lock_wait_timeout=%d", s.osc.OscLockWaitTimeout))
	buf.WriteString(" --charset=utf8 ")
	buf.WriteString(" --chunk-time ")
	buf.WriteString(fmt.Sprintf("%g ", s.osc.OscChunkTime))

	buf.WriteString(" --critical-load ")
	buf.WriteString("Threads_connected:")
	buf.WriteString(strconv.Itoa(s.osc.OscCriticalThreadConnected))
	buf.WriteString(",Threads_running:")
	buf.WriteString(strconv.Itoa(s.osc.OscCriticalThreadRunning))
	buf.WriteString(" ")

	buf.WriteString(" --max-load ")
	buf.WriteString("Threads_connected:")
	buf.WriteString(strconv.Itoa(s.osc.OscMaxThreadConnected))
	buf.WriteString(",Threads_running:")
	buf.WriteString(strconv.Itoa(s.osc.OscMaxThreadRunning))
	buf.WriteString(" ")

	buf.WriteString(" --sleep=")
	buf.WriteString(fmt.Sprintf("%g", s.osc.OscSleep))

	buf.WriteString(" --recurse=1 ")

	buf.WriteString(" --check-interval ")
	buf.WriteString(fmt.Sprintf("%d ", s.osc.OscCheckInterval))

	if !s.osc.OscDropNewTable {
		buf.WriteString(" --no-drop-new-table ")
	}

	if !s.osc.OscDropOldTable {
		buf.WriteString(" --no-drop-old-table ")
	}

	if !s.osc.OscCheckReplicationFilters {
		buf.WriteString(" --no-check-replication-filters ")
	}

	if !s.osc.OscCheckUniqueKeyChange {
		buf.WriteString(" --no-check-unique-key-change ")
	}

	if !s.osc.OscCheckAlter {
		buf.WriteString(" --no-check-alter ")
	}

	buf.WriteString(" --alter-foreign-keys-method=")
	buf.WriteString(s.osc.OscAlterForeignKeysMethod)

	if s.osc.OscAlterForeignKeysMethod == "none" {
		buf.WriteString(" --force ")
	}

	buf.WriteString(" --execute ")
	buf.WriteString(" --statistics ")
	buf.WriteString(" --max-lag=")
	buf.WriteString(fmt.Sprintf("%d", s.osc.OscMaxLag))

	if s.isClusterNode && s.dbVersion > 50600 && s.osc.OscMaxFlowCtl >= 0 {
		buf.WriteString(" --max-flow-ctl=")
		buf.WriteString(fmt.Sprintf("%d", s.osc.OscMaxFlowCtl))
	}

	buf.WriteString(" --no-version-check ")
	buf.WriteString(" --recursion-method=")
	buf.WriteString(s.osc.OscRecursionMethod)

	buf.WriteString(" --progress ")
	buf.WriteString("percentage,1 ")

	buf.WriteString(" --user=\"")
	buf.WriteString(s.opt.User)
	buf.WriteString("\" --password='")
	buf.WriteString(strings.Replace(s.opt.Password, "'", "'\"'\"'", -1))
	buf.WriteString("' --host=")
	buf.WriteString(s.opt.Host)
	buf.WriteString(" --port=")
	buf.WriteString(strconv.Itoa(s.opt.Port))

	buf.WriteString(" D=")
	buf.WriteString(r.TableInfo.Schema)
	buf.WriteString(",t='")
	buf.WriteString(r.TableInfo.Name)
	buf.WriteString("'")
	if s.opt.Port != 3306 {
		buf.WriteString(",P=")
		buf.WriteString(strconv.Itoa(s.opt.Port))
	}

	str := buf.String()
	_ = s.execCommand(r, "", "sh", []string{"-c", str})
}

// getSocketFile return gh-ost socket file
// unix socket file max 104 characters (or 107)
func (s *session) getSocketFile(r *Record) string {
	socketFile := fmt.Sprintf("/tmp/gh-ost.%s.%d.%s.%s.sock", s.opt.Host, s.opt.Port,
		r.TableInfo.Schema, r.TableInfo.Name)
	if len(socketFile) > 100 {
		// 字符串过长时转换为hash值
		host := truncateString(s.opt.Host, 30)
		dbName := truncateString(r.TableInfo.Schema, 30)
		tableName := truncateString(r.TableInfo.Name, 30)
		socketFile = fmt.Sprintf("/tmp/gh-ost.%s.%d.%s.%s.sock", host, s.opt.Port,
			dbName, tableName)
		if len(socketFile) > 100 {
			socketFile = fmt.Sprintf("/tmp/gh%s%d%s%s.sock", host, s.opt.Port,
				dbName, tableName)
		}
	}
	return socketFile
}

func (s *session) mysqlExecuteWithGhost(r *Record) {
	err := os.Setenv("PATH", fmt.Sprintf("%s%s%s",
		s.ghost.GhostBinDir, string(os.PathListSeparator), os.Getenv("PATH")))
	if err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
		return
	}

	if _, err := exec.LookPath("gh-ost"); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
		return
	}

	buf := bytes.NewBufferString("gh-ost")

	buf.WriteString(" --alter \"")
	buf.WriteString(s.getAlterTablePostPart(r.Sql, true))

	if s.hasError() {
		return
	}

	buf.WriteString("\" ")
	if s.osc.OscPrintSql {
		buf.WriteString(" --verbose ")
	}

	// RDS数据库需要做特殊处理
	var masterHost string
	if s.ghost.GhostAliyunRds && s.ghost.GhostAssumeMasterHost == "" {
		masterHost = fmt.Sprintf("%s:%d", s.opt.Host, s.opt.Port)
	} else {
		masterHost = s.ghost.GhostAssumeMasterHost
	}

	buf.WriteString(fmt.Sprintf(" --assume-master-host=%s", masterHost))
	buf.WriteString(fmt.Sprintf(" --exact-rowcount=%t", s.ghost.GhostExactRowcount))
	buf.WriteString(fmt.Sprintf(" --concurrent-rowcount=%t", s.ghost.GhostConcurrentRowcount))
	buf.WriteString(fmt.Sprintf(" --allow-on-master=%t", s.ghost.GhostAllowOnMaster))
	buf.WriteString(fmt.Sprintf(" --allow-master-master=%t", s.ghost.GhostAllowMasterMaster))
	buf.WriteString(fmt.Sprintf(" --allow-nullable-unique-key=%t", s.ghost.GhostAllowNullableUniqueKey))
	buf.WriteString(fmt.Sprintf(" --approve-renamed-columns=%t",
		s.ghost.GhostApproveRenamedColumns))
	buf.WriteString(fmt.Sprintf(" --tungsten=%t", s.ghost.GhostTungsten))
	buf.WriteString(fmt.Sprintf(" --discard-foreign-keys=%t", s.ghost.GhostDiscardForeignKeys))
	buf.WriteString(fmt.Sprintf(" --skip-foreign-key-checks=%t", s.ghost.GhostSkipForeignKeyChecks))
	buf.WriteString(fmt.Sprintf(" --aliyun-rds=%t", s.ghost.GhostAliyunRds))
	buf.WriteString(fmt.Sprintf(" --gcp=%t", s.ghost.GhostGcp))
	buf.WriteString(fmt.Sprintf(" --ok-to-drop-table=%t", s.ghost.GhostOkToDropTable))
	buf.WriteString(fmt.Sprintf(" --initially-drop-old-table=%t", s.ghost.GhostInitiallyDropOldTable))
	buf.WriteString(fmt.Sprintf(" --initially-drop-ghost-table=%t", s.ghost.GhostInitiallyDropGhostTable))
	buf.WriteString(fmt.Sprintf(" --cut-over=%s", s.ghost.GhostCutOver))
	buf.WriteString(fmt.Sprintf(" --force-named-cut-over=%t", s.ghost.GhostForceNamedCutOver))
	buf.WriteString(fmt.Sprintf(" --assume-rbr=%t", s.ghost.GhostAssumeRbr))
	buf.WriteString(fmt.Sprintf(" --cut-over-exponential-backoff=%t", s.ghost.GhostCutOverExponentialBackoff))
	buf.WriteString(fmt.Sprintf(" --exponential-backoff-max-interval=%d", s.ghost.GhostExponentialBackoffMaxInterval))
	buf.WriteString(fmt.Sprintf(" --chunk-size=%d", s.ghost.GhostChunkSize))
	buf.WriteString(fmt.Sprintf(" --dml-batch-size=%d", s.ghost.GhostDmlBatchSize))
	buf.WriteString(fmt.Sprintf(" --default-retries=%d", s.ghost.GhostDefaultRetries))
	buf.WriteString(fmt.Sprintf(" --cut-over-lock-timeout-seconds=%d", s.ghost.GhostCutOverLockTimeoutSeconds))
	buf.WriteString(fmt.Sprintf(" --nice-ratio=%f", s.ghost.GhostNiceRatio))
	buf.WriteString(fmt.Sprintf(" --max-lag-millis=%d", s.ghost.GhostMaxLagMillis))
	buf.WriteString(fmt.Sprintf(" --replication-lag-query=%s", s.ghost.GhostReplicationLagQuery))
	if s.ghost.GhostThrottleControlReplicas != "" {
		buf.WriteString(fmt.Sprintf(" --throttle-control-replicas=%s", s.ghost.GhostThrottleControlReplicas))
	}
	if s.ghost.GhostThrottleQuery != "" {
		buf.WriteString(fmt.Sprintf(" --throttle-query=%s", s.ghost.GhostThrottleQuery))
	}
	if s.ghost.GhostThrottleHTTP != "" {
		buf.WriteString(fmt.Sprintf(" --throttle-http=%s", s.ghost.GhostThrottleHTTP))
	}

	buf.WriteString(fmt.Sprintf(" --heartbeat-interval-millis=%d", s.ghost.GhostHeartbeatIntervalMillis))
	buf.WriteString(fmt.Sprintf(" --throttle-flag-file=%s", s.ghost.GhostThrottleFlagFile))
	buf.WriteString(fmt.Sprintf(" --throttle-additional-flag-file=%s", s.ghost.GhostThrottleAdditionalFlagFile))
	buf.WriteString(fmt.Sprintf(" --postpone-cut-over-flag-file=%s", s.ghost.GhostPostponeCutOverFlagFile))
	buf.WriteString(fmt.Sprintf(" --initially-drop-socket-file=%t", s.ghost.GhostInitiallyDropSocketFile))

	// unix socket file max 104 characters (or 107)
	socketFile := s.getSocketFile(r)
	if !s.ghost.GhostInitiallyDropSocketFile {
		if _, err := os.Stat(socketFile); err == nil {
			s.appendErrorMessage("listen unix socket file already in use, need to clean up manually")
			return
		} else if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
			log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
			s.appendErrorMessage(err.Error())
			return
		}
	}

	buf.WriteString(fmt.Sprintf(" --serve-socket-file=%s", socketFile))

	panicFile := fmt.Sprintf("%s.panic", strings.TrimRight(socketFile, ".sock"))
	buf.WriteString(fmt.Sprintf(" --panic-flag-file=%s", panicFile))

	// 清理panic file
	os.Remove(panicFile)

	buf.WriteString(fmt.Sprintf(" --replica-server-id=%d",
		2000100000+uint(s.sessionVars.ConnectionID%10000)))
	buf.WriteString(fmt.Sprintf(" --critical-load-interval-millis=%d", s.ghost.GhostCriticalLoadIntervalMillis))
	buf.WriteString(fmt.Sprintf(" --critical-load-hibernate-seconds=%d", s.ghost.GhostCriticalLoadHibernateSeconds))

	if s.ghost.GhostForceTableNames != "" {
		buf.WriteString(fmt.Sprintf(" --force-table-names=%s", s.ghost.GhostForceTableNames))
	}

	buf.WriteString(" --critical-load='")
	buf.WriteString("Threads_running=")
	buf.WriteString(strconv.Itoa(s.osc.OscCriticalThreadRunning))
	buf.WriteString(",threads_connected=")
	buf.WriteString(strconv.Itoa(s.osc.OscCriticalThreadConnected))
	buf.WriteString("' ")

	buf.WriteString(" --max-load='")
	buf.WriteString("Threads_running=")
	buf.WriteString(strconv.Itoa(s.osc.OscMaxThreadRunning))
	buf.WriteString(",threads_connected=")
	buf.WriteString(strconv.Itoa(s.osc.OscMaxThreadConnected))
	buf.WriteString("' ")

	buf.WriteString(" --execute ")

	buf.WriteString(" --user=\"")
	buf.WriteString(s.opt.User)
	buf.WriteString("\" --password='")
	buf.WriteString(strings.Replace(s.opt.Password, "'", "'\"'\"'", -1))
	buf.WriteString("' --host=")
	buf.WriteString(s.opt.Host)
	buf.WriteString(" --port=")
	buf.WriteString(strconv.Itoa(s.opt.Port))

	buf.WriteString(" --database='")
	buf.WriteString(r.TableInfo.Schema)
	buf.WriteString("' --table='")
	buf.WriteString(r.TableInfo.Name)
	buf.WriteString("'")

	str := buf.String()
	// log.Info(str)
	if err := s.execCommand(r, socketFile, "sh", []string{"-c", str}); err != nil {
		// 当失败时自动清理socket文件
		if !strings.Contains(err.Error(), "file already in use") {
			os.Remove(socketFile)
		}
	}
}

func (s *session) mysqlExecuteAlterTableGhost(r *Record) {
	migrationContext := base.NewMigrationContext()
	// flag.StringVar(&migrationContext.InspectorConnectionConfig.Key.Hostname, "host", "127.0.0.1", "MySQL hostname (preferably a replica, not the master)")
	migrationContext.InspectorConnectionConfig.Key.Hostname = s.opt.Host
	// flag.StringVar(&migrationContext.AssumeMasterHostname, "assume-master-host", "", "(optional) explicitly tell gh-ost the identity of the master. Format: some.host.com[:port] This is useful in master-master setups where you wish to pick an explicit master, or in a tungsten-replicator where gh-ost is unable to determine the master")

	// RDS数据库需要做特殊处理
	if s.ghost.GhostAliyunRds && s.ghost.GhostAssumeMasterHost == "" {
		migrationContext.AssumeMasterHostname = fmt.Sprintf("%s:%d", s.opt.Host, s.opt.Port)
	} else {
		migrationContext.AssumeMasterHostname = s.ghost.GhostAssumeMasterHost
	}

	log.Debug("assume_master_host: ", migrationContext.AssumeMasterHostname)
	// flag.IntVar(&migrationContext.InspectorConnectionConfig.Key.Port, "port", 3306, "MySQL port (preferably a replica, not the master)")
	migrationContext.InspectorConnectionConfig.Key.Port = s.opt.Port
	// flag.StringVar(&migrationContext.CliUser, "user", "", "MySQL user")
	migrationContext.CliUser = s.opt.User
	// flag.StringVar(&migrationContext.CliPassword, "password", "", "MySQL password")
	migrationContext.CliPassword = s.opt.Password
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
	migrationContext.CountTableRows = s.ghost.GhostExactRowcount
	// flag.BoolVar(&migrationContext.CountTableRows, "exact-rowcount", false, "actually count table rows as opposed to estimate them (results in more accurate progress estimation)")
	migrationContext.ConcurrentCountTableRows = s.ghost.GhostConcurrentRowcount
	// flag.BoolVar(&migrationContext.ConcurrentCountTableRows, "concurrent-rowcount", true, "(with --exact-rowcount), when true (default): count rows after row-copy begins, concurrently, and adjust row estimate later on; when false: first count rows, then start row copy")
	migrationContext.AllowedRunningOnMaster = s.ghost.GhostAllowOnMaster
	// flag.BoolVar(&migrationContext.AllowedRunningOnMaster, "allow-on-master", false, "allow this migration to run directly on master. Preferably it would run on a replica")
	migrationContext.AllowedMasterMaster = s.ghost.GhostAllowMasterMaster
	// flag.BoolVar(&migrationContext.AllowedMasterMaster, "allow-master-master", false, "explicitly allow running in a master-master setup")
	migrationContext.NullableUniqueKeyAllowed = s.ghost.GhostAllowNullableUniqueKey
	// flag.BoolVar(&migrationContext.NullableUniqueKeyAllowed, "allow-nullable-unique-key", false, "allow gh-ost to migrate based on a unique key with nullable columns. As long as no NULL values exist, this should be OK. If NULL values exist in chosen key, data may be corrupted. Use at your own risk!")
	migrationContext.ApproveRenamedColumns = s.ghost.GhostApproveRenamedColumns
	// flag.BoolVar(&migrationContext.ApproveRenamedColumns, "approve-renamed-columns", false, "in case your `ALTER` statement renames columns, gh-ost will note that and offer its interpretation of the rename. By default gh-ost does not proceed to execute. This flag approves that gh-ost's interpretation is correct")
	migrationContext.SkipRenamedColumns = false
	// flag.BoolVar(&migrationContext.SkipRenamedColumns, "skip-renamed-columns", false, "in case your `ALTER` statement renames columns, gh-ost will note that and offer its interpretation of the rename. By default gh-ost does not proceed to execute. This flag tells gh-ost to skip the renamed columns, i.e. to treat what gh-ost thinks are renamed columns as unrelated columns. NOTE: you may lose column data")
	migrationContext.IsTungsten = s.ghost.GhostTungsten
	// flag.BoolVar(&migrationContext.IsTungsten, "tungsten", false, "explicitly let gh-ost know that you are running on a tungsten-replication based topology (you are likely to also provide --assume-master-host)")
	migrationContext.DiscardForeignKeys = s.ghost.GhostDiscardForeignKeys
	// flag.BoolVar(&migrationContext.DiscardForeignKeys, "discard-foreign-keys", false, "DANGER! This flag will migrate a table that has foreign keys and will NOT create foreign keys on the ghost table, thus your altered table will have NO foreign keys. This is useful for intentional dropping of foreign keys")
	migrationContext.SkipForeignKeyChecks = s.ghost.GhostSkipForeignKeyChecks
	// flag.BoolVar(&migrationContext.SkipForeignKeyChecks, "skip-foreign-key-checks", false, "set to 'true' when you know for certain there are no foreign keys on your table, and wish to skip the time it takes for gh-ost to verify that")
	migrationContext.AliyunRDS = s.ghost.GhostAliyunRds
	// flag.BoolVar(&migrationContext.AliyunRDS, "aliyun-rds", false, "set to 'true' when you execute on Aliyun RDS.")
	migrationContext.GoogleCloudPlatform = s.ghost.GhostGcp
	// flag.BoolVar(&migrationContext.GoogleCloudPlatform, "gcp", false, "set to 'true' when you execute on a 1st generation Google Cloud Platform (GCP).")

	// executeFlag := flag.Bool("execute", false, "actually execute the alter & migrate the table. Default is noop: do some tests and exit")
	migrationContext.TestOnReplica = false
	// flag.BoolVar(&migrationContext.TestOnReplica, "test-on-replica", false, "Have the migration run on a replica, not on the master. At the end of migration replication is stopped, and tables are swapped and immediately swap-revert. Replication remains stopped and you can compare the two tables for building trust")
	migrationContext.TestOnReplicaSkipReplicaStop = false
	// flag.BoolVar(&migrationContext.TestOnReplicaSkipReplicaStop, "test-on-replica-skip-replica-stop", false, "When --test-on-replica is enabled, do not issue commands stop replication (requires --test-on-replica)")
	migrationContext.MigrateOnReplica = false
	// flag.BoolVar(&migrationContext.MigrateOnReplica, "migrate-on-replica", false, "Have the migration run on a replica, not on the master. This will do the full migration on the replica including cut-over (as opposed to --test-on-replica)")

	migrationContext.OkToDropTable = s.ghost.GhostOkToDropTable
	// flag.BoolVar(&migrationContext.OkToDropTable, "ok-to-drop-table", false, "Shall the tool drop the old table at end of operation. DROPping tables can be a long locking operation, which is why I'm not doing it by default. I'm an online tool, yes?")
	migrationContext.InitiallyDropOldTable = s.ghost.GhostInitiallyDropOldTable
	// flag.BoolVar(&migrationContext.InitiallyDropOldTable, "initially-drop-old-table", false, "Drop a possibly existing OLD table (remains from a previous run?) before beginning operation. Default is to panic and abort if such table exists")
	migrationContext.InitiallyDropGhostTable = s.ghost.GhostInitiallyDropGhostTable
	// flag.BoolVar(&migrationContext.InitiallyDropGhostTable, "initially-drop-ghost-table", false, "Drop a possibly existing Ghost table (remains from a previous run?) before beginning operation. Default is to panic and abort if such table exists")
	migrationContext.TimestampOldTable = s.ghost.GhostTimestampOldTable
	// flag.BoolVar(&migrationContext.TimestampOldTable, "timestamp-old-table", false, "Use a timestamp in old table name. This makes old table names unique and non conflicting cross migrations")
	cutOver := s.ghost.GhostCutOver
	// cutOver := flag.String("cut-over", "atomic", "choose cut-over type (default|atomic, two-step)")
	migrationContext.ForceNamedCutOverCommand = s.ghost.GhostForceNamedCutOver
	// flag.BoolVar(&migrationContext.ForceNamedCutOverCommand, "force-named-cut-over", false, "When true, the 'unpostpone|cut-over' interactive command must name the migrated table")
	migrationContext.SwitchToRowBinlogFormat = false
	// flag.BoolVar(&migrationContext.SwitchToRowBinlogFormat, "switch-to-rbr", false, "let this tool automatically switch binary log format to 'ROW' on the replica, if needed. The format will NOT be switched back. I'm too scared to do that, and wish to protect you if you happen to execute another migration while this one is running")
	migrationContext.AssumeRBR = s.ghost.GhostAssumeRbr
	// flag.BoolVar(&migrationContext.AssumeRBR, "assume-rbr", false, "set to 'true' when you know for certain your server uses 'ROW' binlog_format. gh-ost is unable to tell, event after reading binlog_format, whether the replication process does indeed use 'ROW', and restarts replication to be certain RBR setting is applied. Such operation requires SUPER privileges which you might not have. Setting this flag avoids restarting replication and you can proceed to use gh-ost without SUPER privileges")
	migrationContext.CutOverExponentialBackoff = s.ghost.GhostCutOverExponentialBackoff
	// flag.BoolVar(&migrationContext.CutOverExponentialBackoff, "cut-over-exponential-backoff", false, "Wait exponentially longer intervals between failed cut-over attempts. Wait intervals obey a maximum configurable with 'exponential-backoff-max-interval').")
	exponentialBackoffMaxInterval := s.ghost.GhostExponentialBackoffMaxInterval
	// exponentialBackoffMaxInterval := flag.Int64("exponential-backoff-max-interval", 64, "Maximum number of seconds to wait between attempts when performing various operations with exponential backoff.")
	chunkSize := s.ghost.GhostChunkSize
	// chunkSize := flag.Int64("chunk-size", 1000, "amount of rows to handle in each iteration (allowed range: 100-100,000)")
	dmlBatchSize := s.ghost.GhostDmlBatchSize
	// dmlBatchSize := flag.Int64("dml-batch-size", 10, "batch size for DML events to apply in a single transaction (range 1-100)")
	defaultRetries := s.ghost.GhostDefaultRetries
	// defaultRetries := flag.Int64("default-retries", 60, "Default number of retries for various operations before panicking")
	cutOverLockTimeoutSeconds := s.ghost.GhostCutOverLockTimeoutSeconds
	// cutOverLockTimeoutSeconds := flag.Int64("cut-over-lock-timeout-seconds", 3, "Max number of seconds to hold locks on tables while attempting to cut-over (retry attempted when lock exceeds timeout)")
	niceRatio := s.ghost.GhostNiceRatio
	// niceRatio := flag.Float64("nice-ratio", 0, "force being 'nice', imply sleep time per chunk time; range: [0.0..100.0]. Example values: 0 is aggressive. 1: for every 1ms spent copying rows, sleep additional 1ms (effectively doubling runtime); 0.7: for every 10ms spend in a rowcopy chunk, spend 7ms sleeping immediately after")
	maxLagMillis := s.ghost.GhostMaxLagMillis
	// maxLagMillis := flag.Int64("max-lag-millis", 1500, "replication lag at which to throttle operation")
	replicationLagQuery := s.ghost.GhostReplicationLagQuery
	// replicationLagQuery := flag.String("replication-lag-query", "", "Deprecated. gh-ost uses an internal, subsecond resolution query")
	throttleControlReplicas := s.ghost.GhostThrottleControlReplicas
	// throttleControlReplicas := flag.String("throttle-control-replicas", "", "List of replicas on which to check for lag; comma delimited. Example: myhost1.com:3306,myhost2.com,myhost3.com:3307")
	throttleQuery := s.ghost.GhostThrottleQuery
	// throttleQuery := flag.String("throttle-query", "", "when given, issued (every second) to check if operation should throttle. Expecting to return zero for no-throttle, >0 for throttle. Query is issued on the migrated server. Make sure this query is lightweight")
	throttleHTTP := s.ghost.GhostThrottleHTTP
	// throttleHTTP := flag.String("throttle-http", "", "when given, gh-ost checks given URL via HEAD request; any response code other than 200 (OK) causes throttling; make sure it has low latency response")
	heartbeatIntervalMillis := s.ghost.GhostHeartbeatIntervalMillis
	// heartbeatIntervalMillis := flag.Int64("heartbeat-interval-millis", 100, "how frequently would gh-ost inject a heartbeat value")
	migrationContext.ThrottleFlagFile = s.ghost.GhostThrottleFlagFile
	// flag.StringVar(&migrationContext.ThrottleFlagFile, "throttle-flag-file", "", "operation pauses when this file exists; hint: use a file that is specific to the table being altered")
	migrationContext.ThrottleAdditionalFlagFile = s.ghost.GhostThrottleAdditionalFlagFile
	// flag.StringVar(&migrationContext.ThrottleAdditionalFlagFile, "throttle-additional-flag-file", "/tmp/gh-ost.throttle", "operation pauses when this file exists; hint: keep default, use for throttling multiple gh-ost operations")
	migrationContext.PostponeCutOverFlagFile = s.ghost.GhostPostponeCutOverFlagFile
	// flag.StringVar(&migrationContext.PostponeCutOverFlagFile, "postpone-cut-over-flag-file", "", "while this file exists, migration will postpone the final stage of swapping tables, and will keep on syncing the ghost table. Cut-over/swapping would be ready to perform the moment the file is deleted.")
	// migrationContext.PanicFlagFile = s.Ghost.GhostPanicFlagFile
	// flag.StringVar(&migrationContext.PanicFlagFile, "panic-flag-file", "", "when this file is created, gh-ost will immediately terminate, without cleanup")
	migrationContext.DropServeSocket = s.ghost.GhostInitiallyDropSocketFile
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
		s.osc.OscCriticalThreadRunning, s.osc.OscCriticalThreadConnected)

	maxLoad := fmt.Sprintf("Threads_running=%d,Threads_connected=%d",
		s.osc.OscMaxThreadRunning, s.osc.OscMaxThreadConnected)
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

	migrationContext.CriticalLoadIntervalMilliseconds = s.ghost.GhostCriticalLoadIntervalMillis
	// flag.Int64Var(&migrationContext.CriticalLoadIntervalMilliseconds, "critical-load-interval-millis", 0, "When 0, migration immediately bails out upon meeting critical-load. When non-zero, a second check is done after given interval, and migration only bails out if 2nd check still meets critical load")
	migrationContext.CriticalLoadHibernateSeconds = s.ghost.GhostCriticalLoadHibernateSeconds
	// flag.Int64Var(&migrationContext.CriticalLoadHibernateSeconds, "critical-load-hibernate-seconds", 0, "When nonzero, critical-load does not panic and bail out; instead, gh-ost goes into hibernate for the specified duration. It will not read/write anything to from/to any server")
	// quiet := false
	// verbose := false
	// debug := false
	// stack := false
	// help := false
	// version := false
	migrationContext.ForceTmpTableName = s.ghost.GhostForceTableNames
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
		s.appendErrorMessage("--allow-on-master and --test-on-replica are mutually exclusive")
	}
	if migrationContext.AllowedRunningOnMaster && migrationContext.MigrateOnReplica {
		log.Error("--allow-on-master and --migrate-on-replica are mutually exclusive")
		s.appendErrorMessage("--allow-on-master and --migrate-on-replica are mutually exclusive")
	}
	if migrationContext.MigrateOnReplica && migrationContext.TestOnReplica {
		log.Error("--migrate-on-replica and --test-on-replica are mutually exclusive")
		s.appendErrorMessage("--migrate-on-replica and --test-on-replica are mutually exclusive")
	}
	if migrationContext.SwitchToRowBinlogFormat && migrationContext.AssumeRBR {
		log.Error("--switch-to-rbr and --assume-rbr are mutually exclusive")
		s.appendErrorMessage("--switch-to-rbr and --assume-rbr are mutually exclusive")
	}
	if migrationContext.TestOnReplicaSkipReplicaStop {
		if !migrationContext.TestOnReplica {
			log.Error("--test-on-replica-skip-replica-stop requires --test-on-replica to be enabled")
			s.appendErrorMessage("--test-on-replica-skip-replica-stop requires --test-on-replica to be enabled")
		}
		log.Warning("--test-on-replica-skip-replica-stop enabled. We will not stop replication before cut-over. Ensure you have a plugin that does this.")
	}
	if migrationContext.CliMasterUser != "" && migrationContext.AssumeMasterHostname == "" {
		log.Error("--master-user requires --assume-master-host")
		s.appendErrorMessage("--master-user requires --assume-master-host")
	}
	if migrationContext.CliMasterPassword != "" && migrationContext.AssumeMasterHostname == "" {
		log.Error("--master-password requires --assume-master-host")
		s.appendErrorMessage("--master-password requires --assume-master-host")
	}
	if migrationContext.TLSCACertificate != "" && !migrationContext.UseTLS {
		log.Error("--ssl-ca requires --ssl")
		s.appendErrorMessage("--ssl-ca requires --ssl")
	}
	if migrationContext.TLSAllowInsecure && !migrationContext.UseTLS {
		log.Error("--ssl-allow-insecure requires --ssl")
		s.appendErrorMessage("--ssl-allow-insecure requires --ssl")
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
		s.appendErrorMessage(fmt.Sprintf("Unknown cut-over: %s", cutOver))
	}
	if err := migrationContext.ReadConfigFile(); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
	}
	if err := migrationContext.ReadThrottleControlReplicaKeys(throttleControlReplicas); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
	}
	if err := migrationContext.ReadMaxLoad(maxLoad); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
	}
	if err := migrationContext.ReadCriticalLoad(criticalLoad); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
	}
	if migrationContext.ServeSocketFile == "" {
		// unix socket file max 104 characters (or 107)
		socketFile := s.getSocketFile(r)
		if _, err := os.Stat(socketFile); err == nil {
			s.appendErrorMessage("listen unix socket file already in use, need to clean up manually")
			return
		} else if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
			log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
			s.appendErrorMessage(err.Error())
			return
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
		s.appendErrorMessage(err.Error())
	}
	if err := migrationContext.SetCutOverLockTimeoutSeconds(cutOverLockTimeoutSeconds); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
	}
	if err := migrationContext.SetExponentialBackoffMaxInterval(exponentialBackoffMaxInterval); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
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
		PanicAbort: make(chan util.ProcessOperation),
		RW:         &sync.RWMutex{},
	}
	s.sessionManager.AddOscProcess(p)

	// defer func() {
	// 	log.Error("complete -------")
	// 	log.Errorf("defer runtime.NumGoroutine: %v", runtime.NumGoroutine())
	// }()

	buf := bytes.NewBufferString("")
	migrator := logic.NewMigrator(migrationContext)

	stop := make(chan bool)

	// log.Errorf("start runtime.NumGoroutine: %v", runtime.NumGoroutine())

	//实时循环读取输出流中的一行内容
	f := func(reader *bufio.Reader) {
		statusTick := time.NewTicker(500 * time.Millisecond)
	FOR:
		for {
			<-statusTick.C

			select {
			case <-stop:
				break FOR
			default:
				line, err2 := reader.ReadString('\n')
				if err2 != nil || io.EOF == err2 || line == "" {
					// 此处不要中止for循环，仅跳过当次循环
					continue
				}
				buf.WriteString(line)
				buf.WriteString("\n")
				if ok := s.mysqlAnalyzeGhostOutput(line, p); ok {
					statusTick.Stop()
					break FOR
				}
			}
		}

		// log.Error("end reader -------")
		// log.Errorf("defer runtime.NumGoroutine: %v", runtime.NumGoroutine())
	}

	migrator.Log = bytes.NewBufferString("")

	// 监听Kill操作
	go func() {
		for oper := range p.PanicAbort {
			p.RW.Lock()
			switch oper {
			case util.ProcessOperationKill:
				migrationContext.PanicAbort <- fmt.Errorf("Execute has been abort in percent: %d, remain time: %s",
					p.Percent, p.RemainTime)
				p.Killed = true
				if _, ok := <-stop; ok {
					stop <- true
				}
			case util.ProcessOperationPause:
				p.Pause = true
				if migrationContext.ThrottleCommandedByUser == 0 {
					atomic.StoreInt64(&migrationContext.ThrottleCommandedByUser, 1)
				}
			case util.ProcessOperationResume:
				p.Pause = false
				if migrationContext.ThrottleCommandedByUser == 1 {
					atomic.StoreInt64(&migrationContext.ThrottleCommandedByUser, 0)
				}
			}
			p.RW.Unlock()
		}

		// log.Error("end panic -------")
		// log.Errorf("defer runtime.NumGoroutine: %v", runtime.NumGoroutine())
	}()
	go f(bufio.NewReader(migrator.Log))

	if err := migrator.Migrate(); err != nil {
		log.Error(err)
		s.appendErrorMessage(err.Error())
	}

	s.sessionManager.OscLock()
	close(p.PanicAbort)
	close(stop)
	s.sessionManager.OscUnLock()

	if s.hasError() {
		r.StageStatus = StatusExecFail
	} else {
		r.StageStatus = StatusExecOK
		r.ExecComplete = true
	}

	if s.hasError() || s.osc.OscPrintNone {
		r.Buf.WriteString(buf.String())
		r.Buf.WriteString("\n")
	}
}

func (s *session) execCommand(r *Record, socketFile string, commandName string, params []string) error {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command(commandName, params...)

	// log.Infof("%s %s", commandName, params)

	//StdoutPipe方法返回一个在命令Start后与命令标准输出关联的管道。Wait方法获知命令结束后会关闭这个管道，一般不需要显式的关闭该管道。
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.appendErrorMessage(err.Error())
		log.Error(err)
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.appendErrorMessage(err.Error())
		log.Error(err)
		return err
	}

	// 保证关闭输出流
	defer stdout.Close()
	defer stderr.Close()

	// 运行命令
	if err := cmd.Start(); err != nil {
		s.appendErrorMessage(err.Error())
		log.Error(err)
		return err
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
		IsGhost:    socketFile != "",
		SocketFile: socketFile,
		PanicAbort: make(chan util.ProcessOperation),
		RW:         &sync.RWMutex{},
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

			if s.ghost.GhostOn {
				if strings.Contains("[info]", line) {
					continue
				}
				buf.WriteString(line)
				if ok := s.mysqlAnalyzeGhostOutput(line, p); ok {
					wg.Done()
					break
				}
			} else {
				buf.WriteString(line)
				s.mysqlAnalyzeOscOutput(line, p)
			}
			buf.WriteString("\n")
		}
	}

	// 监听Kill操作
	go func() {
		oper := <-p.PanicAbort
		if oper == util.ProcessOperationKill {
			p.RW.Lock()
			p.Killed = true
			p.RW.Unlock()
			if err := cmd.Process.Kill(); err != nil {
				s.appendErrorMessage(err.Error())
			} else {
				s.appendErrorMessage(
					fmt.Sprintf("Execute has been abort in percent: %d, remain time: %s",
						p.Percent, p.RemainTime))
			}
		}
	}()
	go f(reader)
	go f(reader2)

	wg.Wait()

	//阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	err = cmd.Wait()
	if err != nil {
		s.appendErrorMessage(err.Error())
		log.Errorf("%s %s", commandName, params)
		log.Error(err)
	}

	close(p.PanicAbort)

	allMessage := buf.String()
	if p.Percent < 100 || s.myRecord.ErrLevel == 2 {
		s.recordSets.MaxLevel = 2
		r.ErrLevel = 2
		r.StageStatus = StatusExecFail
	} else {
		r.StageStatus = StatusExecOK
		r.ExecComplete = true
	}

	if p.Percent < 100 || s.osc.OscPrintNone {
		r.Buf.WriteString(allMessage)
		r.Buf.WriteString("\n")
	}

	if p.Percent < 100 || s.myRecord.ErrLevel == 2 {
		return fmt.Errorf(allMessage)
	}

	// 执行完成或中止后清理osc进程信息
	// pl := s.sessionManager.ShowOscProcessList()
	// delete(pl, p.Sqlsha1)
	return nil
}

func (s *session) mysqlAnalyzeOscOutput(out string, p *util.OscProcessInfo) {
	firsts := regOscPercent.FindStringSubmatch(out)
	p.RW.Lock()
	defer p.RW.Unlock()

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

func (s *session) mysqlAnalyzeGhostOutput(out string, p *util.OscProcessInfo) (complete bool) {
	firsts := regGhostPercent.FindStringSubmatch(out)

	p.RW.Lock()
	defer p.RW.Unlock()

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

	if pct >= 100 {
		pct = 100
		complete = true
	}

	p.Percent = pct
	p.RemainTime = remain
	p.Info = strings.TrimSpace(out)
	return
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
		s.appendErrorMessage(err.Error())
		return "", false
	}
	var builder strings.Builder
	var columns []string

	if len(stmtNodes) == 0 {
		s.appendErrorMessage(fmt.Sprintf("未正确解析ALTER语句: %s", sql))
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
					s.appendErrorMessage(err.Error())
					return "", false
				}
				restoreSQL := builder.String()
				columns = append(columns, restoreSQL)
			}
		default:
			s.appendErrorMessage(fmt.Sprintf("无效类型: %v", stmtNode))
			return "", false
		}
	}

	if len(columns) > 0 {
		return strings.Join(columns, ", "), true
	}

	s.appendErrorMessage(fmt.Sprintf("未正确解析SQL: %s", sql))
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

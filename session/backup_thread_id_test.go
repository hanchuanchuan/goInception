package session_test

import (
	"context"
	gosql "database/sql"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/session"
	"github.com/hanchuanchuan/goInception/util"
	repllog "github.com/siddontang/go-log/log"
	log "github.com/sirupsen/logrus"
)

// 备份校验线程号一致性测试。
//
// 背景：DML 执行与 select connection_id() 原本经由连接池分别执行，连接池静默
// 换连（坏连接替换等）会导致 record.ThreadId 与实际执行连接不一致，binlog 解析
// (checkFilter) 匹配不到变更行，语句被误判为 "Backup failed" 且无错误信息。
//
// 本测试要求本地 MySQL 开启 ROW binlog（docker 示例见 MR 描述），
// 目标库同时充当备份库。通过环境变量覆盖连接信息，未配置且默认地址连不上时跳过：
//
//	GOI_TEST_MYSQL_HOST / GOI_TEST_MYSQL_PORT / GOI_TEST_MYSQL_USER / GOI_TEST_MYSQL_PASSWORD
type backupThreadIDSuite struct {
	host     string
	port     int
	user     string
	password string

	admin *gosql.DB
}

type testEnvMySQL struct {
	host     string
	port     string
	user     string
	password string
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func newBackupThreadIDSuite(t *testing.T) *backupThreadIDSuite {
	env := testEnvMySQL{
		host:     envOrDefault("GOI_TEST_MYSQL_HOST", "127.0.0.1"),
		port:     envOrDefault("GOI_TEST_MYSQL_PORT", "3306"),
		user:     envOrDefault("GOI_TEST_MYSQL_USER", "test"),
		password: envOrDefault("GOI_TEST_MYSQL_PASSWORD", "test"),
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?timeout=3s",
		env.user, env.password, env.host, env.port)
	admin, err := gosql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("open mysql failed: %v", err)
	}
	if err := admin.Ping(); err != nil {
		admin.Close()
		t.Skipf("mysql not available at %s:%s (set GOI_TEST_MYSQL_* to enable): %v",
			env.host, env.port, err)
	}

	return &backupThreadIDSuite{
		host:     env.host,
		port:     mustAtoi(t, env.port),
		user:     env.user,
		password: env.password,
		admin:    admin,
	}
}

func mustAtoi(t *testing.T, s string) int {
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		t.Fatalf("invalid port %q: %v", s, err)
	}
	return v
}

func (s *backupThreadIDSuite) close() {
	s.admin.Close()
}

func (s *backupThreadIDSuite) exec(t *testing.T, query string) {
	if _, err := s.admin.Exec(query); err != nil {
		t.Fatalf("exec %q failed: %v", query, err)
	}
}

// loadIncConfig 加载默认配置并把备份库指向测试 MySQL
func (s *backupThreadIDSuite) loadIncConfig(t *testing.T) {
	log.SetLevel(log.ErrorLevel)
	repllog.SetLevel(repllog.LevelFatal)

	cfg := config.GetGlobalConfig()
	_, localFile, _, _ := runtime.Caller(0)
	dir := path.Dir(localFile)
	configFile := path.Join(dir[0:len(dir)-len("session")], "config/config.toml.default")
	if err := cfg.Load(configFile); err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	inc := &config.GetGlobalConfig().Inc
	inc.BackupHost = s.host
	inc.BackupPort = uint(s.port)
	inc.BackupUser = s.user
	inc.BackupPassword = s.password
	inc.Lang = "en-US"
}

// noOpSessionManager 提供 RunExecute 所需的最小 SessionManager 实现
type noOpSessionManager struct {
	mu       sync.RWMutex
	oscProcs map[string]*util.OscProcessInfo
}

func (m *noOpSessionManager) ShowProcessList() map[uint64]util.ProcessInfo { return nil }
func (m *noOpSessionManager) Kill(connectionID uint64, query bool)         {}
func (m *noOpSessionManager) AddOscProcess(p *util.OscProcessInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oscProcs[p.Sqlsha1] = p
}
func (m *noOpSessionManager) ShowOscProcessList() map[string]*util.OscProcessInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.oscProcs
}
func (m *noOpSessionManager) ShowOscProcessListWithWrite() map[string]*util.OscProcessInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.oscProcs
}
func (m *noOpSessionManager) OscLock()                           { m.mu.Lock() }
func (m *noOpSessionManager) OscUnLock()                         { m.mu.Unlock() }
func (m *noOpSessionManager) KillOscProcess(connectionID uint64) {}

// runExecute 按备份模式执行 SQL，返回逐条记录
func (s *backupThreadIDSuite) runExecute(t *testing.T, sqlText string) []session.Record {
	se := session.NewInception()
	se.SetSessionManager(&noOpSessionManager{oscProcs: map[string]*util.OscProcessInfo{}})
	if err := se.LoadOptions(session.SourceOptions{
		Host:           s.host,
		Port:           s.port,
		User:           s.user,
		Password:       s.password,
		Backup:         true,
		IgnoreWarnings: true,
	}); err != nil {
		t.Fatalf("load options failed: %v", err)
	}

	records, err := se.RunExecute(context.Background(), sqlText)
	if err != nil {
		t.Fatalf("run execute failed: %v", err)
	}
	return records
}

// findRecord 按 SQL 关键字找到对应记录
func findRecord(t *testing.T, records []session.Record, keyword string) *session.Record {
	for i := range records {
		if strings.Contains(strings.ToLower(records[i].Sql), keyword) {
			return &records[i]
		}
	}
	t.Fatalf("record with keyword %q not found in %d records", keyword, len(records))
	return nil
}

// TestBackupDMLThreadIDConsistency 验证整条备份链路：
// DML 在独占连接上执行并取线程号 → binlog 解析按线程号过滤 → 校验通过。
// 校验通过本身就意味着线程号与 binlog 事件一致（checkFilter 要求完全相等）。
func TestBackupDMLThreadIDConsistency(t *testing.T) {
	s := newBackupThreadIDSuite(t)
	// t.Cleanup 按注册逆序执行：先 drop 表，最后关闭连接
	t.Cleanup(s.close)
	s.loadIncConfig(t)

	s.exec(t, "CREATE DATABASE IF NOT EXISTS test_inc")
	t.Cleanup(func() { s.exec(t, "DROP TABLE IF EXISTS test_inc.t_thread_id") })
	s.exec(t, "DROP TABLE IF EXISTS test_inc.t_thread_id")
	s.exec(t, "CREATE TABLE test_inc.t_thread_id(id int primary key, conn_id bigint unsigned not null)")

	// conn_id 列落库的是实际执行 INSERT 的连接号，用于交叉验证 record.ThreadId
	sqlText := "use test_inc;\ninsert into t_thread_id(id, conn_id) values (1, CONNECTION_ID());"

	records := s.runExecute(t, sqlText)
	record := findRecord(t, records, "insert into t_thread_id")

	if record.StageStatus != session.StatusBackupOK {
		t.Fatalf("expected Backup Successfully, got %q (errlevel=%d, message=%q)",
			session.StatusList[record.StageStatus], record.ErrLevel, record.ErrorMessage)
	}
	if record.ErrLevel == 2 {
		t.Fatalf("unexpected error: %q", record.ErrorMessage)
	}
	if record.ErrorMessage != "" {
		t.Fatalf("expected empty error message, got %q", record.ErrorMessage)
	}
	if record.OPID == "" {
		t.Fatal("expected non-empty opid")
	}
	if record.AffectedRows != 1 {
		t.Fatalf("expected 1 affected row, got %d", record.AffectedRows)
	}
	if record.ThreadId == 0 {
		t.Fatal("expected non-zero thread id")
	}

	// 直接证明线程号来自执行语句的连接：落库的 CONNECTION_ID() 必须等于 record.ThreadId。
	// 旧实现中线程号取自另一条池化连接的 select connection_id()，不保证该等式成立。
	var connID uint64
	if err := s.admin.QueryRow(
		"select conn_id from test_inc.t_thread_id where id = 1").Scan(&connID); err != nil {
		t.Fatalf("read back conn_id failed: %v", err)
	}
	if uint64(record.ThreadId) != connID {
		t.Fatalf("thread id mismatch: record.ThreadId=%d, executing connection id=%d",
			record.ThreadId, connID)
	}
	t.Logf("thread id %d consistent with executing connection", record.ThreadId)
}

// TestBackupDMLUseSwitch 验证独占连接上的会话状态对齐：
// 语句前 use 其他库后，不带库名前缀的 DML 仍应落在正确的库上执行。
func TestBackupDMLUseSwitch(t *testing.T) {
	s := newBackupThreadIDSuite(t)
	t.Cleanup(s.close)
	s.loadIncConfig(t)

	s.exec(t, "CREATE DATABASE IF NOT EXISTS test_inc2")
	s.exec(t, "CREATE TABLE IF NOT EXISTS test_inc2.t_use_switch(id int primary key, conn_id bigint unsigned not null)")
	t.Cleanup(func() {
		s.exec(t, "DROP TABLE IF EXISTS test_inc2.t_use_switch")
		s.exec(t, "DROP DATABASE IF EXISTS test_inc2")
	})
	s.exec(t, "DELETE FROM test_inc2.t_use_switch")

	sqlText := "use test_inc2;\ninsert into t_use_switch(id, conn_id) values (1, CONNECTION_ID());"
	records := s.runExecute(t, sqlText)
	record := findRecord(t, records, "insert into t_use_switch")

	if record.StageStatus != session.StatusBackupOK {
		t.Fatalf("expected Backup Successfully, got %q (errlevel=%d, message=%q)",
			session.StatusList[record.StageStatus], record.ErrLevel, record.ErrorMessage)
	}

	var count int
	if err := s.admin.QueryRow("select count(*) from test_inc2.t_use_switch").Scan(&count); err != nil {
		t.Fatalf("count rows failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row in test_inc2.t_use_switch, got %d (USE sync broken?)", count)
	}
}

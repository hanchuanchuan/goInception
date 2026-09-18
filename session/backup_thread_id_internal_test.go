package session

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
	"github.com/hanchuanchuan/goInception/util"
	repllog "github.com/siddontang/go-log/log"
	log "github.com/sirupsen/logrus"
)

// 本文件从内部直接驱动 s.exec / s.fetchThreadID / s.execDMLWithThreadID，
// 用确定性的方式复现"连接池换连导致备份校验线程号错配"的问题。
//
// 复现原理（依赖 database/sql 的池语义：连接取出取 freeConn[last]，
// 归还 append 到尾部，即 LIFO）：
//  1. 把池里的空闲连接全部扣在池外（held）；
//  2. 执行 DML —— 池内无空闲连接，必然落在新开的连接 X 上；
//  3. 归还一条被扣住的连接 A —— A 成为 freeConn[last]；
//  4. 旧的 fetchThreadID() 经池查询 select connection_id() ——
//     必然落在 A 上，返回 A 的线程号，而不是执行 DML 的 X。
//
// 由于第 2 步执行期间 A 被扣在池外，A 不可能是执行连接，错配是必然的。
// 这正是线上 "Backup failed 无错误信息" 误报的根因（binlog 校验按线程号
// 过滤事件，线程号来自另一条连接时匹配不到变更行）。

type internalTestEnv struct {
	host     string
	port     int
	user     string
	password string

	admin *gosql.DB
}

func newInternalTestEnv(t *testing.T) *internalTestEnv {
	host := envOrDefaultInternal("GOI_TEST_MYSQL_HOST", "127.0.0.1")
	port := envOrDefaultInternal("GOI_TEST_MYSQL_PORT", "3306")
	user := envOrDefaultInternal("GOI_TEST_MYSQL_USER", "test")
	password := envOrDefaultInternal("GOI_TEST_MYSQL_PASSWORD", "test")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?timeout=3s", user, password, host, port)
	admin, err := gosql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("open mysql failed: %v", err)
	}
	if err := admin.Ping(); err != nil {
		admin.Close()
		t.Skipf("mysql not available at %s:%s (set GOI_TEST_MYSQL_* to enable): %v", host, port, err)
	}

	var portNum int
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		t.Fatalf("invalid port %q: %v", port, err)
	}
	return &internalTestEnv{host: host, port: portNum, user: user, password: password, admin: admin}
}

func envOrDefaultInternal(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// newInternalSession 构造一个完成建连（checkOptions）的内部会话
func newInternalSession(t *testing.T, env *internalTestEnv) *session {
	log.SetLevel(log.ErrorLevel)
	repllog.SetLevel(repllog.LevelFatal)

	cfg := config.GetGlobalConfig()
	_, localFile, _, _ := runtime.Caller(0)
	dir := path.Dir(localFile)
	configFile := path.Join(dir[0:len(dir)-len("session")], "config/config.toml.default")
	if err := cfg.Load(configFile); err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	se := NewInception()
	se.SetSessionManager(&internalNoOpSessionManager{oscProcs: map[string]*util.OscProcessInfo{}})
	if err := se.LoadOptions(SourceOptions{
		Host:     env.host,
		Port:     env.port,
		User:     env.user,
		Password: env.password,
	}); err != nil {
		t.Fatalf("load options failed: %v", err)
	}
	// 与 RunExecute 的初始化顺序保持一致：init() 后 checkOptions() 建连
	se.init()
	if err := se.checkOptions(); err != nil {
		t.Fatalf("check options failed: %v", err)
	}
	return se
}

type internalNoOpSessionManager struct {
	mu       sync.RWMutex
	oscProcs map[string]*util.OscProcessInfo
}

func (m *internalNoOpSessionManager) ShowProcessList() map[uint64]util.ProcessInfo { return nil }
func (m *internalNoOpSessionManager) Kill(connectionID uint64, query bool)         {}
func (m *internalNoOpSessionManager) AddOscProcess(p *util.OscProcessInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oscProcs[p.Sqlsha1] = p
}
func (m *internalNoOpSessionManager) ShowOscProcessList() map[string]*util.OscProcessInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.oscProcs
}
func (m *internalNoOpSessionManager) ShowOscProcessListWithWrite() map[string]*util.OscProcessInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.oscProcs
}
func (m *internalNoOpSessionManager) OscLock()                           { m.mu.Lock() }
func (m *internalNoOpSessionManager) OscUnLock()                         { m.mu.Unlock() }
func (m *internalNoOpSessionManager) KillOscProcess(connectionID uint64) {}

// drainIdleConns 把池里的空闲连接全部扣在池外并返回。
func drainIdleConns(t *testing.T, db *gosql.DB) []*gosql.Conn {
	var held []*gosql.Conn
	for db.Stats().Idle > 0 {
		c, err := db.Conn(context.Background())
		if err != nil {
			t.Fatalf("grab idle conn failed: %v", err)
		}
		held = append(held, c)
	}
	return held
}

// execConnID 落库用的表：exec_conn 列记录真正执行该语句的连接号
const rotationTestTable = "test_inc.t_rotation"

func setupRotationTable(t *testing.T, env *internalTestEnv) {
	if _, err := env.admin.Exec("CREATE DATABASE IF NOT EXISTS test_inc"); err != nil {
		t.Fatalf("create database failed: %v", err)
	}
	if _, err := env.admin.Exec("DROP TABLE IF EXISTS " + rotationTestTable); err != nil {
		t.Fatalf("drop table failed: %v", err)
	}
	if _, err := env.admin.Exec(
		"CREATE TABLE " + rotationTestTable + "(id int primary key, exec_conn bigint unsigned not null)"); err != nil {
		t.Fatalf("create table failed: %v", err)
	}
}

func readbackExecConn(t *testing.T, env *internalTestEnv, id int) uint64 {
	var execConn uint64
	if err := env.admin.QueryRow(
		fmt.Sprintf("select exec_conn from %s where id = %d", rotationTestTable, id)).Scan(&execConn); err != nil {
		t.Fatalf("read back exec conn failed: %v", err)
	}
	return execConn
}

// TestLegacyFetchThreadIDPoolRotationMismatch 确定性复现旧实现的错配：
// DML 经 s.exec 落在连接 X 上，随后 s.fetchThreadID() 经池查询取到的是
// 另一条被归还的连接 A，返回的线程号属于 A 而非 X。
//
// 若该用例失败（线程号恰好一致），说明 database/sql 的取连接顺序语义
// 发生变化，本复现方式需要随之调整。
func TestLegacyFetchThreadIDPoolRotationMismatch(t *testing.T) {
	env := newInternalTestEnv(t)
	defer env.admin.Close()
	setupRotationTable(t, env)
	se := newInternalSession(t, env)

	db := se.db.DB()
	held := drainIdleConns(t, db)
	if len(held) == 0 {
		t.Fatal("expected at least one idle conn to hold")
	}

	// 旧实现第一步：DML 经池执行（空闲连接已扣空，必然落在新连接 X 上）
	if _, err := se.exec(
		fmt.Sprintf("insert into %s(id, exec_conn) values (1, CONNECTION_ID())", rotationTestTable),
		false); err != nil {
		t.Fatalf("legacy exec failed: %v", err)
	}

	// 归还一条被扣住的连接 A：A 成为池中最后归还的连接，
	// 后续池化查询按 LIFO 语义必然取到它
	held[0].Close()
	for _, c := range held[1:] {
		c.Close()
	}

	// 旧实现第二步：另起池化查询取线程号
	se.threadID = 0 // fetchThreadID 有会话级缓存，先清空
	legacyTid := se.fetchThreadID()

	execConn := readbackExecConn(t, env, 1)
	if legacyTid == 0 {
		t.Fatal("fetchThreadID returned 0")
	}
	if uint64(legacyTid) == execConn {
		t.Fatalf("预期复现线程号错配，但两者一致 (tid=%d)：连接池取连接顺序语义可能已变化，需调整复现方式",
			legacyTid)
	}
	t.Logf("复现成功: fetchThreadID=%d 来自另一条池化连接, 实际执行 DML 的连接为 %d — 线程号错配",
		legacyTid, execConn)
}

// TestExecDMLWithThreadIDPoolRotationConsistency 在同样的强制换连条件下
// 验证新实现：execDMLWithThreadID 在同一条独占连接上执行并取号，
// 线程号与实际执行连接恒一致。
func TestExecDMLWithThreadIDPoolRotationConsistency(t *testing.T) {
	env := newInternalTestEnv(t)
	defer env.admin.Close()
	setupRotationTable(t, env)
	se := newInternalSession(t, env)

	db := se.db.DB()
	// 同样的强制换连条件：扣住全部空闲连接，DML 只能落在新连接上
	held := drainIdleConns(t, db)

	_, err, tid := se.execDMLWithThreadID(
		fmt.Sprintf("insert into %s(id, exec_conn) values (2, CONNECTION_ID())", rotationTestTable))
	if err != nil {
		t.Fatalf("execDMLWithThreadID failed: %v", err)
	}
	if tid == 0 {
		t.Fatal("execDMLWithThreadID returned 0 thread id")
	}

	// 归还被扣住的连接（新实现下这不影响结果：执行与取号已经
	// 在同一条连接上完成）
	for _, c := range held {
		c.Close()
	}

	execConn := readbackExecConn(t, env, 2)
	if uint64(tid) != execConn {
		t.Fatalf("线程号与执行连接不一致: execDMLWithThreadID=%d, 实际执行连接=%d", tid, execConn)
	}
	t.Logf("新实现一致: execDMLWithThreadID=%d == 实际执行连接=%d（独占连接上执行并取号）",
		tid, execConn)
}

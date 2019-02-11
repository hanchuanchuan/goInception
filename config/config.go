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
	"crypto/tls"
	"crypto/x509"
	// "fmt"
	"io/ioutil"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/hanchuanchuan/goInception/util/logutil"
	"github.com/pingcap/errors"
	tracing "github.com/uber/jaeger-client-go/config"
)

// Config number limitations
const (
	MaxLogFileSize = 4096 // MB
)

// Valid config maps
var (
	ValidStorage = map[string]bool{
		"mocktikv": true,
		"tikv":     true,
	}
	// checkTableBeforeDrop enable to execute `admin check table` before `drop table`.
	CheckTableBeforeDrop = false
	// checkBeforeDropLDFlag is a go build flag.
	checkBeforeDropLDFlag = "None"
)

// Config contains configuration options.
type Config struct {
	Host             string          `toml:"host" json:"host"`
	AdvertiseAddress string          `toml:"advertise-address" json:"advertise-address"`
	Port             uint            `toml:"port" json:"port"`
	Store            string          `toml:"store" json:"store"`
	Path             string          `toml:"path" json:"path"`
	Socket           string          `toml:"socket" json:"socket"`
	Lease            string          `toml:"lease" json:"lease"`
	RunDDL           bool            `toml:"run-ddl" json:"run-ddl"`
	SplitTable       bool            `toml:"split-table" json:"split-table"`
	TokenLimit       uint            `toml:"token-limit" json:"token-limit"`
	OOMAction        string          `toml:"oom-action" json:"oom-action"`
	MemQuotaQuery    int64           `toml:"mem-quota-query" json:"mem-quota-query"`
	EnableStreaming  bool            `toml:"enable-streaming" json:"enable-streaming"`
	TxnLocalLatches  TxnLocalLatches `toml:"txn-local-latches" json:"txn-local-latches"`
	// Set sys variable lower-case-table-names, ref: https://dev.mysql.com/doc/refman/5.7/en/identifier-case-sensitivity.html.
	// TODO: We actually only support mode 2, which keeps the original case, but the comparison is case-insensitive.
	LowerCaseTableNames int `toml:"lower-case-table-names" json:"lower-case-table-names"`

	Log                 Log               `toml:"log" json:"log"`
	Security            Security          `toml:"security" json:"security"`
	Status              Status            `toml:"status" json:"status"`
	Performance         Performance       `toml:"performance" json:"performance"`
	XProtocol           XProtocol         `toml:"xprotocol" json:"xprotocol"`
	PreparedPlanCache   PreparedPlanCache `toml:"prepared-plan-cache" json:"prepared-plan-cache"`
	OpenTracing         OpenTracing       `toml:"opentracing" json:"opentracing"`
	ProxyProtocol       ProxyProtocol     `toml:"proxy-protocol" json:"proxy-protocol"`
	TiKVClient          TiKVClient        `toml:"tikv-client" json:"tikv-client"`
	Binlog              Binlog            `toml:"binlog" json:"binlog"`
	Inc                 Inc               `toml:"inc" json:"inc"`
	Osc                 Osc               `toml:"osc" json:"osc"`
	CompatibleKillQuery bool              `toml:"compatible-kill-query" json:"compatible-kill-query"`
}

// Log is the log section of config.
type Log struct {
	// Log level.
	Level string `toml:"level" json:"level"`
	// Log format. one of json, text, or console.
	Format string `toml:"format" json:"format"`
	// Disable automatic timestamps in output.
	DisableTimestamp bool `toml:"disable-timestamp" json:"disable-timestamp"`
	// File log config.
	File logutil.FileLogConfig `toml:"file" json:"file"`

	SlowQueryFile      string `toml:"slow-query-file" json:"slow-query-file"`
	SlowThreshold      uint   `toml:"slow-threshold" json:"slow-threshold"`
	ExpensiveThreshold uint   `toml:"expensive-threshold" json:"expensive-threshold"`
	QueryLogMaxLen     uint   `toml:"query-log-max-len" json:"query-log-max-len"`
}

// Security is the security section of the config.
type Security struct {
	SkipGrantTable bool   `toml:"skip-grant-table" json:"skip-grant-table"`
	SSLCA          string `toml:"ssl-ca" json:"ssl-ca"`
	SSLCert        string `toml:"ssl-cert" json:"ssl-cert"`
	SSLKey         string `toml:"ssl-key" json:"ssl-key"`
	ClusterSSLCA   string `toml:"cluster-ssl-ca" json:"cluster-ssl-ca"`
	ClusterSSLCert string `toml:"cluster-ssl-cert" json:"cluster-ssl-cert"`
	ClusterSSLKey  string `toml:"cluster-ssl-key" json:"cluster-ssl-key"`
}

// ToTLSConfig generates tls's config based on security section of the config.
func (s *Security) ToTLSConfig() (*tls.Config, error) {
	var tlsConfig *tls.Config
	if len(s.ClusterSSLCA) != 0 {
		var certificates = make([]tls.Certificate, 0)
		if len(s.ClusterSSLCert) != 0 && len(s.ClusterSSLKey) != 0 {
			// Load the client certificates from disk
			certificate, err := tls.LoadX509KeyPair(s.ClusterSSLCert, s.ClusterSSLKey)
			if err != nil {
				return nil, errors.Errorf("could not load client key pair: %s", err)
			}
			certificates = append(certificates, certificate)
		}

		// Create a certificate pool from the certificate authority
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(s.ClusterSSLCA)
		if err != nil {
			return nil, errors.Errorf("could not read ca certificate: %s", err)
		}

		// Append the certificates from the CA
		if !certPool.AppendCertsFromPEM(ca) {
			return nil, errors.New("failed to append ca certs")
		}

		tlsConfig = &tls.Config{
			Certificates: certificates,
			RootCAs:      certPool,
		}
	}

	return tlsConfig, nil
}

// Status is the status section of the config.
type Status struct {
	ReportStatus    bool   `toml:"report-status" json:"report-status"`
	StatusPort      uint   `toml:"status-port" json:"status-port"`
	MetricsAddr     string `toml:"metrics-addr" json:"metrics-addr"`
	MetricsInterval uint   `toml:"metrics-interval" json:"metrics-interval"`
}

// Performance is the performance section of the config.
type Performance struct {
	MaxProcs            uint    `toml:"max-procs" json:"max-procs"`
	TCPKeepAlive        bool    `toml:"tcp-keep-alive" json:"tcp-keep-alive"`
	CrossJoin           bool    `toml:"cross-join" json:"cross-join"`
	StatsLease          string  `toml:"stats-lease" json:"stats-lease"`
	RunAutoAnalyze      bool    `toml:"run-auto-analyze" json:"run-auto-analyze"`
	StmtCountLimit      uint    `toml:"stmt-count-limit" json:"stmt-count-limit"`
	FeedbackProbability float64 `toml:"feedback-probability" json:"feedback-probability"`
	QueryFeedbackLimit  uint    `toml:"query-feedback-limit" json:"query-feedback-limit"`
	PseudoEstimateRatio float64 `toml:"pseudo-estimate-ratio" json:"pseudo-estimate-ratio"`
	ForcePriority       string  `toml:"force-priority" json:"force-priority"`
}

// XProtocol is the XProtocol section of the config.
type XProtocol struct {
	XServer bool   `toml:"xserver" json:"xserver"`
	XHost   string `toml:"xhost" json:"xhost"`
	XPort   uint   `toml:"xport" json:"xport"`
	XSocket string `toml:"xsocket" json:"xsocket"`
}

// PlanCache is the PlanCache section of the config.
type PlanCache struct {
	Enabled  bool `toml:"enabled" json:"enabled"`
	Capacity uint `toml:"capacity" json:"capacity"`
	Shards   uint `toml:"shards" json:"shards"`
}

// TxnLocalLatches is the TxnLocalLatches section of the config.
type TxnLocalLatches struct {
	Enabled  bool `toml:"enabled" json:"enabled"`
	Capacity uint `toml:"capacity" json:"capacity"`
}

// PreparedPlanCache is the PreparedPlanCache section of the config.
type PreparedPlanCache struct {
	Enabled  bool `toml:"enabled" json:"enabled"`
	Capacity uint `toml:"capacity" json:"capacity"`
}

// OpenTracing is the opentracing section of the config.
type OpenTracing struct {
	Enable     bool                `toml:"enable" json:"enbale"`
	Sampler    OpenTracingSampler  `toml:"sampler" json:"sampler"`
	Reporter   OpenTracingReporter `toml:"reporter" json:"reporter"`
	RPCMetrics bool                `toml:"rpc-metrics" json:"rpc-metrics"`
}

// OpenTracingSampler is the config for opentracing sampler.
// See https://godoc.org/github.com/uber/jaeger-client-go/config#SamplerConfig
type OpenTracingSampler struct {
	Type                    string        `toml:"type" json:"type"`
	Param                   float64       `toml:"param" json:"param"`
	SamplingServerURL       string        `toml:"sampling-server-url" json:"sampling-server-url"`
	MaxOperations           int           `toml:"max-operations" json:"max-operations"`
	SamplingRefreshInterval time.Duration `toml:"sampling-refresh-interval" json:"sampling-refresh-interval"`
}

// OpenTracingReporter is the config for opentracing reporter.
// See https://godoc.org/github.com/uber/jaeger-client-go/config#ReporterConfig
type OpenTracingReporter struct {
	QueueSize           int           `toml:"queue-size" json:"queue-size"`
	BufferFlushInterval time.Duration `toml:"buffer-flush-interval" json:"buffer-flush-interval"`
	LogSpans            bool          `toml:"log-spans" json:"log-spans"`
	LocalAgentHostPort  string        `toml:"local-agent-host-port" json:"local-agent-host-port"`
}

// ProxyProtocol is the PROXY protocol section of the config.
type ProxyProtocol struct {
	// PROXY protocol acceptable client networks.
	// Empty string means disable PROXY protocol,
	// * means all networks.
	Networks string `toml:"networks" json:"networks"`
	// PROXY protocol header read timeout, Unit is second.
	HeaderTimeout uint `toml:"header-timeout" json:"header-timeout"`
}

// TiKVClient is the config for tikv client.
type TiKVClient struct {
	// GrpcConnectionCount is the max gRPC connections that will be established
	// with each tikv-server.
	GrpcConnectionCount uint `toml:"grpc-connection-count" json:"grpc-connection-count"`
	// After a duration of this time in seconds if the client doesn't see any activity it pings
	// the server to see if the transport is still alive.
	GrpcKeepAliveTime uint `toml:"grpc-keepalive-time" json:"grpc-keepalive-time"`
	// After having pinged for keepalive check, the client waits for a duration of Timeout in seconds
	// and if no activity is seen even after that the connection is closed.
	GrpcKeepAliveTimeout uint `toml:"grpc-keepalive-timeout" json:"grpc-keepalive-timeout"`
	// CommitTimeout is the max time which command 'commit' will wait.
	CommitTimeout string `toml:"commit-timeout" json:"commit-timeout"`
}

// Binlog is the config for binlog.
type Binlog struct {
	BinlogSocket string `toml:"binlog-socket" json:"binlog-socket"`
	WriteTimeout string `toml:"write-timeout" json:"write-timeout"`
	// If IgnoreError is true, when writing binlog meets error, TiDB would
	// ignore the error.
	IgnoreError bool `toml:"ignore-error" json:"ignore-error"`
}

// Inc is the inception section of the config.
type Inc struct {

	// character-set-server = utf8
	// inception_check_autoincrement_datatype = 1
	// inception_check_autoincrement_init_value = 1
	// inception_check_autoincrement_name = 1
	// inception_check_column_comment = 'OFF'
	// inception_check_column_default_value = 0
	// inception_check_dml_limit = 1
	// inception_check_dml_orderby = 1
	// inception_check_dml_where = 'OFF'
	// inception_check_identifier = 1
	// inception_check_index_prefix = 0
	// inception_check_insert_field = 'OFF'
	// inception_check_primary_key = 'OFF'
	// inception_check_table_comment = 'OFF'
	// inception_check_timestamp_default = 0
	// inception_enable_autoincrement_unsigned = 1
	// inception_enable_blob_type = 0
	// inception_enable_nullable = 'ON'
	// inception_enable_column_charset = 0
	// inception_support_charset = utf8mb4

	BackupHost     string `toml:"backup_host" json:"backup_host"` // 远程备份库信息
	BackupPassword string `toml:"backup_password" json:"backup_password"`
	BackupPort     uint   `toml:"backup_port" json:"backup_port"`
	BackupUser     string `toml:"backup_user" json:"backup_user"`

	CheckAutoIncrementDataType  bool `toml:"check_autoincrement_datatype" json:"check_autoincrement_datatype"`
	CheckAutoIncrementInitValue bool `toml:"check_autoincrement_init_value" json:"check_autoincrement_init_value"`
	CheckAutoIncrementName      bool `toml:"check_autoincrement_name" json:"check_autoincrement_name"`
	CheckColumnComment          bool `toml:"check_column_comment" json:"check_column_comment"`
	CheckColumnDefaultValue     bool `toml:"check_column_default_value" json:"check_column_default_value"`
	CheckDMLLimit               bool `toml:"check_dml_limit" json:"check_dml_limit"`
	CheckDMLOrderBy             bool `toml:"check_dml_orderby" json:"check_dml_orderby"`
	CheckDMLWhere               bool `toml:"check_dml_where" json:"check_dml_where"`
	CheckIdentifier             bool `toml:"check_identifier" json:"check_identifier"`
	CheckIndexPrefix            bool `toml:"check_index_prefix" json:"check_index_prefix"`
	CheckInsertField            bool `toml:"check_insert_field" json:"check_insert_field"`
	CheckPrimaryKey             bool `toml:"check_primary_key" json:"check_primary_key"`
	CheckTableComment           bool `toml:"check_table_comment" json:"check_table_comment"`
	CheckTimestampDefault       bool `toml:"check_timestamp_default" json:"check_timestamp_default"`

	EnableAutoIncrementUnsigned bool `toml:"enable_autoincrement_unsigned" json:"enable_autoincrement_unsigned"`
	EnableBlobType              bool `toml:"enable_blob_type" json:"enable_blob_type"`
	EnableColumnCharset         bool `toml:"enable_column_charset" json:"enable_column_charset"`
	EnableDropTable             bool `toml:"enable_drop_table" json:"enable_drop_table"` // 允许删除表
	EnableEnumSetBit            bool `toml:"enable_enum_set_bit" json:"enable_enum_set_bit"`
	EnableForeignKey            bool `toml:"enable_foreign_key" json:"enable_foreign_key"`
	EnableIdentiferKeyword      bool `toml:"enable_identifer_keyword" json:"enable_identifer_keyword"`
	EnableNotInnodb             bool `toml:"enable_not_innodb" json:"enable_not_innodb"`
	EnableNullable              bool `toml:"enable_nullable" json:"enable_nullable"` // 允许空列
	EnableOrderByRand           bool `toml:"enable_orderby_rand" json:"enable_orderby_rand"`
	EnablePartitionTable        bool `toml:"enable_partition_table" json:"enable_partition_table"`
	EnablePKColumnsOnlyInt      bool `toml:"enable_pk_columns_only_int" json:"enable_pk_columns_only_int"`
	EnableSelectStar            bool `toml:"enable_select_star" json:"enable_select_star"`

	MaxCharLength uint `toml:"max_char_length" json:"max_char_length"`
	MaxKeys       uint `toml:"max_keys" json:"max_keys"`
	MaxKeyParts   uint `toml:"max_key_parts" json:"max_key_parts"`
	MaxUpdateRows uint `toml:"max_update_rows" json:"max_update_rows"`

	MaxPrimaryKeyParts uint `toml:"max_primary_key_parts" json:"max_primary_key_parts"` // 主键最多允许有几列组合
	MergeAlterTable    bool `toml:"merge_alter_table" json:"merge_alter_table"`

	// 安全更新是否开启.
	// -1 表示不做操作,基于远端数据库 [默认值]
	// 0  表示关闭安全更新
	// 1  表示开启安全更新
	SqlSafeUpdates int `toml:"sql_safe_updates" json:"sql_safe_updates"`
}

// Osc online schema change 工具参数配置
type Osc struct {

	// 用来设置在arkit返回结果集中，对于原来OSC在执行过程的标准输出信息是不是要打印到结果集对应的错误信息列中，
	// 如果设置为1，就不打印，如果设置为0，就打印。而如果出现了错误，则都会打印。默认值：OFF
	OscPrintNone bool `toml:"osc_print_none" json:"osc_print_none"`

	// 对应参数pt-online-schema-change中的参数--print。默认值：OFF
	OscPrintSql bool `toml:"osc_print_sql" json:"osc_print_sql"`

	// 全局的OSC开关，默认是打开的，如果想要关闭则设置为OFF，这样就会直接修改。默认值：OFF
	OscOn bool `toml:"osc_on" json:"osc_on"`

	// 这个参数实际上是一个OSC开关，如果设置为0，则全部ALTER语句都使用OSC方式，
	// 如果设置为非0，则当这个表占用空间大小大于这个值时才使用OSC方式。
	// 单位为M，这个表大小的计算方式是通过语句
	// select (DATA_LENGTH + INDEX_LENGTH)/1024/1024 from information_schema.tables
	// where table_schema = 'dbname' and table_name = 'tablename' 来实现的。默认值：16
	// [0-1048576]
	OscMinTableSize uint `toml:"osc_min_table_size" json:"osc_min_table_size"`

	// 对应参数pt-online-schema-change中的参数alter-foreign-keys-method，具体意义可以参考OSC官方手册。默认值：none
	// [auto | none | rebuild_constraints | drop_swap]
	OscAlterForeignKeysMethod string `toml:"osc_alter_foreign_keys_method" json:"osc_alter_foreign_keys_method"`

	// 对应参数pt-online-schema-change中的参数recursion_method，具体意义可以参考OSC官方手册。默认值：processlist
	// [processlist | hosts | none]
	OscRecursionMethod string `toml:"osc_recursion_method" json:"osc_recursion_method"`

	// 对应参数pt-online-schema-change中的参数--max-lag。默认值：3
	OscMaxLag int `toml:"osc_max_lag" json:"osc_max_lag"`

	// 对应参数pt-online-schema-change中的参数--[no]check-alter。默认值：ON
	OscCheckAlter bool `toml:"osc_check_alter" json:"osc_check_alter"`

	// 对应参数pt-online-schema-change中的参数--[no]check-replication-filters。默认值：ON
	OscCheckReplicationFilters bool `toml:"osc_check_replication_filters" json:"osc_check_replication_filters"`

	// 对应参数pt-online-schema-change中的参数--[no]drop-old-table。默认值：ON
	oscDropOldTable bool `toml:"osc_drop_old_table" json:"osc_drop_old_table"`

	// 对应参数pt-online-schema-change中的参数--[no]drop-new-table。默认值：ON
	OscDropNewTable bool `toml:"osc_drop_new_table" json:"osc_drop_new_table"`

	// 对应参数pt-online-schema-change中的参数--max-load中的thread_running部分。默认值：80
	OscMaxRunning int `toml:"osc_max_running" json:"osc_max_running"`

	// 对应参数pt-online-schema-change中的参数--max-load中的thread_connected部分。默认值：1000
	OscMaxConnected int `toml:"osc_max_connected" json:"osc_max_connected"`

	// 对应参数pt-online-schema-change中的参数--critical-load中的thread_running部分。默认值：80
	OscCriticalRunning int `toml:"osc_critical_running" json:"osc_critical_running"`

	// 对应参数pt-online-schema-change中的参数--critical-load中的thread_connected部分。默认值：1000
	OscCriticalConnected int `toml:"osc_critical_connected" json:"osc_critical_connected"`

	// 对应参数pt-online-schema-change中的参数--chunk-time。默认值：1
	OscChunkTime int `toml:"osc_chunk_time" json:"osc_chunk_time"`

	// 对应参数pt-online-schema-change中的参数--chunk-size-limit。默认值：4
	OscChunkSizeLimit int `toml:"osc_chunk_size_limit" json:"osc_chunk_size_limit"`

	// 对应参数pt-online-schema-change中的参数--chunk-size。默认值：4
	OscChunkSize int `toml:"osc_chunk_size" json:"osc_chunk_size"`

	// 对应参数pt-online-schema-change中的参数--check-interval，意义是Sleep time between checks for --max-lag。默认值：5
	OscCheckInterval int `toml:"osc_check_interval" json:"osc_check_interval"`

	OscBinDir string `toml:"osc_bin_dir" json:"osc_bin_dir"`
}

var defaultConf = Config{
	Host:             "0.0.0.0",
	AdvertiseAddress: "",
	Port:             4000,
	Store:            "mocktikv",
	Path:             "/tmp/tidb",
	RunDDL:           true,
	SplitTable:       true,
	Lease:            "45s",
	TokenLimit:       1000,
	OOMAction:        "log",
	MemQuotaQuery:    32 << 30,
	EnableStreaming:  false,
	TxnLocalLatches: TxnLocalLatches{
		Enabled:  true,
		Capacity: 2048000,
	},
	LowerCaseTableNames: 2,
	Log: Log{
		Level:  "info",
		Format: "text",
		File: logutil.FileLogConfig{
			LogRotate: true,
			MaxSize:   logutil.DefaultLogMaxSize,
		},
		SlowThreshold:      300,
		ExpensiveThreshold: 10000,
		QueryLogMaxLen:     2048,
	},
	Status: Status{
		ReportStatus:    false,
		StatusPort:      10080,
		MetricsInterval: 15,
	},
	Performance: Performance{
		TCPKeepAlive:        false,
		CrossJoin:           true,
		StatsLease:          "3s",
		RunAutoAnalyze:      false,
		StmtCountLimit:      5000,
		FeedbackProbability: 0.05,
		QueryFeedbackLimit:  1024,
		PseudoEstimateRatio: 0.8,
		ForcePriority:       "NO_PRIORITY",
	},
	XProtocol: XProtocol{
		XHost: "",
		XPort: 0,
	},
	ProxyProtocol: ProxyProtocol{
		Networks:      "",
		HeaderTimeout: 5,
	},
	PreparedPlanCache: PreparedPlanCache{
		Enabled:  false,
		Capacity: 100,
	},
	OpenTracing: OpenTracing{
		Enable: false,
		Sampler: OpenTracingSampler{
			Type:  "const",
			Param: 1.0,
		},
		Reporter: OpenTracingReporter{},
	},
	TiKVClient: TiKVClient{
		GrpcConnectionCount:  16,
		GrpcKeepAliveTime:    10,
		GrpcKeepAliveTimeout: 3,
		CommitTimeout:        "41s",
	},
	Binlog: Binlog{
		WriteTimeout: "15s",
	},
	// 默认跳过权限校验 2019-1-26
	Security: Security{
		SkipGrantTable: true,
	},
	Inc: Inc{
		EnableNullable:     true,
		EnableDropTable:    false,
		CheckTableComment:  false,
		CheckColumnComment: false,
		SqlSafeUpdates:     -1,
	},
	Osc: Osc{
		OscPrintNone:               false,
		OscPrintSql:                false,
		OscOn:                      false,
		OscMinTableSize:            16,
		OscAlterForeignKeysMethod:  "none",
		OscRecursionMethod:         "processlist",
		OscMaxLag:                  3,
		OscCheckAlter:              true,
		OscCheckReplicationFilters: true,
		oscDropOldTable:            true,
		OscDropNewTable:            true,
		OscMaxRunning:              80,
		OscMaxConnected:            1000,
		OscCriticalRunning:         80,
		OscCriticalConnected:       1000,
		OscChunkTime:               1,
		OscChunkSizeLimit:          4,
		OscChunkSize:               4,
		OscCheckInterval:           5,
		OscBinDir:                  "/usr/local/bin",
	},
}

var globalConf = defaultConf

// NewConfig creates a new config instance with default value.
func NewConfig() *Config {
	conf := defaultConf
	return &conf
}

// GetGlobalConfig returns the global configuration for this server.
// It should store configuration from command line and configuration file.
// Other parts of the system can read the global configuration use this function.
func GetGlobalConfig() *Config {
	return &globalConf
}

// Load loads config options from a toml file.
func (c *Config) Load(confFile string) error {
	_, err := toml.DecodeFile(confFile, c)
	if c.TokenLimit <= 0 {
		c.TokenLimit = 1000
	}
	return errors.Trace(err)
}

// ToLogConfig converts *Log to *logutil.LogConfig.
func (l *Log) ToLogConfig() *logutil.LogConfig {
	return &logutil.LogConfig{
		Level:            l.Level,
		Format:           l.Format,
		DisableTimestamp: l.DisableTimestamp,
		File:             l.File,
		SlowQueryFile:    l.SlowQueryFile,
	}
}

// ToTracingConfig converts *OpenTracing to *tracing.Configuration.
func (t *OpenTracing) ToTracingConfig() *tracing.Configuration {
	ret := &tracing.Configuration{
		Disabled:   !t.Enable,
		RPCMetrics: t.RPCMetrics,
		Reporter:   &tracing.ReporterConfig{},
		Sampler:    &tracing.SamplerConfig{},
	}
	ret.Reporter.QueueSize = t.Reporter.QueueSize
	ret.Reporter.BufferFlushInterval = t.Reporter.BufferFlushInterval
	ret.Reporter.LogSpans = t.Reporter.LogSpans
	ret.Reporter.LocalAgentHostPort = t.Reporter.LocalAgentHostPort

	ret.Sampler.Type = t.Sampler.Type
	ret.Sampler.Param = t.Sampler.Param
	ret.Sampler.SamplingServerURL = t.Sampler.SamplingServerURL
	ret.Sampler.MaxOperations = t.Sampler.MaxOperations
	ret.Sampler.SamplingRefreshInterval = t.Sampler.SamplingRefreshInterval
	return ret
}

func init() {
	if checkBeforeDropLDFlag == "1" {
		CheckTableBeforeDrop = true
	}
}

// The following constants represents the valid action configurations for OOMAction.
// NOTE: Althrough the values is case insensitiv, we should use lower-case
// strings because the configuration value will be transformed to lower-case
// string and compared with these constants in the further usage.
const (
	OOMActionCancel = "cancel"
	OOMActionLog    = "log"
)

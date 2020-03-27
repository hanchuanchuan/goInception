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
	"strings"

	// "fmt"
	"io/ioutil"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/hanchuanchuan/goInception/mysql"
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
	PreparedPlanCache   PreparedPlanCache `toml:"prepared-plan-cache" json:"prepared-plan-cache"`
	OpenTracing         OpenTracing       `toml:"opentracing" json:"opentracing"`
	ProxyProtocol       ProxyProtocol     `toml:"proxy-protocol" json:"proxy-protocol"`
	TiKVClient          TiKVClient        `toml:"tikv-client" json:"tikv-client"`
	Binlog              Binlog            `toml:"binlog" json:"binlog"`
	Inc                 Inc               `toml:"inc" json:"inc"`
	Osc                 Osc               `toml:"osc" json:"osc"`
	Ghost               Ghost             `toml:"ghost" json:"ghost"`
	IncLevel            IncLevel          `toml:"inc_level" json:"inc_level"`
	CompatibleKillQuery bool              `toml:"compatible-kill-query" json:"compatible-kill-query"`

	// 是否跳过用户权限校验
	SkipGrantTable bool `toml:"skip_grant_table" json:"skip_grant_table"`
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
	SkipGrantTable bool   `toml:"skip_grant_table" json:"skip_grant_table"`
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
	ReportStatus bool `toml:"report-status" json:"report-status"`
	StatusPort   uint `toml:"status-port" json:"status-port"`
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
	BackupHost     string `toml:"backup_host" json:"backup_host"` // 远程备份库信息
	BackupPassword string `toml:"backup_password" json:"backup_password"`
	BackupPort     uint   `toml:"backup_port" json:"backup_port"`
	BackupUser     string `toml:"backup_user" json:"backup_user"`

	CheckAutoIncrementDataType  bool `toml:"check_autoincrement_datatype" json:"check_autoincrement_datatype"`
	CheckAutoIncrementInitValue bool `toml:"check_autoincrement_init_value" json:"check_autoincrement_init_value"`
	CheckAutoIncrementName      bool `toml:"check_autoincrement_name" json:"check_autoincrement_name"`
	CheckColumnComment          bool `toml:"check_column_comment" json:"check_column_comment"`
	CheckColumnDefaultValue     bool `toml:"check_column_default_value" json:"check_column_default_value"`
	// 检查列顺序变更 #40
	CheckColumnPositionChange bool `toml:"check_column_position_change" json:"check_column_position_change"`
	// 检查列类型变更(允许长度变更,类型变更时警告)
	CheckColumnTypeChange       bool `toml:"check_column_type_change" json:"check_column_type_change"`
	CheckDMLLimit               bool `toml:"check_dml_limit" json:"check_dml_limit"`
	CheckDMLOrderBy             bool `toml:"check_dml_orderby" json:"check_dml_orderby"`
	CheckDMLWhere               bool `toml:"check_dml_where" json:"check_dml_where"`
	CheckIdentifier             bool `toml:"check_identifier" json:"check_identifier"`
	CheckImplicitTypeConversion bool `toml:"check_implicit_type_conversion"` // 检查where条件中的隐式类型转换
	CheckIndexPrefix            bool `toml:"check_index_prefix" json:"check_index_prefix"`
	CheckInsertField            bool `toml:"check_insert_field" json:"check_insert_field"`
	CheckPrimaryKey             bool `toml:"check_primary_key" json:"check_primary_key"`
	CheckTableComment           bool `toml:"check_table_comment" json:"check_table_comment"`
	CheckTimestampDefault       bool `toml:"check_timestamp_default" json:"check_timestamp_default"`
	CheckTimestampCount         bool `toml:"check_timestamp_count" json:"check_timestamp_count"`

	EnableTimeStampType  bool `toml:"enable_timestamp_type" json:"enable_timestamp_type"`
	EnableZeroDate       bool `toml:"enable_zero_date" json:"enable_zero_date"`
	CheckDatetimeDefault bool `toml:"check_datetime_default" json:"check_datetime_default"`
	CheckDatetimeCount   bool `toml:"check_datetime_count" json:"check_datetime_count"`

	// 将 float/double 转成 decimal, 默认为 false
	CheckFloatDouble bool `toml:"check_float_double" json:"check_float_double"`

	CheckIdentifierUpper bool `toml:"check_identifier_upper" json:"check_identifier_upper"`

	// 连接服务器的默认字符集,默认值为utf8mb4
	DefaultCharset              string `toml:"default_charset" json:"default_charset"`
	EnableAutoIncrementUnsigned bool   `toml:"enable_autoincrement_unsigned" json:"enable_autoincrement_unsigned"`
	// 允许blob,text,json列设置为NOT NULL
	EnableBlobNotNull   bool `toml:"enable_blob_not_null" json:"enable_blob_not_null"`
	EnableBlobType      bool `toml:"enable_blob_type" json:"enable_blob_type"`
	EnableChangeColumn  bool `toml:"enable_change_column" json:"enable_change_column"` // 允许change column操作
	EnableColumnCharset bool `toml:"enable_column_charset" json:"enable_column_charset"`
	EnableDropDatabase  bool `toml:"enable_drop_database" json:"enable_drop_database"`
	EnableDropTable     bool `toml:"enable_drop_table" json:"enable_drop_table"` // 允许删除表
	EnableEnumSetBit    bool `toml:"enable_enum_set_bit" json:"enable_enum_set_bit"`

	// DML指纹功能,开启后,在审核时,类似DML将直接复用审核结果,可大幅优化审核效率
	EnableFingerprint      bool `toml:"enable_fingerprint" json:"enable_fingerprint"`
	EnableForeignKey       bool `toml:"enable_foreign_key" json:"enable_foreign_key"`
	EnableIdentiferKeyword bool `toml:"enable_identifer_keyword" json:"enable_identifer_keyword"`
	EnableJsonType         bool `toml:"enable_json_type" json:"enable_json_type"`
	// 是否启用自定义审核级别设置
	// EnableLevel bool `toml:"enable_level" json:"enable_level"`
	// 是否启用最小化回滚SQL设置,当开启时,update语句中未变更的值不再记录到回滚语句中
	EnableMinimalRollback bool `toml:"enable_minimal_rollback" json:"enable_minimal_rollback"`
	// 是否允许指定存储引擎
	EnableSetEngine        bool `toml:"enable_set_engine" json:"enable_set_engine"`
	EnableNullable         bool `toml:"enable_nullable" json:"enable_nullable"`               // 允许空列
	EnableNullIndexName    bool `toml:"enable_null_index_name" json:"enable_null_index_name"` //是否允许不指定索引名
	EnableOrderByRand      bool `toml:"enable_orderby_rand" json:"enable_orderby_rand"`
	EnablePartitionTable   bool `toml:"enable_partition_table" json:"enable_partition_table"`
	EnablePKColumnsOnlyInt bool `toml:"enable_pk_columns_only_int" json:"enable_pk_columns_only_int"`
	EnableSelectStar       bool `toml:"enable_select_star" json:"enable_select_star"`

	// 是否允许设置字符集和排序规则
	EnableSetCharset   bool `toml:"enable_set_charset" json:"enable_set_charset"`
	EnableSetCollation bool `toml:"enable_set_collation" json:"enable_set_collation"`
	// 开启sql统计
	EnableSqlStatistic bool `toml:"enable_sql_statistic" json:"enable_sql_statistic"`

	// explain判断受影响行数时使用的规则, 默认值"first"
	// 可选值: "first", "max"
	// 		"first": 	使用第一行的explain结果作为受影响行数
	// 		"max": 		使用explain结果中的最大值作为受影响行数
	ExplainRule string `toml:"explain_rule" json:"explain_rule"`

	// 全量日志
	GeneralLog bool `toml:"general_log" json:"general_log"`
	// 使用十六进制表示法转储二进制列
	// 受影响的数据类型为BINARY，VARBINARY，BLOB类型
	HexBlob bool `toml:"hex_blob" json:"hex_blob"`

	// 表名/索引名前缀，为空时不作限制
	IndexPrefix     string `toml:"index_prefix" json:"index_prefix"`
	UniqIndexPrefix string `toml:"uniq_index_prefix" json:"uniq_index_prefix"`
	TablePrefix     string `toml:"table_prefix" json:"table_prefix"`

	Lang string `toml:"lang" json:"lang"`
	// 连接服务器允许的最大包大小,以字节为单位 默认值为4194304(即4MB)
	MaxAllowedPacket uint `toml:"max_allowed_packet" json:"max_allowed_packet"`
	MaxCharLength    uint `toml:"max_char_length" json:"max_char_length"`

	// DDL操作最大允许的受影响行数. 默认值0,即不限制
	MaxDDLAffectRows uint `toml:"max_ddl_affect_rows" json:"max_ddl_affect_rows"`

	// 一次最多写入的行数, 仅判断insert values语法
	MaxInsertRows uint `toml:"max_insert_rows" json:"max_insert_rows"`

	MaxKeys       uint `toml:"max_keys" json:"max_keys"`
	MaxKeyParts   uint `toml:"max_key_parts" json:"max_key_parts"`
	MaxUpdateRows uint `toml:"max_update_rows" json:"max_update_rows"`

	MaxPrimaryKeyParts uint `toml:"max_primary_key_parts" json:"max_primary_key_parts"` // 主键最多允许有几列组合
	MergeAlterTable    bool `toml:"merge_alter_table" json:"merge_alter_table"`

	// 建表必须创建的列. 可指定多个列,以逗号分隔.列类型可选. 格式: 列名 [列类型,可选],...
	MustHaveColumns string `toml:"must_have_columns" json:"must_have_columns"`
	// 如果表包含以下列，列必须有索引。可指定多个列,以逗号分隔.列类型可选.   格式: 列名 [列类型,可选],...
	ColumnsMustHaveIndex string `toml:"columns_must_have_index" json:"columns_must_have_index"`

	// 是否跳过用户权限校验
	SkipGrantTable bool `toml:"skip_grant_table" json:"skip_grant_table"`
	// 要跳过的sql语句, 多个时以分号分隔
	SkipSqls string `toml:"skip_sqls" json:"skip_sqls"`

	// 安全更新是否开启.
	// -1 表示不做操作,基于远端数据库 [默认值]
	// 0  表示关闭安全更新
	// 1  表示开启安全更新
	SqlSafeUpdates int `toml:"sql_safe_updates" json:"sql_safe_updates"`

	// 支持的字符集
	SupportCharset string `toml:"support_charset" json:"support_charset"`

	// 支持的排序规则
	SupportCollation string `toml:"support_collation" json:"support_collation"`
	// Version *string

	// 支持的存储引擎,多个时以分号分隔
	SupportEngine string `toml:"support_engine" json:"support_engine"`
	// 远端数据库等待超时时间，单位:秒
	WaitTimeout int `toml:"wait_timeout" json:"wait_timeout"`

	// 版本信息
	Version string `toml:"version" json:"version"`
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

	// 类似--max-lag，检查集群暂停流量控制所花费的平均时间（仅适用于PXC 5.6及以上版本）
	OscMaxFlowCtl int `toml:"osc_max_flow_ctl" json:"osc_max_flow_ctl"`

	// 对应参数pt-online-schema-change中的参数--[no]check-alter。默认值：ON
	OscCheckAlter bool `toml:"osc_check_alter" json:"osc_check_alter"`

	// 对应参数pt-online-schema-change中的参数--[no]check-replication-filters。默认值：ON
	OscCheckReplicationFilters bool `toml:"osc_check_replication_filters" json:"osc_check_replication_filters"`
	// 是否检查唯一索引,默认检查,如果是,则禁止
	OscCheckUniqueKeyChange bool `toml:"osc_check_unique_key_change" json:"osc_check_unique_key_change"`

	// 对应参数pt-online-schema-change中的参数--[no]drop-old-table。默认值：ON
	OscDropOldTable bool `toml:"osc_drop_old_table" json:"osc_drop_old_table"`

	// 对应参数pt-online-schema-change中的参数--[no]drop-new-table。默认值：ON
	OscDropNewTable bool `toml:"osc_drop_new_table" json:"osc_drop_new_table"`

	// 对应参数pt-online-schema-change中的参数--max-load中的thread_running部分。默认值：80
	OscMaxThreadRunning int `toml:"osc_max_thread_running" json:"osc_max_thread_running"`

	// 对应参数pt-online-schema-change中的参数--max-load中的thread_connected部分。默认值：1000
	OscMaxThreadConnected int `toml:"osc_max_thread_connected" json:"osc_max_thread_connected"`

	// 对应参数pt-online-schema-change中的参数--critical-load中的thread_running部分。默认值：80
	OscCriticalThreadRunning int `toml:"osc_critical_thread_running" json:"osc_critical_thread_running"`

	// 对应参数pt-online-schema-change中的参数--critical-load中的thread_connected部分。默认值：1000
	OscCriticalThreadConnected int `toml:"osc_critical_thread_connected" json:"osc_critical_thread_connected"`

	// 对应参数pt-online-schema-change中的参数--chunk-time。默认值：1
	OscChunkTime float32 `toml:"osc_chunk_time" json:"osc_chunk_time"`

	// 对应参数pt-online-schema-change中的参数--chunk-size-limit。默认值：4
	OscChunkSizeLimit int `toml:"osc_chunk_size_limit" json:"osc_chunk_size_limit"`

	// 对应参数pt-online-schema-change中的参数--chunk-size。默认值：1000
	OscChunkSize int `toml:"osc_chunk_size" json:"osc_chunk_size"`

	// 对应参数pt-online-schema-change中的参数--check-interval，意义是Sleep time between checks for --max-lag。默认值：5
	OscCheckInterval int `toml:"osc_check_interval" json:"osc_check_interval"`

	OscBinDir string `toml:"osc_bin_dir" json:"osc_bin_dir"`
}

// GhOst online schema change 工具参数配置
type Ghost struct {

	// 阿里云数据库
	GhostAliyunRds bool `toml:"ghost_aliyun_rds"`
	// 允许gh-ost运行在双主复制架构中，一般与-assume-master-host参数一起使用。
	GhostAllowMasterMaster bool `toml:"ghost_allow_master_master"`
	// 允许gh-ost在数据迁移(migrate)依赖的唯一键可以为NULL，默认为不允许为NULL的唯一键。如果数据迁移(migrate)依赖的唯一键允许NULL值，则可能造成数据不正确，请谨慎使用。
	GhostAllowNullableUniqueKey bool `toml:"ghost_allow_nullable_unique_key"`
	// 允许gh-ost直接运行在主库上。默认gh-ost连接的从库。
	GhostAllowOnMaster bool `toml:"ghost_allow_on_master"`

	// 如果你修改一个列的名字(如change column)，gh-ost将会识别到并且需要提供重命名列名的原因，默认情况下gh-ost是不继续执行的，除非提供-approve-renamed-columns ALTER。
	// ALTER
	GhostApproveRenamedColumns bool `toml:"ghost_approve_renamed_columns"`
	// 为gh-ost指定一个主库，格式为"ip:port"或者"hostname:port"。默认推荐gh-ost连接从库。
	GhostAssumeMasterHost string `toml:"ghost_assume_master_host"`
	// 确认gh-ost连接的数据库实例的binlog_format=ROW的情况下，可以指定-assume-rbr，
	// 这样可以禁止从库上运行stop slave,start slave,执行gh-ost用户也不需要SUPER权限。
	GhostAssumeRbr bool `toml:"ghost_assume_rbr"`

	// 该参数如果为True(默认值)，则进行row-copy之后，估算统计行数(使用explain select count(*)方式)，
	// 并调整ETA时间，否则，gh-ost首先预估统计行数，然后开始row-copy。
	GhostConcurrentRowcount bool `toml:"ghost_concurrent_rowcount"`

	// 一系列逗号分隔的status-name=values组成，当MySQL中status超过对应的values，gh-ost将会退出
	// 	e.g:
	// -critical-load Threads_connected=20,Connections=1500
	// 指的是当MySQL中的状态值Threads_connected>20,Connections>1500的时候，gh-ost将会由于该数据库严重负载而停止并退出。
	// GhostCriticalLoad string `toml:"ghost_critical_load"`

	// 当值为0时，当达到-critical-load，gh-ost立即退出。当值不为0时，当达到-critical-load，
	// gh-ost会在-critical-load-interval-millis秒数后，再次进行检查，再次检查依旧达到-critical-load，gh-ost将会退出。
	GhostCriticalLoadIntervalMillis   int64 `toml:"ghost_critical_load_interval_millis"`
	GhostCriticalLoadHibernateSeconds int64 `toml:"ghost_critical_load_hibernate_seconds"`

	// 选择cut-over类型:atomic/two-step，atomic(默认)类型的cut-over是github的算法，two-step采用的是facebook-OSC的算法。
	GhostCutOver string `toml:"ghost_cut_over"`

	GhostCutOverExponentialBackoff bool `toml:"ghost_cut_over_exponential_backoff"`
	// 在每次迭代中处理的行数量(允许范围：100-100000)，默认值为1000。
	GhostChunkSize int64 `toml:"ghost_chunk_size"`

	// gh-ost在cut-over阶段最大的锁等待时间，当锁超时时，gh-ost的cut-over将重试。(默认值：3)
	GhostCutOverLockTimeoutSeconds int64 `toml:"ghost_cut_over_lock_timeout_seconds"`

	// 很危险的参数，慎用！
	// 该参数针对一个有外键的表，在gh-ost创建ghost表时，并不会为ghost表创建外键。该参数很适合用于删除外键，除此之外，请谨慎使用。
	GhostDiscardForeignKeys bool `toml:"ghost_discard_foreign_keys"`

	// 各种操作在panick前重试次数。(默认为60)
	GhostDefaultRetries int64 `toml:"ghost_default_retries"`

	GhostDmlBatchSize int64 `toml:"ghost_dml_batch_size"`

	// 准确统计表行数(使用select count(*)的方式)，得到更准确的预估时间。
	GhostExactRowcount bool `toml:"ghost_exact_rowcount"`

	// 实际执行alter&migrate表，默认为不执行，仅仅做测试并退出，如果想要ALTER TABLE语句真正落实到数据库中去，需要明确指定-execute
	// GhostExecute bool `toml:"ghost_execute"`

	GhostExponentialBackoffMaxInterval int64 `toml:"ghost_exponential_backoff_max_interval"`
	// When true, the 'unpostpone|cut-over' interactive command must name the migrated table。
	GhostForceTableNames   string `toml:"ghost_force_table_names"`
	GhostForceNamedCutOver bool   `toml:"ghost_force_named_cut_over"`
	GhostGcp               bool   `toml:"ghost_gcp"`

	// gh-ost心跳频率值，默认为500。
	GhostHeartbeatIntervalMillis int64 `toml:"ghost_heartbeat_interval_millis"`

	// gh-ost操作之前，检查并删除已经存在的ghost表。该参数不建议使用，请手动处理原来存在的ghost表。默认不启用该参数，gh-ost直接退出操作。
	GhostInitiallyDropGhostTable bool `toml:"ghost_initially_drop_ghost_table"`

	// gh-ost操作之前，检查并删除已经存在的旧表。该参数不建议使用，请手动处理原来存在的ghost表。默认不启用该参数，gh-ost直接退出操作。
	GhostInitiallyDropOldTable bool `toml:"ghost_initially_drop_old_table"`

	// gh-ost强制删除已经存在的socket文件。该参数不建议使用，可能会删除一个正在运行的gh-ost程序，导致DDL失败。
	GhostInitiallyDropSocketFile bool `toml:"ghost_initially_drop_socket_file"`

	// 主从复制最大延迟时间，当主从复制延迟时间超过该值后，gh-ost将采取节流(throttle)措施，默认值：1500s。
	GhostMaxLagMillis int64 `toml:"ghost_max_lag_millis"`

	// 	一系列逗号分隔的status-name=values组成，当MySQL中status超过对应的values，gh-ost将采取节流(throttle)措施。
	// e.g:
	// -max-load Threads_connected=20,Connections=1500
	// 指的是当MySQL中的状态值Threads_connected>20,Connections>1500的时候，gh-ost将采取节流(throttle)措施。
	// GhostMaxLoad string `toml:"ghost_max_load"`

	GhostNiceRatio float64 `toml:"ghost_nice_ratio"`
	// gh-ost的数据迁移(migrate)运行在从库上，而不是主库上。
	// Have the migration run on a replica, not on the master.
	// This will do the full migration on the replica including cut-over (as opposed to --test-on-replica)
	// GhostMigrateOnReplica bool `toml:"ghost_migrate_on_replica"`

	// 开启标志
	GhostOn bool `toml:"ghost_on"`
	// gh-ost操作结束后，删除旧表，默认状态是不删除旧表，会存在_tablename_del表。
	GhostOkToDropTable bool `toml:"ghost_ok_to_drop_table"`
	// 当这个文件存在的时候，gh-ost的cut-over阶段将会被推迟，直到该文件被删除。
	GhostPostponeCutOverFlagFile string `toml:"ghost_postpone_cut_over_flag_file"`
	// GhostPanicFlagFile           string `toml:"ghost_panic_flag_file"`
	// GhostReplicaServerID      bool `toml:"ghost_replica_server_id"`
	GhostSkipForeignKeyChecks bool `toml:"ghost_skip_foreign_key_checks"`

	// 如果你修改一个列的名字(如change column)，gh-ost将会识别到并且需要提供重命名列名的原因，
	// 默认情况下gh-ost是不继续执行的。该参数告诉gh-ost跳该列的数据迁移，
	// 让gh-ost把重命名列作为无关紧要的列。该操作很危险，你会损失该列的所有值。
	// GhostSkipRenamedColumns bool `toml:"ghost_skip_renamed_columns"`

	// GhostSsl              bool `toml:"ghost_ssl"`
	// GhostSslAllowInsecure bool `toml:"ghost_ssl_allow_insecure"`
	// GhostSslCa            bool `toml:"ghost_ssl_ca"`

	// 在从库上测试gh-ost，包括在从库上数据迁移(migration)，数据迁移完成后stop slave，
	// 原表和ghost表立刻交换而后立刻交换回来。继续保持stop slave，使你可以对比两张表。
	// GhostTestOnReplica bool `toml:"ghost_test_on_replica"`

	// 当-test-on-replica执行时，该参数表示该过程中不用stop slave。
	// GhostTestOnReplicaSkipReplicaStop bool `toml:"ghost_test_on_replica_skip_replica_stop"`

	// 	列出所有需要被检查主从复制延迟的从库。
	// e.g:
	// -throttle-control-replica=192.16.12.22:3306,192.16.12.23:3307,192.16.13.12:3308
	GhostThrottleControlReplicas    string `toml:"ghost_throttle_control_replicas"`
	GhostThrottleHTTP               string `toml:"ghost_throttle_http"`
	GhostTimestampOldTable          bool   `toml:"ghost_timestamp_old_table"`
	GhostThrottleQuery              string `toml:"ghost_throttle_query"`
	GhostThrottleFlagFile           string `toml:"ghost_throttle_flag_file"`
	GhostThrottleAdditionalFlagFile string `toml:"ghost_throttle_additional_flag_file"`

	// 告诉gh-ost你正在运行的是一个tungsten-replication拓扑结构。
	GhostTungsten            bool   `toml:"ghost_tungsten"`
	GhostReplicationLagQuery string `toml:"ghost_replication_lag_query"`
}

type IncLevel struct {
	ER_ALTER_TABLE_ONCE             int8 `toml:"er_alter_table_once"`
	ER_AUTO_INCR_ID_WARNING         int8 `toml:"er_auto_incr_id_warning"`
	ER_AUTOINC_UNSIGNED             int8 `toml:"er_autoinc_unsigned"`
	ER_BLOB_CANT_HAVE_DEFAULT       int8 `toml:"er_blob_cant_have_default"`
	ER_CANT_SET_CHARSET             int8 `toml:"er_cant_set_charset"`
	ER_CANT_SET_COLLATION           int8 `toml:"er_cant_set_collation"`
	ER_CANT_SET_ENGINE              int8 `toml:"er_cant_set_engine"`
	ER_CHANGE_COLUMN_TYPE           int8 `toml:"er_change_column_type"`
	ER_CHANGE_TOO_MUCH_ROWS         int8 `toml:"er_change_too_much_rows"`
	ER_CHAR_TO_VARCHAR_LEN          int8 `toml:"er_char_to_varchar_len"`
	ER_CHARSET_ON_COLUMN            int8 `toml:"er_charset_on_column"`
	ER_COLUMN_HAVE_NO_COMMENT       int8 `toml:"er_column_have_no_comment"`
	ER_DATETIME_DEFAULT             int8 `toml:"er_datetime_default"`
	ER_FOREIGN_KEY                  int8 `toml:"er_foreign_key"`
	ER_IDENT_USE_KEYWORD            int8 `toml:"er_ident_use_keyword"`
	ER_INC_INIT_ERR                 int8 `toml:"er_inc_init_err"`
	ER_INDEX_NAME_IDX_PREFIX        int8 `toml:"er_index_name_idx_prefix"`
	ER_INDEX_NAME_UNIQ_PREFIX       int8 `toml:"er_index_name_uniq_prefix"`
	ER_INSERT_TOO_MUCH_ROWS         int8 `toml:"er_insert_too_much_rows"`
	ER_INVALID_DATA_TYPE            int8 `toml:"er_invalid_data_type"`
	ER_INVALID_IDENT                int8 `toml:"er_invalid_ident"`
	ER_MUST_HAVE_COLUMNS            int8 `toml:"er_must_have_columns"`
	ErrColumnsMustHaveIndex         int8 `toml:"er_columns_must_have_index"`
	ErrColumnsMustHaveIndexTypeErr  int8 `toml:"er_columns_must_have_index_type_err"`
	ER_NO_WHERE_CONDITION           int8 `toml:"er_no_where_condition"`
	ER_NOT_ALLOWED_NULLABLE         int8 `toml:"er_not_allowed_nullable"`
	ER_ORDERY_BY_RAND               int8 `toml:"er_ordery_by_rand"`
	ER_PARTITION_NOT_ALLOWED        int8 `toml:"er_partition_not_allowed"`
	ER_PK_COLS_NOT_INT              int8 `toml:"er_pk_cols_not_int"`
	ER_PK_TOO_MANY_PARTS            int8 `toml:"er_pk_too_many_parts"`
	ER_SELECT_ONLY_STAR             int8 `toml:"er_select_only_star"`
	ER_SET_DATA_TYPE_INT_BIGINT     int8 `toml:"er_set_data_type_int_bigint"`
	ER_TABLE_CHARSET_MUST_NULL      int8 `toml:"er_table_charset_must_null"`
	ER_TABLE_CHARSET_MUST_UTF8      int8 `toml:"er_table_charset_must_utf8"`
	ER_TABLE_MUST_HAVE_COMMENT      int8 `toml:"er_table_must_have_comment"`
	ER_TABLE_MUST_HAVE_PK           int8 `toml:"er_table_must_have_pk"`
	ER_TABLE_PREFIX                 int8 `toml:"er_table_prefix"`
	ER_TEXT_NOT_NULLABLE_ERROR      int8 `toml:"er_text_not_nullable_error"`
	ER_TIMESTAMP_DEFAULT            int8 `toml:"er_timestamp_default"`
	ER_TOO_MANY_KEY_PARTS           int8 `toml:"er_too_many_key_parts"`
	ER_TOO_MANY_KEYS                int8 `toml:"er_too_many_keys"`
	ER_TOO_MUCH_AUTO_DATETIME_COLS  int8 `toml:"er_too_much_auto_datetime_cols"`
	ER_TOO_MUCH_AUTO_TIMESTAMP_COLS int8 `toml:"er_too_much_auto_timestamp_cols"`
	ER_UDPATE_TOO_MUCH_ROWS         int8 `toml:"er_udpate_too_much_rows"`
	ER_USE_ENUM                     int8 `toml:"er_use_enum"`
	ER_USE_TEXT_OR_BLOB             int8 `toml:"er_use_text_or_blob"`
	ER_WITH_DEFAULT_ADD_COLUMN      int8 `toml:"er_with_default_add_column"`
	ER_WITH_INSERT_FIELD            int8 `toml:"er_with_insert_field"`
	ER_WITH_LIMIT_CONDITION         int8 `toml:"er_with_limit_condition"`
	ER_WITH_ORDERBY_CONDITION       int8 `toml:"er_with_orderby_condition"`
	ErCantChangeColumn              int8 `toml:"er_cant_change_column"`
	ErCantChangeColumnPosition      int8 `toml:"er_cant_change_column_position"`
	ErJsonTypeSupport               int8 `toml:"er_json_type_support"`
	ErrImplicitTypeConversion       int8 `toml:"er_implicit_type_conversion"`
	ErrJoinNoOnCondition            int8 `toml:"er_join_no_on_condition"`
	ErrUseValueExpr                 int8 `toml:"er_use_value_expr"`
	ErrWrongAndExpr                 int8 `toml:"er_wrong_and_expr"`
}

var defaultConf = Config{
	Host:             "0.0.0.0",
	AdvertiseAddress: "",
	Port:             4000,
	Store:            "mocktikv",
	Path:             "/tmp/tidb",
	// 关闭ddl线程(不能直接使用该设置,在main中启动服务后自动关闭)
	RunDDL:     true,
	SplitTable: true,
	// 更新远程架构的时间
	Lease:           "0s",
	TokenLimit:      1000,
	OOMAction:       "log",
	MemQuotaQuery:   32 << 30,
	EnableStreaming: false,
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
		ReportStatus: false,
		StatusPort:   10080,
	},
	Performance: Performance{
		TCPKeepAlive: false,
		CrossJoin:    true,
		// 设置0s时关闭统计信息更新
		StatsLease:     "0s",
		RunAutoAnalyze: false,
		StmtCountLimit: 5000,
		// 统计直方图采样率,为0时不作采样
		FeedbackProbability: 0.0,
		// 内存最大采样数
		QueryFeedbackLimit:  0,
		PseudoEstimateRatio: 0.8,
		ForcePriority:       "NO_PRIORITY",
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
	// 为配置方便,在config节点也添加相同参数
	SkipGrantTable: true,
	Security: Security{
		SkipGrantTable: true,
	},
	Inc: Inc{
		EnableZeroDate:        true,
		EnableNullable:        true,
		EnableDropTable:       false,
		EnableSetEngine:       true,
		CheckTableComment:     false,
		CheckColumnComment:    false,
		EnableChangeColumn:    true,
		CheckTimestampCount:   true,
		EnableTimeStampType:   true,
		CheckFloatDouble:      false,
		CheckIdentifierUpper:  false,
		SqlSafeUpdates:        -1,
		SupportCharset:        "utf8,utf8mb4",
		SupportEngine:         "innodb",
		Lang:                  "en-US",
		CheckColumnTypeChange: true,

		// 连接服务器选项
		DefaultCharset:   "utf8mb4",
		MaxAllowedPacket: 4194304,
		ExplainRule:      "first",

		// 为配置方便,在config节点也添加相同参数
		SkipGrantTable: true,

		// 默认参数不指定,避免test时失败
		// Version:            mysql.TiDBReleaseVersion,

		IndexPrefix:     "idx_",  // 默认不检查,由CheckIndexPrefix控制
		UniqIndexPrefix: "uniq_", // 默认不检查,由CheckIndexPrefix控制
		TablePrefix:     "",      // 默认不检查表前缀
	},
	Osc: Osc{
		OscPrintNone:               false,
		OscPrintSql:                false,
		OscOn:                      false,
		OscMinTableSize:            16,
		OscAlterForeignKeysMethod:  "none",
		OscRecursionMethod:         "processlist",
		OscMaxLag:                  3,
		OscMaxFlowCtl:              -1,
		OscCheckAlter:              true,
		OscCheckReplicationFilters: true,
		OscCheckUniqueKeyChange:    true,
		OscDropOldTable:            true,
		OscDropNewTable:            true,
		OscMaxThreadRunning:        80,
		OscMaxThreadConnected:      1000,
		OscCriticalThreadRunning:   80,
		OscCriticalThreadConnected: 1000,
		OscChunkTime:               1,
		OscChunkSizeLimit:          4,
		OscChunkSize:               1000,
		OscCheckInterval:           5,
		OscBinDir:                  "/usr/local/bin",
	},
	Ghost: Ghost{
		GhostOn:                            false,
		GhostAllowOnMaster:                 true,
		GhostAssumeRbr:                     true,
		GhostChunkSize:                     1000,
		GhostConcurrentRowcount:            true,
		GhostCutOver:                       "atomic",
		GhostCutOverLockTimeoutSeconds:     3,
		GhostDefaultRetries:                60,
		GhostHeartbeatIntervalMillis:       500,
		GhostMaxLagMillis:                  1500,
		GhostApproveRenamedColumns:         true,
		GhostExponentialBackoffMaxInterval: 64,
		GhostDmlBatchSize:                  10,
		GhostOkToDropTable:                 true,
		GhostSkipForeignKeyChecks:          true,
	},
	IncLevel: IncLevel{
		ER_ALTER_TABLE_ONCE:             1,
		ER_AUTO_INCR_ID_WARNING:         1,
		ER_AUTOINC_UNSIGNED:             1,
		ER_BLOB_CANT_HAVE_DEFAULT:       1,
		ER_CANT_SET_CHARSET:             1,
		ER_CANT_SET_COLLATION:           1,
		ER_CANT_SET_ENGINE:              1,
		ER_CHANGE_COLUMN_TYPE:           1,
		ER_CHANGE_TOO_MUCH_ROWS:         1,
		ER_CHAR_TO_VARCHAR_LEN:          1,
		ER_CHARSET_ON_COLUMN:            1,
		ER_COLUMN_HAVE_NO_COMMENT:       1,
		ER_DATETIME_DEFAULT:             1,
		ER_FOREIGN_KEY:                  2,
		ER_IDENT_USE_KEYWORD:            1,
		ER_INC_INIT_ERR:                 1,
		ER_INDEX_NAME_IDX_PREFIX:        1,
		ER_INDEX_NAME_UNIQ_PREFIX:       1,
		ER_INSERT_TOO_MUCH_ROWS:         1,
		ER_INVALID_DATA_TYPE:            1,
		ER_INVALID_IDENT:                1,
		ER_MUST_HAVE_COLUMNS:            1,
		ErrColumnsMustHaveIndex:         1,
		ErrColumnsMustHaveIndexTypeErr:  1,
		ER_NO_WHERE_CONDITION:           1,
		ER_NOT_ALLOWED_NULLABLE:         1,
		ER_ORDERY_BY_RAND:               1,
		ER_PARTITION_NOT_ALLOWED:        1,
		ER_PK_COLS_NOT_INT:              1,
		ER_PK_TOO_MANY_PARTS:            1,
		ER_SELECT_ONLY_STAR:             1,
		ER_SET_DATA_TYPE_INT_BIGINT:     2,
		ER_TABLE_CHARSET_MUST_NULL:      1,
		ER_TABLE_CHARSET_MUST_UTF8:      1,
		ER_TABLE_MUST_HAVE_COMMENT:      1,
		ER_TABLE_MUST_HAVE_PK:           1,
		ER_TABLE_PREFIX:                 1,
		ER_TEXT_NOT_NULLABLE_ERROR:      1,
		ER_TIMESTAMP_DEFAULT:            1,
		ER_TOO_MANY_KEY_PARTS:           1,
		ER_TOO_MANY_KEYS:                1,
		ER_TOO_MUCH_AUTO_DATETIME_COLS:  2,
		ER_TOO_MUCH_AUTO_TIMESTAMP_COLS: 2,
		ER_UDPATE_TOO_MUCH_ROWS:         1,
		ER_USE_ENUM:                     1,
		ER_USE_TEXT_OR_BLOB:             2,
		ER_WITH_DEFAULT_ADD_COLUMN:      1,
		ER_WITH_INSERT_FIELD:            1,
		ER_WITH_LIMIT_CONDITION:         1,
		ER_WITH_ORDERBY_CONDITION:       1,
		ErCantChangeColumn:              1,
		ErCantChangeColumnPosition:      1,
		ErJsonTypeSupport:               2,
		ErrImplicitTypeConversion:       1,
		ErrJoinNoOnCondition:            1,
		ErrUseValueExpr:                 1,
		ErrWrongAndExpr:                 1,
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
	// 自动设置版本号
	globalConf.Inc.Version = strings.TrimRight(mysql.TiDBReleaseVersion, "-dirty")
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

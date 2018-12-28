// Copyright 2016 PingCAP, Inc.
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

package session

import (
	"bytes"
	"strings"

	"github.com/hanchuanchuan/tidb/ast"
	"github.com/hanchuanchuan/tidb/model"
	"github.com/hanchuanchuan/tidb/mysql"
	"github.com/hanchuanchuan/tidb/types"
	"github.com/hanchuanchuan/tidb/types/json"
	"github.com/hanchuanchuan/tidb/util/chunk"
	// log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	StageOK byte = iota
	StageCheck
	StageExec
)

const (
	StatusAuditOk byte = iota
	StatusExecFail
	StatusExecOK
	StatusBackupFail
	StatusBackupOK
)

var (
	stageList  = [3]string{"RERUN", "CHECKED", "EXECUTED"}
	statusList = [5]string{"Audit Completed", "Execute failed", "Execute Successfully",
		"Execute Successfully\nBackup failed", "Execute Successfully\nBackup Successfully"}
)

var Keywords = map[string]bool{
	"ACTION":                   true,
	"ADD":                      true,
	"ADDDATE":                  true,
	"ADMIN":                    true,
	"AFTER":                    true,
	"ALL":                      true,
	"ALGORITHM":                true,
	"ALTER":                    true,
	"ALWAYS":                   true,
	"ANALYZE":                  true,
	"AND":                      true,
	"ANY":                      true,
	"AS":                       true,
	"ASC":                      true,
	"ASCII":                    true,
	"AUTO_INCREMENT":           true,
	"AVG":                      true,
	"AVG_ROW_LENGTH":           true,
	"BEGIN":                    true,
	"BETWEEN":                  true,
	"BIGINT":                   true,
	"BINARY":                   true,
	"BINLOG":                   true,
	"BIT":                      true,
	"BIT_AND":                  true,
	"BIT_OR":                   true,
	"BIT_XOR":                  true,
	"BLOB":                     true,
	"BOOL":                     true,
	"BOOLEAN":                  true,
	"BOTH":                     true,
	"BTREE":                    true,
	"BUCKETS":                  true,
	"BY":                       true,
	"BYTE":                     true,
	"CANCEL":                   true,
	"CASCADE":                  true,
	"CASCADED":                 true,
	"CASE":                     true,
	"CAST":                     true,
	"CHANGE":                   true,
	"CHAR":                     true,
	"CHARACTER":                true,
	"CHARSET":                  true,
	"CHECK":                    true,
	"CHECKSUM":                 true,
	"CLEANUP":                  true,
	"CLIENT":                   true,
	"COALESCE":                 true,
	"COLLATE":                  true,
	"COLLATION":                true,
	"COLUMN":                   true,
	"COLUMNS":                  true,
	"COMMENT":                  true,
	"COMMIT":                   true,
	"INCEPTION":                true,
	"INCEPTION_MAGIC_START":    true,
	"INCEPTION_MAGIC_COMMIT":   true,
	"COMMITTED":                true,
	"COMPACT":                  true,
	"COMPRESSED":               true,
	"COMPRESSION":              true,
	"CONNECTION":               true,
	"CONSISTENT":               true,
	"CONSTRAINT":               true,
	"CONVERT":                  true,
	"COPY":                     true,
	"COUNT":                    true,
	"CREATE":                   true,
	"CROSS":                    true,
	"CURRENT_DATE":             true,
	"CURRENT_TIME":             true,
	"CURRENT_TIMESTAMP":        true,
	"CURRENT_USER":             true,
	"CURTIME":                  true,
	"DATA":                     true,
	"DATABASE":                 true,
	"DATABASES":                true,
	"DATE":                     true,
	"DATE_ADD":                 true,
	"DATE_SUB":                 true,
	"DATETIME":                 true,
	"DAY":                      true,
	"DAY_HOUR":                 true,
	"DAY_MICROSECOND":          true,
	"DAY_MINUTE":               true,
	"DAY_SECOND":               true,
	"DDL":                      true,
	"DEALLOCATE":               true,
	"DEC":                      true,
	"DECIMAL":                  true,
	"DEFAULT":                  true,
	"DEFINER":                  true,
	"DELAY_KEY_WRITE":          true,
	"DELAYED":                  true,
	"DELETE":                   true,
	"DESC":                     true,
	"DESCRIBE":                 true,
	"DISABLE":                  true,
	"DISTINCT":                 true,
	"DISTINCTROW":              true,
	"DIV":                      true,
	"DO":                       true,
	"DOUBLE":                   true,
	"DROP":                     true,
	"DUAL":                     true,
	"DUPLICATE":                true,
	"DYNAMIC":                  true,
	"ELSE":                     true,
	"ENABLE":                   true,
	"ENCLOSED":                 true,
	"END":                      true,
	"ENGINE":                   true,
	"ENGINES":                  true,
	"ENUM":                     true,
	"ESCAPE":                   true,
	"ESCAPED":                  true,
	"EVENT":                    true,
	"EVENTS":                   true,
	"EXCLUSIVE":                true,
	"EXECUTE":                  true,
	"EXISTS":                   true,
	"EXPLAIN":                  true,
	"EXTRACT":                  true,
	"FALSE":                    true,
	"FIELDS":                   true,
	"FIRST":                    true,
	"FIXED":                    true,
	"FLOAT":                    true,
	"FLUSH":                    true,
	"FOR":                      true,
	"FORCE":                    true,
	"FOREIGN":                  true,
	"FORMAT":                   true,
	"FROM":                     true,
	"FULL":                     true,
	"FULLTEXT":                 true,
	"FUNCTION":                 true,
	"GENERATED":                true,
	"GET_FORMAT":               true,
	"GLOBAL":                   true,
	"GRANT":                    true,
	"GRANTS":                   true,
	"GROUP":                    true,
	"GROUP_CONCAT":             true,
	"HASH":                     true,
	"HAVING":                   true,
	"HIGH_PRIORITY":            true,
	"HOUR":                     true,
	"HOUR_MICROSECOND":         true,
	"HOUR_MINUTE":              true,
	"HOUR_SECOND":              true,
	"IDENTIFIED":               true,
	"IF":                       true,
	"IGNORE":                   true,
	"IN":                       true,
	"INDEX":                    true,
	"INDEXES":                  true,
	"INFILE":                   true,
	"INNER":                    true,
	"INPLACE":                  true,
	"INSERT":                   true,
	"INT":                      true,
	"INT1":                     true,
	"INT2":                     true,
	"INT3":                     true,
	"INT4":                     true,
	"INT8":                     true,
	"INTEGER":                  true,
	"INTERVAL":                 true,
	"INTERNAL":                 true,
	"INTO":                     true,
	"INVOKER":                  true,
	"IS":                       true,
	"ISOLATION":                true,
	"JOBS":                     true,
	"JOB":                      true,
	"JOIN":                     true,
	"JSON":                     true,
	"KEY":                      true,
	"KEY_BLOCK_SIZE":           true,
	"KEYS":                     true,
	"KILL":                     true,
	"LEADING":                  true,
	"LEFT":                     true,
	"LESS":                     true,
	"LEVEL":                    true,
	"LIKE":                     true,
	"LIMIT":                    true,
	"LINES":                    true,
	"LOAD":                     true,
	"LOCAL":                    true,
	"LOCALTIME":                true,
	"LOCALTIMESTAMP":           true,
	"LOCK":                     true,
	"LONG":                     true,
	"LONGBLOB":                 true,
	"LONGTEXT":                 true,
	"LOW_PRIORITY":             true,
	"MASTER":                   true,
	"MAX":                      true,
	"MAX_CONNECTIONS_PER_HOUR": true,
	"MAX_EXECUTION_TIME":       true,
	"MAX_QUERIES_PER_HOUR":     true,
	"MAX_ROWS":                 true,
	"MAX_UPDATES_PER_HOUR":     true,
	"MAX_USER_CONNECTIONS":     true,
	"MAXVALUE":                 true,
	"MEDIUMBLOB":               true,
	"MEDIUMINT":                true,
	"MEDIUMTEXT":               true,
	"MERGE":                    true,
	"MICROSECOND":              true,
	"MIN":                      true,
	"MIN_ROWS":                 true,
	"MINUTE":                   true,
	"MINUTE_MICROSECOND":       true,
	"MINUTE_SECOND":            true,
	"MOD":                      true,
	"MODE":                     true,
	"MODIFY":                   true,
	"MONTH":                    true,
	"NAMES":                    true,
	"NATIONAL":                 true,
	"NATURAL":                  true,
	"NO":                       true,
	"NO_WRITE_TO_BINLOG":       true,
	"NONE":                     true,
	"NOT":                      true,
	"NOW":                      true,
	"NULL":                     true,
	"NUMERIC":                  true,
	"NVARCHAR":                 true,
	"OFFSET":                   true,
	"ON":                       true,
	"ONLY":                     true,
	"OPTION":                   true,
	"OR":                       true,
	"ORDER":                    true,
	"OUTER":                    true,
	"PACK_KEYS":                true,
	"PARTITION":                true,
	"PARTITIONS":               true,
	"PASSWORD":                 true,
	"PLUGINS":                  true,
	"POSITION":                 true,
	"PRECISION":                true,
	"PREPARE":                  true,
	"PRIMARY":                  true,
	"PRIVILEGES":               true,
	"PROCEDURE":                true,
	"PROCESS":                  true,
	"PROCESSLIST":              true,
	"PROFILES":                 true,
	"QUARTER":                  true,
	"QUERY":                    true,
	"QUERIES":                  true,
	"QUICK":                    true,
	"SHARD_ROW_ID_BITS":        true,
	"RANGE":                    true,
	"RECOVER":                  true,
	"READ":                     true,
	"REAL":                     true,
	"RECENT":                   true,
	"REDUNDANT":                true,
	"REFERENCES":               true,
	"REGEXP":                   true,
	"RELOAD":                   true,
	"RENAME":                   true,
	"REPEAT":                   true,
	"REPEATABLE":               true,
	"REPLACE":                  true,
	"REPLICATION":              true,
	"RESTRICT":                 true,
	"REVERSE":                  true,
	"REVOKE":                   true,
	"RIGHT":                    true,
	"RLIKE":                    true,
	"ROLLBACK":                 true,
	"ROUTINE":                  true,
	"ROW":                      true,
	"ROW_COUNT":                true,
	"ROW_FORMAT":               true,
	"SCHEMA":                   true,
	"SCHEMAS":                  true,
	"SECOND":                   true,
	"SECOND_MICROSECOND":       true,
	"SECURITY":                 true,
	"SELECT":                   true,
	"SERIALIZABLE":             true,
	"SESSION":                  true,
	"SET":                      true,
	"SEPARATOR":                true,
	"SHARE":                    true,
	"SHARED":                   true,
	"SHOW":                     true,
	"SIGNED":                   true,
	"SLAVE":                    true,
	"SLOW":                     true,
	"SMALLINT":                 true,
	"SNAPSHOT":                 true,
	"SOME":                     true,
	"SQL":                      true,
	"SQL_CACHE":                true,
	"SQL_CALC_FOUND_ROWS":      true,
	"SQL_NO_CACHE":             true,
	"START":                    true,
	"STARTING":                 true,
	"STATS":                    true,
	"STATS_BUCKETS":            true,
	"STATS_HISTOGRAMS":         true,
	"STATS_HEALTHY":            true,
	"STATS_META":               true,
	"STATS_PERSISTENT":         true,
	"STATUS":                   true,
	"STORED":                   true,
	"STRAIGHT_JOIN":            true,
	"SUBDATE":                  true,
	"SUBPARTITION":             true,
	"SUBPARTITIONS":            true,
	"SUBSTR":                   true,
	"SUBSTRING":                true,
	"SUM":                      true,
	"SUPER":                    true,
	"TABLE":                    true,
	"TABLES":                   true,
	"TABLESPACE":               true,
	"TEMPORARY":                true,
	"TEMPTABLE":                true,
	"TERMINATED":               true,
	"TEXT":                     true,
	"THAN":                     true,
	"THEN":                     true,
	"TIDB":                     true,
	"TIDB_HJ":                  true,
	"TIDB_INLJ":                true,
	"TIDB_SMJ":                 true,
	"TIME":                     true,
	"TIMESTAMP":                true,
	"TIMESTAMPADD":             true,
	"TIMESTAMPDIFF":            true,
	"TINYBLOB":                 true,
	"TINYINT":                  true,
	"TINYTEXT":                 true,
	"TO":                       true,
	"TOP":                      true,
	"TRACE":                    true,
	"TRAILING":                 true,
	"TRANSACTION":              true,
	"TRIGGER":                  true,
	"TRIGGERS":                 true,
	"TRIM":                     true,
	"TRUE":                     true,
	"TRUNCATE":                 true,
	"UNCOMMITTED":              true,
	"UNDEFINED":                true,
	"UNION":                    true,
	"UNIQUE":                   true,
	"UNKNOWN":                  true,
	"UNLOCK":                   true,
	"UNSIGNED":                 true,
	"UPDATE":                   true,
	"USAGE":                    true,
	"USE":                      true,
	"USER":                     true,
	"USING":                    true,
	"UTC_DATE":                 true,
	"UTC_TIME":                 true,
	"UTC_TIMESTAMP":            true,
	"VALUE":                    true,
	"VALUES":                   true,
	"VARBINARY":                true,
	"VARCHAR":                  true,
	"VARIABLES":                true,
	"VIEW":                     true,
	"VIRTUAL":                  true,
	"WARNINGS":                 true,
	"ERRORS":                   true,
	"WEEK":                     true,
	"WHEN":                     true,
	"WHERE":                    true,
	"WITH":                     true,
	"WRITE":                    true,
	"XOR":                      true,
	"YEAR":                     true,
	"YEAR_MONTH":               true,
	"ZEROFILL":                 true,
}

type Record struct {
	// 阶段   RERUN EXECUTED CHECKED
	Stage byte
	// 阶段说明 Execute Successfully / 审核完成 / 失败...
	// Audit completed
	// Execute failed
	// Execute Successfully
	// Execute Successfully,Backup successfully
	// Execute Successfully,Backup failed
	StageStatus byte

	// 审核级别,0为成功,1为警告,2为错误
	ErrLevel uint8
	// 错误/警告信息
	ErrorMessage string

	Sql string

	// 受影响行
	AffectedRows int

	// 对应备份库的opid,用来找到对应的回滚语句
	// Sequence string	改用属性OPID

	// 备份库的库名
	BackupDBName string

	// 执行用时
	ExecTime string

	// 备份用时
	BackupCostTime string

	// sql的hash值,osc使用
	Sqlsha1 string

	Buf *bytes.Buffer

	Type ast.StmtNode

	// 备份相关
	ExecTimestamp int64
	StartFile     string
	StartPosition int
	EndFile       string
	EndPosition   int
	ThreadId      uint32
	SeqNo         int

	DBName    string
	TableName string
	TableInfo *TableInfo
	// ddl回滚
	DDLRollback string
	OPID        string

	ExecComplete bool
}

type recordSet struct {
	firstIsID  bool
	data       [][]types.Datum
	count      int
	cursor     int
	fields     []*ast.ResultField
	fieldCount int
}

func (r *recordSet) Fields() []*ast.ResultField {
	return r.fields
}

func (r *recordSet) setFields(tps ...uint8) {
	r.fields = make([]*ast.ResultField, len(tps))
	for i := 0; i < len(tps); i++ {
		rf := new(ast.ResultField)
		rf.Column = new(model.ColumnInfo)
		rf.Column.FieldType = *types.NewFieldType(tps[i])
		r.fields[i] = rf
	}
}

func (r *recordSet) getNext() []types.Datum {
	if r.cursor == r.count {
		return nil
	}
	r.cursor++
	row := make([]types.Datum, 0, len(r.fields))
	if r.firstIsID {
		row = append(row, types.NewIntDatum(int64(r.cursor)))
	}
	row = append(row, r.data[r.cursor-1]...)
	return row
}

func (r *recordSet) Next(ctx context.Context, chk *chunk.Chunk) error {
	chk.Reset()
	row := r.getNext()
	if row != nil {
		for i := 0; i < len(row); i++ {
			chk.AppendDatum(i, &row[i])
		}
	}
	return nil
}

func (r *recordSet) NewChunk() *chunk.Chunk {
	fields := make([]*types.FieldType, 0, len(r.fields))
	for _, field := range r.fields {
		fields = append(fields, &field.Column.FieldType)
	}
	return chunk.NewChunkWithCapacity(fields, 32)
}

func (r *recordSet) Close() error {
	r.cursor = 0
	return nil
}

func (r *recordSet) CreateFiled(name string, tp uint8) {
	n := model.NewCIStr(name)
	r.fields[r.fieldCount] = &ast.ResultField{
		Column: &model.ColumnInfo{
			FieldType: *types.NewFieldType(tp),
			Name:      n,
		},
		ColumnAsName: n,
	}
	r.fieldCount++
}

type MyRecordSets struct {
	count   int
	samples []types.Datum
	rc      *recordSet
	pk      ast.RecordSet

	records  []*Record
	MaxLevel uint8

	SeqNo int

	// 作为record的游标
	cursor int
}

func NewRecordSets() *MyRecordSets {
	t := &MyRecordSets{
		records: []*Record{},
	}

	rc := &recordSet{
		// data:       make([][]types.Datum, 0),
		count:      0,
		cursor:     0,
		fieldCount: 0,
	}

	rc.fields = make([]*ast.ResultField, 12)

	// 序号
	rc.CreateFiled("order_id", mysql.TypeLong)
	// 阶段   RERUN EXECUTED CHECKED
	rc.CreateFiled("stage", mysql.TypeString)
	// 审核级别,0为成功,1为警告,2为错误
	rc.CreateFiled("ErrLevel", mysql.TypeShort)
	// 阶段说明 Execute Successfully / 审核完成 / 失败...
	rc.CreateFiled("stagestatus", mysql.TypeString)
	// 错误/警告信息
	rc.CreateFiled("errormessage", mysql.TypeString)
	rc.CreateFiled("sql", mysql.TypeString)
	// 受影响行
	rc.CreateFiled("affected_rows", mysql.TypeLong)
	// 对应备份库的opid,用来找到对应的回滚语句
	rc.CreateFiled("sequence", mysql.TypeString)
	// 备份库的库名
	rc.CreateFiled("backup_dbname", mysql.TypeString)
	rc.CreateFiled("execute_time", mysql.TypeString)
	rc.CreateFiled("backup_time", mysql.TypeString)
	// sql的hash值,osc使用
	rc.CreateFiled("sqlsha1", mysql.TypeString)

	t.rc = rc
	return t
}

func (s *MyRecordSets) Append(r *Record) {
	s.MaxLevel = uint8(Max(int(s.MaxLevel), int(r.ErrLevel)))

	r.SeqNo = s.SeqNo
	s.SeqNo++
	s.records = append(s.records, r)
	s.count++
}

func (s *MyRecordSets) setFields(r *Record) {
	row := make([]types.Datum, s.rc.fieldCount)

	row[0].SetInt64(int64(s.rc.count + 1))

	row[1].SetString(stageList[r.Stage])
	row[2].SetInt64(int64(r.ErrLevel))
	row[3].SetString(statusList[r.StageStatus])

	if r.ErrorMessage != "" {
		row[4].SetString(r.ErrorMessage)
	} else {
		row[4].SetString(strings.TrimRight(r.Buf.String(), "\n"))
	}

	row[5].SetString(r.Sql)
	row[6].SetInt64(int64(r.AffectedRows))
	if r.OPID == "" {
		// record.OPID =
		// row[7].SetNull()
		row[7].SetString(makeOPIDByTime(r.ExecTimestamp,
			r.ThreadId, r.SeqNo))
	} else {
		row[7].SetString(r.OPID)
	}

	// if r.StageStatus == StatusBackupOK {
	// 	row[8].SetString(r.BackupDBName)
	// }

	if r.BackupDBName == "" {
		row[8].SetNull()
	} else {
		row[8].SetString(r.BackupDBName)
	}

	if r.ExecTime == "" {
		row[9].SetString("0")
	} else {
		row[9].SetString(r.ExecTime)
	}

	if r.BackupCostTime == "" {
		row[10].SetString("0")
	} else {
		row[10].SetString(r.BackupCostTime)
	}

	if r.Sqlsha1 == "" {
		row[11].SetNull()
	} else {
		row[11].SetString(r.Sqlsha1)
	}

	s.rc.data[s.rc.count] = row
	s.rc.count++
}

func (s *MyRecordSets) Rows() []ast.RecordSet {

	s.rc.count = 0
	s.rc.data = make([][]types.Datum, len(s.records))

	for _, r := range s.records {
		s.setFields(r)
	}

	s.records = nil

	return []ast.RecordSet{s.rc}
}

func (s *MyRecordSets) All() []*Record {
	return s.records
}

func (s *MyRecordSets) Next() *Record {
	if s.cursor == s.count {
		return nil
	} else {
		s.cursor++
	}
	return s.records[s.cursor-1]
}

type VariableSets struct {
	count   int
	samples []types.Datum
	rc      *recordSet
	pk      ast.RecordSet
}

func NewVariableSets(count int) *VariableSets {
	t := &VariableSets{}

	rc := &recordSet{
		data:       make([][]types.Datum, count),
		count:      0,
		cursor:     0,
		fieldCount: 0,
	}

	rc.fields = make([]*ast.ResultField, 2)

	rc.CreateFiled("Variable_name", mysql.TypeString)
	rc.CreateFiled("Value", mysql.TypeString)
	t.rc = rc

	return t
}

func (s *VariableSets) Append(name string, value string) {
	row := make([]types.Datum, s.rc.fieldCount)

	row[0].SetString(name)
	row[1].SetString(value)

	// s.rc.data = append(s.rc.data, row)
	s.rc.data[s.rc.count] = row
	s.rc.count++
}

func (s *VariableSets) Rows() []ast.RecordSet {
	return []ast.RecordSet{s.rc}
}

type ProcessListSets struct {
	count   int
	samples []types.Datum
	rc      *recordSet
	pk      ast.RecordSet
}

func NewProcessListSets(count int) *ProcessListSets {
	t := &ProcessListSets{}

	rc := &recordSet{
		data:       make([][]types.Datum, count),
		count:      0,
		cursor:     0,
		fieldCount: 0,
	}

	rc.fields = make([]*ast.ResultField, 10)

	rc.CreateFiled("Id", mysql.TypeLonglong)
	//目标数据库用户名
	rc.CreateFiled("Dest_User", mysql.TypeString)
	//目标主机
	rc.CreateFiled("Dest_Host", mysql.TypeString)
	//目标端口
	rc.CreateFiled("Dest_Port", mysql.TypeLong)
	//连接来源主机
	rc.CreateFiled("From_Host", mysql.TypeString)
	//操作类型
	rc.CreateFiled("Command", mysql.TypeString)
	//操作类型
	rc.CreateFiled("STATE", mysql.TypeString)
	rc.CreateFiled("Time", mysql.TypeLonglong)
	rc.CreateFiled("Info", mysql.TypeString)
	// rc.CreateFiled("Current_Execute", mysql.TypeString)
	rc.CreateFiled("Percent", mysql.TypeString)

	t.rc = rc

	return t
}

func (s *ProcessListSets) appendRow(fields []interface{}) {
	row := make([]types.Datum, s.rc.fieldCount)

	for i, col := range fields {
		if col == nil {
			row[i].SetNull()
			continue
		}
		switch x := col.(type) {
		case nil:
			row[i].SetNull()
		case int:
			row[i].SetInt64(int64(x))
		case int64:
			row[i].SetInt64(x)
		case uint64:
			row[i].SetUint64(x)
		case float64:
			row[i].SetFloat64(x)
		case float32:
			row[i].SetFloat32(x)
		case string:
			row[i].SetString(x)
		case []byte:
			row[i].SetBytes(x)
		case types.BinaryLiteral:
			row[i].SetBytes(x)
		case *types.MyDecimal:
			row[i].SetMysqlDecimal(x)
		case types.Time:
			row[i].SetMysqlTime(x)
		case json.BinaryJSON:
			row[i].SetMysqlJSON(x)
		case types.Duration:
			row[i].SetMysqlDuration(x)
		case types.Enum:
			row[i].SetMysqlEnum(x)
		case types.Set:
			row[i].SetMysqlSet(x)
		default:
			row[i].SetNull()
		}
	}

	s.rc.data[s.rc.count] = row
	s.rc.count++
}

func (s *ProcessListSets) Rows() []ast.RecordSet {
	return []ast.RecordSet{s.rc}
}

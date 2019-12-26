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

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/model"
	"github.com/hanchuanchuan/goInception/mysql"
	"github.com/hanchuanchuan/goInception/types"
	"github.com/hanchuanchuan/goInception/types/json"
	"github.com/hanchuanchuan/goInception/util/chunk"
	"github.com/hanchuanchuan/goInception/util/sqlexec"
	// log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

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

	// 是否开启OSC
	useOsc bool

	// update多表时,记录多余的表
	// update多表时,默认set第一列的表为主表,其余表才会记录到该处
	// 仅在发现多表操作时,初始化该参数
	MultiTables map[string]*TableInfo
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
	pk      sqlexec.RecordSet

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
	rc.CreateFiled("error_level", mysql.TypeShort)
	// 阶段说明 Execute Successfully / 审核完成 / 失败...
	rc.CreateFiled("stage_status", mysql.TypeString)
	// 错误/警告信息
	rc.CreateFiled("error_message", mysql.TypeString)
	rc.CreateFiled("sql", mysql.TypeString)
	// 受影响行
	rc.CreateFiled("affected_rows", mysql.TypeLong)
	// 对应备份库的opid,用来找到对应的回滚语句
	rc.CreateFiled("sequence", mysql.TypeString)
	// 备份库的库名
	rc.CreateFiled("backup_dbname", mysql.TypeString)
	rc.CreateFiled("execute_time", mysql.TypeString)
	// sql的hash值,osc使用
	rc.CreateFiled("sqlsha1", mysql.TypeString)
	// 备份用时
	rc.CreateFiled("backup_time", mysql.TypeString)

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
		e := strings.TrimSpace(r.Buf.String())
		if e == "" {
			row[4].SetNull()
		} else {
			row[4].SetString(e)
		}
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

	if r.Sqlsha1 == "" {
		row[10].SetNull()
	} else {
		row[10].SetString(r.Sqlsha1)
	}

	if r.BackupCostTime == "" {
		row[11].SetString("0")
	} else {
		row[11].SetString(r.BackupCostTime)
	}

	s.rc.data[s.rc.count] = row
	s.rc.count++
}

func (s *MyRecordSets) Rows() []sqlexec.RecordSet {

	s.rc.count = 0
	s.rc.data = make([][]types.Datum, len(s.records))

	for _, r := range s.records {
		s.setFields(r)
	}

	s.records = nil

	return []sqlexec.RecordSet{s.rc}
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
	pk      sqlexec.RecordSet
}

func NewVariableSets(count int) *VariableSets {
	t := &VariableSets{}

	rc := &recordSet{
		data:       make([][]types.Datum, 0, count),
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

	s.rc.data = append(s.rc.data, row)
	// s.rc.data[s.rc.count] = row
	s.rc.count++
}

func (s *VariableSets) Rows() []sqlexec.RecordSet {
	return []sqlexec.RecordSet{s.rc}
}

type ProcessListSets struct {
	count   int
	samples []types.Datum
	rc      *recordSet
	pk      sqlexec.RecordSet
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

func (s *ProcessListSets) Rows() []sqlexec.RecordSet {
	return []sqlexec.RecordSet{s.rc}
}

func NewOscProcessListSets(count int, hideCommand bool) *ProcessListSets {
	t := &ProcessListSets{}

	rc := &recordSet{
		data:       make([][]types.Datum, count),
		count:      0,
		cursor:     0,
		fieldCount: 0,
	}

	if hideCommand {
		rc.fields = make([]*ast.ResultField, 6)
	} else {
		rc.fields = make([]*ast.ResultField, 7)
	}

	rc.CreateFiled("DBNAME", mysql.TypeString)
	rc.CreateFiled("TABLENAME", mysql.TypeString)
	if !hideCommand {
		rc.CreateFiled("COMMAND", mysql.TypeString)
	}
	rc.CreateFiled("SQLSHA1", mysql.TypeString)
	rc.CreateFiled("PERCENT", mysql.TypeLong)
	rc.CreateFiled("REMAINTIME", mysql.TypeString)
	rc.CreateFiled("INFOMATION", mysql.TypeString)
	// rc.CreateFiled("STATUS", mysql.TypeString)

	t.rc = rc

	return t
}

type PrintSets struct {
	count   int
	samples []types.Datum
	rc      *recordSet
	pk      sqlexec.RecordSet
}

func NewPrintSets() *PrintSets {
	t := &PrintSets{}

	rc := &recordSet{
		// data:       make([][]types.Datum, 0, count),
		count:      0,
		cursor:     0,
		fieldCount: 0,
	}

	rc.fields = make([]*ast.ResultField, 5)

	rc.CreateFiled("id", mysql.TypeLong)
	rc.CreateFiled("statement", mysql.TypeString)
	rc.CreateFiled("errlevel", mysql.TypeLong)
	rc.CreateFiled("query_tree", mysql.TypeString)
	rc.CreateFiled("errmsg", mysql.TypeString)
	t.rc = rc

	return t
}

func (s *PrintSets) Append(errLevel int64, sql, tree, errmsg string) {
	row := make([]types.Datum, s.rc.fieldCount)

	row[0].SetInt64(int64(s.rc.count + 1))
	row[1].SetString(sql)
	row[2].SetInt64(errLevel)
	row[3].SetString(tree)
	if errmsg == "" {
		row[4].SetNull()
	} else {
		row[4].SetString(errmsg)
	}

	s.rc.data = append(s.rc.data, row)
	s.rc.count++
}

func (s *PrintSets) Rows() []sqlexec.RecordSet {
	return []sqlexec.RecordSet{s.rc}
}

type SplitSets struct {
	count   int
	samples []types.Datum
	rc      *recordSet
	pk      sqlexec.RecordSet

	// 分组id,每当变化一次分组时,自动加1.默认值为1
	id int64

	sqlBuf *bytes.Buffer

	ddlflag   int64
	tableList map[string]bool
}

func NewSplitSets() *SplitSets {
	t := &SplitSets{}

	rc := &recordSet{
		// data:       make([][]types.Datum, 0, count),
		count:      0,
		cursor:     0,
		fieldCount: 0,
	}

	rc.fields = make([]*ast.ResultField, 4)

	rc.CreateFiled("id", mysql.TypeLong)
	rc.CreateFiled("sql_statement", mysql.TypeString)
	rc.CreateFiled("ddlflag", mysql.TypeLong)
	rc.CreateFiled("error_message", mysql.TypeString)
	t.rc = rc

	return t
}

func (s *SplitSets) Append(sql string, errmsg string) {
	row := make([]types.Datum, s.rc.fieldCount)

	row[0].SetInt64(s.id)
	row[1].SetString(sql)
	row[2].SetInt64(s.ddlflag)
	if errmsg == "" {
		row[3].SetNull()
	} else {
		row[3].SetString(errmsg)
	}

	s.rc.data = append(s.rc.data, row)
	s.rc.count++
}

// id累加
func (s *SplitSets) Increment() {
	s.id += 1
}

// CurrentId 当前ID
func (s *SplitSets) CurrentId() int64 {
	return s.id
}

func (s *SplitSets) Rows() []sqlexec.RecordSet {
	return []sqlexec.RecordSet{s.rc}
}

type LevelSets struct {
	count   int
	samples []types.Datum
	rc      *recordSet
	pk      sqlexec.RecordSet
}

func NewLevelSets(count int) *LevelSets {
	t := &LevelSets{}

	rc := &recordSet{
		data:       make([][]types.Datum, 0, count),
		count:      0,
		cursor:     0,
		fieldCount: 0,
	}

	rc.fields = make([]*ast.ResultField, 3)

	rc.CreateFiled("Name", mysql.TypeString)
	rc.CreateFiled("Value", mysql.TypeLong)
	rc.CreateFiled("Desc", mysql.TypeString)

	t.rc = rc

	return t
}

func (s *LevelSets) Append(name string, value int64, desc string) {
	row := make([]types.Datum, s.rc.fieldCount)

	row[0].SetString(name)
	row[1].SetInt64(value)
	row[2].SetString(desc)

	s.rc.data = append(s.rc.data, row)
	// s.rc.data[s.rc.count] = row
	s.rc.count++
}

func (s *LevelSets) Rows() []sqlexec.RecordSet {
	return []sqlexec.RecordSet{s.rc}
}

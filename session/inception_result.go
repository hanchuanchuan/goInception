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
	// "math"
	// "testing"
	// "time"
	"bytes"
	"strings"
	// "fmt"
	// . "github.com/pingcap/check"
	"github.com/hanchuanchuan/tidb/ast"
	"github.com/hanchuanchuan/tidb/model"
	"github.com/hanchuanchuan/tidb/mysql"
	// "github.com/hanchuanchuan/tidb/sessionctx"
	// "github.com/hanchuanchuan/tidb/sessionctx/stmtctx"
	"github.com/hanchuanchuan/tidb/types"
	// "github.com/hanchuanchuan/tidb/types/json"
	"github.com/hanchuanchuan/tidb/util/chunk"
	// "github.com/hanchuanchuan/tidb/util/codec"
	// "github.com/hanchuanchuan/tidb/util/mock"
	// "github.com/hanchuanchuan/tidb/util/ranger"
	// "github.com/pkg/errors"
	"golang.org/x/net/context"
)

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

type recordSet struct {
	firstIsID  bool
	data       [][]types.Datum
	count      int
	cursor     int
	fields     []*ast.ResultField
	fieldCount int
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

	rc.fields = make([]*ast.ResultField, 11)

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
	// sql的hash值,osc使用
	rc.CreateFiled("sqlsha1", mysql.TypeString)

	t.rc = rc
	return t
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

// func (s *MyRecordSets) AppendRow(sql string, ErrLevel int) {

// 	row := make([]types.Datum, s.rc.fieldCount)

// 	row[0].SetInt64(int64(s.rc.count + 1))
// 	row[1].SetString("error")
// 	row[2].SetInt64(int64(ErrLevel))
// 	row[3].SetString("1")
// 	row[4].SetString("testadadsf")
// 	row[5].SetString(sql)
// 	row[6].SetInt64(int64(1))
// 	row[7].SetString("")
// 	row[8].SetString("")
// 	row[9].SetMysqlTime(types.CurrentTime(mysql.TypeTimestamp))
// 	row[10].SetString("")

// 	s.rc.data[s.rc.count] = row
// 	s.rc.count++
// }

func (s *MyRecordSets) Append(r *Record) {
	s.MaxLevel = uint8(Max(int(s.MaxLevel), int(r.ErrLevel)))

	s.SeqNo++
	r.SeqNo = s.SeqNo
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
		row[7].SetNull()
	} else {
		row[7].SetString(r.OPID)
	}

	if r.StageStatus == StatusBackupOK {
		row[8].SetString(r.BackupDBName)

		// if r.BackupDBName == "" {
		// 	row[8].SetNull()
		// } else {
		// 	row[8].SetString(r.BackupDBName)
		// }
	}

	if r.ExecTime == "" {
		row[9].SetString("0")
	} else {
		row[9].SetString(r.ExecTime)
	}
	// row[9].SetMysqlTime(types.CurrentTime(mysql.TypeTimestamp))

	if r.Sqlsha1 == "" {
		row[10].SetNull()
	} else {
		row[10].SetString(r.Sqlsha1)
	}

	s.rc.data[s.rc.count] = row
	s.rc.count++
}

// func (s *MyRecordSets) AppentRows() []ast.RecordSet {

// 	s.Append(&Record{
// 		Sql:      "insert into t1 select 1",
// 		ErrLevel: 1,
// 	})

// 	return []ast.RecordSet{s.rc}
// }

func (s *MyRecordSets) Rows() []ast.RecordSet {

	s.rc.data = make([][]types.Datum, len(s.records))

	for _, r := range s.records {
		s.setFields(r)
	}

	s.records = nil

	return []ast.RecordSet{s.rc}
}

func (r *Record) AnlyzeExplain(rows []ExplainInfo) {
	if len(rows) > 0 {
		r.AffectedRows = rows[0].Rows
	}
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

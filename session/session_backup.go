package session

import (
	"context"
	"fmt"
	"strings"
	"sync"

	mysqlDriver "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

// chanBackup 备份channal数据,用来传递备份的sql等信息
type chanBackup struct {
	// values []interface{}
	// 库名
	dbname string
	values []interface{}
	record *Record
}

func (s *session) ProcessChanBackup(wg *sync.WaitGroup) {
	for {
		r := <-s.chBackupRecord

		if r == nil {
			s.flushBackupRecord(s.lastBackupTable, s.myRecord)
			wg.Done()
			break
		}
		// flush标志. 不能在外面调用flush函数,会导致线程并发操作,写入数据错误
		// 如数据尚未进入到ch通道,此时调用flush,数据无法正确入库

		if r.values == nil {
			s.flushBackupRecord(r.dbname, r.record)
		} else {
			s.writeBackupRecord(r.dbname, r.record, r.values)
		}
	}
}

func (s *session) runBackup(ctx context.Context) {

	var wg sync.WaitGroup
	wg.Add(1)
	s.chBackupRecord = make(chan *chanBackup, 50)
	go s.ProcessChanBackup(&wg)
	defer func() {
		close(s.chBackupRecord)
		wg.Wait()
		// 清空临时的库名
		s.lastBackupTable = ""
	}()

	for _, record := range s.recordSets.All() {

		if s.checkSqlIsDML(record) || s.checkSqlIsDDL(record) {
			s.myRecord = record

			longDataType := s.mysqlCreateBackupTable(record)
			// errno := s.mysqlCreateBackupTable(record)
			// if errno == 2 {
			// 	break
			// }
			if record.TableInfo == nil {
				s.AppendErrorNo(ErrNotFoundTableInfo)
			} else {
				s.mysqlBackupSql(record, longDataType)
			}

			if s.hasError() {
				break
			}
		}

		// // 进程Killed
		// if err := checkClose(ctx); err != nil {
		//     log.Warn("Killed: ", err)
		//     s.AppendErrorMessage("Operation has been killed!")
		//     break
		// }
	}
}

// 解析的sql写入缓存,并定期入库
func (s *session) writeBackupRecord(dbname string, record *Record, values []interface{}) {

	s.insertBuffer = append(s.insertBuffer, values...)

	// 每500行insert提交一次
	if len(s.insertBuffer) >= 500*11 {
		s.flushBackupRecord(dbname, record)
	}
}

// flush用以写入当前insert缓存,并清空缓存.
func (s *session) flushBackupRecord(dbname string, record *Record) {
	// log.Info("flush ", len(s.insertBuffer))

	if len(s.insertBuffer) > 0 {
		const backupRecordColumnCount int = 11
		const rowSQL = "(?,?,?,?,?,?,?,?,?,?,NOW(),?),"
		tableName := fmt.Sprintf("`%s`.`%s`", dbname, remoteBackupTable)

		sql := "insert into %s values%s"
		values := strings.TrimRight(
			strings.Repeat(rowSQL, len(s.insertBuffer)/backupRecordColumnCount), ",")

		err := s.backupdb.Exec(fmt.Sprintf(sql, tableName, values),
			s.insertBuffer...).Error
		if err != nil {
			log.Error(err)
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
				s.recordSets.MaxLevel = 2
				record.StageStatus = StatusBackupFail
				record.AppendErrorMessage(myErr.Message)
			}
		}

		// s.BackupTotalRows += len(s.insertBuffer) / backupRecordColumnCount
		// s.SetMyProcessInfo(record.Sql, time.Now(),
		//     float64(s.BackupTotalRows)/float64(s.TotalChangeRows))

		s.insertBuffer = nil
	}
}

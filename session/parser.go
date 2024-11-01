package session

import (
	"context"
	"crypto/tls"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/juju/errors"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const digits01 = "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789"
const digits10 = "0000000000111111111122222222223333333333444444444455555555556666666666777777777788888888889999999999"

// chanData 备份channal数据,用来传递备份的sql等信息
type chanData struct {
	sql    []byte // dml动态解析的语句
	sqlStr string // ddl回滚语句
	e      *replication.BinlogEvent
	gtid   []byte
	opid   string
	table  string
	record *Record
}

func (s *session) processChan(wg *sync.WaitGroup) {
	for {
		r := <-s.ch

		if r == nil {
			// log.Info("剩余ch", len(s.ch), "cap ch", cap(s.ch), "通道关闭,跳出循环")
			// log.Info("ProcessChan,close")
			s.flush(s.lastBackupTable, s.myRecord)
			wg.Done()
			break
		}
		// flush标志. 不能在外面调用flush函数,会导致线程并发操作,写入数据错误
		// 如数据尚未进入到ch通道,此时调用flush,数据无法正确入库

		if len(r.sqlStr) > 0 {
			s.myWriteDDL(r.sqlStr, r.opid, r.table, r.record)
		} else if len(r.sql) == 0 {
			// log.Info("flush标志")
			s.flush(r.table, r.record)
		} else {
			s.myWrite(r.sql, r.e, r.opid, r.table, r.record)
		}
	}
}

func (s *session) getNextBackupRecord() *Record {
	for {
		r := s.recordSets.Next()
		if r == nil {
			return nil
		}

		if r.TableInfo != nil {

			lastBackupTable := fmt.Sprintf("`%s`.`%s`", s.getRemoteBackupDBName(r), r.TableInfo.Name)

			if s.lastBackupTable == "" {
				s.lastBackupTable = lastBackupTable
			}

			if s.checkSqlIsDDL(r) {
				if s.lastBackupTable != lastBackupTable {
					s.ch <- &chanData{sql: nil, table: s.lastBackupTable, record: s.myRecord}
					s.lastBackupTable = lastBackupTable
				}

				s.ch <- &chanData{sqlStr: r.DDLRollback, opid: r.OPID,
					table: s.lastBackupTable, record: r}

				if r.StageStatus != StatusExecFail {
					r.StageStatus = StatusBackupOK
				}

				continue

			} else if (r.AffectedRows > 0 || r.StageStatus == StatusExecFail) && s.checkSqlIsDML(r) {

				// if s.opt.middlewareExtend != "" {
				// 	continue
				// }

				// 如果开始位置和结果位置相同,说明无变更(受影响行数为0)
				if r.StartFile == r.EndFile && r.StartPosition == r.EndPosition {
					continue
				}

				if s.lastBackupTable != lastBackupTable {
					s.ch <- &chanData{sql: nil, table: s.lastBackupTable, record: s.myRecord}
					s.lastBackupTable = lastBackupTable
				}

				// 先置默认值为备份失败,在备份完成后置为成功
				// if r.AffectedRows > 0 {
				if r.StageStatus != StatusExecFail {
					r.StageStatus = StatusBackupFail
				}
				// 清理已删除的列
				clearDeleteColumns(r.TableInfo)
				if r.MultiTables != nil {
					for _, t := range r.MultiTables {
						clearDeleteColumns(t)
					}
				}

				return r
			}

		}
	}
}

func configPrimaryKey(t *TableInfo) {
	// var primarys map[int]bool
	primarys := make(map[int]bool)
	// var uniques map[int]bool
	uniques := make(map[int]bool)

	for i, r := range t.Fields {
		if r.Key == "PRI" {
			primarys[i] = true
		}
		if r.Key == "UNI" {
			uniques[i] = true
		}
	}

	if len(primarys) > 0 {
		t.primarys = primarys
		t.hasPrimary = true
	} else if len(uniques) > 0 {
		t.primarys = uniques
		t.hasPrimary = true
	} else {
		t.hasPrimary = false
	}
}

// clearDeleteColumns 清理已删除的列,方便解析binlog
func clearDeleteColumns(t *TableInfo) {
	if t == nil || t.IsClear {
		return
	}

	fields := t.Fields
	fs := make([]FieldInfo, 0, len(t.Fields))
	for _, f := range fields {
		if !f.IsDeleted {
			fs = append(fs, f)
		}
	}

	t.Fields = fs

	configPrimaryKey(t)

	t.IsClear = true
}

func (s *session) parserBinlog(ctx context.Context) {

	// var err error
	var wg sync.WaitGroup

	wg.Add(1)
	s.ch = make(chan *chanData, 50)
	go s.processChan(&wg)

	// 最终关闭和返回
	defer func() {
		close(s.ch)
		wg.Wait()

		// log.Info("操作完成", "rows", i)
		// kwargs := map[string]interface{}{"ok": "1"}
		// sendMsg(p.cfg.SocketUser, "rollback_binlog_parse_complete", "binlog解析进度", "", kwargs)
	}()

	// 获取binlog解析起点
	record := s.getNextBackupRecord()
	if record == nil {
		return
	}

	s.myRecord = record

	log.Debug("Parser")

	// 启用Logger，显示详细日志
	// s.backupdb.LogMode(true)

	flavor := "mysql"
	if s.dbType == DBTypeMariaDB {
		flavor = "mariadb"
	}

	var (
		host string
		port uint16
	)
	if s.isMiddleware() {
		host = s.opt.parseHost
		port = uint16(s.opt.parsePort)
	} else {
		host = s.opt.Host
		port = uint16(s.opt.Port)
	}
	cfg := replication.BinlogSyncerConfig{
		ServerID: 2000111111 + uint32(s.sessionVars.ConnectionID%10000),
		Flavor:   flavor,

		Host:       host,
		Port:       port,
		User:       s.opt.User,
		Password:   s.opt.Password,
		UseDecimal: true,
		// RawModeEnabled:  p.cfg.RawMode,
		// SemiSyncEnabled: p.cfg.SemiSync,
	}

	if s.opt.ssl != "" {
		switch s.opt.ssl {
		case "preferred", "true", "required":
			cfg.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}

	b := replication.NewBinlogSyncer(cfg)
	defer b.Close()

	startPosition := mysql.Position{Name: record.StartFile,
		Pos: uint32(record.StartPosition)}
	stopPosition := mysql.Position{Name: record.EndFile,
		Pos: uint32(record.EndPosition)}
	s.lastBackupTable = fmt.Sprintf("`%s`.`%s`", record.BackupDBName, record.TableInfo.Name)
	startTime := time.Now()

	logSync, err := b.StartSync(startPosition)
	if err != nil {
		log.Infof("Start sync error: %v\n", errors.ErrorStack(err))
		s.appendErrorMsg(err.Error())
		return
	}

	currentPosition := startPosition
	var currentThreadID uint32

	// 已解析行数，如果行数大于等于record的实际受影响行数，则可以直接切换到下一record
	var changeRows int

	defer func() {
		if err := recover(); err != nil {
			log.Errorf("ERROR!! BUG!!")
			if s.myRecord != nil {
				log.Errorf("error parsing table `%s`.`%s`. sql: %s",
					s.myRecord.DBName, s.myRecord.TableName, s.myRecord.Sql)
				log.Errorf("binlog info: start pos: %s:%d, end pos: %s:%d. thread id: %d",
					s.myRecord.StartFile, s.myRecord.StartPosition,
					s.myRecord.EndFile, s.myRecord.EndPosition,
					s.myRecord.ThreadId)
				log.Errorf("%#v", s.myRecord)
			}
			log.Errorf("current pos: %s", currentPosition.String())
			log.Errorf("%#v", err)
			s.appendErrorMsg(fmt.Sprintf("%#v", err))
		}
	}()

	for {
		e, err := logSync.GetEvent(context.Background())
		if err != nil {
			log.Infof("Get event error: %v\n", errors.ErrorStack(err))
			s.appendErrorMsg(err.Error())
			break
		}

		if e.Header.LogPos > 0 {
			currentPosition.Pos = e.Header.LogPos
		}

		if e.Header.EventType == replication.ROTATE_EVENT {
			if event, ok := e.Event.(*replication.RotateEvent); ok {
				currentPosition = mysql.Position{Name: string(event.NextLogName),
					Pos: uint32(event.Position)}
			}
		}

		// 如果还没有到操作的binlog范围,跳过
		if currentPosition.Compare(startPosition) == -1 {
			continue
		}

		// log.Errorf("binlog pos: %d", int(currentPosition.Pos))
		// log.Errorf("binlog type: %#v,%#v", e.Header.EventType, e.Header.EventType.String())
		// log.Errorf("currentThreadID: %v", currentThreadID)
		// e.Dump(os.Stdout)
		// os.Stdout.Sync()

		switch e.Header.EventType {
		case replication.TABLE_MAP_EVENT:
			if event, ok := e.Event.(*replication.TableMapEvent); ok {
				if !strings.EqualFold(string(event.Schema), record.TableInfo.Schema) ||
					!strings.EqualFold(string(event.Table), record.TableInfo.Name) {
					goto ENDCHECK
				}
			}

		case replication.QUERY_EVENT:
			// if event, ok := e.Event.(*replication.QueryEvent); ok {
			// 	// currentThreadID = event.SlaveProxyID
			// 	log.Error(string(event.Query))
			// }

			if s.dbType == DBTypeMariaDB && s.dbVersion >= 100000 {
				goto ENDCHECK
			}

			if event, ok := e.Event.(*replication.QueryEvent); ok {
				currentThreadID = event.SlaveProxyID
			}

		case replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
			if event, ok := e.Event.(*replication.RowsEvent); ok {
				if s.checkFilter(event, record, currentThreadID) {
					changeRows += len(event.Rows)
					_, err = s.generateDeleteSql(record.TableInfo, event, e)
					if err != nil {
						log.Error(err)
					}
				} else {
					goto ENDCHECK
				}
			}

		case replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:

			if event, ok := e.Event.(*replication.RowsEvent); ok {
				if ok, t := s.checkUpdateFilter(event, record, currentThreadID); ok {
					changeRows += len(event.Rows)
					if t != nil {
						_, err = s.generateInsertSql(t, event, e)
					} else {
						_, err = s.generateInsertSql(record.TableInfo, event, e)
					}
					if err != nil {
						log.Error(err)
					}
				} else {
					goto ENDCHECK
				}
			}

		case replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
			if event, ok := e.Event.(*replication.RowsEvent); ok {
				if ok, t := s.checkUpdateFilter(event, record, currentThreadID); ok {
					changeRows += len(event.Rows) / 2
					if t != nil {
						_, err = s.generateUpdateSql(t, event, e)
					} else {
						_, err = s.generateUpdateSql(record.TableInfo, event, e)
					}
					if err != nil {
						log.Error(err)
					}
				} else {
					goto ENDCHECK
				}
			}
		}

	ENDCHECK:
		// 如果操作已超过binlog范围,切换到下一日志
		if currentPosition.Compare(stopPosition) > -1 {
			// sql被kill后,如果备份时可以检测到行,则认为执行成功
			// 工单只有执行成功,才允许标记为备份成功
			// if (record.StageStatus == StatusExecFail && record.AffectedRows > 0) ||
			// 	record.StageStatus == StatusExecOK || record.StageStatus == StatusBackupFail {
			if record.AffectedRows > 0 {
				if int64(changeRows) >= record.AffectedRows {
					record.StageStatus = StatusBackupOK
				}
			}

			record.BackupCostTime = fmt.Sprintf("%.3f", time.Since(startTime).Seconds())

			changeRows = 0
			next := s.getNextBackupRecord()
			if next != nil {
				startPosition = mysql.Position{Name: next.StartFile,
					Pos: uint32(next.StartPosition)}
				stopPosition = mysql.Position{Name: next.EndFile,
					Pos: uint32(next.EndPosition)}
				startTime = time.Now()

				s.myRecord = next
				record = next
			} else {
				break
			}
		} else if s.opt.tranBatch > 1 {
			if int64(changeRows) >= record.AffectedRows &&
				(s.dbType != DBTypeMariaDB || s.dbVersion < 100000) {
				if record.AffectedRows > 0 {
					record.StageStatus = StatusBackupOK
				}
				record.BackupCostTime = fmt.Sprintf("%.3f", time.Since(startTime).Seconds())

				changeRows = 0
				next := s.getNextBackupRecord()
				if next != nil {
					startPosition = mysql.Position{Name: next.StartFile,
						Pos: uint32(next.StartPosition)}
					stopPosition = mysql.Position{Name: next.EndFile,
						Pos: uint32(next.EndPosition)}
					startTime = time.Now()

					s.myRecord = next
					record = next
				} else {
					break
				}
			}
		}

		// 如果执行阶段已经kill,则不再检查
		if !s.killExecute {
			// 进程Killed
			if err := checkClose(ctx); err != nil {
				log.Warn("Killed: ", err)
				s.appendErrorMsg("Operation has been killed!")
				break
			}
		}

		// if s.hasErrorBefore() {
		// 	break
		// }
	}
}

// checkFilter 检查限制条件
// 库表正确,线程号正确
// 由于mariadb v10版本后无法获取thread id,所以采用模糊方式解析binlog,并添加警告
func (s *session) checkFilter(event *replication.RowsEvent,
	record *Record, currentThreadID uint32) bool {
	if !strings.EqualFold(string(event.Table.Schema), record.TableInfo.Schema) ||
		!strings.EqualFold(string(event.Table.Table), record.TableInfo.Name) {
		return false
	}

	if currentThreadID == 0 && s.dbType == DBTypeMariaDB {
		if record.ErrLevel != 1 {
			record.appendErrorNo(s.inc.Lang, ErrMariaDBRollbackWarn, s.dbVersion)
		}
		return true
	} else if record.ThreadId != currentThreadID {
		return false
	}
	return true
}

// checkUpdateFilter 检查update的筛选条件
// update会涉及多表更新问题,所以需要把匹配到的表返回
func (s *session) checkUpdateFilter(event *replication.RowsEvent,
	record *Record, currentThreadID uint32) (bool, *TableInfo) {
	var multiTable *TableInfo
	if record.MultiTables == nil {
		if !strings.EqualFold(string(event.Table.Schema), record.TableInfo.Schema) ||
			!strings.EqualFold(string(event.Table.Table), record.TableInfo.Name) {
			return false, nil
		}
	} else {
		found := false
		if strings.EqualFold(string(event.Table.Schema), record.TableInfo.Schema) &&
			strings.EqualFold(string(event.Table.Table), record.TableInfo.Name) {
			found = true
		} else {
			for _, t := range record.MultiTables {
				if strings.EqualFold(string(event.Table.Schema), t.Schema) &&
					strings.EqualFold(string(event.Table.Table), t.Name) {
					multiTable = t
					found = true
					break
				}
			}
		}
		if !found {
			return false, nil
		}
	}

	if currentThreadID == 0 && s.dbType == DBTypeMariaDB {
		if record.ErrLevel != 1 {
			record.appendErrorNo(s.inc.Lang, ErrMariaDBRollbackWarn, s.dbVersion)
		}
		return true, multiTable
	} else if record.ThreadId != currentThreadID {
		return false, nil
	}
	return true, multiTable
}

// 解析的sql写入缓存,并定期入库
func (s *session) myWrite(b []byte, binEvent *replication.BinlogEvent,
	opid string, table string, record *Record) {

	b = append(b, ";"...)

	s.insertBuffer = append(s.insertBuffer, string(b), opid)

	if len(s.insertBuffer) >= 1000 {
		s.flush(table, record)
	}
}

// 解析的sql写入缓存,并定期入库
func (s *session) myWriteDDL(sql string, opid string, table string, record *Record) {

	s.insertBuffer = append(s.insertBuffer, sql, opid)

	if len(s.insertBuffer) >= 1000 {
		s.flush(table, record)
	}
}

// flush用以写入当前insert缓存,并清空缓存.
func (s *session) flush(table string, record *Record) {
	// log.Info("flush ", len(s.insertBuffer))
	if len(s.insertBuffer) > 0 {

		const rowSQL = "(?,?),"
		sql := "insert into %s(rollback_statement,opid_time) values%s"
		values := strings.TrimRight(strings.Repeat(rowSQL, len(s.insertBuffer)/2), ",")
		sql = fmt.Sprintf(sql, table, values)
		err := s.backupdb.Exec(sql,
			s.insertBuffer...).Error
		if err != nil {
			record.StageStatus = StatusBackupFail
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
				record.appendErrorMessage(myErr.Message)
			} else {
				s.appendErrorMsg(err.Error())
			}
			log.Errorf("con:%d %v sql:%s params:%v",
				s.sessionVars.ConnectionID, err, sql, s.insertBuffer)
		}
		s.backupTotalRows += len(s.insertBuffer) / 2
		s.SetMyProcessInfo(record.Sql, time.Now(),
			float64(s.backupTotalRows)/float64(s.totalChangeRows))
	}
	s.insertBuffer = nil
}

func (s *session) checkError(e error) {
	if e != nil {
		log.Error(e)

		// if len(p.cfg.SocketUser) > 0 {
		// 	kwargs := map[string]interface{}{"error": e.Error()}
		// 	sendMsg(p.cfg.SocketUser, "rollback_binlog_parse_complete", "binlog解析进度",
		// 		"", kwargs)
		// }
		// panic(errors.ErrorStack(e))
	}
}

func (s *session) write(b []byte, binEvent *replication.BinlogEvent) {
	// 此处执行状态不确定的记录
	if s.myRecord.StageStatus == StatusExecFail {
		log.Info("auto fix record:", s.myRecord.OPID)
		s.myRecord.AffectedRows += 1
		s.totalChangeRows += 1
	}

	s.ch <- &chanData{sql: b, e: binEvent, opid: s.myRecord.OPID,
		table: s.lastBackupTable, record: s.myRecord}
}

func (s *session) generateInsertSql(t *TableInfo, e *replication.RowsEvent,
	binEvent *replication.BinlogEvent) (string, error) {
	var buf []byte
	if len(t.Fields) < int(e.ColumnCount) {
		return "", errors.Errorf("表%s.%s缺少列!当前列数:%d,binlog的列数%d",
			e.Table.Schema, e.Table.Table, len(t.Fields), e.ColumnCount)
	}

	var columnNames []string
	c := "`%s`"
	template := "INSERT INTO `%s`.`%s`(%s) VALUES(%s)"
	for i, col := range t.Fields {
		if i < int(e.ColumnCount) && !col.IsGenerated() {
			columnNames = append(columnNames, fmt.Sprintf(c, col.Field))
		}
	}

	paramValues := strings.Repeat("?,", t.EffectiveFieldCount())
	paramValues = strings.TrimRight(paramValues, ",")

	sql := fmt.Sprintf(template, e.Table.Schema, e.Table.Table,
		strings.Join(columnNames, ","), paramValues)

	for _, rows := range e.Rows {

		var vv []driver.Value
		for i, d := range rows {
			if t.Fields[i].IsGenerated() {
				continue
			}
			if t.Fields[i].isUnsigned() {
				d = processValue(d, GetDataTypeBase(t.Fields[i].Type))
			}
			vv = append(vv, d)
			// if _, ok := d.([]byte); ok {
			//  log.Info().Msgf("%s:%q\n", t.Fields[j].Field, d)
			// } else {
			//  log.Info().Msgf("%s:%#v\n", t.Fields[j].Field, d)
			// }
		}

		r, err := interpolateParams(sql, vv, s.inc.HexBlob)
		if err != nil {
			log.Error(err)
		}

		s.write(r, binEvent)
	}

	return string(buf), nil
}

func (s *session) generateDeleteSql(t *TableInfo, e *replication.RowsEvent,
	binEvent *replication.BinlogEvent) (string, error) {
	var buf []byte
	if len(t.Fields) < int(e.ColumnCount) {
		return "", errors.Errorf("表%s.%s缺少列!当前列数:%d,binlog的列数%d",
			e.Table.Schema, e.Table.Table, len(t.Fields), e.ColumnCount)
	}

	template := "DELETE FROM `%s`.`%s` WHERE"

	sql := fmt.Sprintf(template, e.Table.Schema, e.Table.Table)

	c_null := " `%s` IS ?"
	c := " `%s`=?"
	var columnNames []string
	for _, rows := range e.Rows {

		columnNames = nil

		var vv []driver.Value
		for i, d := range rows {
			if t.hasPrimary {
				_, ok := t.primarys[i]
				if ok {
					if t.Fields[i].isUnsigned() {
						d = processValue(d, GetDataTypeBase(t.Fields[i].Type))
					}
					vv = append(vv, d)
					if d == nil {
						columnNames = append(columnNames,
							fmt.Sprintf(c_null, t.Fields[i].Field))
					} else {
						columnNames = append(columnNames,
							fmt.Sprintf(c, t.Fields[i].Field))
					}
				}
			} else {
				if t.Fields[i].isUnsigned() {
					d = processValue(d, GetDataTypeBase(t.Fields[i].Type))
				}
				vv = append(vv, d)

				if d == nil {
					columnNames = append(columnNames,
						fmt.Sprintf(c_null, t.Fields[i].Field))
				} else {
					columnNames = append(columnNames,
						fmt.Sprintf(c, t.Fields[i].Field))
				}
			}

		}
		newSql := strings.Join([]string{sql, strings.Join(columnNames, " AND")}, "")

		r, err := interpolateParams(newSql, vv, s.inc.HexBlob)
		if err != nil {
			log.Error(err)
		}

		s.write(r, binEvent)

	}

	return string(buf), nil
}

// processValue 处理无符号值(unsigned)
func processValue(value driver.Value, dataType string) driver.Value {
	if value == nil {
		return value
	}

	switch v := value.(type) {
	case int8:
		if v >= 0 {
			return value
		}
		return int64(1<<8 + int64(v))
	case int16:
		if v >= 0 {
			return value
		}
		return int64(1<<16 + int64(v))
	case int32:
		if v >= 0 {
			return value
		}
		if dataType == "mediumint" {
			return int64(1<<24 + int64(v))
		}
		return int64(1<<32 + int64(v))
	case int64:
		if v >= 0 {
			return value
		}
		return math.MaxUint64 - uint64(abs(v)) + 1
	// case int:
	// case float32:
	// case float64:

	default:
		// log.Error("解析错误")
		// log.Errorf("%T", v)
		return value
	}
}

func abs(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

func (s *session) generateUpdateSql(t *TableInfo, e *replication.RowsEvent,
	binEvent *replication.BinlogEvent) (string, error) {
	var buf []byte
	if len(t.Fields) < int(e.ColumnCount) {
		return "", errors.Errorf("表%s.%s缺少列!当前列数:%d,binlog的列数%d",
			e.Table.Schema, e.Table.Table, len(t.Fields), e.ColumnCount)
	}

	template := "UPDATE `%s`.`%s` SET%s WHERE"

	setValue := " `%s`=?"

	var columnNames []string
	c_null := " `%s` IS ?"
	c := " `%s`=?"
	var sql string

	var sets []string

	// 最小化回滚语句, 当开启时,update语句中未变更的值不再记录到回滚语句中
	minimalMode := s.inc.EnableMinimalRollback

	if !minimalMode {
		for i, col := range t.Fields {
			// 日志是minimal模式时, 只取有值的新列
			// && uint8(e.ColumnBitmap2[i/8])&(1<<(uint(i)%8)) == uint8(e.ColumnBitmap2[i/8])

			if i < int(e.ColumnCount) && !col.IsGenerated() {
				sets = append(sets, fmt.Sprintf(setValue, col.Field))
			}
		}
		sql = fmt.Sprintf(template, e.Table.Schema, e.Table.Table,
			strings.Join(sets, ","))
	}

	var (
		oldValues []driver.Value
		newValues []driver.Value
		newSql    string
	)

	// update时, Rows为2的倍数, 双数index为旧值,单数index为新值
	for i, rows := range e.Rows {
		if i%2 == 0 {
			// 旧值
			for j, d := range rows {
				// // 日志是minimal模式时, 只取有值的新列
				// if uint8(e.ColumnBitmap2[j/8])&(1<<(uint(j)%8)) != uint8(e.ColumnBitmap2[j/8]) {
				// 	continue
				// }
				if minimalMode {
					// 最小化模式下,列如果相等则省略
					if !compareValue(d, e.Rows[i+1][j]) {
						if t.Fields[j].IsGenerated() {
							continue
						}
						if t.Fields[j].isUnsigned() {
							d = processValue(d, GetDataTypeBase(t.Fields[j].Type))
						}
						newValues = append(newValues, d)
						if j < len(t.Fields) {
							sets = append(sets, fmt.Sprintf(setValue, t.Fields[j].Field))
						}
					}
				} else {
					if t.Fields[j].IsGenerated() {
						continue
					}
					if t.Fields[j].isUnsigned() {
						d = processValue(d, GetDataTypeBase(t.Fields[j].Type))
					}
					newValues = append(newValues, d)
				}
			}
		} else {
			// 新值
			columnNames = nil
			for j, d := range rows {
				if t.hasPrimary {
					if _, ok := t.primarys[j]; ok {
						if t.Fields[j].IsGenerated() {
							continue
						}
						if t.Fields[j].isUnsigned() {
							d = processValue(d, GetDataTypeBase(t.Fields[j].Type))
						}
						oldValues = append(oldValues, d)

						if d == nil {
							columnNames = append(columnNames,
								fmt.Sprintf(c_null, t.Fields[j].Field))
						} else {
							columnNames = append(columnNames,
								fmt.Sprintf(c, t.Fields[j].Field))
						}
					}
				} else {
					if t.Fields[j].IsGenerated() {
						continue
					}
					if t.Fields[j].isUnsigned() {
						d = processValue(d, GetDataTypeBase(t.Fields[j].Type))
					}
					oldValues = append(oldValues, d)

					if d == nil {
						columnNames = append(columnNames,
							fmt.Sprintf(c_null, t.Fields[j].Field))
					} else {
						columnNames = append(columnNames,
							fmt.Sprintf(c, t.Fields[j].Field))
					}
				}
			}

			if minimalMode {
				sql = fmt.Sprintf(template, e.Table.Schema, e.Table.Table,
					strings.Join(sets, ","))
				sets = nil
			}
			newSql = strings.Join([]string{sql, strings.Join(columnNames, " AND")}, "")
			newValues = append(newValues, oldValues...)
			r, err := interpolateParams(newSql, newValues, s.inc.HexBlob)
			if err != nil {
				log.Error(err)
			}

			s.write(r, binEvent)

			oldValues = nil
			newValues = nil
		}
	}

	return string(buf), nil
}

func interpolateParams(query string, args []driver.Value, hexBlob bool) ([]byte, error) {
	// Number of ? should be same to len(args)
	if strings.Count(query, "?") != len(args) {
		log.Error("sql", query, "需要参数", strings.Count(query, "?"),
			"提供参数", len(args), "sql的参数个数不匹配")

		return nil, errors.New("driver: skip fast-path; continue as if unimplemented")
	}

	var buf []byte

	argPos := 0

	for i := 0; i < len(query); i++ {
		q := strings.IndexByte(query[i:], '?')
		if q == -1 {
			buf = append(buf, query[i:]...)
			break
		}
		buf = append(buf, query[i:i+q]...)
		i += q

		arg := args[argPos]
		argPos++

		if arg == nil {
			buf = append(buf, "NULL"...)
			continue
		}

		// log.Info(arg)
		// log.Infof("%T", arg)

		switch v := arg.(type) {
		case int8:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int16:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int32:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int64:
			buf = strconv.AppendInt(buf, v, 10)
		case uint64:
			buf = strconv.AppendUint(buf, uint64(v), 10)
		case int:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case decimal.Decimal:
			buf = append(buf, v.String()...)
		case float32:
			buf = strconv.AppendFloat(buf, float64(v), 'g', -1, 32)
		case float64:
			buf = strconv.AppendFloat(buf, v, 'g', -1, 64)
		case bool:
			if v {
				buf = append(buf, '1')
			} else {
				buf = append(buf, '0')
			}
		case time.Time:
			if v.IsZero() {
				buf = append(buf, "'0000-00-00'"...)
			} else {
				v := v.In(time.UTC)
				v = v.Add(time.Nanosecond * 500) // To round under microsecond
				year := v.Year()
				year100 := year / 100
				year1 := year % 100
				month := v.Month()
				day := v.Day()
				hour := v.Hour()
				minute := v.Minute()
				second := v.Second()
				micro := v.Nanosecond() / 1000

				buf = append(buf, []byte{
					'\'',
					digits10[year100], digits01[year100],
					digits10[year1], digits01[year1],
					'-',
					digits10[month], digits01[month],
					'-',
					digits10[day], digits01[day],
					' ',
					digits10[hour], digits01[hour],
					':',
					digits10[minute], digits01[minute],
					':',
					digits10[second], digits01[second],
				}...)

				if micro != 0 {
					micro10000 := micro / 10000
					micro100 := micro / 100 % 100
					micro1 := micro % 100
					buf = append(buf, []byte{
						'.',
						digits10[micro10000], digits01[micro10000],
						digits10[micro100], digits01[micro100],
						digits10[micro1], digits01[micro1],
					}...)
				}
				buf = append(buf, '\'')
			}
		case string:
			if hexBlob {
				if utf8.ValidString(v) {
					buf = append(buf, '\'')
					buf = escapeBytesBackslash(buf, []byte(v))
				} else {
					buf = append(buf, 'X')
					buf = append(buf, '\'')
					b := hex.EncodeToString([]byte(v))
					buf = append(buf, b...)
				}
			} else {
				buf = append(buf, '\'')
				buf = escapeBytesBackslash(buf, []byte(v))
			}

			buf = append(buf, '\'')
		case []byte:
			if v == nil {
				buf = append(buf, "NULL"...)
			} else {
				// buf = append(buf, "_binary'"...)
				if hexBlob {
					if utf8.Valid(v) {
						buf = append(buf, '\'')
						buf = escapeBytesBackslash(buf, v)
					} else {
						buf = append(buf, 'X')
						buf = append(buf, '\'')
						b := hex.EncodeToString(v)
						buf = append(buf, b...)
					}
				} else {
					buf = append(buf, '\'')
					buf = escapeBytesBackslash(buf, v)
				}
				buf = append(buf, '\'')
			}
		default:
			log.Printf("%T", v)
			log.Error("解析错误")
			return nil, errors.New("driver: skip fast-path; continue as if unimplemented")
		}

		// 4 << 20 , 4MB
		if len(buf)+4 > 4<<20 {
			log.Error("解析错误")
			return nil, errors.New("driver: skip fast-path; continue as if unimplemented")
		}
	}
	if argPos != len(args) {
		log.Error("解析错误")
		return nil, errors.New("driver: skip fast-path; continue as if unimplemented")
	}
	return buf, nil
}

// reserveBuffer checks cap(buf) and expand buffer to len(buf) + appendSize.
// If cap(buf) is not enough, reallocate new buffer.
func reserveBuffer(buf []byte, appendSize int) []byte {
	newSize := len(buf) + appendSize
	if cap(buf) < newSize {
		// Grow buffer exponentially
		newBuf := make([]byte, len(buf)*2+appendSize)
		copy(newBuf, buf)
		buf = newBuf
	}
	return buf[:newSize]
}

// escapeBytesBackslash escapes []byte with backslashes (\)
// This escapes the contents of a string (provided as []byte) by adding backslashes before special
// characters, and turning others into specific escape sequences, such as
// turning newlines into \n and null bytes into \0.
// https://github.com/mysql/mysql-server/blob/mysql-5.7.5/mysys/charset.c#L823-L932
func escapeBytesBackslash(buf, v []byte) []byte {
	pos := len(buf)
	buf = reserveBuffer(buf, len(v)*2)

	for _, c := range v {
		switch c {
		case '\x00':
			buf[pos] = '\\'
			buf[pos+1] = '0'
			pos += 2
		case '\n':
			buf[pos] = '\\'
			buf[pos+1] = 'n'
			pos += 2
		case '\r':
			buf[pos] = '\\'
			buf[pos+1] = 'r'
			pos += 2
		case '\x1a':
			buf[pos] = '\\'
			buf[pos+1] = 'Z'
			pos += 2
		case '\'':
			buf[pos] = '\\'
			buf[pos+1] = '\''
			pos += 2
		case '"':
			buf[pos] = '\\'
			buf[pos+1] = '"'
			pos += 2
		case '\\':
			buf[pos] = '\\'
			buf[pos+1] = '\\'
			pos += 2
		default:
			buf[pos] = c
			pos++
		}
	}

	return buf[:pos]
}

// compareValue 比较两值是否相等
func compareValue(v1 interface{}, v2 interface{}) bool {
	equal := false

	// 处理特殊情况
	if v1 == nil && v2 == nil {
		return true
	}
	if v1 == nil || v2 == nil {
		return false
	}

	switch v := v1.(type) {
	case []byte:
		equal = byteEquals(v, v2.([]byte))
	case decimal.Decimal:
		if newDec, ok := v2.(decimal.Decimal); ok {
			equal = v.Equal(newDec)
		} else {
			equal = false
		}
	default:
		equal = v1 == v2
	}

	return equal
}

// byteEquals 判断字节切片是否相等
func byteEquals(v1, v2 []byte) bool {
	if len(v1) != len(v2) {
		return false
	}
	for i, v := range v1 {
		if v != v2[i] {
			return false
		}
	}
	return true
}

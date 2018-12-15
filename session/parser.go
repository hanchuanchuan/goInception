package session

import (
	"context"
	"database/sql/driver"
	"fmt"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/juju/errors"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"sync"
	"time"
)

const digits01 = "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789"
const digits10 = "0000000000111111111122222222223333333333444444444455555555556666666666777777777788888888889999999999"

type Column struct {
	gorm.Model
	Field     string `gorm:"Column:COLUMN_NAME"`
	Type      string `gorm:"Column:COLUMN_TYPE"`
	Collation string `gorm:"Column:COLLATION_NAME"`
	Key       string `gorm:"Column:COLUMN_KEY"`
	Comment   string `gorm:"Column:COLUMN_COMMENT"`

	// CharacterSetName string `gorm:"Column:CHARACTER_SET_NAME"`
}

type Table struct {
	tableId    uint64
	Fields     []Column
	hasPrimary bool
	primarys   map[int]bool
}

type ChanData struct {
	sql    []byte
	e      *replication.BinlogEvent
	gtid   []byte
	opid   string
	table  string
	record *Record
}

func (s *session) ProcessChan(wg *sync.WaitGroup) {
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
		if len(r.sql) == 0 {
			// log.Info("flush标志")
			s.flush(r.table, r.record)
		} else {
			s.myWrite(r.sql, r.e, r.opid, r.table, r.record)
		}
	}
}

func (s *session) GetNextBackupRecord() *Record {
	for {
		r := s.recordSets.Next()
		if r == nil {
			return nil
		}

		if r.AffectedRows > 0 && s.checkSqlIsDML(r) && r.TableInfo != nil {
			// 如果开始位置和结果位置相同,说明无变更(受影响行数为0)
			// if r.StartFile == r.EndFile && r.StartPosition == r.EndPosition {
			// 	continue
			// }

			// 先置默认值为备份失败,在备份完成后置为成功
			r.StageStatus = StatusBackupFail
			return r
		}
	}
}

func (s *session) Parser() {

	// 获取binlog解析起点
	record := s.GetNextBackupRecord()
	if record == nil {
		return
	}

	s.myRecord = record

	log.Debug("Parser")

	var err error
	var wg sync.WaitGroup

	// 启用Logger，显示详细日志
	// s.backupdb.LogMode(true)

	cfg := replication.BinlogSyncerConfig{
		ServerID: 2000000112,
		// Flavor:   p.cfg.Flavor,

		Host:     s.opt.host,
		Port:     uint16(s.opt.port),
		User:     s.opt.user,
		Password: s.opt.password,
		// RawModeEnabled:  p.cfg.RawMode,
		// SemiSyncEnabled: p.cfg.SemiSync,
	}

	b := replication.NewBinlogSyncer(cfg)
	defer b.Close()

	startPosition := mysql.Position{record.StartFile, uint32(record.StartPosition)}
	stopPosition := mysql.Position{record.EndFile, uint32(record.EndPosition)}
	s.lastBackupTable = fmt.Sprintf("`%s`.`%s`", record.BackupDBName, record.TableInfo.Name)
	startTime := time.Now()

	logSync, err := b.StartSync(startPosition)
	if err != nil {
		log.Infof("Start sync error: %v\n", errors.ErrorStack(err))
		return
	}

	wg.Add(1)
	s.ch = make(chan *ChanData, 50)
	go s.ProcessChan(&wg)

	// 最终关闭和返回
	defer func() {
		close(s.ch)
		wg.Wait()

		// log.Info("操作完成", "rows", i)
		// kwargs := map[string]interface{}{"ok": "1"}
		// sendMsg(p.cfg.SocketUser, "rollback_binlog_parse_complete", "binlog解析进度", "", kwargs)
	}()

	currentPosition := startPosition
	var currentThreadID uint32
	for {
		e, err := logSync.GetEvent(context.Background())
		if err != nil {
			log.Infof("Get event error: %v\n", errors.ErrorStack(err))
			break
		}

		if e.Header.LogPos > 0 {
			currentPosition.Pos = e.Header.LogPos
		}

		if e.Header.EventType == replication.ROTATE_EVENT {
			if event, ok := e.Event.(*replication.RotateEvent); ok {
				currentPosition = mysql.Position{string(event.NextLogName),
					uint32(event.Position)}
			}
		}

		// 如果还没有到操作的binlog范围,跳过
		if currentPosition.Compare(startPosition) == -1 {
			continue
		}

		if e.Header.EventType == replication.TABLE_MAP_EVENT {
			if event, ok := e.Event.(*replication.TableMapEvent); ok {
				if !strings.EqualFold(string(event.Schema), record.TableInfo.Schema) ||
					!strings.EqualFold(string(event.Table), record.TableInfo.Name) {
					goto ENDCHECK
				}
			}

		} else if e.Header.EventType == replication.QUERY_EVENT {
			if event, ok := e.Event.(*replication.QueryEvent); ok {
				currentThreadID = event.SlaveProxyID
			}

		} else if e.Header.EventType == replication.WRITE_ROWS_EVENTv2 {
			if event, ok := e.Event.(*replication.RowsEvent); ok {
				if !strings.EqualFold(string(event.Table.Schema), record.TableInfo.Schema) ||
					!strings.EqualFold(string(event.Table.Table), record.TableInfo.Name) {
					goto ENDCHECK
				}
				if record.ThreadId != currentThreadID {
					goto ENDCHECK
				}

				_, err = s.generateDeleteSql(record.TableInfo, event, e)
				s.checkError(err)
			}
		} else if e.Header.EventType == replication.DELETE_ROWS_EVENTv2 {
			if event, ok := e.Event.(*replication.RowsEvent); ok {
				if !strings.EqualFold(string(event.Table.Schema), record.TableInfo.Schema) ||
					!strings.EqualFold(string(event.Table.Table), record.TableInfo.Name) {
					goto ENDCHECK
				}
				if record.ThreadId != currentThreadID {
					goto ENDCHECK
				}

				_, err = s.generateInsertSql(record.TableInfo, event, e)
				s.checkError(err)
			}
		} else if e.Header.EventType == replication.UPDATE_ROWS_EVENTv2 {
			if event, ok := e.Event.(*replication.RowsEvent); ok {
				if !strings.EqualFold(string(event.Table.Schema), record.TableInfo.Schema) ||
					!strings.EqualFold(string(event.Table.Table), record.TableInfo.Name) {
					goto ENDCHECK
				}
				if record.ThreadId != currentThreadID {
					goto ENDCHECK
				}

				_, err = s.generateUpdateSql(record.TableInfo, event, e)
				s.checkError(err)
			}
		}

	ENDCHECK:
		// 如果操作已超过binlog范围,切换到下一日志
		if currentPosition.Compare(stopPosition) > -1 {
			record.StageStatus = StatusBackupOK
			record.BackupCostTime = fmt.Sprintf("%.3f", time.Since(startTime).Seconds())

			next := s.GetNextBackupRecord()
			if next != nil {
				startPosition = mysql.Position{next.StartFile, uint32(next.StartPosition)}
				stopPosition = mysql.Position{next.EndFile, uint32(next.EndPosition)}
				lastBackupTable := fmt.Sprintf("`%s`.`%s`", next.BackupDBName, next.TableInfo.Name)
				startTime = time.Now()

				if s.lastBackupTable != lastBackupTable {
					// log.Info(s.lastBackupTable, " >>> ", lastBackupTable)
					s.ch <- &ChanData{sql: nil, table: s.lastBackupTable, record: record}
					s.lastBackupTable = lastBackupTable
				}
				s.myRecord = next
				record = next

			} else {
				// log.Info("备份已全部解析,操作结束")
				break
			}
		}
	}
}

// 解析的sql写入缓存,并定期入库
func (s *session) myWrite(b []byte, binEvent *replication.BinlogEvent,
	opid string, table string, record *Record) {
	// log.Info("myWrite")
	b = append(b, ";"...)

	s.insertBuffer = append(s.insertBuffer, string(b), opid)

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

		err := s.backupdb.Exec(fmt.Sprintf(sql, table, values),
			s.insertBuffer...).Error
		if err != nil {
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
				record.StageStatus = StatusBackupFail
				record.AppendErrorMessage(myErr.Message)
				log.Error(myErr)
			}
		}
		s.BackupTotalRows += len(s.insertBuffer) / 2
		s.SetMyProcessInfo(record.Sql, time.Now(), s.BackupTotalRows*100/s.TotalChangeRows)
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
	s.ch <- &ChanData{sql: b, e: binEvent, opid: s.myRecord.OPID,
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
		if i < int(e.ColumnCount) {
			columnNames = append(columnNames, fmt.Sprintf(c, col.Field))
		}
	}

	paramValues := strings.Repeat("?,", int(e.ColumnCount))
	paramValues = strings.TrimRight(paramValues, ",")

	sql := fmt.Sprintf(template, e.Table.Schema, e.Table.Table,
		strings.Join(columnNames, ","), paramValues)

	for _, rows := range e.Rows {

		var vv []driver.Value
		for _, d := range rows {
			vv = append(vv, d)
			// if _, ok := d.([]byte); ok {
			//  log.Info().Msgf("%s:%q\n", t.Fields[j].Field, d)
			// } else {
			//  log.Info().Msgf("%s:%#v\n", t.Fields[j].Field, d)
			// }
		}

		r, err := InterpolateParams(sql, vv)
		s.checkError(err)

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

		r, err := InterpolateParams(newSql, vv)
		s.checkError(err)

		s.write(r, binEvent)

	}

	return string(buf), nil
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

	var sets []string

	for i, col := range t.Fields {
		if i < int(e.ColumnCount) {
			sets = append(sets, fmt.Sprintf(setValue, col.Field))
		}
	}

	sql := fmt.Sprintf(template, e.Table.Schema, e.Table.Table,
		strings.Join(sets, ","))

	// log.Info().Msg(sql)

	var (
		oldValues []driver.Value
		newValues []driver.Value
		newSql    string
	)
	// update时, Rows为2的倍数, 双数index为旧值,单数index为新值
	for i, rows := range e.Rows {

		if i%2 == 0 {
			// 旧值
			for _, d := range rows {
				newValues = append(newValues, d)
			}
		} else {
			// 新值
			columnNames = nil
			for j, d := range rows {

				if t.hasPrimary {
					_, ok := t.primarys[j]
					if ok {
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

			newValues = append(newValues, oldValues...)

			newSql = strings.Join([]string{sql, strings.Join(columnNames, " AND")}, "")
			// log.Info().Msg(newSql, len(newValues))
			r, err := InterpolateParams(newSql, newValues)
			s.checkError(err)

			s.write(r, binEvent)

			oldValues = nil
			newValues = nil
		}

	}

	return string(buf), nil
}

func InterpolateParams(query string, args []driver.Value) ([]byte, error) {
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

		switch v := arg.(type) {
		case int8:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int16:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int32:
			buf = strconv.AppendInt(buf, int64(v), 10)
		case int64:
			buf = strconv.AppendInt(buf, v, 10)
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
			buf = append(buf, '\'')
			buf = escapeBytesBackslash(buf, []byte(v))
			buf = append(buf, '\'')
		case []byte:
			if v == nil {
				buf = append(buf, "NULL"...)
			} else {
				// buf = append(buf, "_binary'"...)
				buf = append(buf, '\'')

				buf = escapeBytesBackslash(buf, v)

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

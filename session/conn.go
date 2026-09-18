// Copyright 2013 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/QL-LICENSE file.

// Copyright 2015 PingCAP, Inc.
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
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

const maxBadConnRetries = 2

// createNewConnection 用来创建新的连接
// 注意: 该方法可能导致driver: bad connection异常
func (s *session) createNewConnection(dbName string) {
	addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&autocommit=1&maxAllowedPacket=%d",
		s.opt.User, s.opt.Password, s.opt.Host, s.opt.Port,
		dbName, s.inc.DefaultCharset, s.inc.MaxAllowedPacket)

	db, err := gorm.Open("mysql", addr)

	if err != nil {
		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
		s.appendErrorMsg(err.Error())
		return
	}

	if s.db != nil {
		s.db.Close()
	}

	// 禁用日志记录器，不显示任何日志
	db.LogMode(false)

	// 为保证连接成功关闭,此处等待10ms
	time.Sleep(10 * time.Millisecond)

	s.db = db
}

// raw 执行sql语句,连接失败时自动重连,自动重置当前数据库
func (s *session) raw(sqlStr string) (rows *sql.Rows, err error) {
	// 连接断开无效时,自动重试
	for i := 0; i < maxBadConnRetries; i++ {
		rows, err = s.db.DB().Query(sqlStr)
		if err == nil {
			return
		}
		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
		if err == mysqlDriver.ErrInvalidConn {
			err1 := s.initConnection()
			if err1 != nil {
				return rows, err1
			}
			s.appendErrorMsg(err.Error())
			continue
		}
		return
	}
	return
}

// exec 执行sql语句,连接失败时自动重连,自动重置当前数据库
func (s *session) exec(sqlStr string, retry bool) (res sql.Result, err error) {
	// 连接断开无效时,自动重试
	for i := 0; i < maxBadConnRetries; i++ {
		res, err = s.db.DB().Exec(sqlStr)
		if err == nil {
			return
		}
		log.Errorf("con:%d [retry:%v] %v sql:%s",
			s.sessionVars.ConnectionID, i, err, sqlStr)

		if err == mysqlDriver.ErrInvalidConn {
			err1 := s.initConnection()
			if err1 != nil {
				return res, err1
			}
			if retry {
				s.appendWarningMessage(err.Error())
				continue
			}
			return
		}

		// 连接超时时自动重连数据库. 仅在超时设置超过10min时开启该功能
		if s.inc.WaitTimeout >= 600 {
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok &&
				myErr.Number == 1046 && s.dbName != "" {
				err1 := s.initConnection()
				if err1 != nil {
					return res, err1
				}
				s.appendWarningMessage("Database timeout reconnect.")
				continue
			}
		}
		return
	}
	return
}

// execDMLWithThreadID 在一条独占连接上执行 DML，并取回该连接的线程号。
//
// 备份解析 binlog 时按线程号过滤事件（checkFilter），线程号必须来自真正执行
// 语句的连接。原实现先经连接池执行语句，再用另一条池化查询 select connection_id()
// 取线程号；一旦连接池换连（如坏连接被 database/sql 透明替换、代理按语句断连），
// 线程号与实际执行连接不一致，binlog 中匹配不到变更行，语句会被误判为
// "Backup failed" 且无任何错误信息（相关 issue #666）。
//
// 使用独占连接（*sql.Conn）保证执行与取号在同一条连接上：用户持有的 *sql.Conn
// 在连接失效时返回错误而不是静默切换到别的连接。同时把影响 DML 语义的会话状态
// （当前库、备份所需的 binlog 变量、sql_safe_updates、lock_wait_timeout）在该连接
// 上对齐——这些 SET 原本只作用于池化连接，换连后即失效。
//
// 与 s.exec 的 retry=false 语义一致：不做坏连接重试，避免语句实际已执行导致重复提交。
func (s *session) execDMLWithThreadID(sqlStr string) (res sql.Result, err error, threadID uint32) {
	db := s.db.DB()
	ctx := context.Background()

	conn, err := db.Conn(ctx)
	if err != nil {
		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
		// 获取独占连接失败时退回常规执行路径，线程号由 fetchThreadID 兜底
		res, err = s.exec(sqlStr, false)
		return res, err, s.fetchThreadID()
	}
	defer conn.Close()

	if setErr := s.alignSessionStateOnConn(ctx, conn); setErr != nil {
		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, setErr)
		s.appendErrorMsg(setErr.Error())
		return nil, setErr, 0
	}

	res, err = conn.ExecContext(ctx, sqlStr)
	if err != nil {
		return res, err, 0
	}

	query := "select connection_id();"
	if s.isMiddleware() {
		query = s.opt.middlewareExtend + query
	}
	var tid uint64
	if qErr := conn.QueryRowContext(ctx, query).Scan(&tid); qErr != nil {
		// 语句本身已执行成功，取号失败仅记录日志，线程号交由调用方兜底
		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, qErr)
		return res, err, 0
	}

	// thread_id溢出处理，与 fetchThreadID 保持一致
	if tid > math.MaxUint32 {
		tid %= 1 << 32
	}
	return res, err, uint32(tid)
}

// alignSessionStateOnConn 把会影响 DML 执行语义的会话变量对齐到独占连接上。
// 这些变量在 initConnection/executeCommit 中通过 s.exec 只设置在池化连接上，
// 独占连接上需要重新设置。任一设置失败即返回错误（此时语句尚未执行）。
func (s *session) alignSessionStateOnConn(ctx context.Context, conn *sql.Conn) error {
	if s.dbName != "" {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE `%s`", s.dbName)); err != nil {
			return fmt.Errorf("use database %s: %w", s.dbName, err)
		}
	}

	if s.opt.Backup {
		if _, err := conn.ExecContext(ctx, "set session binlog_format='row';"); err != nil {
			return fmt.Errorf("set binlog_format=row: %w", err)
		}
		if _, err := conn.ExecContext(ctx, "set session binlog_row_image='FULL';"); err != nil {
			return fmt.Errorf("set binlog_row_image=FULL: %w", err)
		}
	}

	if s.inc.SqlSafeUpdates == 0 || s.inc.SqlSafeUpdates == 1 {
		if _, err := conn.ExecContext(ctx,
			fmt.Sprintf("set session sql_safe_updates=%d;", s.inc.SqlSafeUpdates)); err != nil {
			return fmt.Errorf("set sql_safe_updates=%d: %w", s.inc.SqlSafeUpdates, err)
		}
	}

	if s.inc.LockWaitTimeout > 0 {
		if _, err := conn.ExecContext(ctx,
			fmt.Sprintf("set session lock_wait_timeout=%d;", s.inc.LockWaitTimeout)); err != nil {
			return fmt.Errorf("set lock_wait_timeout=%d: %w", s.inc.LockWaitTimeout, err)
		}
	}

	return nil
}

// execDDL 执行sql语句,连接失败时自动重连,自动重置当前数据库
func (s *session) execDDL(sqlStr string, retry bool) (res sql.Result, err error) {
	// 连接断开无效时,自动重试
	for i := 0; i < maxBadConnRetries; i++ {
		res, err = s.ddlDB.DB().Exec(sqlStr)
		if err == nil {
			return
		}
		log.Errorf("con:%d %v sql:%s", s.sessionVars.ConnectionID, err, sqlStr)
		if err == mysqlDriver.ErrInvalidConn {
			err1 := s.initConnection()
			if err1 != nil {
				return res, err1
			}
			if retry {
				s.appendWarningMessage(err.Error())
				continue
			}
			return
		}
		return
	}
	return
}

// Raw 执行sql语句,连接失败时自动重连,自动重置当前数据库
func (s *session) rawScan(sqlStr string, dest interface{}) (err error) {
	// 连接断开无效时,自动重试
	for i := 0; i < maxBadConnRetries; i++ {
		err = s.db.Raw(sqlStr).Scan(dest).Error
		if err == nil {
			return
		}
		if err == mysqlDriver.ErrInvalidConn {
			log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
			err1 := s.initConnection()
			if err1 != nil {
				return err1
			}
			s.appendErrorMsg(err.Error())
			continue
		}
		return
	}
	return
}

// Raw 执行sql语句,连接失败时自动重连,自动重置当前数据库
func (s *session) rawDB(dest interface{}, sqlStr string, values ...interface{}) (err error) {
	// 连接断开无效时,自动重试
	for i := 0; i < maxBadConnRetries; i++ {
		err = s.db.Raw(sqlStr, values...).Scan(dest).Error
		if err == nil {
			return
		}
		if err == mysqlDriver.ErrInvalidConn {
			log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
			err1 := s.initConnection()
			if err1 != nil {
				return err1
			}
			s.appendErrorMsg(err.Error())
			continue
		}
		return
	}
	return
}

// initConnection 连接失败时自动重连,重连后重置当前数据库
func (s *session) initConnection() (err error) {
	name := s.dbName
	if name == "" {
		name = s.opt.db
	}

	// 连接断开无效时,自动重试
	for i := 0; i < maxBadConnRetries; i++ {
		if name == "" {
			err = s.db.DB().Ping()
		} else {
			err = s.db.Exec(fmt.Sprintf("USE `%s`", name)).Error
		}
		if err == nil {
			// 连接重连时,清除线程ID缓存
			// s.threadID = 0
			log.Infof("con:%d Database timeout reconnect", s.sessionVars.ConnectionID)
			return
		}

		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
		if err != mysqlDriver.ErrInvalidConn {
			if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
				s.appendErrorMsg(myErr.Message)
			} else {
				s.appendErrorMsg(err.Error())
			}
			return
		}
	}

	if err != nil {
		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
			s.appendErrorMsg(myErr.Message)
		} else {
			s.appendErrorMsg(err.Error())
		}
	}
	return
}

// // SwitchDatabase USE切换到当前数据库. (避免连接断开后当前数据库置空)
// func (s *session) SwitchDatabase(db *gorm.DB) error {
// 	name := s.DBName
// 	if name == "" {
// 		name = s.opt.db
// 	}
// 	if name == "" {
// 		return nil
// 	}

// 	// log.Infof("SwitchDatabase: %v", name)
// 	_, err := db.DB().Exec(fmt.Sprintf("USE `%s`", name))
// 	if err != nil {
// 		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
// 		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
// 			s.AppendErrorMessage(myErr.Message)
// 		} else {
// 			s.AppendErrorMessage(err.Error())
// 		}
// 	}
// 	return err
// }

// // GetDatabase 获取当前数据库
// func (s *session) GetDatabase() string {
// 	log.Debug("GetDatabase")

// 	var value string
// 	sql := "select database();"

// 	rows, err := s.Raw(sql)
// 	if rows != nil {
// 		defer rows.Close()
// 	}

// 	if err != nil {
// 		log.Errorf("con:%d %v", s.sessionVars.ConnectionID, err)
// 		if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
// 			s.AppendErrorMessage(myErr.Message)
// 		} else {
// 			s.AppendErrorMessage(err.Error())
// 		}
// 	} else {
// 		for rows.Next() {
// 			rows.Scan(&value)
// 		}
// 	}

// 	return value
// }

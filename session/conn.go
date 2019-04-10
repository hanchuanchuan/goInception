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
    "fmt"
    // "math"
    // "strings"
    "database/sql"
    "time"

    // "github.com/hanchuanchuan/goInception/ast"
    // "github.com/hanchuanchuan/goInception/expression"
    // "github.com/hanchuanchuan/goInception/mysql"
    // "github.com/hanchuanchuan/goInception/sessionctx/stmtctx"
    mysqlDriver "github.com/go-sql-driver/mysql"
    "github.com/jinzhu/gorm"
    log "github.com/sirupsen/logrus"
)

const maxBadConnRetries = 3

// createNewConnection 用来创建新的连接
// 注意: 该方法可能导致driver: bad connection异常
func (s *session) createNewConnection(dbName string) {
    addr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&maxAllowedPacket=4194304",
        s.opt.user, s.opt.password, s.opt.host, s.opt.port, dbName)

    db, err := gorm.Open("mysql", addr)

    if err != nil {
        log.Error(err)
        s.AppendErrorMessage(err.Error())
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

// Raw 执行sql语句,连接失败时自动重连,自动重置当前数据库
func (s *session) Raw(sqlStr string) (rows *sql.Rows, err error) {

    // 连接断开无效时,自动重试
    for i := 0; i < maxBadConnRetries; i++ {
        // rows, err = s.db.Raw(sqlStr).Rows()
        rows, err = s.db.DB().Query(sqlStr)
        if err == nil {
            return
        } else {
            log.Error(err)
            if err == mysqlDriver.ErrInvalidConn {
                err = s.initConnection()
                if err != nil {
                    return
                }
            } else {
                return
            }
        }
    }

    return
}

// Raw 执行sql语句,连接失败时自动重连,自动重置当前数据库
func (s *session) Exec(sqlStr string) (res sql.Result, err error) {

    // 连接断开无效时,自动重试
    for i := 0; i < maxBadConnRetries; i++ {
        res, err = s.db.DB().Exec(sqlStr)
        // err = res.Error
        if err == nil {
            return
        } else {
            if err == mysqlDriver.ErrInvalidConn {
                log.Error(err)
                err = s.initConnection()
                if err != nil {
                    return
                }
            } else {
                return
                // if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
                //     s.AppendErrorMessage(myErr.Message)
                // } else {
                //     s.AppendErrorMessage(err.Error())
                // }
                // break
            }
        }
    }

    return
}

// Raw 执行sql语句,连接失败时自动重连,自动重置当前数据库
func (s *session) RawScan(sqlStr string, dest interface{}) (err error) {

    // 连接断开无效时,自动重试
    for i := 0; i < maxBadConnRetries; i++ {
        err = s.db.Raw(sqlStr).Scan(dest).Error
        if err == nil {
            return
        } else {
            if err == mysqlDriver.ErrInvalidConn {
                log.Error(err)
                err = s.initConnection()
                if err != nil {
                    return
                }
            } else {
                return
            }
        }
    }

    return
}

// initConnection 连接失败时自动重连,重连后重置当前数据库
func (s *session) initConnection() (err error) {

    name := s.DBName
    if name == "" {
        name = "mysql"
    }

    // 连接断开无效时,自动重试
    for i := 0; i < maxBadConnRetries; i++ {
        if err = s.db.Exec(fmt.Sprintf("USE `%s`", name)).Error; err == nil {
            // 连接重连时,清除线程ID缓存
            s.threadID = 0
            return
        } else {
            if err != mysqlDriver.ErrInvalidConn {
                if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
                    s.AppendErrorMessage(myErr.Message)
                } else {
                    s.AppendErrorMessage(err.Error())
                }
                return
            }
            log.Error(err)
        }
    }

    if err != nil {
        log.Error(err)
        if myErr, ok := err.(*mysqlDriver.MySQLError); ok {
            s.AppendErrorMessage(myErr.Message)
        } else {
            s.AppendErrorMessage(err.Error())
        }
    }

    return
}

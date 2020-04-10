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
	"math"
	"strings"
	"time"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/mysql"
	"github.com/hanchuanchuan/goInception/sessionctx"
	"github.com/hanchuanchuan/goInception/sessionctx/variable"
	"github.com/hanchuanchuan/goInception/terror"
	"github.com/hanchuanchuan/goInception/types"
	"github.com/hanchuanchuan/goInception/util/timeutil"
	"github.com/pingcap/errors"
)

var (
	// All the exported errors are defined here:
	ErrIncorrectParameterCount = terror.ClassExpression.New(mysql.ErrWrongParamcountToNativeFct, mysql.MySQLErrName[mysql.ErrWrongParamcountToNativeFct])
	ErrDivisionByZero          = terror.ClassExpression.New(mysql.ErrDivisionByZero, mysql.MySQLErrName[mysql.ErrDivisionByZero])
	ErrRegexp                  = terror.ClassExpression.New(mysql.ErrRegexp, mysql.MySQLErrName[mysql.ErrRegexp])
	ErrOperandColumns          = terror.ClassExpression.New(mysql.ErrOperandColumns, mysql.MySQLErrName[mysql.ErrOperandColumns])
	ErrCutValueGroupConcat     = terror.ClassExpression.New(mysql.ErrCutValueGroupConcat, mysql.MySQLErrName[mysql.ErrCutValueGroupConcat])

	// All the un-exported errors are defined here:
	errFunctionNotExists             = terror.ClassExpression.New(mysql.ErrSpDoesNotExist, mysql.MySQLErrName[mysql.ErrSpDoesNotExist])
	errZlibZData                     = terror.ClassTypes.New(mysql.ErrZlibZData, mysql.MySQLErrName[mysql.ErrZlibZData])
	errIncorrectArgs                 = terror.ClassExpression.New(mysql.ErrWrongArguments, mysql.MySQLErrName[mysql.ErrWrongArguments])
	errUnknownCharacterSet           = terror.ClassExpression.New(mysql.ErrUnknownCharacterSet, mysql.MySQLErrName[mysql.ErrUnknownCharacterSet])
	errDefaultValue                  = terror.ClassExpression.New(mysql.ErrInvalidDefault, "invalid default value")
	errDeprecatedSyntaxNoReplacement = terror.ClassExpression.New(mysql.ErrWarnDeprecatedSyntaxNoReplacement, mysql.MySQLErrName[mysql.ErrWarnDeprecatedSyntaxNoReplacement])
	errBadField                      = terror.ClassExpression.New(mysql.ErrBadField, mysql.MySQLErrName[mysql.ErrBadField])
	errWarnAllowedPacketOverflowed   = terror.ClassExpression.New(mysql.ErrWarnAllowedPacketOverflowed, mysql.MySQLErrName[mysql.ErrWarnAllowedPacketOverflowed])
	errWarnOptionIgnored             = terror.ClassExpression.New(mysql.WarnOptionIgnored, mysql.MySQLErrName[mysql.WarnOptionIgnored])
	errTruncatedWrongValue           = terror.ClassExpression.New(mysql.ErrTruncatedWrongValue, mysql.MySQLErrName[mysql.ErrTruncatedWrongValue])
)

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

// IsCurrentTimestampExpr returns whether e is CurrentTimestamp expression.
func IsCurrentTimestampExpr(e ast.ExprNode) bool {
	if fn, ok := e.(*ast.FuncCallExpr); ok && fn.FnName.L == ast.CurrentTimestamp {
		return true
	}
	return false
}

// GetTimeValue gets the time value with type tp.
func GetTimeValue(ctx sessionctx.Context, v interface{}, tp byte, fsp int) (d types.Datum, err error) {
	value := types.Time{
		Type: tp,
		Fsp:  fsp,
	}

	defaultTime, err := getSystemTimestamp(ctx)
	if err != nil {
		return d, errors.Trace(err)
	}
	sc := ctx.GetSessionVars().StmtCtx
	if sc.TimeZone == nil {
		sc.TimeZone = timeutil.SystemLocation()
	}
	switch x := v.(type) {
	case string:
		upperX := strings.ToUpper(x)
		if upperX == strings.ToUpper(ast.CurrentTimestamp) {
			value.Time = types.FromGoTime(defaultTime.Truncate(time.Duration(math.Pow10(9-fsp)) * time.Nanosecond))
			if tp == mysql.TypeTimestamp || tp == mysql.TypeDatetime {
				err = value.ConvertTimeZone(time.Local, ctx.GetSessionVars().Location())
				if err != nil {
					return d, errors.Trace(err)
				}
			}
		} else if upperX == types.ZeroDatetimeStr {
			value, err = types.ParseTimeFromNum(sc, 0, tp, fsp)
			terror.Log(errors.Trace(err))
		} else {
			value, err = types.ParseTime(sc, x, tp, fsp)
			if err != nil {
				return d, errors.Trace(err)
			}
		}
	case *ast.ValueExpr:
		switch x.Kind() {
		case types.KindString:
			value, err = types.ParseTime(sc, x.GetString(), tp, fsp)
			if err != nil {
				return d, errors.Trace(err)
			}
		case types.KindInt64:
			value, err = types.ParseTimeFromNum(sc, x.GetInt64(), tp, fsp)
			if err != nil {
				return d, errors.Trace(err)
			}
		case types.KindNull:
			return d, nil
		default:
			return d, errors.Trace(errDefaultValue)
		}
	case *ast.FuncCallExpr:
		if x.FnName.L == ast.CurrentTimestamp {
			d.SetString(strings.ToUpper(ast.CurrentTimestamp))
			return d, nil
		}
		return d, errors.Trace(errDefaultValue)
	// case *ast.UnaryOperationExpr:
	// 	// support some expression, like `-1`
	// 	v, err := EvalAstExpr(ctx, x)
	// 	if err != nil {
	// 		return d, errors.Trace(err)
	// 	}
	// 	ft := types.NewFieldType(mysql.TypeLonglong)
	// 	xval, err := v.ConvertTo(ctx.GetSessionVars().StmtCtx, ft)
	// 	if err != nil {
	// 		return d, errors.Trace(err)
	// 	}

	// 	value, err = types.ParseTimeFromNum(sc, xval.GetInt64(), tp, fsp)
	// 	if err != nil {
	// 		return d, errors.Trace(err)
	// 	}
	default:
		return d, nil
	}
	d.SetMysqlTime(value)
	return d, nil
}

func getSystemTimestamp(ctx sessionctx.Context) (time.Time, error) {
	now := time.Now()

	if ctx == nil {
		return now, nil
	}

	sessionVars := ctx.GetSessionVars()
	timestampStr, err := variable.GetSessionSystemVar(sessionVars, "timestamp")
	if err != nil {
		return now, errors.Trace(err)
	}

	if timestampStr == "" {
		return now, nil
	}
	timestamp, err := types.StrToInt(sessionVars.StmtCtx, timestampStr)
	if err != nil {
		return time.Time{}, errors.Trace(err)
	}
	if timestamp <= 0 {
		return now, nil
	}
	return time.Unix(timestamp, 0), nil
}

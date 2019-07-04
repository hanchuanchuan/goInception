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
	"strings"

	"github.com/hanchuanchuan/goInception/mysql"
	"github.com/hanchuanchuan/goInception/terror"
	log "github.com/sirupsen/logrus"
)

var ErrorsMessage = map[int]string{}

var (
	ErrWrongValueForVar = terror.ClassVariable.New(mysql.ErrWrongValueForVar,
		mysql.MySQLErrName[mysql.ErrWrongValueForVar])
	ErrTruncatedWrongValue = terror.ClassVariable.New(mysql.ErrTruncatedWrongValue,
		mysql.MySQLErrName[mysql.ErrTruncatedWrongValue])
	ErrWrongTypeForVar = terror.ClassVariable.New(mysql.ErrWrongTypeForVar,
		mysql.MySQLErrName[mysql.ErrWrongTypeForVar])
)

const (
	ER_ERROR_FIRST = iota
	ER_NOT_SUPPORTED_YET
	ER_SQL_NO_SOURCE
	ER_SQL_NO_OP_TYPE
	ER_SQL_INVALID_OP_TYPE
	ER_PARSE_ERROR
	ER_SYNTAX_ERROR
	ER_REMOTE_EXE_ERROR
	ER_SHUTDOWN_COMPLETE
	ER_WITH_INSERT_FIELD
	ER_WITH_INSERT_VALUES
	ER_WRONG_VALUE_COUNT_ON_ROW
	ER_BAD_FIELD_ERROR
	ER_FIELD_SPECIFIED_TWICE
	ER_BAD_NULL_ERROR
	ER_NO_WHERE_CONDITION
	ER_NORMAL_SHUTDOWN
	ER_FORCING_CLOSE
	ER_CON_COUNT_ERROR
	ER_INVALID_COMMAND
	ER_SQL_INVALID_SOURCE
	ER_WRONG_DB_NAME
	EXIT_UNKNOWN_VARIABLE
	EXIT_UNKNOWN_OPTION
	ER_NO_DB_ERROR
	ER_WITH_LIMIT_CONDITION
	ER_WITH_ORDERBY_CONDITION
	ER_SELECT_ONLY_STAR
	ER_ORDERY_BY_RAND
	ER_ID_IS_UPER
	ER_UNKNOWN_COLLATION
	ER_INVALID_DATA_TYPE
	ER_NOT_ALLOWED_NULLABLE
	ER_DUP_FIELDNAME
	ER_WRONG_COLUMN_NAME
	ER_WRONG_AUTO_KEY
	ER_TABLE_CANT_HANDLE_AUTO_INCREMENT
	ER_FOREIGN_KEY
	ER_TOO_MANY_KEY_PARTS
	ER_TOO_LONG_IDENT
	ER_UDPATE_TOO_MUCH_ROWS
	ER_INSERT_TOO_MUCH_ROWS
	ER_WRONG_NAME_FOR_INDEX
	ER_TOO_MANY_KEYS
	ER_NOT_SUPPORTED_KEY_TYPE
	ER_WRONG_SUB_KEY
	ER_WRONG_KEY_COLUMN
	ER_TOO_LONG_KEY
	ER_MULTIPLE_PRI_KEY
	ER_DUP_KEYNAME
	ER_TOO_LONG_INDEX_COMMENT
	ER_DUP_INDEX
	ER_TEMP_TABLE_TMP_PREFIX
	ER_TABLE_CHARSET_MUST_UTF8
	ER_TABLE_CHARSET_MUST_NULL
	ER_TABLE_MUST_HAVE_COMMENT
	ER_COLUMN_HAVE_NO_COMMENT
	ER_TABLE_MUST_HAVE_PK
	ER_PARTITION_NOT_ALLOWED
	ER_USE_ENUM
	ER_USE_TEXT_OR_BLOB
	ER_COLUMN_EXISTED
	ER_COLUMN_NOT_EXISTED
	ER_CANT_DROP_FIELD_OR_KEY
	ER_INVALID_DEFAULT
	ER_USERNAME
	ER_HOSTNAME
	ER_NOT_VALID_PASSWORD
	ER_WRONG_STRING_LENGTH
	ER_BLOB_USED_AS_KEY
	ER_TOO_LONG_BAKDB_NAME
	ER_INVALID_BACKUP_HOST_INFO
	ER_BINLOG_CORRUPTED
	ER_NET_READ_ERROR
	ER_NETWORK_READ_EVENT_CHECKSUM_FAILURE
	ER_SLAVE_RELAY_LOG_WRITE_FAILURE
	ER_INCORRECT_GLOBAL_LOCAL_VAR
	ER_START_AS_BEGIN
	ER_OUTOFMEMORY
	ER_HAVE_BEGIN
	ER_NET_READ_INTERRUPTED
	ER_BINLOG_FORMAT_STATEMENT
	EXIT_NO_ARGUMENT_ALLOWED
	EXIT_ARGUMENT_REQUIRED
	EXIT_AMBIGUOUS_OPTION
	ER_ERROR_EXIST_BEFORE
	ER_UNKNOWN_SYSTEM_VARIABLE
	ER_UNKNOWN_CHARACTER_SET
	ER_END_WITH_COMMIT
	ER_DB_NOT_EXISTED_ERROR
	ER_TABLE_EXISTS_ERROR
	ER_INDEX_NAME_IDX_PREFIX
	ER_INDEX_NAME_UNIQ_PREFIX
	ER_AUTOINC_UNSIGNED
	ER_VARCHAR_TO_TEXT_LEN
	ER_CHAR_TO_VARCHAR_LEN
	ER_KEY_COLUMN_DOES_NOT_EXITS
	ER_INC_INIT_ERR
	ER_WRONG_ARGUMENTS
	ER_SET_DATA_TYPE_INT_BIGINT
	ER_TIMESTAMP_DEFAULT
	ER_CHARSET_ON_COLUMN
	ER_AUTO_INCR_ID_WARNING
	ER_ALTER_TABLE_ONCE
	ER_BLOB_CANT_HAVE_DEFAULT
	ER_END_WITH_SEMICOLON
	ER_NON_UNIQ_ERROR
	ER_TABLE_NOT_EXISTED_ERROR
	ER_UNKNOWN_TABLE
	ER_INVALID_GROUP_FUNC_USE
	ER_INDEX_USE_ALTER_TABLE
	ER_WITH_DEFAULT_ADD_COLUMN
	ER_TRUNCATED_WRONG_VALUE
	ER_TEXT_NOT_NULLABLE_ERROR
	ER_WRONG_VALUE_FOR_VAR
	ER_TOO_MUCH_AUTO_TIMESTAMP_COLS
	ER_INVALID_ON_UPDATE
	ER_DDL_DML_COEXIST
	ER_SLAVE_CORRUPT_EVENT
	ER_COLLATION_CHARSET_MISMATCH
	ER_NOT_SUPPORTED_ALTER_OPTION
	ER_CONFLICTING_DECLARATIONS
	ER_IDENT_USE_KEYWORD
	ER_VIEW_SELECT_CLAUSE
	ER_OSC_KILL_FAILED
	ER_NET_PACKETS_OUT_OF_ORDER
	ER_NOT_SUPPORTED_ITEM_TYPE
	ER_INVALID_IDENT
	ER_INCEPTION_EMPTY_QUERY
	ER_PK_COLS_NOT_INT
	ER_PK_TOO_MANY_PARTS
	ER_REMOVED_SPACES
	ER_CHANGE_COLUMN_TYPE
	ER_CANT_DROP_TABLE
	ER_CANT_DROP_DATABASE
	ER_WRONG_TABLE_NAME
	ER_CANT_SET_CHARSET
	ER_CANT_SET_COLLATION
	ER_CANT_SET_ENGINE
	ER_MUST_AT_LEAST_ONE_COLUMN
	ER_MUST_HAVE_COLUMNS
	ER_PRIMARY_CANT_HAVE_NULL
	ErrCantRemoveAllFields
	ErrNotFoundTableInfo
	ErrNotFoundThreadId
	ErrNotFoundMasterStatus
	ErrNonUniqTable
	ErrWrongUsage
	ErrDataTooLong
	ErrCharsetNotSupport
	ErrCollationNotSupport
	ErrTableCollationNotSupport
	ErrJsonTypeSupport
	ErrEngineNotSupport
	//ER_NULL_NAME_FOR_INDEX
	ErrMixOfGroupFuncAndFields
	ErrFieldNotInGroupBy
	ErrCantChangeColumnPosition
	ER_ERROR_LAST
)

var ErrorsDefault = map[int]string{
	ER_ERROR_FIRST:                         "HelloWorld",
	ER_NOT_SUPPORTED_YET:                   "Not supported statement type.",
	ER_SQL_NO_SOURCE:                       "The sql have no source information.",
	ER_SQL_NO_OP_TYPE:                      "The sql have no operation type.",
	ER_SQL_INVALID_OP_TYPE:                 "Invalid sql operation type.",
	ER_PARSE_ERROR:                         "%s near '%s' at line %d",
	ER_SYNTAX_ERROR:                        "You have an error in your SQL syntax, ",
	ER_REMOTE_EXE_ERROR:                    "Execute in source server failed.",
	ER_SHUTDOWN_COMPLETE:                   "Shutdown complete.",
	ER_WITH_INSERT_FIELD:                   "Set the field list for insert statements.",
	ER_WITH_INSERT_VALUES:                  "Set the values list for insert statements.",
	ER_WRONG_VALUE_COUNT_ON_ROW:            "Column count doesn't match value count at row %d.",
	ER_BAD_FIELD_ERROR:                     "Unknown column '%s' in '%s'.",
	ER_FIELD_SPECIFIED_TWICE:               "Column '%s' specified twice in table '%s'.",
	ER_BAD_NULL_ERROR:                      "Column '%s' cannot be null in %d row.",
	ER_NO_WHERE_CONDITION:                  "set the where condition for select statement.",
	ER_NORMAL_SHUTDOWN:                     "%s: Normal shutdown\n",
	ER_FORCING_CLOSE:                       "%s: Forcing close of thread %ld  user: '%s'\n",
	ER_CON_COUNT_ERROR:                     "Too many connections",
	ER_INVALID_COMMAND:                     "Invalid command.",
	ER_SQL_INVALID_SOURCE:                  "Invalid source infomation.",
	ER_WRONG_DB_NAME:                       "Incorrect database name '%s'.",
	EXIT_UNKNOWN_VARIABLE:                  "Exist incorrect variable.",
	EXIT_UNKNOWN_OPTION:                    "Exist incorrect option.",
	ER_NO_DB_ERROR:                         "No database selected.",
	ER_WITH_LIMIT_CONDITION:                "Limit is not allowed in update/delete statement.",
	ER_WITH_ORDERBY_CONDITION:              "Order by is not allowed in update/delete statement.",
	ER_SELECT_ONLY_STAR:                    "Select only star is not allowed.",
	ER_ORDERY_BY_RAND:                      "Order by rand is not allowed in select statement.",
	ER_ID_IS_UPER:                          "Identifier is not allowed to been upper-case.",
	ER_UNKNOWN_COLLATION:                   "Unknown collation: '%s'.",
	ER_INVALID_DATA_TYPE:                   "Not supported data type on field: '%s'.",
	ER_NOT_ALLOWED_NULLABLE:                "Column '%s' in table '%s' is not allowed to been nullable.",
	ER_DUP_FIELDNAME:                       "Duplicate column name '%s'.",
	ER_WRONG_COLUMN_NAME:                   "Incorrect column name '%s'.",
	ER_WRONG_AUTO_KEY:                      "Incorrect table definition; there can be only one auto column and it must be defined as a key.",
	ER_TABLE_CANT_HANDLE_AUTO_INCREMENT:    "The used table type doesn't support AUTO_INCREMENT columns.",
	ER_FOREIGN_KEY:                         "Foreign key is not allowed in table '%s'.",
	ER_TOO_MANY_KEY_PARTS:                  "Too many key parts in Key '%s' in table '%s' specified, max %d parts allowed.",
	ER_TOO_LONG_IDENT:                      "Identifier name '%s' is too long.",
	ER_UDPATE_TOO_MUCH_ROWS:                "Update(%d rows) more then %d rows.",
	ER_INSERT_TOO_MUCH_ROWS:                "Insert(%d rows) more then %d rows.",
	ER_WRONG_NAME_FOR_INDEX:                "Incorrect index name '%s' in table '%s'.",
	ER_TOO_MANY_KEYS:                       "Too many keys specified in table '%s', max %d keys allowed.",
	ER_NOT_SUPPORTED_KEY_TYPE:              "Not supported key type: '%s'.",
	ER_WRONG_SUB_KEY:                       "Incorrect prefix key; the used key part isn't a string, the used length is longer than the key part, or the storagengine doesn't support unique prefix keys",
	ER_WRONG_KEY_COLUMN:                    "The used storage engine can't index column '%s'.",
	ER_TOO_LONG_KEY:                        "Specified key '%s' was too long; max key length is %d bytes.",
	ER_MULTIPLE_PRI_KEY:                    "Multiple primary key defined.",
	ER_DUP_KEYNAME:                         "Duplicate key name '%s'.",
	ER_TOO_LONG_INDEX_COMMENT:              "Comment for index '%s' is too long (max = %lu).",
	ER_DUP_INDEX:                           "Duplicate index '%s' defined on the table '%s.%s'.",
	ER_TEMP_TABLE_TMP_PREFIX:               "Set 'tmp' prefix for temporary table.",
	ER_TABLE_CHARSET_MUST_UTF8:             "Set charset to one of '%s' for table '%s'.",
	ER_TABLE_CHARSET_MUST_NULL:             "Not allowed set charset for table '%s'.",
	ErrTableCollationNotSupport:            "Not allowed set collation for table '%s'.",
	ER_TABLE_MUST_HAVE_COMMENT:             "Set comments for table '%s'.",
	ER_COLUMN_HAVE_NO_COMMENT:              "Column '%s' in table '%s' have no comments.",
	ER_TABLE_MUST_HAVE_PK:                  "Set a primary key for table '%s'.",
	ER_PARTITION_NOT_ALLOWED:               "Partition is not allowed in table.",
	ER_USE_ENUM:                            "Type enum is used in column.",
	ER_USE_TEXT_OR_BLOB:                    "Type blob/text is used in column '%s'.",
	ER_COLUMN_EXISTED:                      "Column '%s' have existed.",
	ER_COLUMN_NOT_EXISTED:                  "Column '%s' not existed.",
	ER_CANT_DROP_FIELD_OR_KEY:              "Can't DROP '%s'; check that column/key exists.",
	ER_INVALID_DEFAULT:                     "Invalid default value for column '%s'.",
	ER_USERNAME:                            "user name",
	ER_HOSTNAME:                            "host name",
	ER_NOT_VALID_PASSWORD:                  "Your password does not satisfy the current policy requirements.",
	ER_WRONG_STRING_LENGTH:                 "String '%s' is too long for %s (should be no longer than %d).",
	ER_BLOB_USED_AS_KEY:                    "BLOB column '%s' can't be used in key specification with the used table type.",
	ER_TOO_LONG_BAKDB_NAME:                 "The backup dbname '%-s-%d-%s' is too long.",
	ER_INVALID_BACKUP_HOST_INFO:            "Invalid remote backup information.",
	ER_BINLOG_CORRUPTED:                    "Binlog is corrupted.",
	ER_NET_READ_ERROR:                      "Got an error reading communication packets.",
	ER_NETWORK_READ_EVENT_CHECKSUM_FAILURE: "Replication event checksum verification failed while reading from network.",
	ER_SLAVE_RELAY_LOG_WRITE_FAILURE:       "Relay log write failure: %s.",
	ER_INCORRECT_GLOBAL_LOCAL_VAR:          "Variable '%s' is a %s variable.",
	ER_START_AS_BEGIN:                      "Must start as begin statement.",
	ER_OUTOFMEMORY:                         "Out of memory; restart server and try again (needed %d bytes).",
	ER_HAVE_BEGIN:                          "Have you begin twice? Or you didn't commit last time, if so, you can execute commit explicitly.",
	ER_NET_READ_INTERRUPTED:                "Got timeout reading communication packets.",
	ER_BINLOG_FORMAT_STATEMENT:             "The binlog_format is statement, backup is disabled.",
	EXIT_NO_ARGUMENT_ALLOWED:               "Not allow set argument.",
	EXIT_ARGUMENT_REQUIRED:                 "Require argument.",
	EXIT_AMBIGUOUS_OPTION:                  "Ambiguous argument.",
	ER_ERROR_EXIST_BEFORE:                  "Exist error at before statement.",
	ER_UNKNOWN_SYSTEM_VARIABLE:             "Unknown system variable '%s'.",
	ER_UNKNOWN_CHARACTER_SET:               "Unknown character set: '%s'.",
	ER_END_WITH_COMMIT:                     "Must end with commit.",
	ER_DB_NOT_EXISTED_ERROR:                "Selected Database '%s' not existed.",
	ER_TABLE_EXISTS_ERROR:                  "Table '%s' already exists.",
	ER_INDEX_NAME_IDX_PREFIX:               "Index '%s' in table '%s' need 'idx_' prefix.",
	ER_INDEX_NAME_UNIQ_PREFIX:              "Index '%s' in table '%s' need 'uniq_' prefix.",
	ER_AUTOINC_UNSIGNED:                    "Set unsigned attribute on auto increment column in table '%s'.",
	ER_VARCHAR_TO_TEXT_LEN:                 "Set column '%s' to TEXT type.",
	ER_CHAR_TO_VARCHAR_LEN:                 "Set column '%s' to VARCHAR type.",
	ER_KEY_COLUMN_DOES_NOT_EXITS:           "Key column '%s' doesn't exist in table.",
	ER_INC_INIT_ERR:                        "Set auto-increment initialize value to 1.",
	ER_WRONG_ARGUMENTS:                     "Incorrect arguments to %s.",
	ER_SET_DATA_TYPE_INT_BIGINT:            "Set auto-increment data type to int or bigint.",
	ER_TIMESTAMP_DEFAULT:                   "Set default value for timestamp column '%s'.",
	ER_CHARSET_ON_COLUMN:                   "Not Allowed set charset for column '%s.%s'.",
	ER_AUTO_INCR_ID_WARNING:                "Auto increment column '%s' is meaningful? it's dangerous!",
	ER_ALTER_TABLE_ONCE:                    "Merge the alter statement for table '%s' to ONE.",
	ER_BLOB_CANT_HAVE_DEFAULT:              "BLOB, TEXT, GEOMETRY or JSON column '%s' can't have a default value.",
	ER_END_WITH_SEMICOLON:                  "Add ';' after the last sql statement.",
	ER_NON_UNIQ_ERROR:                      "Column '%s' in %s is ambiguous.",
	ER_TABLE_NOT_EXISTED_ERROR:             "Table '%s' doesn't exist.",
	ER_UNKNOWN_TABLE:                       "Unknown table '%s' in %s.",
	ER_INVALID_GROUP_FUNC_USE:              "Invalid use of group function.",
	ER_INDEX_USE_ALTER_TABLE:               "Create/drop index and rename is not allowed, please replace with alter statement.",
	ER_WITH_DEFAULT_ADD_COLUMN:             "Set Default value for column '%s' in table '%s'",
	ER_TRUNCATED_WRONG_VALUE:               "Truncated incorrect %s value: '%s'",
	ER_TEXT_NOT_NULLABLE_ERROR:             "TEXT/BLOB Column '%s' in table '%s' can't  been not null.",
	ER_WRONG_VALUE_FOR_VAR:                 "Variable '%s' can't be set to the value of '%s'",
	ER_TOO_MUCH_AUTO_TIMESTAMP_COLS:        "Incorrect table definition; there can be only one TIMESTAMP column with CURRENT_TIMESTAMP in DEFAULT or ON UPDATE clause",
	ER_INVALID_ON_UPDATE:                   "Invalid ON UPDATE clause for '%s' column",
	ER_DDL_DML_COEXIST:                     "DDL can not coexist with the DML for table '%s'.",
	ER_SLAVE_CORRUPT_EVENT:                 "Corrupted replication event was detected.",
	ER_COLLATION_CHARSET_MISMATCH:          "COLLATION '%s' is not valid for CHARACTER SET '%s'",
	ER_NOT_SUPPORTED_ALTER_OPTION:          "Not supported statement of alter option",
	ER_CONFLICTING_DECLARATIONS:            "Conflicting declarations: '%s%s' and '%s%s'",
	ER_IDENT_USE_KEYWORD:                   "Identifier '%s' is keyword in MySQL.",
	ER_VIEW_SELECT_CLAUSE:                  "View's SELECT contains a '%s' clause",
	ER_OSC_KILL_FAILED:                     "Can not find OSC executing task",
	ER_NET_PACKETS_OUT_OF_ORDER:            "Got packets out of order",
	ER_NOT_SUPPORTED_ITEM_TYPE:             "Not supported expression type '%s'.",
	ER_INVALID_IDENT:                       "Identifier '%s' is invalid, valid options: [a-z|A-Z|0-9|_].",
	ER_INCEPTION_EMPTY_QUERY:               "Inception error, Query was empty.",
	ER_PK_COLS_NOT_INT:                     "Primary key column '%s' is not int or bigint type in table '%s'.'%s'.",
	ER_PK_TOO_MANY_PARTS:                   "Too many primary key part in table '%s'.'%s', max parts: %d",
	ER_REMOVED_SPACES:                      "Leading spaces are removed from name '%s'",
	ER_CHANGE_COLUMN_TYPE:                  "Type conversion warning for column '%s' %s -> %s.",
	ER_CANT_DROP_TABLE:                     "Drop/truncate '%s' is not allowed, please replace with alter rename statement.",
	ER_CANT_DROP_DATABASE:                  "Command is forbidden! Cannot delete database '%s'.",
	ER_WRONG_TABLE_NAME:                    "Incorrect table name '%-.100s'",
	ER_CANT_SET_CHARSET:                    "Cannot set charset '%s'",
	ER_CANT_SET_COLLATION:                  "Cannot set collation '%s'",
	ER_CANT_SET_ENGINE:                     "Cannot set engine '%s'",
	ER_MUST_AT_LEAST_ONE_COLUMN:            "A table must have at least 1 column.",
	ER_MUST_HAVE_COLUMNS:                   "Must have the specified column: '%s'.",
	ER_PRIMARY_CANT_HAVE_NULL:              "All parts of a PRIMARY KEY must be NOT NULL; if you need NULL in a key, use UNIQUE instead",
	ErrCantRemoveAllFields:                 "You can't delete all columns with ALTER TABLE; use DROP TABLE instead",
	ErrNotFoundTableInfo:                   "Skip backup because there is no table structure information.",
	ErrNotFoundThreadId:                    "MariaDB v%d not supported yet,please confirm that the rollback sql is correct",
	ErrNotFoundMasterStatus:                "Can't found master binlog position.",
	ErrNonUniqTable:                        "Not unique table/alias: '%-.192s'.", // mysql.MySQLErrName[mysql.ErrNonuniqTable],
	ErrWrongUsage:                          "Incorrect usage of %s and %s.",
	ErrDataTooLong:                         "Data too long for column '%s' at row %d",
	ErrCharsetNotSupport:                   "Set charset to one of '%s'.",
	ErrCollationNotSupport:                 "Set collation to one of '%s'",
	ErrEngineNotSupport:                    "Set engine to one of '%s'",
	ErrJsonTypeSupport:                     "Json type not allowed in column '%s'.",
	ER_ERROR_LAST:                          "TheLastError,ByeBye",
	ErrMixOfGroupFuncAndFields:             "In aggregated query without GROUP BY, expression #%d of SELECT list contains nonaggregated column '%s'; this is incompatible with sql_mode=only_full_group_by.",
	ErrFieldNotInGroupBy:                   "Expression #%d of %s is not in GROUP BY clause and contains nonaggregated column '%s' which is not functionally dependent on columns in GROUP BY clause; this is incompatible with sql_mode=only_full_group_by.",
	ErrCantChangeColumnPosition:            "Cannot change the position of the column '%s'",
	// ErrMixOfGroupFuncAndFields:             "Mixing of GROUP columns (MIN(),MAX(),COUNT(),...) with no GROUP columns is illegal if there is no GROUP BY clause",
	//ER_NULL_NAME_FOR_INDEX:                 "Index name cannot be null in table '%s'.",
}

var ErrorsChinese = map[int]string{
	ER_NOT_SUPPORTED_YET:                "不支持的语法类型.",
	ER_SQL_NO_SOURCE:                    "sql没有源信息.",
	ER_SQL_NO_OP_TYPE:                   "sql没有操作类型设置.",
	ER_SQL_INVALID_OP_TYPE:              "无效的sql操作类型.",
	ER_PARSE_ERROR:                      "%s near '%s' at line %d",
	ER_SYNTAX_ERROR:                     "SQL语法有错误, ",
	ER_REMOTE_EXE_ERROR:                 "Execute in source server failed.",
	ER_SHUTDOWN_COMPLETE:                "Shutdown complete.",
	ER_WITH_INSERT_FIELD:                "insert语句需要指定字段列表.",
	ER_WITH_INSERT_VALUES:               "insert语句需要指定值列表.",
	ER_WRONG_VALUE_COUNT_ON_ROW:         "行 %d 的列数和值列表不匹配.",
	ER_BAD_FIELD_ERROR:                  "Unknown column '%s' in '%s'.",
	ER_FIELD_SPECIFIED_TWICE:            "列 '%s' 指定重复(表 '%s').",
	ER_BAD_NULL_ERROR:                   "列 '%s' 不能为null(第 %d 行).",
	ER_NO_WHERE_CONDITION:               "selete语句请指定where条件.",
	ER_NORMAL_SHUTDOWN:                  "%s: Normal shutdown\n",
	ER_FORCING_CLOSE:                    "%s: Forcing close of thread %ld  user: '%s'\n",
	ER_CON_COUNT_ERROR:                  "Too many connections",
	ER_INVALID_COMMAND:                  "Invalid command.",
	ER_SQL_INVALID_SOURCE:               "不正确的数据源信息.",
	ER_WRONG_DB_NAME:                    "不正确的的数据库名 '%s'.",
	EXIT_UNKNOWN_VARIABLE:               "Exist incorrect variable.",
	EXIT_UNKNOWN_OPTION:                 "Exist incorrect option.",
	ER_NO_DB_ERROR:                      "没有选择数据库.",
	ER_WITH_LIMIT_CONDITION:             "update/delete语句不允许Limit.",
	ER_WITH_ORDERBY_CONDITION:           "update/delete语句不允许Order by.",
	ER_SELECT_ONLY_STAR:                 "不允许'select *'语法.",
	ER_ORDERY_BY_RAND:                   "不允许'Order by rand'语法.",
	ER_ID_IS_UPER:                       "标识符不允许大写.",
	ER_UNKNOWN_COLLATION:                "未知的排序规则: '%s'.",
	ER_INVALID_DATA_TYPE:                "列 '%s' 数据类型不支持.",
	ER_NOT_ALLOWED_NULLABLE:             "列 '%s' 不允许为null(表 '%s').",
	ER_DUP_FIELDNAME:                    "重复的列名: '%s'.",
	ER_WRONG_COLUMN_NAME:                "不正确的列名: '%s'.",
	ER_WRONG_AUTO_KEY:                   "不正确的表定义,只能有一个自增列且必须为索引键.",
	ER_TABLE_CANT_HANDLE_AUTO_INCREMENT: "使用的表类型不支持自增列.",
	ER_FOREIGN_KEY:                      "不允许使用外键(表 '%s').",
	ER_TOO_MANY_KEY_PARTS:               "索引 '%s'指定了太多的字段(表 '%s'), 最多允许 %d 个字段.",
	ER_TOO_LONG_IDENT:                   "名称 '%s' 过长.",
	ER_UDPATE_TOO_MUCH_ROWS:             "预计一次更新(%d行)超过 %d 行.",
	ER_INSERT_TOO_MUCH_ROWS:             "一次新增(%d行)超过 %d 行.",
	ER_WRONG_NAME_FOR_INDEX:             "索引 '%s' 名称不正确(表 '%s').",
	ER_TOO_MANY_KEYS:                    "表 '%s' 指定了太多索引, 最多允许 %d 个.",
	ER_NOT_SUPPORTED_KEY_TYPE:           "不允许的键类型: '%s'.",
	ER_WRONG_SUB_KEY:                    "索引列不能指定长度或指定的长度超出字段长度.",
	// ER_WRONG_KEY_COLUMN:                    "The used storage engine can't index column '%s'.",
	ER_TOO_LONG_KEY:                        "索引 '%s' 过长; 最大长度为 %d 字节.",
	ER_MULTIPLE_PRI_KEY:                    "定义了多个主键.",
	ER_DUP_KEYNAME:                         "索引名 '%s' 重复.",
	ER_TOO_LONG_INDEX_COMMENT:              "索引 '%s' 注释过长(max = %lu).",
	ER_DUP_INDEX:                           "索引 '%s' 定义重复(表'%s.%s').",
	ER_TEMP_TABLE_TMP_PREFIX:               "临时表需要指定'tmp'前缀",
	ER_TABLE_CHARSET_MUST_UTF8:             "允许的字符集为: '%s'(表'%s').",
	ER_TABLE_CHARSET_MUST_NULL:             "表 '%s' 禁止设置字符集!",
	ErrTableCollationNotSupport:            "表 '%s' 禁止设置排序规则!",
	ER_TABLE_MUST_HAVE_COMMENT:             "表 '%s' 需要设置注释.",
	ER_COLUMN_HAVE_NO_COMMENT:              "列 '%s' 需要设置注释(表'%s').",
	ER_TABLE_MUST_HAVE_PK:                  "表 '%s' 需要设置主键.",
	ER_PARTITION_NOT_ALLOWED:               "不允许创建分区表.",
	ER_USE_ENUM:                            "不允许使用enum类型.",
	ER_USE_TEXT_OR_BLOB:                    "不允许使用 blob/text 类型(列'%s').",
	ER_COLUMN_EXISTED:                      "列 '%s' 已存在.",
	ER_COLUMN_NOT_EXISTED:                  "列 '%s' 不存在.",
	ER_CANT_DROP_FIELD_OR_KEY:              "无法删除 '%s'; 请检查字段/键是否存在.",
	ER_INVALID_DEFAULT:                     "列 '%s' 默认值无效.",
	ER_USERNAME:                            "user name",
	ER_HOSTNAME:                            "host name",
	ER_NOT_VALID_PASSWORD:                  "Your password does not satisfy the current policy requirements.",
	ER_WRONG_STRING_LENGTH:                 "String '%s' is too long for %s (should be no longer than %d).",
	ER_BLOB_USED_AS_KEY:                    "BLOB列 '%s' 不允许作为索引列.",
	ER_TOO_LONG_BAKDB_NAME:                 "备份库名 '%-s-%d-%s' 过长.",
	ER_INVALID_BACKUP_HOST_INFO:            "无效备份库信息.",
	ER_BINLOG_CORRUPTED:                    "Binlog is corrupted.",
	ER_NET_READ_ERROR:                      "Got an error reading communication packets.",
	ER_NETWORK_READ_EVENT_CHECKSUM_FAILURE: "Replication event checksum verification failed while reading from network.",
	ER_SLAVE_RELAY_LOG_WRITE_FAILURE:       "Relay log write failure: %s.",
	ER_INCORRECT_GLOBAL_LOCAL_VAR:          "Variable '%s' is a %s variable.",
	ER_START_AS_BEGIN:                      "必须以begin语句开始.",
	ER_OUTOFMEMORY:                         "Out of memory; restart server and try again (needed %d bytes).",
	ER_HAVE_BEGIN:                          "指定了多次begin.",
	ER_NET_READ_INTERRUPTED:                "Got timeout reading communication packets.",
	ER_BINLOG_FORMAT_STATEMENT:             "The binlog_format is statement, backup is disabled.",
	EXIT_NO_ARGUMENT_ALLOWED:               "Not allow set argument.",
	EXIT_ARGUMENT_REQUIRED:                 "Require argument.",
	EXIT_AMBIGUOUS_OPTION:                  "Ambiguous argument.",
	ER_ERROR_EXIST_BEFORE:                  "Exist error at before statement.",
	ER_UNKNOWN_SYSTEM_VARIABLE:             "Unknown system variable '%s'.",
	ER_UNKNOWN_CHARACTER_SET:               "Unknown character set: '%s'.",
	ER_END_WITH_COMMIT:                     "Must end with commit.",
	ER_DB_NOT_EXISTED_ERROR:                "选择的数据库 '%s' 不存在.",
	ER_TABLE_EXISTS_ERROR:                  "表 '%s' 已存在.",
	ER_INDEX_NAME_IDX_PREFIX:               "索引 '%s' 需要以'idx_'为前缀(表'%s').",
	ER_INDEX_NAME_UNIQ_PREFIX:              "索引 '%s' 需要以'uniq_'为前缀(表'%s').",
	ER_AUTOINC_UNSIGNED:                    "自增列建议设置无符号标志unsigned(表'%s').",
	ER_VARCHAR_TO_TEXT_LEN:                 "列 '%s' 建议设置为text类型.",
	ER_CHAR_TO_VARCHAR_LEN:                 "列 '%s' 建议设置为varchar类型.",
	ER_KEY_COLUMN_DOES_NOT_EXITS:           "列 '%s' 不存在.",
	ER_INC_INIT_ERR:                        "建议自增列初始值置为 1.",
	ER_WRONG_ARGUMENTS:                     "Incorrect arguments to %s.",
	ER_SET_DATA_TYPE_INT_BIGINT:            "自增列需要设置为int或bigint类型.",
	ER_TIMESTAMP_DEFAULT:                   "请设置timestamp列 '%s' 的默认值.",
	ER_CHARSET_ON_COLUMN:                   "表 '%s' 列 '%s' 禁止设置字符集!",
	ER_AUTO_INCR_ID_WARNING:                "自增列('%s')建议命名为'ID'.",
	ER_ALTER_TABLE_ONCE:                    "表 '%s' 的多个alter操作请合并成一个.",
	ER_BLOB_CANT_HAVE_DEFAULT:              "BLOB,TEXT,GEOMETRY或JSON列 '%s' 禁止设置默认值.",
	ER_END_WITH_SEMICOLON:                  "Add ';' after the last sql statement.",
	ER_NON_UNIQ_ERROR:                      "Column '%s' in %s is ambiguous.",
	ER_TABLE_NOT_EXISTED_ERROR:             "表 '%s' 不存在.",
	ER_UNKNOWN_TABLE:                       "Unknown table '%s' in %s.",
	ER_INVALID_GROUP_FUNC_USE:              "Invalid use of group function.",
	ER_INDEX_USE_ALTER_TABLE:               "暂不支持create/drop index和rename语法,请使用alter语句替换.",
	ER_WITH_DEFAULT_ADD_COLUMN:             "表 '%s' 请设置默认值(表'%s')",
	ER_TRUNCATED_WRONG_VALUE:               "Truncated incorrect %s value: '%s'",
	ER_TEXT_NOT_NULLABLE_ERROR:             "TEXT/BLOB 列 '%s' 禁止设置为not null(表'%s').",
	ER_WRONG_VALUE_FOR_VAR:                 "Variable '%s' can't be set to the value of '%s'",
	ER_TOO_MUCH_AUTO_TIMESTAMP_COLS:        "表定义不正确,只能有一个TIMESTAMP字段在DEFAULT或ON UPDATE指定CURRENT_TIMESTAMP.",
	ER_INVALID_ON_UPDATE:                   "列 %s' ON UPDATE 设置无效",
	ER_DDL_DML_COEXIST:                     "DDL can not coexist with the DML for table '%s'.",
	ER_SLAVE_CORRUPT_EVENT:                 "Corrupted replication event was detected.",
	ER_COLLATION_CHARSET_MISMATCH:          "COLLATION '%s' is not valid for CHARACTER SET '%s'",
	ER_NOT_SUPPORTED_ALTER_OPTION:          "Not supported statement of alter option",
	ER_CONFLICTING_DECLARATIONS:            "Conflicting declarations: '%s%s' and '%s%s'",
	ER_IDENT_USE_KEYWORD:                   "标识符 '%s' 是MySQL关键字.",
	ER_VIEW_SELECT_CLAUSE:                  "View's SELECT contains a '%s' clause",
	ER_OSC_KILL_FAILED:                     "Can not find OSC executing task",
	ER_NET_PACKETS_OUT_OF_ORDER:            "Got packets out of order",
	ER_NOT_SUPPORTED_ITEM_TYPE:             "Not supported expression type '%s'.",
	ER_INVALID_IDENT:                       "标识符 '%s' 无效, 允许字符为 [a-z|A-Z|0-9|_].",
	ER_INCEPTION_EMPTY_QUERY:               "Inception error, Query was empty.",
	ER_PK_COLS_NOT_INT:                     "主键列 '%s' 建议使用int或bigint类型(表'%s'.'%s').",
	ER_PK_TOO_MANY_PARTS:                   "表 '%s'.'%s' 主键指定了太多的字段, 最多允许 %d 个字段",
	ER_REMOVED_SPACES:                      "Leading spaces are removed from name '%s'",
	ER_CHANGE_COLUMN_TYPE:                  "类型转换警告: 列 '%s' %s -> %s.",
	ER_CANT_DROP_TABLE:                     "禁用【DROP】|【TRUNCATE】删除/清空表 '%s', 请改用RENAME重写.",
	ER_CANT_DROP_DATABASE:                  "命令禁止! 无法删除数据库'%s'.",
	ER_WRONG_TABLE_NAME:                    "不正确的表名: '%-.100s'",
	ER_CANT_SET_CHARSET:                    "禁止指定字符集: '%s'",
	ER_CANT_SET_COLLATION:                  "禁止指定排序规则: '%s'",
	ER_CANT_SET_ENGINE:                     "禁止指定存储引擎:'%s'",
	ER_MUST_AT_LEAST_ONE_COLUMN:            "表至少需要有一个列.",
	ER_MUST_HAVE_COLUMNS:                   "表必须包含以下列: '%s'.",
	ER_PRIMARY_CANT_HAVE_NULL:              "主键的所有列必须为NOT NULL,如需要NULL列,请改用唯一索引",
	ErrCantRemoveAllFields:                 "禁止删除表的所有列.",
	ErrNotFoundTableInfo:                   "没有表结构信息,跳过备份.",
	ErrNotFoundThreadId:                    "MariaDB v%d 对回滚支持不完美,请注意确认回滚语句是否正确",
	ErrNotFoundMasterStatus:                "无法获取master binlog信息.",
	ErrNonUniqTable:                        "表名或别名: '%-.192s' 不唯一.",
	ErrDataTooLong:                         "数据过长!(列 '%s',行 '%d')",
	ErrCharsetNotSupport:                   "允许的字符集: '%s'.",
	ErrCollationNotSupport:                 "允许的排序规则: '%s'.",
	ErrEngineNotSupport:                    "允许的存储引擎: '%s'.",
	ErrWrongUsage:                          "%s子句无法使用%s",
	ErrJsonTypeSupport:                     "不允许使用json类型(列'%s').",
	ErrCantChangeColumnPosition:            "不允许改变列顺序(列'%s').",
	//ER_NULL_NAME_FOR_INDEX:                 "在表 '%s' 中, 索引名称不能为空.",
}

func GetErrorLevel(errorNo int) uint8 {
	switch errorNo {
	case ER_WITH_INSERT_FIELD,
		ER_NO_WHERE_CONDITION,
		ER_WITH_ORDERBY_CONDITION,
		ER_SELECT_ONLY_STAR,
		ER_ORDERY_BY_RAND,
		ER_UNKNOWN_COLLATION,
		ER_INVALID_DATA_TYPE,
		ER_NOT_ALLOWED_NULLABLE,
		ER_TOO_MANY_KEY_PARTS,
		ER_UDPATE_TOO_MUCH_ROWS,
		ER_INSERT_TOO_MUCH_ROWS,
		ER_TOO_MANY_KEYS,
		ER_PK_TOO_MANY_PARTS,
		ER_PK_COLS_NOT_INT,
		ER_TIMESTAMP_DEFAULT,
		ER_CHAR_TO_VARCHAR_LEN,
		ER_USE_ENUM,
		ER_OUTOFMEMORY,
		ER_INC_INIT_ERR,
		ER_CHARSET_ON_COLUMN,
		ER_IDENT_USE_KEYWORD,
		ER_TABLE_CHARSET_MUST_UTF8,
		ER_TABLE_CHARSET_MUST_NULL,
		ErrTableCollationNotSupport,
		ER_AUTO_INCR_ID_WARNING,
		ER_ALTER_TABLE_ONCE,
		ER_BLOB_CANT_HAVE_DEFAULT,
		ER_WITH_DEFAULT_ADD_COLUMN,

		ER_NOT_SUPPORTED_ALTER_OPTION,
		ER_COLUMN_HAVE_NO_COMMENT,
		ER_TABLE_MUST_HAVE_COMMENT,
		ER_WITH_LIMIT_CONDITION,
		ER_INDEX_NAME_IDX_PREFIX,
		ER_INDEX_NAME_UNIQ_PREFIX,
		ER_AUTOINC_UNSIGNED,
		ER_PARTITION_NOT_ALLOWED,
		ER_TABLE_MUST_HAVE_PK,
		ER_TOO_LONG_INDEX_COMMENT,
		ER_TEXT_NOT_NULLABLE_ERROR,
		ER_INVALID_IDENT,
		ER_CANT_SET_CHARSET,
		ER_CANT_SET_COLLATION,
		ER_CANT_SET_ENGINE,
		ErrNotFoundTableInfo,
		ErrNotFoundThreadId,
		ER_MUST_HAVE_COLUMNS,
		ErrCantChangeColumnPosition,
		ER_CHANGE_COLUMN_TYPE:
		return 1

	case ER_CONFLICTING_DECLARATIONS,
		ER_NO_DB_ERROR,
		ER_KEY_COLUMN_DOES_NOT_EXITS,
		ER_TOO_LONG_BAKDB_NAME,
		ER_DB_NOT_EXISTED_ERROR,
		ER_TABLE_EXISTS_ERROR,
		ER_COLUMN_EXISTED,
		ER_START_AS_BEGIN,
		ER_COLUMN_NOT_EXISTED,
		ER_WRONG_STRING_LENGTH,
		ER_BLOB_USED_AS_KEY,
		ER_INVALID_DEFAULT,
		ER_NOT_SUPPORTED_KEY_TYPE,
		ER_DUP_INDEX,
		ER_TEMP_TABLE_TMP_PREFIX,
		ER_TOO_LONG_KEY,
		ER_MULTIPLE_PRI_KEY,
		ER_DUP_KEYNAME,
		ER_DUP_FIELDNAME,
		ER_WRONG_KEY_COLUMN,
		ER_WRONG_COLUMN_NAME,
		ER_WRONG_AUTO_KEY,
		ER_WRONG_SUB_KEY,
		ER_WRONG_NAME_FOR_INDEX,
		ER_TOO_LONG_IDENT,
		ER_SQL_INVALID_SOURCE,
		ER_WRONG_DB_NAME,
		ER_WITH_INSERT_VALUES,
		ER_WRONG_VALUE_COUNT_ON_ROW,
		ER_BAD_FIELD_ERROR,
		ER_FIELD_SPECIFIED_TWICE,
		ER_SQL_NO_SOURCE,
		ER_PARSE_ERROR,
		ER_SYNTAX_ERROR,
		ER_END_WITH_SEMICOLON,
		ER_INDEX_USE_ALTER_TABLE,
		ER_INVALID_GROUP_FUNC_USE,
		ER_TABLE_NOT_EXISTED_ERROR,
		ER_UNKNOWN_TABLE,
		ER_TOO_MUCH_AUTO_TIMESTAMP_COLS,
		ER_INVALID_ON_UPDATE,
		ER_NON_UNIQ_ERROR,
		ER_DDL_DML_COEXIST,
		ER_COLLATION_CHARSET_MISMATCH,
		ER_VIEW_SELECT_CLAUSE,
		ER_NOT_SUPPORTED_ITEM_TYPE,
		ER_CANT_DROP_TABLE,
		ER_CANT_DROP_DATABASE,
		ER_CANT_DROP_FIELD_OR_KEY,
		ER_NOT_SUPPORTED_YET,
		ErrCharsetNotSupport,
		ErrCollationNotSupport,
		ErrEngineNotSupport,
		ER_FOREIGN_KEY,
		ER_INCEPTION_EMPTY_QUERY:
		//ER_NULL_NAME_FOR_INDEX:
		return 2

	default:
		return 2
	}
}

func GetErrorMessage(errorNo int) string {
	if v, ok := ErrorsMessage[errorNo]; ok {
		return v
	}
	if v, ok := ErrorsDefault[errorNo]; ok {
		return v
	}
	return "Invalid error code!"
}

// SQLError records an error information, from executing SQL.
type SQLError struct {
	Code    int
	Message string
}

// Error prints errors, with a formatted string.
func (e *SQLError) Error() string {
	return e.Message
}

// NewErr generates a SQL error, with an error code and default format specifier defined in MySQLErrName.
func NewErr(errCode int, args ...interface{}) *SQLError {
	e := &SQLError{Code: errCode}
	e.Message = fmt.Sprintf(GetErrorMessage(errCode), args...)
	return e
}

// NewErrf creates a SQL error, with an error code and a format specifier.
func NewErrf(format string, args ...interface{}) *SQLError {
	e := &SQLError{Code: 0}
	e.Message = fmt.Sprintf(format, args...)
	return e
}

func SetLanguage(langStr string) {
	lang := strings.Replace(strings.ToLower(langStr), "-", "_", 1)
	if lang == "zh_cn" {
		ErrorsMessage = ErrorsChinese
	} else {
		ErrorsMessage = ErrorsDefault
		if lang != "en_us" {
			log.Warning("Lang set Error! use default en-US.")
		}
	}
}

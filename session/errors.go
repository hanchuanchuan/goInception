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
	// "strconv"

	"github.com/hanchuanchuan/goInception/mysql"
	"github.com/hanchuanchuan/goInception/terror"
)

//go:generate stringer -type=ErrorCode
type ErrorCode int

var (
	ErrWrongValueForVar = terror.ClassVariable.New(mysql.ErrWrongValueForVar,
		mysql.MySQLErrName[mysql.ErrWrongValueForVar])
	ErrTruncatedWrongValue = terror.ClassVariable.New(mysql.ErrTruncatedWrongValue,
		mysql.MySQLErrName[mysql.ErrTruncatedWrongValue])
	ErrWrongTypeForVar = terror.ClassVariable.New(mysql.ErrWrongTypeForVar,
		mysql.MySQLErrName[mysql.ErrWrongTypeForVar])
)

const (
	ER_ERROR_FIRST ErrorCode = iota
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
	ER_NO_DB_ERROR
	ER_WITH_LIMIT_CONDITION
	ER_WITH_ORDERBY_CONDITION
	ER_SELECT_ONLY_STAR
	ER_ORDERY_BY_RAND
	ER_ID_IS_UPER
	ErrUnknownCharset
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
	ER_CHANGE_TOO_MUCH_ROWS
	ER_WRONG_NAME_FOR_INDEX
	ER_TOO_MANY_KEYS
	ER_NOT_SUPPORTED_KEY_TYPE
	ER_WRONG_SUB_KEY
	ER_WRONG_KEY_COLUMN
	ER_TOO_LONG_KEY
	ER_MULTIPLE_PRI_KEY
	ER_DUP_KEYNAME
	ER_TOO_LONG_INDEX_COMMENT
	ER_CANT_ADD_PK_OR_UK_COLUMN
	ER_DUP_INDEX
	ER_INDEX_COLUMN_REPEAT
	ER_TEMP_TABLE_TMP_PREFIX
	ER_TABLE_PREFIX
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
	ER_CANT_DROP_INDEX_COLUMN
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
	ER_ERROR_EXIST_BEFORE
	ER_UNKNOWN_SYSTEM_VARIABLE
	ER_UNKNOWN_CHARACTER_SET
	ER_END_WITH_COMMIT
	ER_DB_NOT_EXISTED_ERROR
	ER_TABLE_EXISTS_ERROR
	ER_TABLE_GROUP_EXISTS_ERROR
	ER_TABLE_GROUP_NOT_EXISTED_ERROR
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
	ER_IDENT_USE_CUSTOM_KEYWORD
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
	ER_CANT_CHANGE_COLUMN_TYPE
	ER_CANT_DROP_TABLE
	ER_CANT_DROP_DATABASE
	ER_WRONG_TABLE_NAME
	ER_CANT_SET_CHARSET
	ER_CANT_SET_COLLATION
	ER_CANT_SET_ENGINE
	ER_MUST_AT_LEAST_ONE_COLUMN
	ER_MUST_HAVE_COLUMNS
	ErrColumnsMustHaveIndex
	ErrColumnsMustHaveIndexTypeErr
	ER_PRIMARY_CANT_HAVE_NULL
	ErrCantRemoveAllFields
	ErrNotFoundTableInfo
	ErrMariaDBRollbackWarn
	ErrNotFoundMasterStatus
	ErrNonUniqTable
	ErrWrongUsage
	ErrDataTooLong
	ErrCharsetNotSupport
	ErrCollationNotSupport
	ErrTableCollationNotSupport
	ErrJsonTypeSupport
	ErrEngineNotSupport
	ErrMixOfGroupFuncAndFields
	ErrFieldNotInGroupBy
	ErCantChangeColumnPosition
	ErCantChangeColumn
	ER_DATETIME_DEFAULT
	ER_TOO_MUCH_AUTO_DATETIME_COLS
	ErrFloatDoubleToDecimal
	ErrIdentifierUpper
	ErrIdentifierLower
	ErrWrongAndExpr
	ErrCannotAddForeign
	ErrWrongFkDefWithMatch
	ErrFkDupName
	ErrJoinNoOnCondition
	ErrImplicitTypeConversion
	ErrUseValueExpr
	ErrUseIndexVisibility
	ErrViewSupport
	ErrViewColumnCount
	ErrIncorrectDateTimeValue
	ErrSameNamePartition
	ErrRepeatConstDefinition
	ErrPartitionNotExisted
	ErrIndexNotExisted
	ErrMaxVarcharLength
	ErrMaxColumnCount
	ER_ERROR_LAST
	ER_TOOL_BASED_UNIQUE_INDEX_WARNING
)

var ErrorsDefault = map[ErrorCode]string{
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
	ER_NO_WHERE_CONDITION:                  "Please set the where condition.",
	ER_NORMAL_SHUTDOWN:                     "%s: Normal shutdown\n",
	ER_FORCING_CLOSE:                       "%s: Forcing close of thread %ld  user: '%s'\n",
	ER_CON_COUNT_ERROR:                     "Too many connections",
	ER_INVALID_COMMAND:                     "Invalid command.",
	ER_SQL_INVALID_SOURCE:                  "Invalid source infomation(%s).",
	ER_WRONG_DB_NAME:                       "Incorrect database name '%s'.",
	ER_NO_DB_ERROR:                         "No database selected.",
	ER_WITH_LIMIT_CONDITION:                "Limit is not allowed in update/delete statement.",
	ER_WITH_ORDERBY_CONDITION:              "Order by is not allowed in update/delete statement.",
	ER_SELECT_ONLY_STAR:                    "Select only star is not allowed.",
	ER_ORDERY_BY_RAND:                      "Order by rand is not allowed in select statement.",
	ER_ID_IS_UPER:                          "Identifier is not allowed to been upper-case.",
	ErrUnknownCharset:                      "Unknown charset: '%s'.",
	ER_UNKNOWN_COLLATION:                   "Unknown collation: '%s'.",
	ER_INVALID_DATA_TYPE:                   "Not supported data type on field: '%s'(%s).",
	ER_NOT_ALLOWED_NULLABLE:                "Column '%s' in table '%s' is not allowed to been nullable.",
	ER_DUP_FIELDNAME:                       "Duplicate column name '%s'.",
	ER_WRONG_COLUMN_NAME:                   "Incorrect column name '%s'.",
	ER_WRONG_AUTO_KEY:                      "Incorrect table definition; there can be only one auto column and it must be defined as a key.",
	ER_TABLE_CANT_HANDLE_AUTO_INCREMENT:    "The used table type doesn't support AUTO_INCREMENT columns.",
	ER_FOREIGN_KEY:                         "Foreign key is not allowed in table '%s'.",
	ER_TOO_MANY_KEY_PARTS:                  "Too many key parts in Key '%s' in table '%s' specified, max %d parts allowed.",
	ER_TOO_LONG_IDENT:                      "Identifier name '%s' is too long.",
	ER_UDPATE_TOO_MUCH_ROWS:                "Update(%d rows) more than %d rows.",
	ER_INSERT_TOO_MUCH_ROWS:                "Insert(%d rows) more than %d rows.",
	ER_CHANGE_TOO_MUCH_ROWS:                "%s(%d rows) more than %d rows.",
	ER_WRONG_NAME_FOR_INDEX:                "Incorrect index name '%s' in table '%s'.",
	ER_TOO_MANY_KEYS:                       "Too many keys specified in table '%s', max %d keys allowed.",
	ER_NOT_SUPPORTED_KEY_TYPE:              "Not supported key type: '%s'.",
	ER_WRONG_SUB_KEY:                       "Incorrect prefix key; the used key part isn't a string, the used length is longer than the key part, or the storagengine doesn't support unique prefix keys",
	ER_WRONG_KEY_COLUMN:                    "The used storage engine can't index column '%s'.",
	ER_TOO_LONG_KEY:                        "Specified key '%s' was too long; max key length is %d bytes.",
	ER_MULTIPLE_PRI_KEY:                    "Multiple primary key defined.",
	ER_DUP_KEYNAME:                         "Duplicate key name '%s'.",
	ER_TOO_LONG_INDEX_COMMENT:              "Comment for index '%s' is too long (max = %lu).",
	ER_CANT_ADD_PK_OR_UK_COLUMN:            "Can't add PK or UK column '%s'.",
	ER_DUP_INDEX:                           "Duplicate index '%s' defined on the table '%s.%s'.",
	ER_INDEX_COLUMN_REPEAT:                 "Column repeat index '%s' defined on the table '%s.%s' column('%s').",
	ER_TEMP_TABLE_TMP_PREFIX:               "Set 'tmp' prefix for temporary table.",
	ER_TABLE_PREFIX:                        "Need set '%s' prefix for table.",
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
	ER_CANT_DROP_INDEX_COLUMN:              "Can't DROP index column '%s'.",
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
	ER_ERROR_EXIST_BEFORE:                  "Exist error at before statement.",
	ER_UNKNOWN_SYSTEM_VARIABLE:             "Unknown system variable '%s'.",
	ER_UNKNOWN_CHARACTER_SET:               "Unknown character set: '%s'.",
	ER_END_WITH_COMMIT:                     "Must end with commit.",
	ER_DB_NOT_EXISTED_ERROR:                "Selected Database '%s' not existed.",
	ER_TABLE_EXISTS_ERROR:                  "Table '%s' already exists.",
	ER_TABLE_GROUP_EXISTS_ERROR:            "Table group '%s' already exists.",
	ER_TABLE_GROUP_NOT_EXISTED_ERROR:       "Table group '%s' not existed.",
	ER_INDEX_NAME_IDX_PREFIX:               "Index '%s' need '%s' prefix (table '%s').",
	ER_INDEX_NAME_UNIQ_PREFIX:              "Unique index '%s' need '%s' prefix (table '%s').",
	ER_AUTOINC_UNSIGNED:                    "Set unsigned attribute on auto increment column in table '%s'.",
	ER_VARCHAR_TO_TEXT_LEN:                 "Set column '%s' to TEXT type.",
	ER_CHAR_TO_VARCHAR_LEN:                 "Set column '%s' to VARCHAR type.",
	ER_KEY_COLUMN_DOES_NOT_EXITS:           "Key column '%s' doesn't exist in table.",
	ER_INC_INIT_ERR:                        "Set auto-increment initialize value to 1.",
	ER_WRONG_ARGUMENTS:                     "Incorrect arguments to %s.",
	ER_SET_DATA_TYPE_INT_BIGINT:            "Set auto-increment data type to int or bigint.",
	ER_TIMESTAMP_DEFAULT:                   "Set default value for timestamp column '%s'.",
	ER_CHARSET_ON_COLUMN:                   "Not Allowed set charset or collation for column '%s.%s'.",
	ER_AUTO_INCR_ID_WARNING:                "Auto increment column '%s' is meaningful? it's dangerous!",
	ER_ALTER_TABLE_ONCE:                    "Merge the alter statement for table '%s' to ONE.",
	ER_BLOB_CANT_HAVE_DEFAULT:              "BLOB, TEXT, GEOMETRY or JSON column '%s' can't have a default value.",
	ER_END_WITH_SEMICOLON:                  "Add ';' after the last sql statement.",
	ER_NON_UNIQ_ERROR:                      "Column '%s' in field list is ambiguous.",
	ER_TABLE_NOT_EXISTED_ERROR:             "Table '%s' doesn't exist.",
	ER_UNKNOWN_TABLE:                       "Unknown table '%s' in %s.",
	ER_INVALID_GROUP_FUNC_USE:              "Invalid use of group function.",
	ER_INDEX_USE_ALTER_TABLE:               "Create/drop index and rename is not allowed, please replace with alter statement.",
	ER_WITH_DEFAULT_ADD_COLUMN:             "Set Default value for column '%s' in table '%s'",
	ER_TRUNCATED_WRONG_VALUE:               "Truncated incorrect %s value: '%s'",
	ER_TEXT_NOT_NULLABLE_ERROR:             "TEXT/BLOB/JSON Column '%s' in table '%s' can't been not null.",
	ER_WRONG_VALUE_FOR_VAR:                 "Variable '%s' can't be set to the value of '%s'",
	ER_TOO_MUCH_AUTO_TIMESTAMP_COLS:        "Incorrect table definition; there can be only one TIMESTAMP column with CURRENT_TIMESTAMP in DEFAULT or ON UPDATE clause",
	ER_INVALID_ON_UPDATE:                   "Invalid ON UPDATE clause for '%s' column",
	ER_DDL_DML_COEXIST:                     "DDL can not coexist with the DML for table '%s'.",
	ER_SLAVE_CORRUPT_EVENT:                 "Corrupted replication event was detected.",
	ER_COLLATION_CHARSET_MISMATCH:          "COLLATION '%s' is not valid for CHARACTER SET '%s'",
	ER_NOT_SUPPORTED_ALTER_OPTION:          "Not supported statement of alter option",
	ER_CONFLICTING_DECLARATIONS:            "Conflicting declarations: '%s%s' and '%s%s'",
	ER_IDENT_USE_KEYWORD:                   "Identifier '%s' is keyword in MySQL.",
	ER_IDENT_USE_CUSTOM_KEYWORD:            "Identifier '%s' is custom keyword.",
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
	ER_CANT_CHANGE_COLUMN_TYPE:             "Cannot change column type '%s' %s -> %s.",
	ER_CANT_DROP_TABLE:                     "Drop/truncate '%s' is not allowed, please replace with alter rename statement.",
	ER_CANT_DROP_DATABASE:                  "Command is forbidden! Cannot drop database '%s'.",
	ER_WRONG_TABLE_NAME:                    "Incorrect table name '%-.100s'",
	ER_CANT_SET_CHARSET:                    "Cannot set charset '%s'",
	ER_CANT_SET_COLLATION:                  "Cannot set collation '%s'",
	ER_CANT_SET_ENGINE:                     "Cannot set engine '%s'",
	ER_MUST_AT_LEAST_ONE_COLUMN:            "A table must have at least 1 column.",
	ER_MUST_HAVE_COLUMNS:                   "Must have the specified column: '%s'.",
	ErrColumnsMustHaveIndex:                "The specified column: '%s' must have index.",
	ErrColumnsMustHaveIndexTypeErr:         "The specified column: '%s' type must be '%s',current is '%s'.",
	ER_PRIMARY_CANT_HAVE_NULL:              "All parts of a PRIMARY KEY must be NOT NULL; if you need NULL in a key, use UNIQUE instead",
	ErrCantRemoveAllFields:                 "You can't delete all columns with ALTER TABLE; use DROP TABLE instead",
	ErrNotFoundTableInfo:                   "Skip backup because there is no table structure information.",
	ErrMariaDBRollbackWarn:                 "MariaDB v%d not supported yet,please confirm that the rollback sql is correct",
	ErrNotFoundMasterStatus:                "Can't found master binlog position.",
	ErrNonUniqTable:                        "Not unique table/alias: '%-.192s'.", // mysql.MySQLErrName[mysql.ErrNonuniqTable],
	ErrWrongUsage:                          "Incorrect usage of %s and %s.",
	ErrDataTooLong:                         "Data too long for column '%s' at row %d",
	ErrCharsetNotSupport:                   "Set charset to one of '%s'.",
	ErrCollationNotSupport:                 "Set collation to one of '%s'",
	ErrEngineNotSupport:                    "Set engine to one of '%s'",
	ErrJsonTypeSupport:                     "Json type not allowed in column '%s'.",
	ErrMixOfGroupFuncAndFields:             "In aggregated query without GROUP BY, expression #%d of SELECT list contains nonaggregated column '%s'; this is incompatible with sql_mode=only_full_group_by.",
	ErrFieldNotInGroupBy:                   "Expression #%d of %s is not in GROUP BY clause and contains nonaggregated column '%s' which is not functionally dependent on columns in GROUP BY clause; this is incompatible with sql_mode=only_full_group_by.",
	ErCantChangeColumnPosition:             "Cannot change the position of the column '%s'.",
	ErCantChangeColumn:                     "Not supported statement of change column('%s').",
	// ErrMixOfGroupFuncAndFields:             "Mixing of GROUP columns (MIN(),MAX(),COUNT(),...) with no GROUP columns is illegal if there is no GROUP BY clause",
	//ER_NULL_NAME_FOR_INDEX:                 "Index name cannot be null in table '%s'.",
	ER_DATETIME_DEFAULT:                "Set default value for DATETIME column '%s'.",
	ER_TOO_MUCH_AUTO_DATETIME_COLS:     "Incorrect table definition; there can be only one DATETIME column with CURRENT_TIMESTAMP in DEFAULT or ON UPDATE clause",
	ErrFloatDoubleToDecimal:            "Set column '%s' to DECIMAL type.",
	ErrIdentifierUpper:                 "Identifier '%s' must be capitalized.",
	ErrIdentifierLower:                 "Identifier '%s' must be lowercase.",
	ErrWrongAndExpr:                    "May be the wrong syntax! Separate multiple fields with commas.",
	ErrCannotAddForeign:                "Cannot add foreign key constraint",
	ErrWrongFkDefWithMatch:             "Incorrect foreign key definition for '%-.192s': Key reference and table reference don't match",
	ErrFkDupName:                       "Duplicate foreign key constraint name '%s'",
	ErrJoinNoOnCondition:               "set the on clause for join statement.",
	ErrImplicitTypeConversion:          "Implicit type conversion is not allowed(column '%s.%s',type '%s').",
	ErrUseValueExpr:                    "Please confirm if you want to use value expression in where condition.",
	ErrUseIndexVisibility:              "The back-end database does not support the index to specify the visible option.",
	ErrViewSupport:                     "Not allowed to create or use views '%s'.",
	ErrViewColumnCount:                 "View's SELECT and view's field list have different column counts",
	ErrIncorrectDateTimeValue:          "Incorrect datetime value: '%v'(column '%s')",
	ErrSameNamePartition:               "Duplicate partition name %-.192s",
	ErrRepeatConstDefinition:           "Duplicate partition constant definition: '%v'",
	ErrPartitionNotExisted:             "Partition '%-.64s' does not exist",
	ErrIndexNotExisted:                 "Index '%-.64s' does not exist",
	ErrMaxVarcharLength:                "Column length too big for column '%s' (Custom maximum is %d)",
	ErrMaxColumnCount:                  "Table '%s' has too many columns(limit %d,current %d)",
	ER_ERROR_LAST:                      "TheLastError,ByeBye",
	ER_TOOL_BASED_UNIQUE_INDEX_WARNING: "Existing unique indexes may cause duplicate data loss when executing statements using schema-altering tools. It is recommended to review and assess potential risks.",
}

var ErrorsChinese = map[ErrorCode]string{
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
	ER_NO_WHERE_CONDITION:               "请指定where条件.",
	ER_NORMAL_SHUTDOWN:                  "%s: Normal shutdown\n",
	ER_FORCING_CLOSE:                    "%s: Forcing close of thread %ld  user: '%s'\n",
	ER_CON_COUNT_ERROR:                  "Too many connections",
	ER_INVALID_COMMAND:                  "Invalid command.",
	ER_SQL_INVALID_SOURCE:               "不正确的数据源信息(%s).",
	ER_WRONG_DB_NAME:                    "不正确的的数据库名 '%s'.",
	ER_NO_DB_ERROR:                      "没有选择数据库.",
	ER_WITH_LIMIT_CONDITION:             "update/delete语句不允许Limit.",
	ER_WITH_ORDERBY_CONDITION:           "update/delete语句不允许Order by.",
	ER_SELECT_ONLY_STAR:                 "不允许'select *'语法.",
	ER_ORDERY_BY_RAND:                   "不允许'Order by rand'语法.",
	ER_ID_IS_UPER:                       "标识符不允许大写.",
	ErrUnknownCharset:                   "未知的字符集: '%s'.",
	ER_UNKNOWN_COLLATION:                "未知的排序规则: '%s'.",
	ER_INVALID_DATA_TYPE:                "列 '%s' 数据类型(%s)不支持.",
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
	ER_CHANGE_TOO_MUCH_ROWS:             "预计影响行数(%d行)超过 %d 行.",
	ER_WRONG_NAME_FOR_INDEX:             "索引 '%s' 名称不正确(表 '%s').",
	ER_TOO_MANY_KEYS:                    "表 '%s' 指定了太多索引, 最多允许 %d 个.",
	ER_NOT_SUPPORTED_KEY_TYPE:           "不允许的键类型: '%s'.",
	ER_WRONG_SUB_KEY:                    "索引列不能指定长度或指定的长度超出字段长度.",
	// ER_WRONG_KEY_COLUMN:                    "The used storage engine can't index column '%s'.",
	ER_TOO_LONG_KEY:                        "索引 '%s' 过长; 最大长度为 %d 字节.",
	ER_MULTIPLE_PRI_KEY:                    "定义了多个主键.",
	ER_DUP_KEYNAME:                         "索引名 '%s' 重复.",
	ER_TOO_LONG_INDEX_COMMENT:              "索引 '%s' 注释过长(max = %lu).",
	ER_CANT_ADD_PK_OR_UK_COLUMN:            "禁止添加主键或唯一键列 '%s",
	ER_DUP_INDEX:                           "索引 '%s' 定义重复(表'%s.%s').",
	ER_INDEX_COLUMN_REPEAT:                 "索引 '%s' 的字段与索引 '%s.%s' 存在重复字段('%s').",
	ER_TEMP_TABLE_TMP_PREFIX:               "临时表需要指定'tmp'前缀",
	ER_TABLE_PREFIX:                        "表名需要指定'%s'前缀",
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
	ER_CANT_DROP_INDEX_COLUMN:              "无法删除索引列 '%s'.",
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
	ER_ERROR_EXIST_BEFORE:                  "Exist error at before statement.",
	ER_UNKNOWN_SYSTEM_VARIABLE:             "Unknown system variable '%s'.",
	ER_UNKNOWN_CHARACTER_SET:               "Unknown character set: '%s'.",
	ER_END_WITH_COMMIT:                     "Must end with commit.",
	ER_DB_NOT_EXISTED_ERROR:                "选择的数据库 '%s' 不存在.",
	ER_TABLE_EXISTS_ERROR:                  "表 '%s' 已存在.",
	ER_TABLE_GROUP_EXISTS_ERROR:            "表组 '%s' 已存在.",
	ER_TABLE_GROUP_NOT_EXISTED_ERROR:       "表组 '%s' 不存在.",
	ER_INDEX_NAME_IDX_PREFIX:               "索引 '%s' 需要指定'%s'前缀(表'%s').",
	ER_INDEX_NAME_UNIQ_PREFIX:              "唯一索引 '%s' 需要指定'%s'前缀(表'%s').",
	ER_AUTOINC_UNSIGNED:                    "自增列建议设置无符号标志unsigned(表'%s').",
	ER_VARCHAR_TO_TEXT_LEN:                 "列 '%s' 建议设置为text类型.",
	ER_CHAR_TO_VARCHAR_LEN:                 "列 '%s' 建议设置为varchar类型.",
	ER_KEY_COLUMN_DOES_NOT_EXITS:           "列 '%s' 不存在.",
	ER_INC_INIT_ERR:                        "建议自增列初始值置为 1.",
	ER_WRONG_ARGUMENTS:                     "Incorrect arguments to %s.",
	ER_SET_DATA_TYPE_INT_BIGINT:            "自增列需要设置为int或bigint类型.",
	ER_TIMESTAMP_DEFAULT:                   "请设置timestamp列 '%s' 的默认值.",
	ER_CHARSET_ON_COLUMN:                   "表 '%s' 列 '%s' 禁止设置字符集或排序规则!",
	ER_AUTO_INCR_ID_WARNING:                "自增列('%s')建议命名为'ID'.",
	ER_ALTER_TABLE_ONCE:                    "表 '%s' 的多个alter操作请合并成一个.",
	ER_BLOB_CANT_HAVE_DEFAULT:              "BLOB,TEXT,GEOMETRY或JSON列 '%s' 禁止设置默认值.",
	ER_END_WITH_SEMICOLON:                  "Add ';' after the last sql statement.",
	ER_NON_UNIQ_ERROR:                      "列 '%s' 有歧义,请指明表前缀.",
	ER_TABLE_NOT_EXISTED_ERROR:             "表 '%s' 不存在.",
	ER_UNKNOWN_TABLE:                       "Unknown table '%s' in %s.",
	ER_INVALID_GROUP_FUNC_USE:              "Invalid use of group function.",
	ER_INDEX_USE_ALTER_TABLE:               "暂不支持create/drop index和rename语法,请使用alter语句替换.",
	ER_WITH_DEFAULT_ADD_COLUMN:             "列 '%s' 请设置默认值(表'%s')",
	ER_TRUNCATED_WRONG_VALUE:               "Truncated incorrect %s value: '%s'",
	ER_TEXT_NOT_NULLABLE_ERROR:             "TEXT/BLOB/JSON 列 '%s' 禁止设置为not null(表'%s').",
	ER_WRONG_VALUE_FOR_VAR:                 "Variable '%s' can't be set to the value of '%s'",
	ER_TOO_MUCH_AUTO_TIMESTAMP_COLS:        "表定义不正确,只能有一个TIMESTAMP字段在DEFAULT或ON UPDATE指定CURRENT_TIMESTAMP.",
	ER_INVALID_ON_UPDATE:                   "列 %s' ON UPDATE 设置无效",
	ER_DDL_DML_COEXIST:                     "DDL can not coexist with the DML for table '%s'.",
	ER_SLAVE_CORRUPT_EVENT:                 "Corrupted replication event was detected.",
	ER_COLLATION_CHARSET_MISMATCH:          "COLLATION '%s' is not valid for CHARACTER SET '%s'",
	ER_NOT_SUPPORTED_ALTER_OPTION:          "Not supported statement of alter option",
	ER_CONFLICTING_DECLARATIONS:            "Conflicting declarations: '%s%s' and '%s%s'",
	ER_IDENT_USE_KEYWORD:                   "标识符 '%s' 是MySQL关键字.",
	ER_IDENT_USE_CUSTOM_KEYWORD:            "标识符 '%s' 是自定义关键字.",
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
	ER_CANT_CHANGE_COLUMN_TYPE:             "禁止改变列的类型 '%s' %s -> %s.",
	ER_CANT_DROP_TABLE:                     "禁用【DROP】|【TRUNCATE】删除/清空表 '%s', 请改用RENAME重写.",
	ER_CANT_DROP_DATABASE:                  "命令禁止! 无法删除数据库'%s'.",
	ER_WRONG_TABLE_NAME:                    "不正确的表名: '%-.100s'",
	ER_CANT_SET_CHARSET:                    "禁止指定字符集: '%s'",
	ER_CANT_SET_COLLATION:                  "禁止指定排序规则: '%s'",
	ER_CANT_SET_ENGINE:                     "禁止指定存储引擎:'%s'",
	ER_MUST_AT_LEAST_ONE_COLUMN:            "表至少需要有一个列.",
	ER_MUST_HAVE_COLUMNS:                   "表必须包含以下列: '%s'.",
	ErrColumnsMustHaveIndex:                "列: '%s' 必须建索引.",
	ErrColumnsMustHaveIndexTypeErr:         "列: '%s' 类型必须为 '%s',当前为 '%s'",
	ER_PRIMARY_CANT_HAVE_NULL:              "主键的所有列必须为NOT NULL,如需要NULL列,请改用唯一索引",
	ErrCantRemoveAllFields:                 "禁止删除表的所有列.",
	ErrNotFoundTableInfo:                   "没有表结构信息,跳过备份.",
	ErrMariaDBRollbackWarn:                 "MariaDB v%d 对回滚支持不完美,请注意确认回滚语句是否正确",
	ErrNotFoundMasterStatus:                "无法获取master binlog信息.",
	ErrNonUniqTable:                        "表名或别名: '%-.192s' 不唯一.",
	ErrDataTooLong:                         "数据过长!(列 '%s',行 '%d')",
	ErrCharsetNotSupport:                   "允许的字符集: '%s'.",
	ErrCollationNotSupport:                 "允许的排序规则: '%s'.",
	ErrEngineNotSupport:                    "允许的存储引擎: '%s'.",
	ErrWrongUsage:                          "%s子句无法使用%s",
	ErrJsonTypeSupport:                     "不允许使用json类型(列'%s').",
	ErCantChangeColumnPosition:             "不允许改变列顺序(列'%s').",
	ErCantChangeColumn:                     "不允许change column语法(列'%s').",
	ER_DATETIME_DEFAULT:                    "请设置 datetime 列 '%s' 的默认值.",
	ER_TOO_MUCH_AUTO_DATETIME_COLS:         "表定义不正确,只能有一个 datetime 字段,在 DEFAULT 或 ON UPDATE指定CURRENT_TIMESTAMP.",
	ErrFloatDoubleToDecimal:                "列 '%s' 建议设置为 decimal 类型.",
	ErrIdentifierUpper:                     "标识符 '%s' 必须大写.",
	ErrIdentifierLower:                     "标识符 '%s' 必须小写.",
	ErrWrongAndExpr:                        "可能是错误语法!更新多个字段时请使用逗号分隔.",
	ErrJoinNoOnCondition:                   "join语句请指定on子句.",
	ErrImplicitTypeConversion:              "不允许隐式类型转换(列'%s.%s',类型'%s').",
	ErrUseValueExpr:                        "请确认是否要在where条件中使用值表达式.",
	ErrUseIndexVisibility:                  "后端数据库不支持索引的visible选项",
	ErrViewSupport:                         "不允许创建或使用视图 '%s'.",
	ErrViewColumnCount:                     "视图的SELECT和视图字段列表具有不同的列数",
	ErrIncorrectDateTimeValue:              "不正确的时间:'%v'(列 '%s')",
	ErrSameNamePartition:                   "分区名重复: %-.192s",
	ErrRepeatConstDefinition:               "重复的分区范围定义: '%v'",
	ErrPartitionNotExisted:                 "分区 '%-.64s' 不存在",
	ErrIndexNotExisted:                     "Index '%-.64s' 不存在",
	ErrMaxVarcharLength:                    "列'%s'指定长度过长(自定义上限为%d)",
	ErrMaxColumnCount:                      "表'%s'列数过多(上限:%d,当前:%d)",
	ER_TOOL_BASED_UNIQUE_INDEX_WARNING:     "存在唯一索引，使用改表工具执行语句可能导致重复数据丢失，建议复查是否存在风险",
}

func GetErrorLevel(code ErrorCode) uint8 {

	switch code {
	case ER_ALTER_TABLE_ONCE,
		ER_AUTO_INCR_ID_WARNING,
		ER_AUTOINC_UNSIGNED,
		ER_BLOB_CANT_HAVE_DEFAULT,
		ER_CANT_SET_CHARSET,
		ER_CANT_SET_COLLATION,
		ER_CANT_SET_ENGINE,
		ER_CHANGE_COLUMN_TYPE,
		ER_CHAR_TO_VARCHAR_LEN,
		ER_CHARSET_ON_COLUMN,
		ER_COLUMN_HAVE_NO_COMMENT,
		ER_IDENT_USE_KEYWORD,
		ER_IDENT_USE_CUSTOM_KEYWORD,
		ER_INC_INIT_ERR,
		ER_INDEX_NAME_IDX_PREFIX,
		ER_INDEX_NAME_UNIQ_PREFIX,
		ER_TEMP_TABLE_TMP_PREFIX,
		ER_TABLE_PREFIX,
		ER_INSERT_TOO_MUCH_ROWS,
		ER_INVALID_DATA_TYPE,
		ER_INVALID_IDENT,
		ER_MUST_HAVE_COLUMNS,
		ErrColumnsMustHaveIndex,
		ErrColumnsMustHaveIndexTypeErr,
		ER_NO_WHERE_CONDITION,
		ErrJoinNoOnCondition,
		ER_NOT_ALLOWED_NULLABLE,
		ER_NOT_SUPPORTED_ALTER_OPTION,
		ER_ORDERY_BY_RAND,
		ER_OUTOFMEMORY,
		ER_PARTITION_NOT_ALLOWED,
		ER_PK_COLS_NOT_INT,
		ER_PK_TOO_MANY_PARTS,
		ER_SELECT_ONLY_STAR,
		ER_TABLE_CHARSET_MUST_NULL,
		ER_TABLE_CHARSET_MUST_UTF8,
		ER_TABLE_MUST_HAVE_COMMENT,
		ER_TABLE_MUST_HAVE_PK,
		ER_TEXT_NOT_NULLABLE_ERROR,
		ER_TIMESTAMP_DEFAULT,
		ER_TOO_LONG_INDEX_COMMENT,
		ER_TOO_MANY_KEY_PARTS,
		ER_TOO_MANY_KEYS,
		ER_UDPATE_TOO_MUCH_ROWS,
		ER_CHANGE_TOO_MUCH_ROWS,
		ErrUnknownCharset,
		ER_UNKNOWN_COLLATION,
		ER_USE_ENUM,
		ER_WITH_DEFAULT_ADD_COLUMN,
		ER_WITH_LIMIT_CONDITION,
		ER_WITH_ORDERBY_CONDITION,
		ErCantChangeColumnPosition,
		ErCantChangeColumn,
		ErrNotFoundTableInfo,
		ErrMariaDBRollbackWarn,
		ErrTableCollationNotSupport,
		ER_DATETIME_DEFAULT,
		ErrWrongAndExpr,
		ErrImplicitTypeConversion,
		ErrUseValueExpr,
		ErrMaxColumnCount,
		ER_WITH_INSERT_FIELD,
		ER_TOOL_BASED_UNIQUE_INDEX_WARNING:
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
		ER_INDEX_COLUMN_REPEAT,
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
		ErrMaxVarcharLength,
		ER_FOREIGN_KEY,
		ER_TOO_MUCH_AUTO_DATETIME_COLS,
		ER_INCEPTION_EMPTY_QUERY:
		return 2

	default:
		return 2
	}
}

// GetErrorMessage 获取审核信息,默认为英文
func GetErrorMessage(code ErrorCode, lang string) string {
	if lang == "zh_cn" || lang == "zh-cn" {
		if v, ok := ErrorsChinese[code]; ok {
			return v
		}
	}
	if v, ok := ErrorsDefault[code]; ok {
		return v
	}
	return "Invalid error code!"
}

// SQLError records an error information, from executing SQL.
type SQLError struct {
	Code    ErrorCode
	Level   uint8
	Message string
}

// Error prints errors, with a formatted string.
func (e *SQLError) Error() string {
	return e.Message
}

// NewErr generates a SQL error, with an error code and default format specifier defined in MySQLErrName.
func NewErr(errCode ErrorCode, args ...interface{}) *SQLError {
	e := &SQLError{Code: errCode}
	e.Message = fmt.Sprintf(GetErrorMessage(errCode, "en_us"), args...)
	return e
}

// custom error level. used for inc_level function
func (e *SQLError) SetLevel(l uint8) *SQLError {
	e.Level = l
	return e
}

// NewErrf creates a SQL error, with an error code and a format specifier.
func NewErrf(format string, args ...interface{}) *SQLError {
	e := &SQLError{Code: 0}
	e.Message = fmt.Sprintf(format, args...)
	return e
}

func (e ErrorCode) String() string {
	switch e {
	case ER_ERROR_FIRST:
		return "er_error_first"
	case ER_NOT_SUPPORTED_YET:
		return "er_not_supported_yet"
	case ER_SQL_NO_SOURCE:
		return "er_sql_no_source"
	case ER_SQL_NO_OP_TYPE:
		return "er_sql_no_op_type"
	case ER_SQL_INVALID_OP_TYPE:
		return "er_sql_invalid_op_type"
	case ER_PARSE_ERROR:
		return "er_parse_error"
	case ER_SYNTAX_ERROR:
		return "er_syntax_error"
	case ER_REMOTE_EXE_ERROR:
		return "er_remote_exe_error"
	case ER_SHUTDOWN_COMPLETE:
		return "er_shutdown_complete"
	case ER_WITH_INSERT_FIELD:
		return "er_with_insert_field"
	case ER_WITH_INSERT_VALUES:
		return "er_with_insert_values"
	case ER_WRONG_VALUE_COUNT_ON_ROW:
		return "er_wrong_value_count_on_row"
	case ER_BAD_FIELD_ERROR:
		return "er_bad_field_error"
	case ER_FIELD_SPECIFIED_TWICE:
		return "er_field_specified_twice"
	case ER_BAD_NULL_ERROR:
		return "er_bad_null_error"
	case ER_NO_WHERE_CONDITION:
		return "er_no_where_condition"
	case ER_NORMAL_SHUTDOWN:
		return "er_normal_shutdown"
	case ER_FORCING_CLOSE:
		return "er_forcing_close"
	case ER_CON_COUNT_ERROR:
		return "er_con_count_error"
	case ER_INVALID_COMMAND:
		return "er_invalid_command"
	case ER_SQL_INVALID_SOURCE:
		return "er_sql_invalid_source"
	case ER_WRONG_DB_NAME:
		return "er_wrong_db_name"
	case ER_NO_DB_ERROR:
		return "er_no_db_error"
	case ER_WITH_LIMIT_CONDITION:
		return "er_with_limit_condition"
	case ER_WITH_ORDERBY_CONDITION:
		return "er_with_orderby_condition"
	case ER_SELECT_ONLY_STAR:
		return "er_select_only_star"
	case ER_ORDERY_BY_RAND:
		return "er_ordery_by_rand"
	case ER_ID_IS_UPER:
		return "er_id_is_uper"
	case ErrUnknownCharset:
		return "er_unknown_charset"
	case ER_UNKNOWN_COLLATION:
		return "er_unknown_collation"
	case ER_INVALID_DATA_TYPE:
		return "er_invalid_data_type"
	case ER_NOT_ALLOWED_NULLABLE:
		return "er_not_allowed_nullable"
	case ER_DUP_FIELDNAME:
		return "er_dup_fieldname"
	case ER_WRONG_COLUMN_NAME:
		return "er_wrong_column_name"
	case ER_WRONG_AUTO_KEY:
		return "er_wrong_auto_key"
	case ER_TABLE_CANT_HANDLE_AUTO_INCREMENT:
		return "er_table_cant_handle_auto_increment"
	case ER_FOREIGN_KEY:
		return "er_foreign_key"
	case ER_TOO_MANY_KEY_PARTS:
		return "er_too_many_key_parts"
	case ER_TOO_LONG_IDENT:
		return "er_too_long_ident"
	case ER_UDPATE_TOO_MUCH_ROWS:
		return "er_udpate_too_much_rows"
	case ER_CHANGE_TOO_MUCH_ROWS:
		return "er_change_too_much_rows"
	case ER_INSERT_TOO_MUCH_ROWS:
		return "er_insert_too_much_rows"
	case ER_WRONG_NAME_FOR_INDEX:
		return "er_wrong_name_for_index"
	case ER_TOO_MANY_KEYS:
		return "er_too_many_keys"
	case ER_NOT_SUPPORTED_KEY_TYPE:
		return "er_not_supported_key_type"
	case ER_WRONG_SUB_KEY:
		return "er_wrong_sub_key"
	case ER_WRONG_KEY_COLUMN:
		return "er_wrong_key_column"
	case ER_TOO_LONG_KEY:
		return "er_too_long_key"
	case ER_MULTIPLE_PRI_KEY:
		return "er_multiple_pri_key"
	case ER_DUP_KEYNAME:
		return "er_dup_keyname"
	case ER_TOO_LONG_INDEX_COMMENT:
		return "er_too_long_index_comment"
	case ER_DUP_INDEX:
		return "er_dup_index"
	case ER_INDEX_COLUMN_REPEAT:
		return "er_index_column_repeat"
	case ER_TEMP_TABLE_TMP_PREFIX:
		return "er_temp_table_tmp_prefix"
	case ER_TABLE_PREFIX:
		return "er_table_prefix"
	case ER_TABLE_CHARSET_MUST_UTF8:
		return "er_table_charset_must_utf8"
	case ER_TABLE_CHARSET_MUST_NULL:
		return "er_table_charset_must_null"
	case ER_TABLE_MUST_HAVE_COMMENT:
		return "er_table_must_have_comment"
	case ER_COLUMN_HAVE_NO_COMMENT:
		return "er_column_have_no_comment"
	case ER_TABLE_MUST_HAVE_PK:
		return "er_table_must_have_pk"
	case ER_PARTITION_NOT_ALLOWED:
		return "er_partition_not_allowed"
	case ER_USE_ENUM:
		return "er_use_enum"
	case ER_USE_TEXT_OR_BLOB:
		return "er_use_text_or_blob"
	case ER_COLUMN_EXISTED:
		return "er_column_existed"
	case ER_COLUMN_NOT_EXISTED:
		return "er_column_not_existed"
	case ER_CANT_DROP_FIELD_OR_KEY:
		return "er_cant_drop_field_or_key"
	case ER_INVALID_DEFAULT:
		return "er_invalid_default"
	case ER_USERNAME:
		return "er_username"
	case ER_HOSTNAME:
		return "er_hostname"
	case ER_NOT_VALID_PASSWORD:
		return "er_not_valid_password"
	case ER_WRONG_STRING_LENGTH:
		return "er_wrong_string_length"
	case ER_BLOB_USED_AS_KEY:
		return "er_blob_used_as_key"
	case ER_TOO_LONG_BAKDB_NAME:
		return "er_too_long_bakdb_name"
	case ER_INVALID_BACKUP_HOST_INFO:
		return "er_invalid_backup_host_info"
	case ER_BINLOG_CORRUPTED:
		return "er_binlog_corrupted"
	case ER_NET_READ_ERROR:
		return "er_net_read_error"
	case ER_NETWORK_READ_EVENT_CHECKSUM_FAILURE:
		return "er_network_read_event_checksum_failure"
	case ER_SLAVE_RELAY_LOG_WRITE_FAILURE:
		return "er_slave_relay_log_write_failure"
	case ER_INCORRECT_GLOBAL_LOCAL_VAR:
		return "er_incorrect_global_local_var"
	case ER_START_AS_BEGIN:
		return "er_start_as_begin"
	case ER_OUTOFMEMORY:
		return "er_outofmemory"
	case ER_HAVE_BEGIN:
		return "er_have_begin"
	case ER_NET_READ_INTERRUPTED:
		return "er_net_read_interrupted"
	case ER_BINLOG_FORMAT_STATEMENT:
		return "er_binlog_format_statement"
	case ER_ERROR_EXIST_BEFORE:
		return "er_error_exist_before"
	case ER_UNKNOWN_SYSTEM_VARIABLE:
		return "er_unknown_system_variable"
	case ER_UNKNOWN_CHARACTER_SET:
		return "er_unknown_character_set"
	case ER_END_WITH_COMMIT:
		return "er_end_with_commit"
	case ER_DB_NOT_EXISTED_ERROR:
		return "er_db_not_existed_error"
	case ER_TABLE_EXISTS_ERROR:
		return "er_table_exists_error"
	case ER_TABLE_GROUP_EXISTS_ERROR:
		return "er_table_group_exists_error"
	case ER_TABLE_GROUP_NOT_EXISTED_ERROR:
		return "er_table_group_not_existed_error"
	case ER_INDEX_NAME_IDX_PREFIX:
		return "er_index_name_idx_prefix"
	case ER_INDEX_NAME_UNIQ_PREFIX:
		return "er_index_name_uniq_prefix"
	case ER_AUTOINC_UNSIGNED:
		return "er_autoinc_unsigned"
	case ER_VARCHAR_TO_TEXT_LEN:
		return "er_varchar_to_text_len"
	case ER_CHAR_TO_VARCHAR_LEN:
		return "er_char_to_varchar_len"
	case ER_KEY_COLUMN_DOES_NOT_EXITS:
		return "er_key_column_does_not_exits"
	case ER_INC_INIT_ERR:
		return "er_inc_init_err"
	case ER_WRONG_ARGUMENTS:
		return "er_wrong_arguments"
	case ER_SET_DATA_TYPE_INT_BIGINT:
		return "er_set_data_type_int_bigint"
	case ER_TIMESTAMP_DEFAULT:
		return "er_timestamp_default"
	case ER_CHARSET_ON_COLUMN:
		return "er_charset_on_column"
	case ER_AUTO_INCR_ID_WARNING:
		return "er_auto_incr_id_warning"
	case ER_ALTER_TABLE_ONCE:
		return "er_alter_table_once"
	case ER_BLOB_CANT_HAVE_DEFAULT:
		return "er_blob_cant_have_default"
	case ER_END_WITH_SEMICOLON:
		return "er_end_with_semicolon"
	case ER_NON_UNIQ_ERROR:
		return "er_non_uniq_error"
	case ER_TABLE_NOT_EXISTED_ERROR:
		return "er_table_not_existed_error"
	case ER_UNKNOWN_TABLE:
		return "er_unknown_table"
	case ER_INVALID_GROUP_FUNC_USE:
		return "er_invalid_group_func_use"
	case ER_INDEX_USE_ALTER_TABLE:
		return "er_index_use_alter_table"
	case ER_WITH_DEFAULT_ADD_COLUMN:
		return "er_with_default_add_column"
	case ER_TRUNCATED_WRONG_VALUE:
		return "er_truncated_wrong_value"
	case ER_TEXT_NOT_NULLABLE_ERROR:
		return "er_text_not_nullable_error"
	case ER_WRONG_VALUE_FOR_VAR:
		return "er_wrong_value_for_var"
	case ER_TOO_MUCH_AUTO_TIMESTAMP_COLS:
		return "er_too_much_auto_timestamp_cols"
	case ER_INVALID_ON_UPDATE:
		return "er_invalid_on_update"
	case ER_DDL_DML_COEXIST:
		return "er_ddl_dml_coexist"
	case ER_SLAVE_CORRUPT_EVENT:
		return "er_slave_corrupt_event"
	case ER_COLLATION_CHARSET_MISMATCH:
		return "er_collation_charset_mismatch"
	case ER_NOT_SUPPORTED_ALTER_OPTION:
		return "er_not_supported_alter_option"
	case ER_CONFLICTING_DECLARATIONS:
		return "er_conflicting_declarations"
	case ER_IDENT_USE_KEYWORD, ER_IDENT_USE_CUSTOM_KEYWORD:
		return "er_ident_use_keyword"
	case ER_VIEW_SELECT_CLAUSE:
		return "er_view_select_clause"
	case ER_OSC_KILL_FAILED:
		return "er_osc_kill_failed"
	case ER_NET_PACKETS_OUT_OF_ORDER:
		return "er_net_packets_out_of_order"
	case ER_NOT_SUPPORTED_ITEM_TYPE:
		return "er_not_supported_item_type"
	case ER_INVALID_IDENT:
		return "er_invalid_ident"
	case ER_INCEPTION_EMPTY_QUERY:
		return "er_inception_empty_query"
	case ER_PK_COLS_NOT_INT:
		return "er_pk_cols_not_int"
	case ER_PK_TOO_MANY_PARTS:
		return "er_pk_too_many_parts"
	case ER_REMOVED_SPACES:
		return "er_removed_spaces"
	case ER_CHANGE_COLUMN_TYPE:
		return "er_change_column_type"
	case ER_CANT_DROP_TABLE:
		return "er_cant_drop_table"
	case ER_CANT_DROP_DATABASE:
		return "er_cant_drop_database"
	case ER_WRONG_TABLE_NAME:
		return "er_wrong_table_name"
	case ER_CANT_SET_CHARSET:
		return "er_cant_set_charset"
	case ER_CANT_SET_COLLATION:
		return "er_cant_set_collation"
	case ER_CANT_SET_ENGINE:
		return "er_cant_set_engine"
	case ER_MUST_AT_LEAST_ONE_COLUMN:
		return "er_must_at_least_one_column"
	case ER_MUST_HAVE_COLUMNS:
		return "er_must_have_columns"
	case ErrColumnsMustHaveIndex:
		return "er_columns_must_have_index"
	case ErrColumnsMustHaveIndexTypeErr:
		return "er_columns_must_have_index_type_err"
	case ER_PRIMARY_CANT_HAVE_NULL:
		return "er_primary_cant_have_null"
	case ErrCantRemoveAllFields:
		return "er_cant_remove_all_fields"
	case ErrNotFoundTableInfo:
		return "er_not_found_table_info"
	case ErrMariaDBRollbackWarn:
		return "er_mariadb_rollback_warn"
	case ErrNotFoundMasterStatus:
		return "er_not_found_master_status"
	case ErrNonUniqTable:
		return "er_non_uniq_table"
	case ErrWrongUsage:
		return "er_wrong_usage"
	case ErrDataTooLong:
		return "er_data_too_long"
	case ErrCharsetNotSupport:
		return "er_charset_not_support"
	case ErrCollationNotSupport:
		return "er_collation_not_support"
	case ErrTableCollationNotSupport:
		return "er_table_collation_not_support"
	case ErrJsonTypeSupport:
		return "er_json_type_support"
	case ErrEngineNotSupport:
		return "er_engine_not_support"
	case ErrMixOfGroupFuncAndFields:
		return "er_mix_of_group_func_and_fields"
	case ErrFieldNotInGroupBy:
		return "er_field_not_in_group_by"
	case ErCantChangeColumnPosition:
		return "er_cant_change_column_position"
	case ErCantChangeColumn:
		return "er_cant_change_column"
	case ErrFloatDoubleToDecimal:
		return "er_float_double_to_decimal"
	case ErrIdentifierUpper:
		return "er_identifier_upper"
	case ErrIdentifierLower:
		return "er_identifier_lower"
	case ErrWrongAndExpr:
		return "er_wrong_and_expr"
	case ErrJoinNoOnCondition:
		return "er_join_no_on_condition"
	case ErrImplicitTypeConversion:
		return "er_implicit_type_conversion"
	case ErrUseValueExpr:
		return "er_use_value_expr"
	case ErrUseIndexVisibility:
		return "er_use_index_visibility"
	case ErrViewSupport:
		return "er_view_support"
	case ErrIncorrectDateTimeValue:
		return "er_incorrect_datetime_value"
	case ErrSameNamePartition:
		return "er_same_name_partition"
	case ErrRepeatConstDefinition:
		return "er_repeat_const_definition"
	case ErrPartitionNotExisted:
		return "er_partition_not_existed"
	case ErrIndexNotExisted:
		return "er_index_not_existed"
	case ErrMaxVarcharLength:
		return "er_max_varchar_length"
	case ErrMaxColumnCount:
		return "er_max_column_count"
	case ER_ERROR_LAST:
		return "er_error_last"
	case ER_TOOL_BASED_UNIQUE_INDEX_WARNING:
		return "er_tool_based_unique_index_warning"

	}
	return ""
}

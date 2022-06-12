package config

type IncLevel struct {
	ER_ALTER_TABLE_ONCE       int8 `toml:"er_alter_table_once"`
	ER_AUTO_INCR_ID_WARNING   int8 `toml:"er_auto_incr_id_warning"`
	ER_AUTOINC_UNSIGNED       int8 `toml:"er_autoinc_unsigned"`
	ER_BLOB_CANT_HAVE_DEFAULT int8 `toml:"er_blob_cant_have_default"`
	ER_CANT_SET_CHARSET       int8 `toml:"er_cant_set_charset"`
	ER_CANT_SET_COLLATION     int8 `toml:"er_cant_set_collation"`
	ER_CANT_SET_ENGINE        int8 `toml:"er_cant_set_engine"`
	ER_CHANGE_COLUMN_TYPE     int8 `toml:"er_change_column_type"`
	ER_CHANGE_TOO_MUCH_ROWS   int8 `toml:"er_change_too_much_rows"`
	ER_CHAR_TO_VARCHAR_LEN    int8 `toml:"er_char_to_varchar_len"`
	ER_CHARSET_ON_COLUMN      int8 `toml:"er_charset_on_column"`
	ER_COLUMN_HAVE_NO_COMMENT int8 `toml:"er_column_have_no_comment"`
	ER_DATETIME_DEFAULT       int8 `toml:"er_datetime_default"`
	ErrFloatDoubleToDecimal   int8 `toml:"er_float_double_to_decimal"`
	ER_FOREIGN_KEY            int8 `toml:"er_foreign_key"`
	ER_IDENT_USE_KEYWORD      int8 `toml:"er_ident_use_keyword"`
	ER_INC_INIT_ERR           int8 `toml:"er_inc_init_err"`

	ER_INDEX_NAME_IDX_PREFIX        int8 `toml:"er_index_name_idx_prefix"`
	ER_INDEX_NAME_UNIQ_PREFIX       int8 `toml:"er_index_name_uniq_prefix"`
	ER_INSERT_TOO_MUCH_ROWS         int8 `toml:"er_insert_too_much_rows"`
	ER_INVALID_DATA_TYPE            int8 `toml:"er_invalid_data_type"`
	ER_INVALID_IDENT                int8 `toml:"er_invalid_ident"`
	ErrMariaDBRollbackWarn          int8 `toml:"er_mariadb_rollback_warn"`
	ER_MUST_HAVE_COLUMNS            int8 `toml:"er_must_have_columns"`
	ErrColumnsMustHaveIndex         int8 `toml:"er_columns_must_have_index"`
	ErrColumnsMustHaveIndexTypeErr  int8 `toml:"er_columns_must_have_index_type_err"`
	ER_NO_WHERE_CONDITION           int8 `toml:"er_no_where_condition"`
	ER_NOT_ALLOWED_NULLABLE         int8 `toml:"er_not_allowed_nullable"`
	ER_ORDERY_BY_RAND               int8 `toml:"er_ordery_by_rand"`
	ER_PARTITION_NOT_ALLOWED        int8 `toml:"er_partition_not_allowed"`
	ER_PK_COLS_NOT_INT              int8 `toml:"er_pk_cols_not_int"`
	ER_PK_TOO_MANY_PARTS            int8 `toml:"er_pk_too_many_parts"`
	ER_SELECT_ONLY_STAR             int8 `toml:"er_select_only_star"`
	ER_SET_DATA_TYPE_INT_BIGINT     int8 `toml:"er_set_data_type_int_bigint"`
	ER_TABLE_CHARSET_MUST_NULL      int8 `toml:"er_table_charset_must_null"`
	ER_TABLE_CHARSET_MUST_UTF8      int8 `toml:"er_table_charset_must_utf8"`
	ER_TABLE_MUST_HAVE_COMMENT      int8 `toml:"er_table_must_have_comment"`
	ER_TABLE_MUST_HAVE_PK           int8 `toml:"er_table_must_have_pk"`
	ER_TABLE_PREFIX                 int8 `toml:"er_table_prefix"`
	ER_TEXT_NOT_NULLABLE_ERROR      int8 `toml:"er_text_not_nullable_error"`
	ER_TIMESTAMP_DEFAULT            int8 `toml:"er_timestamp_default"`
	ER_TOO_MANY_KEY_PARTS           int8 `toml:"er_too_many_key_parts"`
	ER_TOO_MANY_KEYS                int8 `toml:"er_too_many_keys"`
	ER_TOO_MUCH_AUTO_DATETIME_COLS  int8 `toml:"er_too_much_auto_datetime_cols"`
	ER_TOO_MUCH_AUTO_TIMESTAMP_COLS int8 `toml:"er_too_much_auto_timestamp_cols"`
	ER_UDPATE_TOO_MUCH_ROWS         int8 `toml:"er_udpate_too_much_rows"`
	ER_USE_ENUM                     int8 `toml:"er_use_enum"`
	ER_USE_TEXT_OR_BLOB             int8 `toml:"er_use_text_or_blob"`
	ER_WITH_DEFAULT_ADD_COLUMN      int8 `toml:"er_with_default_add_column"`
	ER_WITH_INSERT_FIELD            int8 `toml:"er_with_insert_field"`
	ER_WITH_LIMIT_CONDITION         int8 `toml:"er_with_limit_condition"`
	ER_WITH_ORDERBY_CONDITION       int8 `toml:"er_with_orderby_condition"`
	ErCantChangeColumn              int8 `toml:"er_cant_change_column"`
	ErCantChangeColumnPosition      int8 `toml:"er_cant_change_column_position"`
	ErJsonTypeSupport               int8 `toml:"er_json_type_support"`
	ErrImplicitTypeConversion       int8 `toml:"er_implicit_type_conversion"`
	ErrJoinNoOnCondition            int8 `toml:"er_join_no_on_condition"`
	ErrUseValueExpr                 int8 `toml:"er_use_value_expr"`
	ErrWrongAndExpr                 int8 `toml:"er_wrong_and_expr"`
	ErrViewSupport                  int8 `toml:"er_view_support"`
	ErrIncorrectDateTimeValue       int8 `toml:"er_incorrect_datetime_value"`
	ErrMaxVarcharLength             int8 `toml:"er_max_varchar_length"`
	ErrMaxColumnCount               int8 `toml:"er_max_column_count"`
}

var DefaultLevel = IncLevel2{
	ErAlterTableOnce:                  1,
	ErAutoIncrIdWarning:               1,
	ErAutoincUnsigned:                 1,
	ErBadFieldError:                   2,
	ErBadNullError:                    2,
	ErBinlogCorrupted:                 2,
	ErBinlogFormatStatement:           2,
	ErBlobCantHaveDefault:             1,
	ErBlobUsedAsKey:                   2,
	ErCantChangeColumn:                1,
	ErCantChangeColumnPosition:        1,
	ErCantDropDatabase:                2,
	ErCantDropFieldOrKey:              2,
	ErCantDropTable:                   2,
	ErCantRemoveAllFields:             2,
	ErCantSetCharset:                  1,
	ErCantSetCollation:                1,
	ErCantSetEngine:                   1,
	ErChangeColumnType:                1,
	ErChangeTooMuchRows:               1,
	ErCharToVarcharLen:                1,
	ErCharsetNotSupport:               2,
	ErCharsetOnColumn:                 1,
	ErCollationCharsetMismatch:        2,
	ErCollationNotSupport:             2,
	ErColumnExisted:                   2,
	ErColumnHaveNoComment:             1,
	ErColumnNotExisted:                2,
	ErColumnsMustHaveIndex:            1,
	ErColumnsMustHaveIndexTypeErr:     1,
	ErConCountError:                   2,
	ErConflictingDeclarations:         2,
	ErDataTooLong:                     2,
	ErDbNotExistedError:               2,
	ErDdlDmlCoexist:                   2,
	ErDupFieldname:                    2,
	ErDupIndex:                        2,
	ErDupKeyname:                      2,
	ErEndWithCommit:                   2,
	ErEndWithSemicolon:                2,
	ErEngineNotSupport:                2,
	ErErrorExistBefore:                2,
	ErFieldNotInGroupBy:               2,
	ErFieldSpecifiedTwice:             2,
	ErFloatDoubleToDecimal:            2,
	ErForcingClose:                    2,
	ErForeignKey:                      2,
	ErHaveBegin:                       2,
	ErHostname:                        2,
	ErIdIsUper:                        2,
	ErIdentUseKeyword:                 1,
	ErIdentifierLower:                 2,
	ErIdentifierUpper:                 2,
	ErImplicitTypeConversion:          1,
	ErIncInitErr:                      1,
	ErInceptionEmptyQuery:             2,
	ErIncorrectDatetimeValue:          2,
	ErIncorrectGlobalLocalVar:         2,
	ErIndexNameIdxPrefix:              1,
	ErIndexNameUniqPrefix:             1,
	ErIndexUseAlterTable:              2,
	ErInsertTooMuchRows:               1,
	ErInvalidBackupHostInfo:           2,
	ErInvalidCommand:                  2,
	ErInvalidDataType:                 1,
	ErInvalidDefault:                  2,
	ErInvalidGroupFuncUse:             2,
	ErInvalidIdent:                    1,
	ErInvalidOnUpdate:                 2,
	ErJoinNoOnCondition:               1,
	ErJsonTypeSupport:                 2,
	ErKeyColumnDoesNotExits:           2,
	ErMariadbRollbackWarn:             1,
	ErMaxColumnCount:                  1,
	ErMaxVarcharLength:                2,
	ErMixOfGroupFuncAndFields:         2,
	ErMultiplePriKey:                  2,
	ErMustAtLeastOneColumn:            2,
	ErMustHaveColumns:                 1,
	ErNetPacketsOutOfOrder:            2,
	ErNetReadError:                    2,
	ErNetReadInterrupted:              2,
	ErNetworkReadEventChecksumFailure: 2,
	ErNoDbError:                       2,
	ErNoWhereCondition:                1,
	ErNonUniqError:                    2,
	ErNonUniqTable:                    2,
	ErNormalShutdown:                  2,
	ErNotAllowedNullable:              1,
	ErNotFoundMasterStatus:            2,
	ErNotFoundTableInfo:               1,
	ErNotSupportedAlterOption:         1,
	ErNotSupportedItemType:            2,
	ErNotSupportedKeyType:             2,
	ErNotSupportedYet:                 2,
	ErNotValidPassword:                2,
	ErOrderyByRand:                    1,
	ErOscKillFailed:                   2,
	ErOutofmemory:                     1,
	ErParseError:                      2,
	ErPartitionNotAllowed:             1,
	ErPartitionNotExisted:             2,
	ErPkColsNotInt:                    1,
	ErPkTooManyParts:                  1,
	ErPrimaryCantHaveNull:             2,
	ErRemoteExeError:                  2,
	ErRemovedSpaces:                   2,
	ErRepeatConstDefinition:           2,
	ErSameNamePartition:               2,
	ErSelectOnlyStar:                  1,
	ErSetDataTypeIntBigint:            2,
	ErShutdownComplete:                2,
	ErSlaveCorruptEvent:               2,
	ErSlaveRelayLogWriteFailure:       2,
	ErSqlInvalidOpType:                2,
	ErSqlInvalidSource:                2,
	ErSqlNoOpType:                     2,
	ErSqlNoSource:                     2,
	ErStartAsBegin:                    2,
	ErSyntaxError:                     2,
	ErTableCantHandleAutoIncrement:    2,
	ErTableCharsetMustNull:            1,
	ErTableCharsetMustUtf8:            1,
	ErTableCollationNotSupport:        1,
	ErTableExistsError:                2,
	ErTableMustHaveComment:            1,
	ErTableMustHavePk:                 1,
	ErTableNotExistedError:            2,
	ErTablePrefix:                     1,
	ErTempTableTmpPrefix:              1,
	ErTextNotNullableError:            1,
	ErTimestampDefault:                1,
	ErTooLongBakdbName:                2,
	ErTooLongIdent:                    2,
	ErTooLongIndexComment:             1,
	ErTooLongKey:                      2,
	ErTooManyKeyParts:                 1,
	ErTooManyKeys:                     1,
	ErTooMuchAutoTimestampCols:        2,
	ErTruncatedWrongValue:             2,
	ErUdpateTooMuchRows:               1,
	ErUnknownCharacterSet:             2,
	ErUnknownCharset:                  1,
	ErUnknownCollation:                1,
	ErUnknownSystemVariable:           2,
	ErUnknownTable:                    2,
	ErUseEnum:                         1,
	ErUseIndexVisibility:              2,
	ErUseTextOrBlob:                   2,
	ErUseValueExpr:                    1,
	ErUsername:                        2,
	ErVarcharToTextLen:                2,
	ErViewSelectClause:                2,
	ErViewSupport:                     2,
	ErWithDefaultAddColumn:            1,
	ErWithInsertField:                 1,
	ErWithInsertValues:                2,
	ErWithLimitCondition:              1,
	ErWithOrderbyCondition:            1,
	ErWrongAndExpr:                    1,
	ErWrongArguments:                  2,
	ErWrongAutoKey:                    2,
	ErWrongColumnName:                 2,
	ErWrongDbName:                     2,
	ErWrongKeyColumn:                  2,
	ErWrongNameForIndex:               2,
	ErWrongStringLength:               2,
	ErWrongSubKey:                     2,
	ErWrongTableName:                  2,
	ErWrongUsage:                      2,
	ErWrongValueCountOnRow:            2,
	ErWrongValueForVar:                2,
}

var defaultLevel = IncLevel{
	ER_ALTER_TABLE_ONCE:             1,
	ER_AUTO_INCR_ID_WARNING:         1,
	ER_AUTOINC_UNSIGNED:             1,
	ER_BLOB_CANT_HAVE_DEFAULT:       1,
	ER_CANT_SET_CHARSET:             1,
	ER_CANT_SET_COLLATION:           1,
	ER_CANT_SET_ENGINE:              1,
	ER_CHANGE_COLUMN_TYPE:           1,
	ER_CHANGE_TOO_MUCH_ROWS:         1,
	ER_CHAR_TO_VARCHAR_LEN:          1,
	ER_CHARSET_ON_COLUMN:            1,
	ER_COLUMN_HAVE_NO_COMMENT:       1,
	ER_DATETIME_DEFAULT:             1,
	ErrFloatDoubleToDecimal:         2,
	ER_FOREIGN_KEY:                  2,
	ER_IDENT_USE_KEYWORD:            1,
	ER_INC_INIT_ERR:                 1,
	ER_INDEX_NAME_IDX_PREFIX:        1,
	ER_INDEX_NAME_UNIQ_PREFIX:       1,
	ER_INSERT_TOO_MUCH_ROWS:         1,
	ER_INVALID_DATA_TYPE:            1,
	ER_INVALID_IDENT:                1,
	ErrMariaDBRollbackWarn:          1,
	ER_MUST_HAVE_COLUMNS:            1,
	ErrColumnsMustHaveIndex:         1,
	ErrColumnsMustHaveIndexTypeErr:  1,
	ER_NO_WHERE_CONDITION:           1,
	ER_NOT_ALLOWED_NULLABLE:         1,
	ER_ORDERY_BY_RAND:               1,
	ER_PARTITION_NOT_ALLOWED:        1,
	ER_PK_COLS_NOT_INT:              1,
	ER_PK_TOO_MANY_PARTS:            1,
	ER_SELECT_ONLY_STAR:             1,
	ER_SET_DATA_TYPE_INT_BIGINT:     2,
	ER_TABLE_CHARSET_MUST_NULL:      1,
	ER_TABLE_CHARSET_MUST_UTF8:      1,
	ER_TABLE_MUST_HAVE_COMMENT:      1,
	ER_TABLE_MUST_HAVE_PK:           1,
	ER_TABLE_PREFIX:                 1,
	ER_TEXT_NOT_NULLABLE_ERROR:      1,
	ER_TIMESTAMP_DEFAULT:            1,
	ER_TOO_MANY_KEY_PARTS:           1,
	ER_TOO_MANY_KEYS:                1,
	ER_TOO_MUCH_AUTO_DATETIME_COLS:  2,
	ER_TOO_MUCH_AUTO_TIMESTAMP_COLS: 2,
	ER_UDPATE_TOO_MUCH_ROWS:         1,
	ER_USE_ENUM:                     1,
	ER_USE_TEXT_OR_BLOB:             2,
	ER_WITH_DEFAULT_ADD_COLUMN:      1,
	ER_WITH_INSERT_FIELD:            1,
	ER_WITH_LIMIT_CONDITION:         1,
	ER_WITH_ORDERBY_CONDITION:       1,
	ErCantChangeColumn:              1,
	ErCantChangeColumnPosition:      1,
	ErJsonTypeSupport:               2,
	ErrImplicitTypeConversion:       1,
	ErrJoinNoOnCondition:            1,
	ErrUseValueExpr:                 1,
	ErrWrongAndExpr:                 1,
	ErrViewSupport:                  2,
	ErrIncorrectDateTimeValue:       2,
	ErrMaxVarcharLength:             2,
	ErrMaxColumnCount:               1,
}

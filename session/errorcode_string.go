// Code generated by "stringer -type=ErrorCode"; DO NOT EDIT.

package session

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ER_ERROR_FIRST-0]
	_ = x[ER_NOT_SUPPORTED_YET-1]
	_ = x[ER_SQL_NO_SOURCE-2]
	_ = x[ER_SQL_NO_OP_TYPE-3]
	_ = x[ER_SQL_INVALID_OP_TYPE-4]
	_ = x[ER_PARSE_ERROR-5]
	_ = x[ER_SYNTAX_ERROR-6]
	_ = x[ER_REMOTE_EXE_ERROR-7]
	_ = x[ER_SHUTDOWN_COMPLETE-8]
	_ = x[ER_WITH_INSERT_FIELD-9]
	_ = x[ER_WITH_INSERT_VALUES-10]
	_ = x[ER_WRONG_VALUE_COUNT_ON_ROW-11]
	_ = x[ER_BAD_FIELD_ERROR-12]
	_ = x[ER_FIELD_SPECIFIED_TWICE-13]
	_ = x[ER_BAD_NULL_ERROR-14]
	_ = x[ER_NO_WHERE_CONDITION-15]
	_ = x[ER_NORMAL_SHUTDOWN-16]
	_ = x[ER_FORCING_CLOSE-17]
	_ = x[ER_CON_COUNT_ERROR-18]
	_ = x[ER_INVALID_COMMAND-19]
	_ = x[ER_SQL_INVALID_SOURCE-20]
	_ = x[ER_WRONG_DB_NAME-21]
	_ = x[EXIT_UNKNOWN_VARIABLE-22]
	_ = x[EXIT_UNKNOWN_OPTION-23]
	_ = x[ER_NO_DB_ERROR-24]
	_ = x[ER_WITH_LIMIT_CONDITION-25]
	_ = x[ER_WITH_ORDERBY_CONDITION-26]
	_ = x[ER_SELECT_ONLY_STAR-27]
	_ = x[ER_ORDERY_BY_RAND-28]
	_ = x[ER_ID_IS_UPER-29]
	_ = x[ER_UNKNOWN_COLLATION-30]
	_ = x[ER_INVALID_DATA_TYPE-31]
	_ = x[ER_NOT_ALLOWED_NULLABLE-32]
	_ = x[ER_DUP_FIELDNAME-33]
	_ = x[ER_WRONG_COLUMN_NAME-34]
	_ = x[ER_WRONG_AUTO_KEY-35]
	_ = x[ER_TABLE_CANT_HANDLE_AUTO_INCREMENT-36]
	_ = x[ER_FOREIGN_KEY-37]
	_ = x[ER_TOO_MANY_KEY_PARTS-38]
	_ = x[ER_TOO_LONG_IDENT-39]
	_ = x[ER_UDPATE_TOO_MUCH_ROWS-40]
	_ = x[ER_INSERT_TOO_MUCH_ROWS-41]
	_ = x[ER_WRONG_NAME_FOR_INDEX-42]
	_ = x[ER_TOO_MANY_KEYS-43]
	_ = x[ER_NOT_SUPPORTED_KEY_TYPE-44]
	_ = x[ER_WRONG_SUB_KEY-45]
	_ = x[ER_WRONG_KEY_COLUMN-46]
	_ = x[ER_TOO_LONG_KEY-47]
	_ = x[ER_MULTIPLE_PRI_KEY-48]
	_ = x[ER_DUP_KEYNAME-49]
	_ = x[ER_TOO_LONG_INDEX_COMMENT-50]
	_ = x[ER_DUP_INDEX-51]
	_ = x[ER_TEMP_TABLE_TMP_PREFIX-52]
	_ = x[ER_TABLE_CHARSET_MUST_UTF8-53]
	_ = x[ER_TABLE_CHARSET_MUST_NULL-54]
	_ = x[ER_TABLE_MUST_HAVE_COMMENT-55]
	_ = x[ER_COLUMN_HAVE_NO_COMMENT-56]
	_ = x[ER_TABLE_MUST_HAVE_PK-57]
	_ = x[ER_PARTITION_NOT_ALLOWED-58]
	_ = x[ER_USE_ENUM-59]
	_ = x[ER_USE_TEXT_OR_BLOB-60]
	_ = x[ER_COLUMN_EXISTED-61]
	_ = x[ER_COLUMN_NOT_EXISTED-62]
	_ = x[ER_CANT_DROP_FIELD_OR_KEY-63]
	_ = x[ER_INVALID_DEFAULT-64]
	_ = x[ER_USERNAME-65]
	_ = x[ER_HOSTNAME-66]
	_ = x[ER_NOT_VALID_PASSWORD-67]
	_ = x[ER_WRONG_STRING_LENGTH-68]
	_ = x[ER_BLOB_USED_AS_KEY-69]
	_ = x[ER_TOO_LONG_BAKDB_NAME-70]
	_ = x[ER_INVALID_BACKUP_HOST_INFO-71]
	_ = x[ER_BINLOG_CORRUPTED-72]
	_ = x[ER_NET_READ_ERROR-73]
	_ = x[ER_NETWORK_READ_EVENT_CHECKSUM_FAILURE-74]
	_ = x[ER_SLAVE_RELAY_LOG_WRITE_FAILURE-75]
	_ = x[ER_INCORRECT_GLOBAL_LOCAL_VAR-76]
	_ = x[ER_START_AS_BEGIN-77]
	_ = x[ER_OUTOFMEMORY-78]
	_ = x[ER_HAVE_BEGIN-79]
	_ = x[ER_NET_READ_INTERRUPTED-80]
	_ = x[ER_BINLOG_FORMAT_STATEMENT-81]
	_ = x[EXIT_NO_ARGUMENT_ALLOWED-82]
	_ = x[EXIT_ARGUMENT_REQUIRED-83]
	_ = x[EXIT_AMBIGUOUS_OPTION-84]
	_ = x[ER_ERROR_EXIST_BEFORE-85]
	_ = x[ER_UNKNOWN_SYSTEM_VARIABLE-86]
	_ = x[ER_UNKNOWN_CHARACTER_SET-87]
	_ = x[ER_END_WITH_COMMIT-88]
	_ = x[ER_DB_NOT_EXISTED_ERROR-89]
	_ = x[ER_TABLE_EXISTS_ERROR-90]
	_ = x[ER_INDEX_NAME_IDX_PREFIX-91]
	_ = x[ER_INDEX_NAME_UNIQ_PREFIX-92]
	_ = x[ER_AUTOINC_UNSIGNED-93]
	_ = x[ER_VARCHAR_TO_TEXT_LEN-94]
	_ = x[ER_CHAR_TO_VARCHAR_LEN-95]
	_ = x[ER_KEY_COLUMN_DOES_NOT_EXITS-96]
	_ = x[ER_INC_INIT_ERR-97]
	_ = x[ER_WRONG_ARGUMENTS-98]
	_ = x[ER_SET_DATA_TYPE_INT_BIGINT-99]
	_ = x[ER_TIMESTAMP_DEFAULT-100]
	_ = x[ER_CHARSET_ON_COLUMN-101]
	_ = x[ER_AUTO_INCR_ID_WARNING-102]
	_ = x[ER_ALTER_TABLE_ONCE-103]
	_ = x[ER_BLOB_CANT_HAVE_DEFAULT-104]
	_ = x[ER_END_WITH_SEMICOLON-105]
	_ = x[ER_NON_UNIQ_ERROR-106]
	_ = x[ER_TABLE_NOT_EXISTED_ERROR-107]
	_ = x[ER_UNKNOWN_TABLE-108]
	_ = x[ER_INVALID_GROUP_FUNC_USE-109]
	_ = x[ER_INDEX_USE_ALTER_TABLE-110]
	_ = x[ER_WITH_DEFAULT_ADD_COLUMN-111]
	_ = x[ER_TRUNCATED_WRONG_VALUE-112]
	_ = x[ER_TEXT_NOT_NULLABLE_ERROR-113]
	_ = x[ER_WRONG_VALUE_FOR_VAR-114]
	_ = x[ER_TOO_MUCH_AUTO_TIMESTAMP_COLS-115]
	_ = x[ER_INVALID_ON_UPDATE-116]
	_ = x[ER_DDL_DML_COEXIST-117]
	_ = x[ER_SLAVE_CORRUPT_EVENT-118]
	_ = x[ER_COLLATION_CHARSET_MISMATCH-119]
	_ = x[ER_NOT_SUPPORTED_ALTER_OPTION-120]
	_ = x[ER_CONFLICTING_DECLARATIONS-121]
	_ = x[ER_IDENT_USE_KEYWORD-122]
	_ = x[ER_VIEW_SELECT_CLAUSE-123]
	_ = x[ER_OSC_KILL_FAILED-124]
	_ = x[ER_NET_PACKETS_OUT_OF_ORDER-125]
	_ = x[ER_NOT_SUPPORTED_ITEM_TYPE-126]
	_ = x[ER_INVALID_IDENT-127]
	_ = x[ER_INCEPTION_EMPTY_QUERY-128]
	_ = x[ER_PK_COLS_NOT_INT-129]
	_ = x[ER_PK_TOO_MANY_PARTS-130]
	_ = x[ER_REMOVED_SPACES-131]
	_ = x[ER_CHANGE_COLUMN_TYPE-132]
	_ = x[ER_CANT_DROP_TABLE-133]
	_ = x[ER_CANT_DROP_DATABASE-134]
	_ = x[ER_WRONG_TABLE_NAME-135]
	_ = x[ER_CANT_SET_CHARSET-136]
	_ = x[ER_CANT_SET_COLLATION-137]
	_ = x[ER_CANT_SET_ENGINE-138]
	_ = x[ER_MUST_AT_LEAST_ONE_COLUMN-139]
	_ = x[ER_MUST_HAVE_COLUMNS-140]
	_ = x[ER_PRIMARY_CANT_HAVE_NULL-141]
	_ = x[ErrCantRemoveAllFields-142]
	_ = x[ErrNotFoundTableInfo-143]
	_ = x[ErrNotFoundThreadId-144]
	_ = x[ErrNotFoundMasterStatus-145]
	_ = x[ErrNonUniqTable-146]
	_ = x[ErrWrongUsage-147]
	_ = x[ErrDataTooLong-148]
	_ = x[ErrCharsetNotSupport-149]
	_ = x[ErrCollationNotSupport-150]
	_ = x[ErrTableCollationNotSupport-151]
	_ = x[ErrJsonTypeSupport-152]
	_ = x[ErrEngineNotSupport-153]
	_ = x[ErrMixOfGroupFuncAndFields-154]
	_ = x[ErrFieldNotInGroupBy-155]
	_ = x[ErrCantChangeColumnPosition-156]
	_ = x[ER_ERROR_LAST-157]
}

const _ErrorCode_name = "ER_ERROR_FIRSTER_NOT_SUPPORTED_YETER_SQL_NO_SOURCEER_SQL_NO_OP_TYPEER_SQL_INVALID_OP_TYPEER_PARSE_ERRORER_SYNTAX_ERRORER_REMOTE_EXE_ERRORER_SHUTDOWN_COMPLETEER_WITH_INSERT_FIELDER_WITH_INSERT_VALUESER_WRONG_VALUE_COUNT_ON_ROWER_BAD_FIELD_ERRORER_FIELD_SPECIFIED_TWICEER_BAD_NULL_ERRORER_NO_WHERE_CONDITIONER_NORMAL_SHUTDOWNER_FORCING_CLOSEER_CON_COUNT_ERRORER_INVALID_COMMANDER_SQL_INVALID_SOURCEER_WRONG_DB_NAMEEXIT_UNKNOWN_VARIABLEEXIT_UNKNOWN_OPTIONER_NO_DB_ERRORER_WITH_LIMIT_CONDITIONER_WITH_ORDERBY_CONDITIONER_SELECT_ONLY_STARER_ORDERY_BY_RANDER_ID_IS_UPERER_UNKNOWN_COLLATIONER_INVALID_DATA_TYPEER_NOT_ALLOWED_NULLABLEER_DUP_FIELDNAMEER_WRONG_COLUMN_NAMEER_WRONG_AUTO_KEYER_TABLE_CANT_HANDLE_AUTO_INCREMENTER_FOREIGN_KEYER_TOO_MANY_KEY_PARTSER_TOO_LONG_IDENTER_UDPATE_TOO_MUCH_ROWSER_INSERT_TOO_MUCH_ROWSER_WRONG_NAME_FOR_INDEXER_TOO_MANY_KEYSER_NOT_SUPPORTED_KEY_TYPEER_WRONG_SUB_KEYER_WRONG_KEY_COLUMNER_TOO_LONG_KEYER_MULTIPLE_PRI_KEYER_DUP_KEYNAMEER_TOO_LONG_INDEX_COMMENTER_DUP_INDEXER_TEMP_TABLE_TMP_PREFIXER_TABLE_CHARSET_MUST_UTF8ER_TABLE_CHARSET_MUST_NULLER_TABLE_MUST_HAVE_COMMENTER_COLUMN_HAVE_NO_COMMENTER_TABLE_MUST_HAVE_PKER_PARTITION_NOT_ALLOWEDER_USE_ENUMER_USE_TEXT_OR_BLOBER_COLUMN_EXISTEDER_COLUMN_NOT_EXISTEDER_CANT_DROP_FIELD_OR_KEYER_INVALID_DEFAULTER_USERNAMEER_HOSTNAMEER_NOT_VALID_PASSWORDER_WRONG_STRING_LENGTHER_BLOB_USED_AS_KEYER_TOO_LONG_BAKDB_NAMEER_INVALID_BACKUP_HOST_INFOER_BINLOG_CORRUPTEDER_NET_READ_ERRORER_NETWORK_READ_EVENT_CHECKSUM_FAILUREER_SLAVE_RELAY_LOG_WRITE_FAILUREER_INCORRECT_GLOBAL_LOCAL_VARER_START_AS_BEGINER_OUTOFMEMORYER_HAVE_BEGINER_NET_READ_INTERRUPTEDER_BINLOG_FORMAT_STATEMENTEXIT_NO_ARGUMENT_ALLOWEDEXIT_ARGUMENT_REQUIREDEXIT_AMBIGUOUS_OPTIONER_ERROR_EXIST_BEFOREER_UNKNOWN_SYSTEM_VARIABLEER_UNKNOWN_CHARACTER_SETER_END_WITH_COMMITER_DB_NOT_EXISTED_ERRORER_TABLE_EXISTS_ERRORER_INDEX_NAME_IDX_PREFIXER_INDEX_NAME_UNIQ_PREFIXER_AUTOINC_UNSIGNEDER_VARCHAR_TO_TEXT_LENER_CHAR_TO_VARCHAR_LENER_KEY_COLUMN_DOES_NOT_EXITSER_INC_INIT_ERRER_WRONG_ARGUMENTSER_SET_DATA_TYPE_INT_BIGINTER_TIMESTAMP_DEFAULTER_CHARSET_ON_COLUMNER_AUTO_INCR_ID_WARNINGER_ALTER_TABLE_ONCEER_BLOB_CANT_HAVE_DEFAULTER_END_WITH_SEMICOLONER_NON_UNIQ_ERRORER_TABLE_NOT_EXISTED_ERRORER_UNKNOWN_TABLEER_INVALID_GROUP_FUNC_USEER_INDEX_USE_ALTER_TABLEER_WITH_DEFAULT_ADD_COLUMNER_TRUNCATED_WRONG_VALUEER_TEXT_NOT_NULLABLE_ERRORER_WRONG_VALUE_FOR_VARER_TOO_MUCH_AUTO_TIMESTAMP_COLSER_INVALID_ON_UPDATEER_DDL_DML_COEXISTER_SLAVE_CORRUPT_EVENTER_COLLATION_CHARSET_MISMATCHER_NOT_SUPPORTED_ALTER_OPTIONER_CONFLICTING_DECLARATIONSER_IDENT_USE_KEYWORDER_VIEW_SELECT_CLAUSEER_OSC_KILL_FAILEDER_NET_PACKETS_OUT_OF_ORDERER_NOT_SUPPORTED_ITEM_TYPEER_INVALID_IDENTER_INCEPTION_EMPTY_QUERYER_PK_COLS_NOT_INTER_PK_TOO_MANY_PARTSER_REMOVED_SPACESER_CHANGE_COLUMN_TYPEER_CANT_DROP_TABLEER_CANT_DROP_DATABASEER_WRONG_TABLE_NAMEER_CANT_SET_CHARSETER_CANT_SET_COLLATIONER_CANT_SET_ENGINEER_MUST_AT_LEAST_ONE_COLUMNER_MUST_HAVE_COLUMNSER_PRIMARY_CANT_HAVE_NULLErrCantRemoveAllFieldsErrNotFoundTableInfoErrNotFoundThreadIdErrNotFoundMasterStatusErrNonUniqTableErrWrongUsageErrDataTooLongErrCharsetNotSupportErrCollationNotSupportErrTableCollationNotSupportErrJsonTypeSupportErrEngineNotSupportErrMixOfGroupFuncAndFieldsErrFieldNotInGroupByErrCantChangeColumnPositionER_ERROR_LAST"

var _ErrorCode_index = [...]uint16{0, 14, 34, 50, 67, 89, 103, 118, 137, 157, 177, 198, 225, 243, 267, 284, 305, 323, 339, 357, 375, 396, 412, 433, 452, 466, 489, 514, 533, 550, 563, 583, 603, 626, 642, 662, 679, 714, 728, 749, 766, 789, 812, 835, 851, 876, 892, 911, 926, 945, 959, 984, 996, 1020, 1046, 1072, 1098, 1123, 1144, 1168, 1179, 1198, 1215, 1236, 1261, 1279, 1290, 1301, 1322, 1344, 1363, 1385, 1412, 1431, 1448, 1486, 1518, 1547, 1564, 1578, 1591, 1614, 1640, 1664, 1686, 1707, 1728, 1754, 1778, 1796, 1819, 1840, 1864, 1889, 1908, 1930, 1952, 1980, 1995, 2013, 2040, 2060, 2080, 2103, 2122, 2147, 2168, 2185, 2211, 2227, 2252, 2276, 2302, 2326, 2352, 2374, 2405, 2425, 2443, 2465, 2494, 2523, 2550, 2570, 2591, 2609, 2636, 2662, 2678, 2702, 2720, 2740, 2757, 2778, 2796, 2817, 2836, 2855, 2876, 2894, 2921, 2941, 2966, 2988, 3008, 3027, 3050, 3065, 3078, 3092, 3112, 3134, 3161, 3179, 3198, 3224, 3244, 3271, 3284}

func (i ErrorCode) String() string {
	if i < 0 || i >= ErrorCode(len(_ErrorCode_index)-1) {
		return "ErrorCode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ErrorCode_name[_ErrorCode_index[i]:_ErrorCode_index[i+1]]
}

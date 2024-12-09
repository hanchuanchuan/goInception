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

package parser

import (
	"strings"

	"github.com/hanchuanchuan/goInception/util/charset"
)

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentChar(ch rune) bool {
	return isLetter(ch) || isDigit(ch) || ch == '_' || ch == '$' || isIdentExtend(ch)
}

func isIdentExtend(ch rune) bool {
	return ch >= 0x80 && ch <= '\uffff'
}

func isUserVarChar(ch rune) bool {
	return isLetter(ch) || isDigit(ch) || ch == '_' || ch == '$' || ch == '.' || isIdentExtend(ch)
}

type trieNode struct {
	childs [256]*trieNode
	token  int
	fn     func(s *Scanner) (int, Pos, string)
}

var ruleTable trieNode

func initTokenByte(c byte, tok int) {
	if ruleTable.childs[c] == nil {
		ruleTable.childs[c] = &trieNode{}
	}
	ruleTable.childs[c].token = tok
}

func initTokenString(str string, tok int) {
	node := &ruleTable
	for _, c := range str {
		if node.childs[c] == nil {
			node.childs[c] = &trieNode{}
		}
		node = node.childs[c]
	}
	node.token = tok
}

func initTokenFunc(str string, fn func(s *Scanner) (int, Pos, string)) {
	for i := 0; i < len(str); i++ {
		c := str[i]
		if ruleTable.childs[c] == nil {
			ruleTable.childs[c] = &trieNode{}
		}
		ruleTable.childs[c].fn = fn
	}
	return
}

func init() {
	// invalid is a special token defined in parser.y, when parser meet
	// this token, it will throw an error.
	// set root trie node's token to invalid, so when input match nothing
	// in the trie, invalid will be the default return token.
	ruleTable.token = invalid
	initTokenByte('/', int('/'))
	initTokenByte('+', int('+'))
	initTokenByte('>', int('>'))
	initTokenByte('<', int('<'))
	initTokenByte('(', int('('))
	initTokenByte(')', int(')'))
	initTokenByte('[', int('['))
	initTokenByte(']', int(']'))
	initTokenByte(';', int(';'))
	initTokenByte(',', int(','))
	initTokenByte('&', int('&'))
	initTokenByte('%', int('%'))
	initTokenByte(':', int(':'))
	initTokenByte('|', int('|'))
	initTokenByte('!', int('!'))
	initTokenByte('^', int('^'))
	initTokenByte('~', int('~'))
	initTokenByte('\\', int('\\'))
	initTokenByte('?', paramMarker)
	initTokenByte('=', eq)
	initTokenByte('{', int('{'))
	initTokenByte('}', int('}'))

	initTokenString("||", pipes)
	initTokenString("&&", andand)
	initTokenString("&^", andnot)
	initTokenString(":=", assignmentEq)
	initTokenString("<=>", nulleq)
	initTokenString(">=", ge)
	initTokenString("<=", le)
	initTokenString("!=", neq)
	initTokenString("<>", neqSynonym)
	initTokenString("<<", lsh)
	initTokenString(">>", rsh)
	initTokenString("\\N", null)

	initTokenFunc("@", startWithAt)
	initTokenFunc("/", startWithSlash)
	initTokenFunc("*", startWithStar)
	initTokenFunc("-", startWithDash)
	initTokenFunc("#", startWithSharp)
	initTokenFunc("Xx", startWithXx)
	initTokenFunc("Nn", startWithNn)
	initTokenFunc("Bb", startWithBb)
	initTokenFunc(".", startWithDot)
	initTokenFunc("_$ACDEFGHIJKLMOPQRSTUVWYZacdefghijklmopqrstuvwyz", scanIdentifier)
	initTokenFunc("`", scanQuotedIdent)
	initTokenFunc("0123456789", startWithNumber)
	initTokenFunc("'\"", startString)
}

var tokenMap = map[string]int{
	"ACTION":           action,
	"ADD":              add,
	"ADDDATE":          addDate,
	"ADMIN":            admin,
	"AFTER":            after,
	"ALL":              all,
	"ALGORITHM":        algorithm,
	"ALTER":            alter,
	"ALWAYS":           always,
	"ANALYZE":          analyze,
	"AND":              and,
	"ANY":              any,
	"AS":               as,
	"ASC":              asc,
	"ASCII":            ascii,
	"ATTRIBUTES":       attributes,
	"AUTO_INCREMENT":   autoIncrement,
	"AUTO_RANDOM":      autoRandom,
	"AUTO_RANDOM_BASE": autoRandomBase,
	"AVG":              avg,
	"AVG_ROW_LENGTH":   avgRowLength,
	"BEFORE":           before,
	"BEGIN":            begin,
	"BETWEEN":          between,
	"BIGINT":           bigIntType,
	"BINARY":           binaryType,
	"BINLOG":           binlog,
	"BINDING":          binding,
	"BIT":              bitType,
	"BIT_AND":          bitAnd,
	"BIT_OR":           bitOr,
	"BIT_XOR":          bitXor,
	"BLOB":             blobType,
	"BOOL":             boolType,
	"BOOLEAN":          booleanType,
	"BOTH":             both,
	"BTREE":            btree,
	"BUCKETS":          buckets,
	"BY":               by,
	"BYTE":             byteType,
	"CANCEL":           cancel,
	"CASCADE":          cascade,
	"CASCADED":         cascaded,
	"CASE":             caseKwd,
	"CAST":             cast,
	"CHANGE":           change,
	"CHAR":             charType,
	"CHARACTER":        character,
	"CHARSET":          charsetKwd,
	"CHECK":            check,
	"CHECKSUM":         checksum,
	"CLEANUP":          cleanup,
	"CLIENT":           client,
	"COALESCE":         coalesce,
	"COLLATE":          collate,
	"COLLATION":        collation,
	"COLUMN":           column,
	"COLUMNS":          columns,
	"COMMENT":          comment,
	"COMMIT":           commit,

	"COMMITTED":                committed,
	"COMPACT":                  compact,
	"COMPRESSED":               compressed,
	"COMPRESSION":              compression,
	"CONNECTION":               connection,
	"CONSISTENT":               consistent,
	"CONSTRAINT":               constraint,
	"CONVERT":                  convert,
	"COPY":                     copyKwd,
	"COUNT":                    count,
	"CREATE":                   create,
	"CROSS":                    cross,
	"CURRENT":                  current,
	"CURDATE":                  curDate,
	"CURRENT_DATE":             currentDate,
	"CURRENT_TIME":             currentTime,
	"CURRENT_TIMESTAMP":        currentTs,
	"CURRENT_USER":             currentUser,
	"CURTIME":                  curTime,
	"DATA":                     data,
	"DATABASE":                 database,
	"DATABASES":                databases,
	"DATE":                     dateType,
	"DATE_ADD":                 dateAdd,
	"DATE_SUB":                 dateSub,
	"DATETIME":                 datetimeType,
	"DAY":                      day,
	"DAY_HOUR":                 dayHour,
	"DAY_MICROSECOND":          dayMicrosecond,
	"DAY_MINUTE":               dayMinute,
	"DAY_SECOND":               daySecond,
	"DDL":                      ddl,
	"DEALLOCATE":               deallocate,
	"DEC":                      decimalType,
	"DECIMAL":                  decimalType,
	"DEFAULT":                  defaultKwd,
	"DEFINER":                  definer,
	"DELAY_KEY_WRITE":          delayKeyWrite,
	"DELAYED":                  delayed,
	"DELETE":                   deleteKwd,
	"DESC":                     desc,
	"DESCRIBE":                 describe,
	"DIRECTORY":                directory,
	"DISABLE":                  disable,
	"DISCARD":                  discard,
	"DISTINCT":                 distinct,
	"DISTINCTROW":              distinct,
	"DIV":                      div,
	"DO":                       do,
	"DOUBLE":                   doubleType,
	"DROP":                     drop,
	"DUAL":                     dual,
	"DUPLICATE":                duplicate,
	"DYNAMIC":                  dynamic,
	"ELSE":                     elseKwd,
	"ENABLE":                   enable,
	"ENCLOSED":                 enclosed,
	"END":                      end,
	"ENGINE":                   engine,
	"ENGINES":                  engines,
	"ENUM":                     enum,
	"ESCAPE":                   escape,
	"ESCAPED":                  escaped,
	"EVENT":                    event,
	"EVENTS":                   events,
	"EXCLUSIVE":                exclusive,
	"EXCHANGE":                 exchange,
	"EXECUTE":                  execute,
	"EXISTS":                   exists,
	"EXPLAIN":                  explain,
	"EXTRACT":                  extract,
	"FALSE":                    falseKwd,
	"FIELDS":                   fields,
	"FIRST":                    first,
	"FIXED":                    fixed,
	"FLOAT":                    floatType,
	"FLUSH":                    flush,
	"FOR":                      forKwd,
	"FORCE":                    force,
	"FOREIGN":                  foreign,
	"FORMAT":                   format,
	"FROM":                     from,
	"FULL":                     full,
	"FULLTEXT":                 fulltext,
	"FUNCTION":                 function,
	"GENERATED":                generated,
	"GEOMETRY":                 geometryType,
	"POINT":                    geometryType,
	"LINESTRING":               geometryType,
	"POLYGON":                  geometryType,
	"GET":                      get,
	"GET_FORMAT":               getFormat,
	"GLOBAL":                   global,
	"GRANT":                    grant,
	"GRANTS":                   grants,
	"GROUP":                    group,
	"GROUP_CONCAT":             groupConcat,
	"HASH":                     hash,
	"HAVING":                   having,
	"HIGH_PRIORITY":            highPriority,
	"HISTORY":                  history,
	"HOUR":                     hour,
	"HOUR_MICROSECOND":         hourMicrosecond,
	"HOUR_MINUTE":              hourMinute,
	"HOUR_SECOND":              hourSecond,
	"IDENTIFIED":               identified,
	"IF":                       ifKwd,
	"IGNORE":                   ignore,
	"IMPORT":                   importKwd,
	"IMPORTS":                  imports,
	"IN":                       in,
	"INDEX":                    index,
	"INDEXES":                  indexes,
	"INFILE":                   infile,
	"INNER":                    inner,
	"INPLACE":                  inplace,
	"INSTANT":                  instant,
	"INSERT":                   insert,
	"INT":                      intType,
	"INT1":                     int1Type,
	"INT2":                     int2Type,
	"INT3":                     int3Type,
	"INT4":                     int4Type,
	"INT8":                     int8Type,
	"INTEGER":                  integerType,
	"INTERVAL":                 interval,
	"INTERNAL":                 internal,
	"INTO":                     into,
	"INVISIBLE":                invisible,
	"INVOKER":                  invoker,
	"IS":                       is,
	"ISOLATION":                isolation,
	"JOBS":                     jobs,
	"JOB":                      job,
	"JOIN":                     join,
	"JSON":                     jsonType,
	"KEY":                      key,
	"KEY_BLOCK_SIZE":           keyBlockSize,
	"KEYS":                     keys,
	"KILL":                     kill,
	"LEADING":                  leading,
	"LEFT":                     left,
	"LESS":                     less,
	"LEVEL":                    level,
	"LEVELS":                   levels,
	"LIKE":                     like,
	"LIMIT":                    limit,
	"LINEAR":                   linear,
	"LINES":                    lines,
	"LIST":                     list,
	"LOAD":                     load,
	"LOCAL":                    local,
	"LOCALITY":                 locality,
	"LOCALTIME":                localTime,
	"LOCALTIMESTAMP":           localTs,
	"LOCK":                     lock,
	"LONG":                     long,
	"LONGBLOB":                 longblobType,
	"LONGTEXT":                 longtextType,
	"LOW_PRIORITY":             lowPriority,
	"MASTER":                   master,
	"MAX":                      max,
	"MAX_CONNECTIONS_PER_HOUR": maxConnectionsPerHour,
	"MAX_EXECUTION_TIME":       maxExecutionTime,
	"MAX_QUERIES_PER_HOUR":     maxQueriesPerHour,
	"MAX_ROWS":                 maxRows,
	"MAX_UPDATES_PER_HOUR":     maxUpdatesPerHour,
	"MAX_USED_PART_ID":         maxUsedPartId,
	"MAX_USER_CONNECTIONS":     maxUserConnections,
	"MAXVALUE":                 maxValue,
	"MEDIUMBLOB":               mediumblobType,
	"MEDIUMINT":                mediumIntType,
	"MEDIUMTEXT":               mediumtextType,
	"MERGE":                    merge,
	"MICROSECOND":              microsecond,
	"MIN":                      min,
	"MIN_ROWS":                 minRows,
	"MINUTE":                   minute,
	"MINUTE_MICROSECOND":       minuteMicrosecond,
	"MINUTE_SECOND":            minuteSecond,
	"MOD":                      mod,
	"MODE":                     mode,
	"MODIFY":                   modify,
	"MONTH":                    month,
	"NAMES":                    names,
	"NATIONAL":                 national,
	"NATURAL":                  natural,
	"NO":                       no,
	"NO_WRITE_TO_BINLOG":       noWriteToBinLog,
	"NODEGROUP":                nodegroup,
	"NONE":                     none,
	"NOT":                      not,
	"NOW":                      now,
	"NULL":                     null,
	"NUMERIC":                  numericType,
	"NVARCHAR":                 nvarcharType,
	"OFFSET":                   offset,
	"ON":                       on,
	"ONLY":                     only,
	"OPTIMIZE":                 optimize,
	"OPTION":                   option,
	"OR":                       or,
	"ORDER":                    order,
	"OUTER":                    outer,
	"PACK_KEYS":                packKeys,
	"PARSER":                   parser,
	"PARTITION":                partition,
	"PARTITIONING":             partitioning,
	"PARTITIONS":               partitions,
	"PASSWORD":                 password,
	"PLUGINS":                  plugins,
	"POSITION":                 position,
	"PRE_SPLIT_REGIONS":        preSplitRegions,
	"PRECISION":                precisionType,
	"PREPARE":                  prepare,
	"PRIMARY":                  primary,
	"PRIMARY_ZONE":             primaryZone,
	"PRIVILEGES":               privileges,
	"PROCEDURE":                procedure,
	"PROCESS":                  process,
	"PROCESSLIST":              processlist,
	"PROFILES":                 profiles,
	"QUARTER":                  quarter,
	"QUERY":                    query,
	"QUERIES":                  queries,
	"QUICK":                    quick,
	"SHARD_ROW_ID_BITS":        shardRowIDBits,
	"RANGE":                    rangeKwd,
	"RECOVER":                  recover,
	"READ":                     read,
	"REAL":                     realType,
	"RECENT":                   recent,
	"REDUNDANT":                redundant,
	"REFERENCES":               references,
	"REGEXP":                   regexpKwd,
	"RELOAD":                   reload,
	"REMOVE":                   remove,
	"RENAME":                   rename,
	"REORGANIZE":               reorganize,
	"REPAIR":                   repair,
	"REPEAT":                   repeat,
	"REPEATABLE":               repeatable,
	"REPLACE":                  replace,
	"REPLICATION":              replication,
	"RESTRICT":                 restrict,
	"REVERSE":                  reverse,
	"REVOKE":                   revoke,
	"RIGHT":                    right,
	"RLIKE":                    rlike,
	"ROLLBACK":                 rollback,
	"ROUTINE":                  routine,
	"ROW":                      row,
	"ROW_COUNT":                rowCount,
	"ROW_FORMAT":               rowFormat,
	"RTREE":                    rtree,
	"SCHEMA":                   database,
	"SCHEMAS":                  databases,
	"SECOND":                   second,
	"SECOND_MICROSECOND":       secondMicrosecond,
	"SECURITY":                 security,
	"SELECT":                   selectKwd,
	"SERIALIZABLE":             serializable,
	"SESSION":                  session,
	"SET":                      set,
	"SEPARATOR":                separator,
	"SHARE":                    share,
	"SHARED":                   shared,
	"SHOW":                     show,
	"SIGNED":                   signed,
	"SLAVE":                    slave,
	"SLOW":                     slow,
	"SMALLINT":                 smallIntType,
	"SNAPSHOT":                 snapshot,
	"SOME":                     some,
	"SPATIAL":                  spatial,
	"SQL":                      sql,
	"SQL_CACHE":                sqlCache,
	"SQL_CALC_FOUND_ROWS":      sqlCalcFoundRows,
	"SQL_NO_CACHE":             sqlNoCache,
	"START":                    start,
	"STARTING":                 starting,
	"STATS":                    stats,
	"STATS_BUCKETS":            statsBuckets,
	"STATS_HISTOGRAMS":         statsHistograms,
	"STATS_HEALTHY":            statsHealthy,
	"STATS_META":               statsMeta,
	"STATS_PERSISTENT":         statsPersistent,
	"STATUS":                   status,
	"STORED":                   stored,
	"STRAIGHT_JOIN":            straightJoin,
	"SUBDATE":                  subDate,
	"SUBPARTITION":             subpartition,
	"SUBPARTITIONS":            subpartitions,
	"SUBSTR":                   substring,
	"SUBSTRING":                substring,
	"SUM":                      sum,
	"SUPER":                    super,
	"SYSTEM_TIME":              systemTime,
	"TABLE":                    tableKwd,
	"TABLEGROUP":               tablegroup,
	"TABLEGROUP_ID":            tablegroupId,
	"TABLEGROUPS":              tablegroups,
	"TABLES":                   tables,
	"TABLESPACE":               tablespace,
	"TEMPLATE":                 template,
	"TEMPORARY":                temporary,
	"TEMPTABLE":                temptable,
	"TERMINATED":               terminated,
	"TEXT":                     textType,
	"THAN":                     than,
	"THEN":                     then,
	"TIDB":                     tidb,
	"TIDB_HJ":                  tidbHJ,
	"TIDB_INLJ":                tidbINLJ,
	"TIDB_SMJ":                 tidbSMJ,
	"TIME":                     timeType,
	"TIMESTAMP":                timestampType,
	"TIMESTAMPADD":             timestampAdd,
	"TIMESTAMPDIFF":            timestampDiff,
	"TINYBLOB":                 tinyblobType,
	"TINYINT":                  tinyIntType,
	"TINYTEXT":                 tinytextType,
	"TO":                       to,
	"TOP":                      top,
	"TRACE":                    trace,
	"TRAILING":                 trailing,
	"TRANSACTION":              transaction,
	"TRIGGER":                  trigger,
	"TRIGGERS":                 triggers,
	"TRIM":                     trim,
	"TRUE":                     trueKwd,
	"TRUNCATE":                 truncate,
	"TYPE":                     tp,
	"UNCOMMITTED":              uncommitted,
	"UNDEFINED":                undefined,
	"UNION":                    union,
	"UNIQUE":                   unique,
	"UNKNOWN":                  unknown,
	"UNLOCK":                   unlock,
	"UNSIGNED":                 unsigned,
	"UPDATE":                   update,
	"USAGE":                    usage,
	"USE":                      use,
	"USER":                     user,
	"USING":                    using,
	"UTC_DATE":                 utcDate,
	"UTC_TIME":                 utcTime,
	"UTC_TIMESTAMP":            utcTimestamp,
	"VALIDATION":               validation,
	"VALUE":                    value,
	"VALUES":                   values,
	"VARBINARY":                varbinaryType,
	"VARCHAR":                  varcharType,
	"VARIABLES":                variables,
	"VIEW":                     view,
	"VIRTUAL":                  virtual,
	"VISIBLE":                  visible,
	"WARNINGS":                 warnings,
	"ERRORS":                   identSQLErrors,
	"WEEK":                     week,
	"WHEN":                     when,
	"WHERE":                    where,
	"WITH":                     with,
	"WITHOUT":                  without,
	"WRITE":                    write,
	"XOR":                      xor,
	"YEAR":                     yearType,
	"YEAR_MONTH":               yearMonth,
	"ZEROFILL":                 zerofill,

	"INC":                    inception,
	"INCEPTION":              inception,
	"INCEPTION_MAGIC_START":  inception_magic_start,
	"INCEPTION_MAGIC_COMMIT": inception_magic_commit,

	"OSC":         osc,
	"OSC_PERCENT": osc_percent,
	"STOP":        stop,
	"PAUSE":       pause,
	"RESUME":      resume,
}

// See https://dev.mysql.com/doc/refman/5.7/en/function-resolution.html for details
var btFuncTokenMap = map[string]int{
	"ADDDATE":      builtinAddDate,
	"BIT_AND":      builtinBitAnd,
	"BIT_OR":       builtinBitOr,
	"BIT_XOR":      builtinBitXor,
	"CAST":         builtinCast,
	"COUNT":        builtinCount,
	"CURDATE":      builtinCurDate,
	"CURTIME":      builtinCurTime,
	"DATE_ADD":     builtinDateAdd,
	"DATE_SUB":     builtinDateSub,
	"EXTRACT":      builtinExtract,
	"GROUP_CONCAT": builtinGroupConcat,
	"MAX":          builtinMax,
	"MID":          builtinSubstring,
	"MIN":          builtinMin,
	"NOW":          builtinNow,
	"POSITION":     builtinPosition,
	"SESSION_USER": builtinUser,
	"STD":          builtinStddevPop,
	"STDDEV":       builtinStddevPop,
	"STDDEV_POP":   builtinStddevPop,
	"STDDEV_SAMP":  builtinVarSamp,
	"SUBDATE":      builtinSubDate,
	"SUBSTR":       builtinSubstring,
	"SUBSTRING":    builtinSubstring,
	"SUM":          builtinSum,
	"SYSDATE":      builtinSysDate,
	"SYSTEM_USER":  builtinUser,
	"TRIM":         builtinTrim,
	"VARIANCE":     builtinVarPop,
	"VAR_POP":      builtinVarPop,
	"VAR_SAMP":     builtinVarSamp,
}

var windowFuncTokenMap = map[string]int{
	// "CUME_DIST":    cumeDist,
	// "DENSE_RANK":   denseRank,
	// "FIRST_VALUE":  firstValue,
	// "GROUPS":       groups,
	// "LAG":          lag,
	// "LAST_VALUE":   lastValue,
	// "LEAD":         lead,
	// "NTH_VALUE":    nthValue,
	// "NTILE":        ntile,
	// "OVER":         over,
	// "PERCENT_RANK": percentRank,
	// "RANK":         rank,
	// "ROWS":         rows,
	// "ROW_NUMBER":   rowNumber,
	// "WINDOW":       window,
}

// aliases are strings directly map to another string and use the same token.
var aliases = map[string]string{
	"SCHEMA":     "DATABASE",
	"SCHEMAS":    "DATABASES",
	"DEC":        "DECIMAL",
	"SUBSTR":     "SUBSTRING",
	"POINT":      "GEOMETRY",
	"LINESTRING": "GEOMETRY",
	"POLYGON":    "GEOMETRY",
	"INC":        "INCEPTION",
	// "GET":     "SHOW",
}

// hintedTokens is a set of tokens which recognizes a hint.
// According to https://dev.mysql.com/doc/refman/8.0/en/optimizer-hints.html,
// only SELECT, INSERT, REPLACE, UPDATE and DELETE accept optimizer hints.
// additionally we support CREATE and PARTITION for hints at table creation.
var hintedTokens = map[int]struct{}{
	selectKwd: {},
	insert:    {},
	replace:   {},
	update:    {},
	deleteKwd: {},
	create:    {},
	partition: {},
}

var hintTokenMap = map[string]int{
	// MySQL 8.0 hint names
	"JOIN_FIXED_ORDER":      hintJoinFixedOrder,
	"JOIN_ORDER":            hintJoinOrder,
	"JOIN_PREFIX":           hintJoinPrefix,
	"JOIN_SUFFIX":           hintJoinSuffix,
	"BKA":                   hintBKA,
	"NO_BKA":                hintNoBKA,
	"BNL":                   hintBNL,
	"NO_BNL":                hintNoBNL,
	"HASH_JOIN":             hintHashJoin,
	"NO_HASH_JOIN":          hintNoHashJoin,
	"MERGE":                 hintMerge,
	"NO_MERGE":              hintNoMerge,
	"INDEX_MERGE":           hintIndexMerge,
	"NO_INDEX_MERGE":        hintNoIndexMerge,
	"MRR":                   hintMRR,
	"NO_MRR":                hintNoMRR,
	"NO_ICP":                hintNoICP,
	"NO_RANGE_OPTIMIZATION": hintNoRangeOptimization,
	"SKIP_SCAN":             hintSkipScan,
	"NO_SKIP_SCAN":          hintNoSkipScan,
	"SEMIJOIN":              hintSemijoin,
	"NO_SEMIJOIN":           hintNoSemijoin,
	"MAX_EXECUTION_TIME":    hintMaxExecutionTime,
	"SET_VAR":               hintSetVar,
	"RESOURCE_GROUP":        hintResourceGroup,
	"QB_NAME":               hintQBName,

	// TiDB hint names
	"AGG_TO_COP":              hintAggToCop,
	"IGNORE_PLAN_CACHE":       hintIgnorePlanCache,
	"HASH_AGG":                hintHashAgg,
	"IGNORE_INDEX":            hintIgnoreIndex,
	"INL_HASH_JOIN":           hintInlHashJoin,
	"INL_JOIN":                hintInlJoin,
	"INL_MERGE_JOIN":          hintInlMergeJoin,
	"MEMORY_QUOTA":            hintMemoryQuota,
	"NO_SWAP_JOIN_INPUTS":     hintNoSwapJoinInputs,
	"QUERY_TYPE":              hintQueryType,
	"READ_CONSISTENT_REPLICA": hintReadConsistentReplica,
	"READ_FROM_STORAGE":       hintReadFromStorage,
	"MERGE_JOIN":              hintSMJoin,
	"STREAM_AGG":              hintStreamAgg,
	"SWAP_JOIN_INPUTS":        hintSwapJoinInputs,
	"USE_INDEX_MERGE":         hintUseIndexMerge,
	"USE_INDEX":               hintUseIndex,
	"USE_PLAN_CACHE":          hintUsePlanCache,
	"USE_TOJA":                hintUseToja,
	"TIME_RANGE":              hintTimeRange,
	"USE_CASCADES":            hintUseCascades,

	// TiDB hint aliases
	"TIDB_HJ":   hintHashJoin,
	"TIDB_INLJ": hintInlJoin,
	"TIDB_SMJ":  hintSMJoin,

	// Other keywords
	"OLAP":            hintOLAP,
	"OLTP":            hintOLTP,
	"TIKV":            hintTiKV,
	"TIFLASH":         hintTiFlash,
	"FALSE":           hintFalse,
	"TRUE":            hintTrue,
	"MB":              hintMB,
	"GB":              hintGB,
	"DUPSWEEDOUT":     hintDupsWeedOut,
	"FIRSTMATCH":      hintFirstMatch,
	"LOOSESCAN":       hintLooseScan,
	"MATERIALIZATION": hintMaterialization,
}

func (s *Scanner) isTokenIdentifier(lit string, offset int) int {
	// An identifier before or after '.' means it is part of a qualified identifier.
	// We do not parse it as keyword.
	if s.r.peek() == '.' {
		return 0
	}
	if offset > 0 && s.r.s[offset-1] == '.' {
		return 0
	}
	buf := &s.buf
	buf.Reset()
	buf.Grow(len(lit))
	data := buf.Bytes()[:len(lit)]
	for i := 0; i < len(lit); i++ {
		if lit[i] >= 'a' && lit[i] <= 'z' {
			data[i] = lit[i] + 'A' - 'a'
		} else {
			data[i] = lit[i]
		}
	}

	checkBtFuncToken := false
	if s.r.peek() == '(' {
		checkBtFuncToken = true
	} else if s.sqlMode.HasIgnoreSpaceMode() {
		s.skipWhitespace()
		if s.r.peek() == '(' {
			checkBtFuncToken = true
		}
	}
	if checkBtFuncToken {
		if tok := btFuncTokenMap[string(data)]; tok != 0 {
			return tok
		}
	}
	tok, ok := tokenMap[string(data)]
	if !ok && s.supportWindowFunc {
		tok = windowFuncTokenMap[string(data)]
	}
	return tok
}

func handleIdent(lval *yySymType) int {
	s := lval.ident
	// A character string literal may have an optional character set introducer and COLLATE clause:
	// [_charset_name]'string' [COLLATE collation_name]
	// See https://dev.mysql.com/doc/refman/5.7/en/charset-literal.html
	if !strings.HasPrefix(s, "_") {
		return identifier
	}
	cs, _, err := charset.GetCharsetInfo(s[1:])
	if err != nil {
		return identifier
	}
	lval.ident = cs
	return underscoreCS
}

// SpecialCommentsController controls whether special comments like `/*T![xxx] yyy */`
// can be parsed as `yyy`. To add such rules, please use SpecialCommentsController.Register().
// For example:
//
//	SpecialCommentsController.Register("30100");
//
// Now the parser will treat
//
//	select a, /*T![30100] mysterious_keyword */ from t;
//
// and
//
//	select a, mysterious_keyword from t;
//
// equally.
// Similar special comments without registration are ignored by parser.
var SpecialCommentsController = specialCommentsCtrl{
	supportedFeatures: map[string]struct{}{},
}

type specialCommentsCtrl struct {
	supportedFeatures map[string]struct{}
}

func (s *specialCommentsCtrl) Register(featureID string) {
	s.supportedFeatures[featureID] = struct{}{}
}

func (s *specialCommentsCtrl) Unregister(featureID string) {
	delete(s.supportedFeatures, featureID)
}

func (s *specialCommentsCtrl) ContainsAll(featureIDs []string) bool {
	for _, f := range featureIDs {
		if _, found := s.supportedFeatures[f]; !found {
			return false
		}
	}
	return true
}

// GetKeywords is 自定义函数,返回所有的关键字
func GetKeywords() map[string]int {
	return tokenMap
}

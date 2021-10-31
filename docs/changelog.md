# goInception Changelog

## [v1.2.5]-2021-10-31

### Fix
* Fix the problem that the ghost_initially_drop_socket_file parameter of gh-ost does not take effect (#382)
* Fix the inconsistency of index prefix between Chinese and English (#380)
* Fix the problem that the number of affected rows may overflow (#375)
* Fix the problem that the default value of bit type is set to NULL and it cannot be approved (#346)
* Fix the problem that pt-osc fails to execute when connecting to a port other than 3306 (#335)
* Fix the bug that dml of tidb v4.0 version cannot be audited
* Fix the problem that the calculated column is not ignored in the dml rollback statement (#324)
* Fix the problem that the rename table syntax does not take effect logically when multiple table names are modified at one time (#327)

### New Features
* Added the `check_identifier_lower` parameter to enable lowercase requirements for table names, column names, and index names (#389)
* Add `masking function`-syntax tree parsing function upgrade, return objects involved in select (library, table, column) (#355)
* Added ghost_bin_dir parameter setting from switching gh-ost mode, `independent gh-ost module`, using binary method to call to optimize the possible bugs when ddl high concurrency and kill gh-ost process (#334)
* Add option `max_execution_time` to set the maximum execution time of session-level DML statements (#319)
* Added `enable_any_statement` option to support all syntaxes to be executable (#301)

### Update
* Optimize the keyword review logic of alter table modification column clauses (#392)
* Optimize group by clause review logic (#385)
* Add datetime decimal second check (#369)
* Add view support for union
* Fine-tune the insert column value check logic in strict mode
* Optimize the character type length audit logic
* Optimize the handling of database connection timeout (#306)
* Improve character set and collation review logic, add utf8mb4_0900_ai_ci collation support


## [v1.2.4]-2020-12-19

### Fix
* Fixed the problem that dml may not be backed up successfully in MariaDB
* Fix the issue of audit errors when updating the specified table alias (#249)
* Fix the problem that the child query in the select column may not find the parent table column and the having clause may not find the column (#266)
* Fix the problem of falsely reporting error when not calling in the correct format
* Fix the problem that the float type parameter specified in the `inception set` syntax will report an error (#279)
* Fix the status indicator when the backup statement is not parsed correctly (#286)

### New Features
* Add lock_wait_timeout to control the lock wait timeout time during normal SQL execution (#224)
* Increase the sleep parameter of pt-osc to optimize the load situation when the low configuration db executes ddl changes (#260)
* Add view support (#238,#262)
* Add the parameter `ignore_osc_alter_stmt` to configure the syntax to force the osc check to be ignored (#258,#263)
* Added CREATE TABLE AS SELECT syntax support (#246,#264)
* Add auto_random support for tidb column attributes (#270)
* Add sql_mode option when inc is executed (#267)
* Add illegal date review when inserting (#277)

### Update
* Optimize index visibility audit support for multi-version databases (#247)
* Optimize the binlog analysis of the backup function (#250)
* Improve MySQL8.0 keyword list (#210)
* Optimize the configuration of the default database, the use operation can be omitted after specifying the --db option
* Optimize the osc process processing logic and add concurrent lock processing
* Improve alter table partition syntax (#281)
* Optimize the processing of the backup function when ssl is turned on (#287)


## [v1.2.3]-2020-05-22

### Fix
* fix: Fix the problem that pt-osc cannot be killed when getting the table lock stuck (#213, #222)

### New Features
* Add the parameter `ignore_sighup`, the terminal connection disconnection signal is ignored by default (#195)
* Add `osc_lock_wait_timeout` parameter to control pt-osc wait for meta lock time, default 60s (#214, #215)
* Support multiple custom index prefixes (#204)

### Update
* Improve MySQL5.7 keyword list (#210)


## [v1.2.2]-2020-04-04

### Fix
* Fix the problem of erroneous review when `max_char_length` is not specified
* Fix the problem of inaccurate index column length limit (#176)
* Fix the problem that the inception set command returns an error, and fix the problem that the lang setting may not take effect
* Fix the problem that errors may occur during mixed execution of DDL and DML when opening a transaction (#182)

### New Features
* Add session-level variable settings (#157, #166, #167)
* Add the osc parameter `osc_max_flow_ctl` so that the PXC cluster can enable the pt-osc function (#170, #172)
* Add audit item `columns_must_have_index`, index must be added to the specified column (#174, #175)

### Update
* Optimize column character set and collation review logic (#173)
* Add review of value expressions in where conditions to avoid incorrect updates of invalid expressions (#178)
* Optimize the printing of internal SQL that must be executed with privileges when authentication fails, so that the problem can be quickly located


## [v1.2.1]-2020-03-07

### Fix
* Optimize the problem that the comments before and after the ALTER statement when using the pt tool may cause its parsing failure

### New Features
* Add version number information to system variables `show variables like'version'` (#164)

### Update
* When opening the parameter for obtaining the actual number of affected rows, ignore the sql fingerprint function (accuracy first)
* Automatically remove the special spaces at the beginning and end of each SQL line (ASCII code 160)
* Optimized bit type default value audit, now supports audit like `b'1'`
* Optimize the error message when the data source is not configured correctly to make it more friendly
* Cache the setting of the unique key when creating a table
* Improve the display width review of numeric types (#162)


## [v1.2.0]-2020-01-18

### Fix
* Fix the bug that the change column was not properly reviewed when the column name was modified and the old column name was quoted immediately (#150)
* Fixed the problem of incorrect review when using the wrong length of the year type
* Fix the problem that the audit logic of ON clause of join grammar is incorrect

### New Features
* Table name and index name prefix customization (#149)
    * Table name prefix `table_prefix`, the default is empty, that is, unlimited
    * Index prefix `index_prefix`, the default is `idx_`, which is consistent with the previous version and can be customized
    * Unique index prefix `uniq_index_prefix`, the default is `uniq_`, which is consistent with the previous version and can be customized

### Update
* Optimize the alter clause parsing when using the pt tool
* Optimize the log output of gh-ost and backup functions and adjust to a unified format
* Optimize the parameter settings of the default database, adjust the default `mysql` library to be empty by default to avoid affecting the master-slave synchronization under special circumstances

## [v1.1.6]-2020-01-02

### Fix
* Fix the issue of incorrect audit of timestamp type switch
* Fix the bug that the DML operation is not submitted correctly when the instance is not turned on autocommit (#146)

### New Features
* Add user authentication module to realize user management and secure connection function (#132)
* Add [transaction support](trans.html)(batch execution) (#135)

### Update
* Optimize column type change review (#121)
* Implement rollback support when update set with multiple tables (#112,#136)
* Optimize audit rules when case-sensitive (#123)
* Optimize the alter clause split logic when using the pt-osc tool (#142)


## [v1.1.5]-2019-12-09

### Update
* Optimize the case review logic of object names
* Optimize the accuracy of index length audit
* Support update multi-table update syntax (multi-table rollback is not supported temporarily) (#112)
* Optimize the error message when the sql syntax analysis fails

## [v1.1.4]-2019-11-21

### Fix
* Fix the handling of auto-increment columns when inserting non-empty fields (#113)
* Fixed the issue of SQL generation error for the rollback of alter table rename statement
* Fix the problem that limit is not processed when DML transfers to select count when the `real_row_count` option is turned on (#119)

### New Features
* Add new parameter `hex_blob` to support parsing binary types when rolling back (#118)


## [v1.1.3]-2019-11-13

### Fix
* Fix the problem of minimizing the panic generation of rollback statements when there are text, json and other []byte type fields in the table (#105,#107)
* Fix the problem that the `decimal` type becomes scientific notation during reverse analysis (#106,#108)
* Fix a bug that caused thread safety issues when parsing call parameters during multi-threaded high-concurrency testing (#103)

### New Features
* Add the audit option `check_implicit_type_conversion` to audit the implicit type conversion in the where condition (#101)

### Update
* Added TiDB database judgment (tidb backup is not supported)
* Add field ambiguity review when table prefix is not specified


## [v1.1.2]-2019-10-30

### Fix
* Fix the problem that the thread number cannot be backed up when the thread number exceeds the uint32 range

### New Features
* Add the setting parameter `enable_minimal_rollback` to enable the minimized rollback SQL setting (#90)
* Add the setting parameter `wait_timeout` to set the remote database waiting timeout time, the default is 0, that is to keep the database settings
* Add mysql secure connection parameter setting `--ssl`, etc., configurable SSL or CA certificate verification (#92)


## [v1.1.1]-2019-10-13

### Fix
* Fix the problem of explain error in TiDB database (#86)
* Fix the problem that the column number verification of `insert select` syntax may be inaccurate when there are deleted columns

### New Features
* Add the review option `explain_rule` to set how explain gets the number of affected rows

### Update
* Improve the audit rules of `spatial index`
* Adjustments to update syntax are all logically reviewed
* Added ON clause review of join syntax
* Optimize the delete audit rules, skip the explain audit when there are new tables
* When the remote database cannot be connected, optimize the return result and add sql content to return


## [v1.1.0]-2019-9-7

### Fix
* Fix the problem that the add column operation misses the detection of `merge_alter_table` (#79)

### New Features
* Add spatial type syntax analysis, add spatial index support
* Add a new call option `--db` to set the default connection database, the default value is `mysql`

### Update
* Support operations such as creating tables at the same time when building a database (#77)
* Optimize the details of DDL rollback, adjust the rollback SQL for multiple clauses of alter table to reverse (#76)
* Add database read-only status judgment before execution
* Optimize the total length of the index audit, now based on the target library `innodb_large_prefix` parameter judgment
* Review the asterisk column in the select syntax
* Optimize the multi-statement splitting and parsing logic, and optimize the SQL parsing at the end of the semicolon but not the end
* Improve the index check in the column definition


## [v1.0.5]-2019-8-20

### Fix
* Fix the problem that the insert values ​​clause does not support the default syntax

### New Features
* Add the parameter `default_charset` to set the default character set for connecting to the database, the default value is `utf8mb4` (to solve the problem that the lower version does not support utf8mb4)
* Add the pt-osc parameter `osc_check_unique_key_change`, set whether pt-osc checks the unique index, the default is `true`

### Update
* Optimize the rollback function, add binlog_row_image setting check, and automatically modify the session level to full when it is minimal


## [v1.0.4]-2019-8-5

### New Features
* Add syntax support for set names (#69)

### Update
* Optimize the primary key index audit information (#67)
* Improve `update set` multi-field audit rules, add warnings for set multi-column and syntax
* Optimize gh-ost socket file name generation rules to avoid creation failure due to length overflow
* Improve foreign key review rules (#68,#70)


## [v1.0.3]-2019-7-29

### Fix
* `[gh-ost]` fixed the problem that gh-ost did not disconnect the binlog dump connection when abnormal
* `[gh-ost]` Fix the problem that when adding datetime column and default value current_timestamp in gh-ost, incremental data will cause data error due to time zone (timestamp column is normal)

### New Features
* Add parameter `enable_change_column` to set whether to support change column syntax
* Add calling option `real_row_count` to set whether to get the number of really affected rows through `count(*)`. The default value is `false`

### Update
* Add pt-osc to perform change column audit, prohibit multiple change column operations to avoid data loss (pt-osc bug)


## [v1.0.2]-2019-7-26

### Fix
* Fix the bug that the `alter table` command can pass normally without other options (#59)
* Fix the problem that backup records may be written wrongly in the backup library during cross-database operations

### New Features
* Add parameter `max_ddl_affect_rows` to set the maximum number of affected rows allowed by DDL, the default is `0`, which means no limit
* Add parameter `check_float_double`, when it is true, warn that float/double will be converted to decimal data type. Default is false (#62)
* Add parameter `check_identifier_upper` to restrict table names, column names, index names, etc. must be uppercase, the default is `false` (#63)

### Update
* Optimize the implementation of custom audit level, remove the parameter `enable_level`, now the custom audit level and audit switch settings are merged (#52)
* Upgrade parser grammar analysis package, optimize column sorting rules and partition table grammar support (#50)
* Optimize the automatic change of gh-ost's server_id setting to avoid duplication of the same instance


## [v1.0.1]-2019-7-20

### Fix
* Fix the case compatibility problem of `must_have_columns` parameter column type

### New Features
* Added `alter table rename index` syntax support
* Add the parameter `enable_zero_date` to set whether to support the time value of 0, and force an error to be reported when it is closed. The default value is `true` (#55)
* Add the parameter `enable_timestamp_type` to set whether to allow the `timestamp` type field (#57)
* Add `mysql 5.5` version audit support (#54)

### Update
* Optimize the logical storage of modify column information
* Optimize the key definition logic preservation of column attributes


## [v1.0]-2019-7-15

### Fix
* Fixed the problem of pt-osc execution error when the password contains special characters

### New Features
* Add audit result level customization function (#52)

### Update
* Add delete/update self-connection audit support (#51)
* Optimize the automatic change of the specified server_id when binlog is rolled back to avoid duplication of the same instance


## [v1.0-rc4]-2019-7-9

### Fix
* Fix the problem that pt-osc may be executed successfully but the progress is less than 100% (#48)

### New Features
* Add enable_set_engine and support_engine parameters to control whether to allow specifying storage engines and supported storage engine types (#47)

### Update
* Optimize the osc process list, the osc process information of the same session will be cleared later (after the session execution returns) (#48)
* Optimize the generation logic of the backup library library name, and automatically truncate the library name when it is too long (#49)
* Optimize delete and update alias review (#51)

## [v1.0-rc3]-2019-7-2

### Fix
* Fix the problem that may not be supported when using osc to make DDL changes (such as `alter table t engine='innodb'`)

### New Features
* Add sleep execution waiting function to reduce the impact on online databases (#46)
    * Call option `sleep`, how many milliseconds to sleep after executing `sleep_rows` SQL to reduce the impact on online databases
    * Call option `sleep_rows`, sleep once after executing how many SQLs
* Add parameter `max_allowed_packet` to support longer SQL text
* Add parameter `skip_sqls` to be compatible with the default sql of different clients

### Update
* Adjust the sql_statement field type of the backup record table to mediumtext, and it will automatically be compatible with the text type of the old version
* Compatible with mysqlclient client


## [v1.0-rc2]-2019-6-21

### Fix
* Optimize the rollback related table structure, adjust the character set to utf8mb4 (`The structure of the historical table needs to be adjusted manually`)

### Update
* Optimize audit rules, audit subqueries, functions and other expressions (#44)
* Optimize the socket file name format generated by gh-ost by default
* Optimize log output, add thread number display
* Add mariadb judgment when binlog is parsed


## [v1.0-rc1]-2019-6-12

### New Features
* Add split function (#42)


## [v0.9-beta]-2019-6-4

### New Features
* Add statistical function, which can be enabled by parameter `enable_sql_statistic` (#38)
* Add parameter `check_column_position_change` to control whether to check column position/order change (#40, #41)

### Update
* Optimize the logic when using Alibaba Cloud RDS and gh-ost, automatically set the `assume-master-host` parameter (#39)


## [v0.8.3-beta]-2019-5-30

### Fix
* Fix initial-drop-old-table and initial-drop-ghost-table parameter support of gh-ost
* Fix the bug that osc cannot be enabled normally after setting osc_min_table_size greater than 0

### Update
* Compatible syntax inception get processlist
* Built-in pt-osc package in docker image (version 3.0.13)

## [v0.8.2-beta]-2019-5-27

### Fix
* fix: fix the handling of overflow values ​​of unsigned columns during binlog parsing
* fix: Fix the bug of syntax error when gh-ost execution statement has backticks (#33)
* fix: Fix the bug that returns successful execution and backup during kill DDL operation, now it will prompt that the execution result is unknown (#34)


## [v0.8.1-beta]-2019-5-24

### Fix
* Fixed the bug that the returned table does not exist when a table name with inconsistent case is used after a new table is created

### New Features
* Add the general_log parameter to record the full amount of logs

### Update
* Optimize the audit rules for insert select new tables, now you can also audit when you select new tables


## [v0.8-beta]-2019-5-22

### Fix
* Fixed a bug where warnings may be incorrectly marked as errors when the SQL fingerprint function is turned on

### Update
* Optimize sub-query review rules, recursively review all sub-queries
* Review group by syntax and aggregate functions


## [v0.7.5-beta]-2019-5-17

### Fix
* Fix the kill logic in the execution phase to avoid the backup suspension after the kill

### New Features
* Add support for select syntax
* Added ALGORITHM, LOCK, FORCE syntax support for alter table

### Update
* Optimize update subquery review


## [v0.7.4-beta]-2019-5-12

### New Features
* Add alter table option syntax support (#30)
* Redesign the kill operation support, support remote database kill and goInception kill commands (#10)


## [v0.7.3-beta]-2019-5-10

### Fix
* Fix the bug that occasionally marks the execution/backup success when the backup is turned on and the execution error occurs occasionally

### New Features
* Add the `check_column_type_change` parameter to set whether to open the field type change review, the default is `on` (#27)

### Update
* Implement insert select * Review the number of columns


## [v0.7.2-beta]-2019-5-7

### New Features
* Add the `enable_json_type` parameter to set whether to allow json type fields (#26)

### Update
* Implement audit rules based on the system variable explicit_defaults_for_timestamp
* Optimize osc parsing, escape passwords and special characters in alter statements

## [v0.7.1-beta]-2019-5-4

### Update
* Optimize json type field processing logic, no longer check its default value and NOT NULL constraint (#7, #22)
* Optimize must_have_columns parameter value analysis
* Optimize the insert select audit logic

### Fix
* Fix and improve add column(...) syntax support
* Fixed the bug that the execution failed when the alter statement had extra spaces when osc was turned on

### New Features
* Add the `enable_null_index_name` parameter to allow the index name not to be specified (#25)
* Add syntax tree printing function (beta) (#21)


## [v0.7-beta]-2019-4-26

### Update
* Optimized the review of the update associated with the newly created table, now it can associate with the newly created table during update
* Optimized insert new select syntax review, now you can get the estimated number of affected rows
* Automatically ignore warnings during the review phase and optimize the review logic
* Optimize the audit logic of `check_column_default_value`, the primary key will be skipped when auditing the default value
* SQL will be automatically truncated when the backup phase is too long (such as insert values ​​with many rows), `returns warning but does not affect execution and backup operations`

### Fix
* Fix the issue that the column type audit is wrong when the `enable_pk_columns_only_int` option is turned on

### New Features
* Add the `enable_set_collation` parameter to set whether to allow the specified table and database collation rules
* Add the `support_collation` parameter to set the supported collation rules, separated by commas when multiple

## [v0.6.4-beta]-2019-4-23

### Fix
* Fixed the problem that mysql 5.6 and mariadb could not get the number of affected rows


## [v0.6.3-beta]-2019-4-22

### New Features
* Add the `max_insert_rows` parameter to set the maximum number of rows allowed by insert values.
* Add the `must_have_columns` parameter to specify the columns that must be created when the table is built. Separate multiple columns with commas (`format: column name [column type, optional]`)


## [v0.6.2-beta]-2019-4-18
### Update
* Added unsupported syntax warnings (create table as and create table select)
* Support for table structure changes when implementing alter multiple clauses, such as drop column followed by add column

### Fix
* Fix the problem that an error is reported when explain returns a null column
* Fix the problem that the unique identifier of the index is set incorrectly


## [v0.6.1-beta]-2019-4-9
### Update
* Add remote database disconnection and reconnection mechanism, optimize thread number and master status query speed
* Optimize remote database access operations
* Optimize sql content parsing, remove extra semicolons and spaces

### Fix
* Fixed the problem that the column could not be found during cross-database update
* Fix the problem of execution error when the osc clause has double quotes

### New Features
* Add sql fingerprint function
When the dml statements are similar, the explain results can be reused according to the same fingerprint ID to reduce the remote database explain operations and improve the audit speed
	- You can open the global configuration through ```inception set enable_fingerprint=1;``` or configuration file
	- You can also open a single configuration by calling the option ```--fingerprint=1;```
	- Take the union of the two configurations, that is, if either configuration is turned on, the sql fingerprint function will be enabled, and it will be turned off by default.


## [v0.6-beta]-2019-4-3
### Update
* Optimized the performance of backup operations, and changed the backup information to batch writing
* Added backup library connection timeout check
* Performance optimization of explain function
* Optimize the default processing when some functions do not specify the architecture name
* Optimize the default value check, add support for calculated columns (#12, #13, #14)
* Optimize the time format and range check, check the zero-value date according to the database sql_mode
* Upgrade to go 1.12

### Fix
* Fix the index name verification logic, which can be consistent with the column name
* Fix the problem of inaccurate timestamp default value verification

### New Features
* Added support for the kill function, which can be killed during audit and execution, but cannot be killed during the backup phase (#10)
* Add the `check_timestamp_count` parameter to configure whether to check the current_timestamp number (#11, #15)


## [v0.5.3-beta]-2019-3-25
### Update
* Use logical verification when changing column names to avoid explain update failure
* Add union clause verification
* Add table name alias repeatability check

### Fix
* Fix the verification problem when the update set clause specifies the table alias
* Fix the issue of auto-increment column verification
* Fix the verification problem when the default value is an expression

## [v0.5.2-beta]-2019-3-17
### Update
* Optimize the primary key NULL column audit rules (audit `DEFAULT NULL`)
* Optimize the check of the total length of the index, and judge the length of the number of bytes according to the column character set
* Optimize the handling of default values ​​in DDL backup


## [v0.5.1-beta]-2019-3-14
### Update
* Optimize option parsing rules, passwords are compatible with special characters
* Optimize the SQL statement returned when the syntax analysis fails
* Added Chinese exception and warning information
* Add new parameters
	- lang sets the language of the returned exception information, optional values ​​`en-US`, `zh-CN`, default `en-US`
### Fix
* Fix the problem of repeated warning messages in mariadb backup


## [v0.5-beta]-2019-3-10
### Update
* Compatible with backup compatibility of mariadb v10 version (the rollback statement may be wrong in high concurrency, please pay attention to check)
* Update some parameter names of pt-osc to make it consistent with inception
	- osc_critical_running -> osc_critical_thread_running
	- osc_critical_connected -> osc_critical_thread_connected
	- osc_max_running -> osc_max_thread_running
	- osc_max_connected -> osc_max_thread_connected
* Hide some unused parameters of gh-osc
* Add whether to allow deleting the database parameter `enable_drop_database`
* Optimize the display and setting of system variables
* Adjust the default values ​​of some parameters
	- ghost_ok_to_drop_table `true`
	- ghost_skip_foreign_key_checks `true`
	- osc_chunk_size `1000`
### Fix
* Fix json column verification exception problem (#7)

## [v0.4.1-beta]-2019-3-6
### Update
* Compatible with mariadb database (v5.5.60)
	- Added mariadb binlog parsing support (test version **v5.5.60**, v10 version can not resolve thread_id due to the change of binlog format)
	- Optimize the return information when the backup fails


## [v0.4-beta]-2019-3-5
### New Features
* Add gh-ost tool support
	- No need to install gh-ost, built-in functions (v1.0.48)
	- Process list ```inception get osc processlist```
	- Specify process information ```inception get osc_percent'sqlsha1'```
	- Process termination ```inception stop alter'sqlsha1'``` (synonym ```inception kill osc'sqlsha1'```)
	- Process pause ```inception pause alter'sqlsha1'``` (synonym ```inception pause osc'sqlsha1'```)
	- Process resume ```inception resume alter'sqlsha1'``` (synonym ```inception resume osc'sqlsha1'```)
	- Compatible with gh-ost parameters ```inception show variables like'ghost%'```


## [v0.3-beta]-2019-2-13
### New Features
* Add pt-osc tool support
	- ```inception get osc processlist``` to view the osc process list
	- ```inception get osc_percent'sqlsha1'``` to view the specified osc process
	- ```inception stop alter'sqlsha1'``` (synonym ```inception kill osc'sqlsha1'```) abort the specified osc process


## [v0.2-beta]-2019-1-31
### Optimizer
* Optimize the binary build method, compress the installation package size
* Remove vendor dependency and optimize the use of GO111MODULE

* Skip permission verification to avoid failure to log in to goInception
* Remove root authentication to prevent windows from failing to start
* Optimize the type check of inception set variables


## [v0.1-beta]-2019-1-25
#### goInception officially released

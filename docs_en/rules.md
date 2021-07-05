# Audit rules and why

Understand and be familiar with the following audit rules, you can know what goinception roughly does.

**Note**: The rules listed below may not cover all the functions currently implemented by goinception. What specific rules are included must be summarized and discovered during use. At the same time, you can learn about these rules in detail in conjunction with configuration parameters.


## Syntax Support
|syntax|description|
|:----|:----|
|use db|Check if the db exist是, need to connect to the server.|
|set names charset|Only support this set command|
|create database|Create database|
|create table|Create table|
|alter table|Alter table|
|drop table|Drop table|
|truncate table|Truncate table|
|insert|Insert rows|
|update|Update rows|
|delete|Delete rows|
|inception command sets |`Show processing list`. `Show osc processing`, setting config and review level.|
|user command sets |`Create/modify/drop/grant/revoke` user when turn on the safety login.|


## `Insert` rules

|Check item|Check option|
|:----|:----|
|If table exists| default|
|If column exists | default|
|The column which must be insert.|check_insert_field|
|The list of column must by insert.| default|
|The number of insert columns must equal the insert fields.| default|
|Return error if insert NULL to the column cannot be NULL.|default|
|Can not insert duplicate column fields.|default|


## `Update`, `Delete` rules
|Check item|Config option|
|:----|:----|
|table must exists.| default|
|Must use where|check_dml_where|
|Prohibit limit|check_dml_limit|
|Prohibit order by|check_dml_orderby|
|Show warning when the effect rows more than 10000(can setting the number by max_update_rows)|max_update_rows|
|Check Where details.|default|
|All tables and columns in SQL must exist.| default|
|The number limit for insert values|max_insert_rows|
|When update with multi-tables, if do not set prefix, auto match.|default|
|Multi-tables in one SQL, check if column use in ambiguity| default|
|Set Multi-tables in one update, rollback SQL also support.| fefault|
|Use explain to estimate the effect rows or use select count to get the really counts of rows.|realRowCount,explain_rule|
|Set sql_safe_updates|sql_safe_updates|
|Review if has on subquery in join multi-tables.|check_dml_where|
|Check if use implicit type conversion.|check_implicit_type_conversion|
|Check use comma or `and` in update set|default |


## `Create` rules
### table option rules
|Check item|Config option|
|:----|:----|
|If table exists|default|
|If database exists|default|
|For create table like，check if the liked table exists.|default|
|The length of [table/column/index] name cannot be larger than 65 byte.|default|
|Object name must be upper.|check_identifier_upper|
|Object name can contains [a-zA-Z0-9_]|check_identifier|
|Object name can not use keywords.|enable_identifer_keyword|
|Chartet limit|enable_set_charset,support_charset|
|Collation limit|enable_set_collation,support_collation|
|Storage Engine limit|enable_set_engine,support_engine|
|If can use partition table|enable_partition_table|
|Only one auto_increment column|default|
|Only one primary key|default|
|If table must have primary key|check_primary_key|
|If table must have comment|check_table_comment|
|tables have at least one column.|default|
|The columns which tables must have.|must_have_columns|
|The length of the table comment can not overflow|default|
|Prohibit the use of syntax create table as.|default|
|Prohibit use Foreign key|enable_foreign_key|


### column option rules

|Check item|Config option|
|:----|:----|
|If enable to set the charset of column.|enable_column_charset|
|If enable column is any type of collect, enum, bitmap.|enable_enum_set_bit|
|If columns must have comments.|check_column_comment|
|If the length of char is more than 20, you need to change type to varchar.|max_char_length|
|If enable column is BLOB/TEXT.|enable_blob_type|
|If enable column is JSON.|enable_json_type|
|Can not have duplicate column name in one table.|default|
|Only date type can use auto_increment.|default|
|Prohibit use unavailable prefix for schema name or table name.|default|
|If enable not null for every column.|enable_nullable|
|If enable column is timestamp|enable_timestamp_type|
|If the column is timestamp, it must have a default value.|check_timestamp_default|
|If the column is datetime, it must have a default value.|check_datetime_default|
|Prohibit two column are timestamp. If they are datetime, then can not set for DEFAULT `CURRENT_TIMESTAMP` and ON UPDATE `CURRENT_TIMESTAMP`|check_timestamp_count,check_datetime_count|
|Only timestamp or datetime can use on update|default|
|on update express only can use CURRENT_TIMESTAMP|default|
|Suggest use float/double instead of decimal|check_float_double|


### Index rules

|Check item|Config option|
|:----|:----|
|Index must have name|enable_null_index_name|
|Unique index must start by `uniq_` as a prefix.|check_index_prefix|
|Index must start by `idx_` as a prefix.|check_index_prefix|
|The number of columns in an index can not be more than 5.|max_key_parts|
|The limit for the number of columns in primary key.|max_primary_key_parts|
|If the primary key must be int or bigint.|enable_pk_columns_only_int|
|The number of indexes in a table can not be more than 5.|max_keys|
|Index must build on an existing column.|default|
|Can not have duplicate column in an index|default|
|Indexes can not build on the BLOB column.|default|
|The length of index can not more than 767 or 3072, depends on `innodb_large_prefix` of mysql|default|
|Index name can not be PRIMARY|default|
|Index name can not be duplicate.|default|

### `auto_increment` rules
|Check item|Config option|
|:----|:----|
|`auto_increment` must be 1 when create table.|check_autoincrement_init_value|
|If the name of the `auto_increment` column is not id, then the column may have physical meaning, do not do like that.|check_autoincrement_name|
|The type of `auto_increment` column must be `int` or `bigint`.|check_autoincrement_datatype|
|`auto_increment` column must be `unsigned`.|enable_autoincrement_unsigned|

### Default rules
|Check item|Config option|
|:----|:----:|
|`BLOB`/`TEXT` column can not have default value.|enable_blob_not_null|
|If return error, when default value is `NULL` but the column type is `NOT NULL`, or the primary key, or `auto_increment` column.| default|
|`JSON` column can not use default value.| default|
|Expect the column of `auto_increment`/`primary key`/`JSON`/`calculate column`/`BLOB`, every column must have default value.|check_column_default_value|



## `alter` rules
|Check item|Config option|
|:----|:----|
|create index| same as `create` rules |
|Add column|same as `create` rules |
|Default value| same as `create` rules |
|Check storage engine| same as `create` rules |
|Check charset| same as `create` rules |
|Check collation| same as `create` rules|
|If table exists.| default|
|Multi-alter on one table will be merged.|merge_alter_table|
|If column exists.|defalut|
|table option only support change storage `engine/ comment/auto_increment value/charset`.|default|
|If enable change column|enable_change_column|
|If enable modify the order of columns|check_column_position_change|
|If enable modify column type.|check_column_type_change|


# Tips
* SQL review base on mysql 5.7, if any questions, please help us to fix by issue commit.
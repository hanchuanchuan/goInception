# Option List


All goInception audit rules can check with ```inception show variables;```.

```sql
inception show variables;
```

Modify in two ways:

- one at ```inception set ```
```sql
inception set osc_check_interval = 10;
```

- another one at config.toml, and start by ```-config=config.toml```



![variables List](./images/variables.png)


|Option|Default|Value|Description|
|:----|:----|:----|:----|
|check_autoincrement_datatype|FALSE|true,false|If return error when auto_increment column is not int or bigint.|
|check_autoincrement_init_value|FALSE|true,false|If return error when auto_increment column not start from 1.|
|check_autoincrement_name|FALSE|true,false|If return warn when the name of the auto_increment column is not ID, that means the column is meaningful.|
|check_column_comment|FALSE|true,false|If the new column has a comment.|
|check_column_default_value|FALSE|true,false|If the new column, no matter create or alter, has a default value.|
|check_column_position_change `v0.9`|FALSE|true,false|Check if column position changed|
|check_column_type_change `v0.7.3`|TRUE|true,false|Check the change of column type|
|check_dml_limit|FALSE|true,false|If return error when LIMIT used in DML.|
|check_dml_orderby|FALSE|true,false|If return error when Order By used in DML.|
|check_dml_where|FALSE|true,false|If return error when there is no WHERE in DML.|
|check_float_double `v1.0.2`|FALSE|true,false|If turned on, the type of `float/double` will change to decimal auto.|
|check_identifier|FALSE|true,false|Check if Identifier is correct.Rule: [a-z,A-Z,0-9,_]|
|check_identifier_upper `v1.0.2`|FALSE|true,false|If the Identifier, such as table name, column name,index name, must be uppercase. Default false|
|check_implicit_type_conversion `v1.1.3`|FALSE|true,false|If have implicit type conversion at WHERE. Default false|
|check_index_prefix|FALSE|true,false|If check index prefix, setting by `index_prefix` and `uniq_index_prefix`|
|check_insert_field|FALSE|true,false|If check the table and column exist.|
|check_primary_key|FALSE|true,false|If return error when there is no primary key at create table.|
|check_table_comment|FALSE|true,false|If return error when there is no comment at create table.|
|check_timestamp_count `v0.6.0`|FALSE|true,false|If check how many current_timestamp columns.|
|check_timestamp_default|FALSE|true,false|If return error when timestamp column has no default value.|
|columns_must_have_index `v1.2.2`| |string|Set the column must be create index. Split by comma. Format: column name [type, option]|
|default_charset `v1.0.5`|utf8mb4|string|The default connection charset. Default utf8mb4.|
|enable_autoincrement_unsigned|FALSE|true,false|If the auto_increment column should be unsigned.|
|enable_any_statement `v1.2.5`|FALSE|true,false|If all SQL approved.|
|enable_blob_not_null `v1.0`|FALSE|true,false|If set the default value of `blob/text/json` not null are approved, default is false, means not allowed.|
|enable_blob_type|FALSE|true,false|If check support of BLOB column, include create,alter etc.|
|enable_change_column `v1.0.3`|TRUE|true,false|If support change column syntax, default true.|
|enable_column_charset|FALSE|true,false|If allow to set charset in SQL|
|enable_drop_database|FALSE|true,false|If allow to drop database.|
|enable_drop_table|FALSE|true,false|If allow to drop table.|
|enable_enum_set_bit|FALSE|true,false|If can use enum,set,bit|
|enable_fingerprint `v0.6.2`|FALSE|true,false|SQL fingerprint.|
|explain_rule `v1.1.1`|`first`|`first`, `max`|The rule which explain decide the effect of SQL. `first`: use affect rows at the first row of explain shows as the SQL affect rows. `max`: use the max affect rows of explain as the whole explain affect rows.|
|enable_foreign_key|FALSE|true,false|If can use foreign key.|
|enable_identifer_keyword|FALSE|true,false|If use MySQL key words in SQL. default warn.|
|enable_json_type `v0.7.2`|FALSE|true,false|If can use Json type include create, alter etc.|
|enable_minimal_rollback `v1.1.2`|FALSE|true,false|If turn on the min rollback SQL, if on the rollback of update only record the change rows. Default false.|
|enable_nullable|TRUE|true,false|If allow NULL for new column.|
|enable_null_index_name v0.7.1|FALSE|true,false|If allow NULL index name for new index.|
|enable_orderby_rand|FALSE|true,false|If return error, when SQL within order by rand.|
|enable_partition_table|FALSE|true,false|If use partition table.|
|enable_pk_columns_only_int|FALSE|true,false|If the primary key must be int.|
|enable_select_star|FALSE|true,false|If return error, when use Select*|
|enable_set_charset|FALSE|true,false|If enable setting charset|
|enable_set_collation `v0.7`|FALSE|true,false|If enable setting collation|
|enable_set_engine `v1.0-rc4`|TRUE|true,false|If enable setting engine, default true.|
|enable_sql_statistic `v0.9`|FALSE|true,false|Turn on statistic|
|enable_timestamp_type `v1.0.1`|TRUE|true,false|If enable timestamp column, include create and alter, default true.|
|enable_use_view `v1.2.4`|FALSE|true,false|If enable create and use View|
|enable_zero_date `v1.0.1`|TRUE|true,false|If enable time is 0, when turn off return error. Default true, means turn on, followed `NO_ZERO_DATE` in `sql_mode` setting.|
|general_log `v0.8.1`|FALSE|true,false|If record full log|
|hex_blob `v1.1.4`|FALSE|true,false|When decode binlog, if binary type can be saved as string type, then saved as hexadecimal string type. Affect `binary`, `varbinary`, `blob`. Default turn off.|
|ignore_osc_alter_stmt `v1.2.4`| |string|If ignore alter in osc. Format: drop index, add column, etc. divided b comma.|
|lang v0.5.1|`en-US`|`en-US`,`zh-CN`|Return message charset, option: `en-US`,`zh-CN`|
|lock_wait_timeout `v1.2.4`|-1|int|How much seconds Lock wait when session execute SQL.|
|max_allowed_packet `v1.0-rc3`|4194304|int|Max data package size, default 4194304 byte = 4MB|
|max_char_length|0|int|Max length of char, if exceed, `warning to exchange to varchar`.|
|max_ddl_affect_rows `v1.0.2`|0|int|Warning when the DDL affects rows more than the setting value, if setting `0`, no limit.|
|max_insert_rows `v0.6.3`|0|int|The max rows than can be insert in one INSERT values. `0` means no limit.|
|max_key_parts|3|int|The max columns can be contained in one index.|
|max_keys|3|int|The max number of index can be contained in one table.|
|max_primary_key_parts|3|int|The max columns can be contained in the primary key.|
|max_update_rows|5000|int|Warning, when update/delete estimate the effect rows more the setting value.|
|merge_alter_table|FALSE|true,false|If merge the alter SQL on the same table and warning.|
|must_have_columns `v0.6.3`| |string|Setting the columns which must be contained in a new create table. Split by comma,format: column_name `[column_type,option]`|
|skip_sqls `v1.0-rc3`| |string|Setting which sql can ignore review for compatible client.|
|sql_mode `v1.2.4`| |string|Connection sql_mode.|
|sql_safe_updates|-1|-1,0,1|Safe update. -1 means do nothing and followe the remote database. 0 means turn off. 1 means turn on.|
|support_charset|utf8,utf8mb4|string|Charset support, split by comma.|
|support_collation `v0.7`| |string|Collating support, split by comma.|
|support_engine `v1.0-rc4`|innodb|string|Engines support, default innodb, split by comma.|
|index_prefix `v1.2.0`|idx_|string|Index prefix default idx_ , related with the option of check_index_prefix. NULL means no limit.|
|uniq_index_prefix `v1.2.0`|uniq_|string|Unique index prefix, default uniq_, related with check_index_prefix. NULL means no limit.|
|table_prefix `v1.2.0`| |string|The prefix of table name. NULL means no limit.|
|wait_timeout `v1.1.2`|0|int|The wait timeout of remote database. Default 0 seconds. Means use the database setting.|

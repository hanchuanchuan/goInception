

### pt-online-schema-change

- Please download and install `percona-toolkit` (`v3.0.4` or other campatible version) first.
- Please specific pt-osc dir by `osc_bin_dir` first，default `/usr/local/bin`.

#### Options

you can check `pt-osc` options by ```inception show variables like 'osc%';```

```sql
inception show variables like 'osc%';
```
Modify in two ways:
- one at ```inception set ```
```sql
inception set osc_check_interval = 10;
```

- another one at config.toml, and start by ```-config=config.toml```


#### process command


##### check osc process

<!-- sqlsha1 -->
```sql
inception get osc processlist;
```

result return:

DBNAME   | TABLENAME | COMMAND | SQLSHA1  | PERCENT | REMAINTIME | INFOMATION
----------|---------|--------------------|-----------------|-----------------|-------------------|-----
test_inc | t1 | alter table t1 add column c33 int | *E53542EFF4E179BE267210114EC5EDBEF9DC5D8F |       9 | 00:36      | Copying `test_inc`.`t1`:   9% 00:36 remain



##### check specific osc process
```sql
inception get osc_percent '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```

result return:

DBNAME   | TABLENAME | SQLSHA1                                   | PERCENT | REMAINTIME | INFOMATION
----------|---------|--------------------|-----------------|-----------------|-----
test_inc | t1        | *E53542EFF4E179BE267210114EC5EDBEF9DC5D8F |      49 | 00:14      | Copying `test_inc`.`t1`:  49% 00:14 remain


##### stop specific osc process

`need to remove auxiliary tables manual`
```sql
inception kill osc '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
-- or --
inception stop alter '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```


### Option description

Options  |  default  |  type | description
------------ | ------------- | ------------ | ------------
osc_on                                 | false          | bool | OSC switch
osc_alter_foreign_keys_method          | none           | string | correspond OSC option `alter-foreign-keys-method`
osc_bin_dir                            | /usr/local/bin | string | pt-online-schema-change script path
osc_check_alter                        | true           | bool | correspond `--[no]check-alter`
osc_check_interval                     | 5              | int | correspond `--check-interval`, means Sleep time between checks for `--max-lag`.
osc_check_replication_filters          | true           | bool | correspond `--[no]check-replication-filters`
osc_chunk_size                         | 1000           | int | correspond `--chunk-size`
osc_chunk_size_limit                   | 4              | int | correspond `--chunk-size-limit`
osc_chunk_time                         | 1              | int | correspond `--chunk-time`
osc_check_unique_key_change `v1.0.5` | true              | bool | correspond `--[no]check_unique_key_change`, if check unique index.
osc_critical_thread_connected                 | 1000           | int | correspond `--critical-load中的thread_connected`
osc_critical_thread_running                   | 80             | int | correspond `--critical-load中的thread_running`
osc_drop_new_table                     | true           | bool | correspond `--[no]drop-new-table`
osc_drop_old_table                     | true           | bool | correspond `--[no]drop-old-table`
osc_max_lag                            | 3              | int | correspond `--max-lag`
osc_max_thread_connected                      | 1000           | int | correspond `--max-load中的thread_connected`
osc_max_thread_running                        | 80             | int | correspond `--max-load中的thread_running`
osc_min_table_size                     | 16             | int | Switch set，if value is 0, all ALTER will use OSC.if values is not 0, only the table size more than the values, use OSC. unit in MB，table size = `select (DATA_LENGTH + INDEX_LENGTH)/1024/1024 from information_schema.tables where table_schema = "dbname" and table_name = "tablename"`
osc_print_none                         | false          | bool | If value is 1, do not print. If values is 0, print osc output in inception result error sets.
osc_print_sql                          | false          | bool | correspond `--print`
osc_recursion_method                   | processlist    | string | correspond `-ecursion_method`


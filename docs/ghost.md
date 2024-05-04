

### gh-ost

- gh-ost is built-in GoInception. No additional installation. About Stop, Pause, Recover are contained in GoInception command.
- Support calling gh-ost in binary mode, the parameter switch is `ghost_bin_dir`, please refer to the `osc_bin_dir` parameter of pt-osc for usage, if this parameter is not specified, it will still be called with the built-in gh-ost.

#### Options

gh-ost tool Option check in inception show variables like 'ghost%'`;

```sql
inception show variables like 'ghost%';
```

Modify in two ways:

- one at ```inception set ```
```sql
inception set osc_check_interval = 10;
```

- another one at config.toml, and start by ```-config=config.toml```


#### Process command


##### check osc process

<!-- sqlsha1 -->
```sql
inception get osc processlist;
```

results return:

DBNAME   | TABLENAME | COMMAND | SQLSHA1  | PERCENT | REMAINTIME | INFOMATION
----------|---------|--------------------|-----------------|-----------------|-------------------|-----
test_inc | t1 | alter table t1 add column c33 int | *E53542EFF4E179BE267210114EC5EDBEF9DC5D8F |       9 | 00:36      | Copying `test_inc`.`t1`:   9% 00:36 remain



##### check specific osc process

<!-- sqlsha1 -->
```sql
inception get osc_percent '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```

results return:

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

##### pause specific osc process

```sql
inception pause osc '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
-- or --
inception pause alter '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```

##### recover specific osc process

```sql
inception resume osc '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
-- or --
inception resume alter '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```


### reuse pt-osc options

Options  |  default  |  type | description
------------ | ------------- | ------------ | ------------
osc_critical_thread_connected                 | 1000           | int | correspond to `--critical-load中的thread_connected`
osc_critical_thread_running                   | 80             | int | correspond to `--critical-load中的thread_running`
osc_max_thread_connected                      | 1000           | int | correspond to `--max-load中的thread_connected`
osc_max_thread_running                        | 80             | int | correspond `--max-load中的thread_running`
osc_min_table_size                     | 16             | int | Switch set，if value is 0, all ALTER will use OSC.if values is not 0, only the table size more than the values, use OSC. unit in MB，table size = `select (DATA_LENGTH + INDEX_LENGTH)/1024/1024 from information_schema.tables where table_schema = "dbname" and table_name = "tablename"`
osc_print_none                         | false          | bool | If value is 1, do not print. If values is 0, print osc output in inception result error sets.


### Option descriptions

Option |  default  |  type | description
------------ | ------------- | ------------ | ------------
ghost_on                               | false  | bool | gh-ost swtich
ghost_bin_dir `v1.2.5`                 | '' | string | gh-ost binary path
ghost_aliyun_rds                       | false  | bool | Ali rds database label
ghost_allow_master_master              | false  | bool | If gh-ost can run in Dual master structure, with option `-assume-master-host`
|ghost_allow_nullable_unique_key|FALSE|bool|If gh-ost can depend on Unique key in NULL when migrate.Default unique key can not be NULL.If yes, data will inconsistency in migration.|
|ghost_allow_on_master|TRUE|bool|If gh-ost can run on Master.Default on Master|
|ghost_approve_renamed_columns|TRUE|bool|If support to change column name.Default false.|
|ghost_assume_master_host| |string|Gh-ost Maser db addr (hostname:port or ip:port).Default gh-ost connect to Slave.|
|ghost_assume_rbr|TRUE|bool|When the server that gh-ost connect binlog_format = ROW, can use -assume-rbr to probit run `stop/start slave ` on Slave and the user of gh-ost don't need super privileges. `Warning: avoid any effect to live db, set true.`|
|ghost_chunk_size|1000|int|The row number in each loop.Default 1000. Range in [100, 100000]|
|ghost_concurrent_rowcount|TRUE|bool|If value is true(default), estimate table rows by `explain select count(*)` after row-copy，and reject the time of ETA. If value is false, gh-ost will estimate table rows first and then row-copy.|
|ghost_critical_load_hibernate_seconds|0|int|Reach `critical-load`, in sleep.|
|ghost_critical_load_interval_millis|0|int|if the value is 0，exit immediately when reach the `-critical-load`. If the value is not 0, wait `-critical-load-interval-millis` when reach `-critical-load` and check again, if reach again, exit.|
|ghost_cut_over|atomic|string|The type of cut-over: `atomic(default)` use github algorithm. `two-step` use facebook-OSC algorithm|
|ghost_cut_over_exponential_backoff|FALSE|bool|Wait exponentially longer intervals between failed cut-over attempts. Wait intervals obey a maximum configurable with 'exponential-backoff-max-interval').|
|ghost_cut_over_lock_timeout_seconds|3|int|The lock wait timeout when cut-over stage, when timeout, will retry. Default 3.|
|ghost_default_retries|60|int|Retry times before panic at many actions. Default 60.|
|ghost_discard_foreign_keys|FALSE|bool|For foreign key table, gh-ost create will not contain the foreign key. Only use for remove the foreign key.|
|ghost_dml_batch_size|10|int|How many DML in one transaction. Range in [1,100]. Default 10.|
|ghost_exact_rowcount|FALSE|bool|Exactly estimate table rows by `select count(*)`.|
|ghost_exponential_backoff_max_interval|64|int|Maximum number of seconds to wait between attempts when performing various operations with exponential backoff. (default 64)|
|ghost_force_named_cut_over|FALSE|bool|When true, the ‘unpostpone | cut-over’ interactive command must name the migrated table。|
|ghost_force_table_names| |string|table name prefix to be used on the temporary tables|
|ghost_gcp|FALSE|bool|Google Cloud support.|
|ghost_heartbeat_interval_millis|500|int|Gh-ost heartbeat values, default 500ms.|
|ghost_initially_drop_ghost_table|FALSE|bool|If check and delete a existing ghost table. `Warning: do not use it in live env, maybe delete the ghost table in a running gh-ost. Manual check and delete.`|
|ghost_initially_drop_old_table|FALSE|bool|If check and delete old-table when start gh-ost. `Warning: do not use it in live env, maybe delete the table in a running gh-ost. Manual check and delete.`|
|ghost_initially_drop_socket_file|FALSE|bool|Gh-ost will force delete the existing socket file. `Warning: do not use it in live env, maybe delete the socket file of a running gh-ost.`|
|ghost_max_lag_millis|1500|int|When slave latency more the the value, gh-ost will throttling. Default 1500ms.|
|ghost_nice_ratio|0|float|Sleep time at each chunk time. Range in [0.0...100.0]. E.g: 0: no sleep; 1: bewteen each row-copy 1ms, sleep 1ms; 0.7: bewteen row-copy 10ms, sleep 7ms.|
|ghost_ok_to_drop_table|TRUE|bool|If drop old-table, when gh-ost finished. Default true|
|ghost_postpone_cut_over_flag_file| |string|Cut-over stage will be delayed when this file exists until this file is deleted.|
|ghost_replication_lag_query| |string|Check latency by `show slave status ` Seconds_behind_master. If use pt-heartbeat, sql = `SELECT ROUND(UNIX_TIMESTAMP() - MAX(UNIX_TIMESTAMP(ts))) AS delay FROM my_schema.heartbeat`|
|ghost_skip_foreign_key_checks|TRUE|bool|Skip foreign key check, default true.|
|ghost_throttle_additional_flag_file| |string|Gh-ost will stop immediately when this file is created.Used to stop all gh-ost executions. Delete this file, recover all gh-ost executions.|
|ghost_throttle_control_replicas| |string|List all slaves which need to check why latency.|
|ghost_throttle_flag_file| |string|Gh-ost will stop immediately when this file is created. Used for control single gh-ost execution. `--throttle-additional-flag-file` string used for muil-execution.|
|ghost_throttle_http| |string|The `--throttle-http` flag allows for throttling via HTTP. Every 100ms gh-ost issues a HEAD request to the provided URL. If the response status code is not 200 throttling will kick in until a 200 response status code is returned.|
|ghost_throttle_query| |string|Flow check at every minute. When the value is `0`, don't need throttling. When the value `>0`, need throttling. This query will run on a migrated server, please keep it lightly.|
|ghost_timestamp_old_table|FALSE|bool|Use timestamp in old-tablename, keep unique name|
|ghost_tungsten|FALSE|bool|Tell gh-ost running in a tungsten-replication structure|


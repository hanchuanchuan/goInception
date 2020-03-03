

### gh-ost

- 内置gh-ost源码(`v1.0.48`)，因此无须下载。
- 手动终止和暂停及恢复功能已开放相应命令，因此隐藏相关参数。

####参数设置

gh-ost工具的设置参数可以可以通过```inception show variables like 'ghost%';```查看

```sql
inception show variables like 'ghost%';
```

支持以下方式设置:

- 1.通过```inception set ```设置

```sql
inception set osc_check_interval = 10;
```

- 2.配置config.toml,并通过```-config=config.toml```指定配置文件启动


#### 进程命令


#####查看osc进程

<!-- sqlsha1 -->
```sql
inception get osc processlist;
```

返回结果：

DBNAME   | TABLENAME | COMMAND | SQLSHA1  | PERCENT | REMAINTIME | INFOMATION
----------|---------|--------------------|-----------------|-----------------|-------------------|-----
test_inc | t1 | alter table t1 add column c33 int | *E53542EFF4E179BE267210114EC5EDBEF9DC5D8F |       9 | 00:36      | Copying `test_inc`.`t1`:   9% 00:36 remain



#####查看指定osc进程
```sql
inception get osc_percent '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```

返回结果：

DBNAME   | TABLENAME | SQLSHA1                                   | PERCENT | REMAINTIME | INFOMATION
----------|---------|--------------------|-----------------|-----------------|-----
test_inc | t1        | *E53542EFF4E179BE267210114EC5EDBEF9DC5D8F |      49 | 00:14      | Copying `test_inc`.`t1`:  49% 00:14 remain


#####终止指定osc进程
`终止后注意手动清理相关辅助表`
```sql
inception kill osc '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
-- 或同义词
inception stop alter '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```

#####暂停指定osc进程
```sql
inception pause osc '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
-- 或同义词
inception pause alter '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```

#####恢复指定osc进程
```sql
inception resume osc '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
-- 或同义词
inception resume alter '*E53542EFF4E179BE267210114EC5EDBEF9DC5D8F';
```


### 复用pt-osc参数

参数  |  默认值  |  可选范围 | 说明
------------ | ------------- | ------------ | ------------
osc_critical_thread_connected                 | 1000           | int | 对应参数--critical-load中的thread_connected部分
osc_critical_thread_running                   | 80             | int | 对应参数--critical-load中的thread_running部分
osc_max_thread_connected                      | 1000           | int | 对应参数--max-load中的thread_connected部分
osc_max_thread_running                        | 80             | int | 对应参数--max-load中的thread_running部分
osc_min_table_size                     | 16             | int | OSC的开关，如果设置为0，则全部ALTER语句都走OSC，如果设置为非0，则当这个表占用空间大小大于这个值时才使用OSC方式。单位为M，这个表大小的计算方式是通过语句： select (DATA_LENGTH + INDEX_LENGTH)/1024/1024 from information_schema.tables where table_schema = "dbname" and table_name = "tablename"来实现的。
osc_print_none                         | false          | bool | 用来设置在Inception返回结果集中，对于原来OSC在执行过程的标准输出信息是不是要打印到结果集对应的错误信息列中，如果设置为1，就不打印，如果设置为0，就打印。而如果出现错误了，则都会打印


### 参数说明

参数  |  默认值  |  可选范围 | 说明
------------ | ------------- | ------------ | ------------
ghost_on                               | false  | bool | gh-ost开关
ghost_aliyun_rds                       | false  | bool | 阿里云rds数据库标志
ghost_allow_master_master              | false  | bool | 允许gh-ost运行在双主复制架构中，一般与-assume-master-host参数一起使用
ghost_allow_nullable_unique_key        | false  | bool | 允许gh-ost在数据迁移(migrate)依赖的唯一键可以为NULL，默认为不允许为NULL的唯一键。如果数据迁移(migrate)依赖的唯一键允许NULL值，则可能造成数据不正确，请谨慎使用。
ghost_allow_on_master                  | `true`   | bool | 允许gh-ost直接运行在主库上。默认gh-ost连接的`主库`。`(暂未添加从库地址的配置)`
ghost_approve_renamed_columns          | true   | bool | 如果支持修改列名,则需设置此参数为`true`,否则gh-ost不会执行。
ghost_assume_master_host               |        | string | 为gh-ost指定一个主库，格式为"ip:port"或者"hostname:port"。默认推荐gh-ost连接从库。
ghost_assume_rbr                       | true   | bool | 确认gh-ost连接的数据库实例的binlog_format=ROW的情况下，可以指定-assume-rbr，这样可以禁止从库上运行stop slave,start slave,执行gh-ost用户也不需要SUPER权限。`为避免影响生产数据库，此参数建议置为true`
ghost_chunk_size                       | 1000   | int | 在每次迭代中处理的行数量(允许范围：100-100000)，默认值为1000。
ghost_concurrent_rowcount              | true   | bool | 该参数如果为True(默认值)，则进行row-copy之后，估算统计行数(使用explain select count(*)方式)，并调整ETA时间，否则，gh-ost首先预估统计行数，然后开始row-copy。
ghost_critical_load_hibernate_seconds  | 0      | int | 负载达到critical-load时，gh-ost在指定的时间内进入休眠状态。 它不会读/写任何来自任何服务器的任何内容。
ghost_critical_load_interval_millis    | 0      | int | 当值为0时，当达到-critical-load，gh-ost立即退出。当值不为0时，当达到-critical-load，gh-ost会在-critical-load-interval-millis秒数后，再次进行检查，再次检查依旧达到-critical-load，gh-ost将会退出。
ghost_cut_over                         | atomic | string |  选择cut-over类型:atomic/two-step，atomic(默认)类型的cut-over是github的算法，two-step采用的是facebook-OSC的算法。
ghost_cut_over_exponential_backoff     | false  | bool | Wait exponentially longer intervals between failed cut-over attempts. Wait intervals obey a maximum configurable with 'exponential-backoff-max-interval').
ghost_cut_over_lock_timeout_seconds    | 3      | int | gh-ost在cut-over阶段最大的锁等待时间，当锁超时时，gh-ost的cut-over将重试。(默认值：3)
ghost_default_retries                  | 60     | int | 各种操作在panick前重试次数。(默认为60)
ghost_discard_foreign_keys             | false  | bool | 该参数针对一个有外键的表，在gh-ost创建ghost表时，并不会为ghost表创建外键。该参数很适合用于删除外键，除此之外，请谨慎使用。
ghost_dml_batch_size                   | 10     | int | 在单个事务中应用DML事件的批量大小（范围1-100）（默认值为10）
ghost_exact_rowcount                   | false  | bool | 准确统计表行数(使用select count(*)的方式)，得到更准确的预估时间。
ghost_exponential_backoff_max_interval | 64     | int |  Maximum number of seconds to wait between attempts when performing various operations with exponential backoff. (default 64)
ghost_force_named_cut_over             | false  | bool | When true, the ‘unpostpone|cut-over’ interactive command must name the migrated table。
ghost_force_table_names                |        | string | table name prefix to be used on the temporary tables
ghost_gcp                              | false  | bool | google云平台支持
ghost_heartbeat_interval_millis        | 500    | int | gh-ost心跳频率值，默认为500ms。
ghost_initially_drop_ghost_table       | false  | bool | gh-ost操作之前，检查并删除已经存在的ghost表。该参数不建议使用，请手动处理原来存在的ghost表。
ghost_initially_drop_old_table         | false  | bool | gh-ost操作之前，检查并删除已经存在的旧表。该参数不建议使用，请手动处理原来存在的ghost表。
ghost_initially_drop_socket_file       | false  | bool | gh-ost强制删除已经存在的socket文件。该参数不建议使用，可能会删除一个正在运行的gh-ost程序，导致DDL失败。
ghost_max_lag_millis                   | 1500   | int | 主从复制最大延迟时间，当主从复制延迟时间超过该值后，gh-ost将采取节流(throttle)措施，默认值：1500ms。
ghost_nice_ratio                       | 0      | float | 每次chunk时间段的休眠时间，范围[0.0...100.0]。e.g:0：每个chunk时间段不休眠，即一个chunk接着一个chunk执行；1：每row-copy 1毫秒，则另外休眠1毫秒；0.7：每row-copy 10毫秒，则另外休眠7毫秒。
ghost_ok_to_drop_table                 | true   | bool | gh-ost操作结束后，删除旧表，默认状态是`删除旧表`。
ghost_postpone_cut_over_flag_file      |        | string | 当这个文件存在的时候，gh-ost的cut-over阶段将会被推迟，直到该文件被删除。
ghost_replication_lag_query            |        | string | 检查主从复制延迟的SQL语句，默认gh-ost通过show slave status获取Seconds_behind_master作为主从延迟时间依据。如果使用pt-heartbeat工具，检查主从复制延迟的SQL语句类似于:`SELECT ROUND(UNIX_TIMESTAMP() - MAX(UNIX_TIMESTAMP(ts))) AS delay FROM my_schema.heartbeat`;
ghost_skip_foreign_key_checks          | true  | bool | 跳过外键检查,默认为`true`
ghost_throttle_additional_flag_file    |        | string | 当该文件被创建后，gh-ost操作立即停止。该参数可以用在多个gh-ost同时操作的时候，创建一个文件，让所有的gh-ost操作停止，或者删除这个文件，让所有的gh-ost操作恢复。
ghost_throttle_control_replicas        |        | string | 列出所有需要被检查主从复制延迟的从库。
ghost_throttle_flag_file               |        | string | 当该文件被创建后，gh-ost操作立即停止。该参数适合控制单个gh-ost操作。-throttle-additional-flag-file string适合控制多个gh-ost操作。
ghost_throttle_http                    |        | string | The --throttle-http flag allows for throttling via HTTP. Every 100ms gh-ost issues a HEAD request to the provided URL. If the response status code is not 200 throttling will kick in until a 200 response status code is returned.
ghost_throttle_query                   |        | string | 节流查询。每秒钟执行一次。当返回值=0时不需要节流，当返回值>0时，需要执行节流操作。该查询会在数据迁移(migrated)服务器上操作，所以请确保该查询是轻量级的。
ghost_timestamp_old_table              | false  | bool | 在旧表名中使用时间戳。 这会使旧表名称具有唯一且无冲突的交叉迁移
ghost_tungsten                         | false  | bool | 告诉gh-ost你正在运行的是一个tungsten-replication拓扑结构。

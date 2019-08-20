

### pt-online-schema-change

- 使用前需要手动下载安装`percona-toolkit` (`v3.0.4`或参数兼容版本)
- 使用前需要指定pt-osc目录参数`osc_bin_dir`，默认为`/usr/local/bin`

####参数设置

pt-osc工具的设置参数可以可以通过```inception show variables like 'osc%';```查看

```sql
inception show variables like 'osc%';
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


### 参数说明

参数  |  默认值  |  可选范围 | 说明
------------ | ------------- | ------------ | ------------
osc_on                                 | false          | bool | OSC开关
osc_alter_foreign_keys_method          | none           | string | 对应OSC参数alter-foreign-keys-method
osc_bin_dir                            | /usr/local/bin | string | pt-online-schema-change脚本的位置
osc_check_alter                        | true           | bool | 对应参数--[no]check-alter
osc_check_interval                     | 5              | int | 对应参数--check-interval，意义是Sleep time between checks for --max-lag.
osc_check_replication_filters          | true           | bool | 对应参数--[no]check-replication-filters
osc_chunk_size                         | 1000           | int | 对应参数--chunk-size
osc_chunk_size_limit                   | 4              | int | 对应参数--chunk-size-limit
osc_chunk_time                         | 1              | int | 对应参数--chunk-time
osc_check_unique_key_change `v1.0.5` | true              | bool | 对应参数--[no]check_unique_key_change,设置是否检查唯一索引
osc_critical_thread_connected                 | 1000           | int | 对应参数--critical-load中的thread_connected部分
osc_critical_thread_running                   | 80             | int | 对应参数--critical-load中的thread_running部分
osc_drop_new_table                     | true           | bool | 对应参数--[no]drop-new-table
osc_drop_old_table                     | true           | bool | 对应参数--[no]drop-old-table
osc_max_lag                            | 3              | int | 对应参数--max-lag
osc_max_thread_connected                      | 1000           | int | 对应参数--max-load中的thread_connected部分
osc_max_thread_running                        | 80             | int | 对应参数--max-load中的thread_running部分
osc_min_table_size                     | 16             | int | OSC的开关，如果设置为0，则全部ALTER语句都走OSC，如果设置为非0，则当这个表占用空间大小大于这个值时才使用OSC方式。单位为M，这个表大小的计算方式是通过语句： select (DATA_LENGTH + INDEX_LENGTH)/1024/1024 from information_schema.tables where table_schema = "dbname" and table_name = "tablename"来实现的。
osc_print_none                         | false          | bool | 用来设置在Inception返回结果集中，对于原来OSC在执行过程的标准输出信息是不是要打印到结果集对应的错误信息列中，如果设置为1，就不打印，如果设置为0，就打印。而如果出现错误了，则都会打印
osc_print_sql                          | false          | bool | 对应参数--print
osc_recursion_method                   | processlist    | string | 对应参数recursion_method


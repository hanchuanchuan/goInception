# goInception 更新日志


## [v1.0-rc2] - 2019-6-21

### Fix
* 优化回滚相关表结构,字符集调整为utf8mb4 (`历史表结构需要手动调整`)

### Update
* 优化审核规则,审核子查询、函数等各种表达式 (#44)
* 优化gh-ost默认生成的socket文件名格式
* 优化日志输出,添加线程号显示
* binlog解析时添加mariadb判断


## [v1.0-rc1] - 2019-6-12

### New Features
* 添加split分隔功能 (#42)


## [v0.9-beta] - 2019-6-4

### New Features
* 添加统计功能,可通过参数 `enable_sql_statistic` 启用 (#38)
* 添加参数 `check_column_position_change` ,可控制是否检查列位置/顺序变更 (#40, #41)

### Update
* 优化使用阿里云RDS和gh-ost时的逻辑,自动设置 `assume-master-host` 参数 (#39)


## [v0.8.3-beta] - 2019-5-30

### Fix
* 修复gh-ost的initially-drop-old-table和initially-drop-ghost-table参数支持
* 修复设置osc_min_table_size大于0后无法正常启用osc的bug

### Update
* 兼容语法inception get processlist
* docker镜像内置pt-osc包(版本3.0.13)


## [v0.8.2-beta] - 2019-5-27

### Fix
* fix: 修复binlog解析时对unsigned列溢出值的处理
* fix: 修复gh-ost执行语句有反引号时报语法错误的bug (#33)
* fix: 修复kill DDL操作时,返回执行和备份成功的bug,现在会提示执行结果未知了 (#34)


## [v0.8.1-beta] - 2019-5-24

### Fix
* 修复新建表后,使用大小写不一致的表名时返回表不存在bug

### New Features
* 添加general_log参数,用以记录全量日志

### Update
* 优化insert select新表的审核规则,现在select新表时也可以审核了


## [v0.8-beta] - 2019-5-22

### Fix
* 修复当开启sql指纹功能时,可能出现把警告误标记为错误的bug

### Update
* 优化子查询审核规则,递归审核所有子查询
* 审核group by语法和聚合函数


## [v0.7.5-beta] - 2019-5-17

### Fix
* 修复执行阶段kill逻辑,避免kill后备份也中止

### New Features
* 添加select语法支持
* 添加alter table的ALGORITHM,LOCK,FORCE语法支持

### Update
* 优化update子查询审核


## [v0.7.4-beta] - 2019-5-12

### New Features
* 添加alter table表选项语法支持 (#30)
* 重新设计kill操作支持,支持远端数据库kill和goInception kill命令 (#10)


## [v0.7.3-beta] - 2019-5-10

### Fix
* 修复在开启备份时,执行错误时偶尔出现的误标记执行/备份成功bug

### New Features
* 添加`check_column_type_change`参数，设置是否开启字段类型变更审核,默认`开启` (#27)

### Update
* 实现insert select * 列数审核


## [v0.7.2-beta] - 2019-5-7

### New Features
* 添加`enable_json_type`参数，设置是否允许json类型字段 (#26)

### Update
* 实现基于系统变量explicit_defaults_for_timestamp的审核规则
* 优化osc解析,转义密码和alter语句中的特殊字符


## [v0.7.1-beta] - 2019-5-4

### Update
* 优化json类型字段处理逻辑，不再检查其默认值和NOT NULL约束 (#7, #22)
* 优化must_have_columns参数值解析
* 优化insert select审核逻辑

### Fix
* 修复和完善add column(...)语法支持
* 修复开启osc时,alter语句有多余空格时执行失败的bug

### New Features
* 添加`enable_null_index_name`参数，允许不指定索引名 (#25)
* 添加语法树打印功能(beta) (#21)


## [v0.7-beta] - 2019-4-26

### Update
* 优化update关联新建表时的审核，现在update时可以关联新建表了
* 优化insert 新建 select语法审核，现在可以获取预估受影响行数了
* 审核阶段自动忽略警告，优化审核逻辑
* 优化`check_column_default_value`的审核逻辑，默认值审核时会跳过主键
* 备份阶段sql过长时会自动截断(比如insert values很多行)，`返回警告但不影响执行和备份操作`

### Fix
* 修复开启`enable_pk_columns_only_int`选项时列类型审核错误的问题

### New Features
* 添加`enable_set_collation`参数，设置是否允许指定表和数据库的排序规则
* 添加`support_collation`参数，设置支持的排序规则,多个时以逗号分隔


## [v0.6.4-beta] - 2019-4-23

### Fix
* 修复mysql 5.6和mariadb无法获取受影响行数的问题


## [v0.6.3-beta] - 2019-4-22

### New Features
* 添加`max_insert_rows`参数，设置insert values允许的最大行数。
* 添加`must_have_columns`参数，用以指定建表时必须创建的列。多个列时以逗号分隔(`格式: 列名 [列类型,可选]`)


## [v0.6.2-beta] - 2019-4-18
### Update
* 添加不支持的语法警告(create table as和create table select)
* 实现alter多子句时的表结构变化支持,如drop column后跟add column

### Fix
* 修复explain返回null列时报错的问题
* 修复索引的唯一标识设置错误问题


## [v0.6.1-beta] - 2019-4-9
### Update
* 添加远端数据库断开重连机制，优化线程号和master status查询速度
* 优化远端数据库访问操作
* 优化sql内容解析，移除多余分号和空格

### Fix
* 修复跨库update时无法找到列的问题
* 修复osc子句有双引号时执行错误的问题

### New Features
* 添加sql指纹功能
dml语句相似时，可以根据相同的指纹ID复用explain结果，以减少远端数据库explain操作，并提高审核速度
	- 可以通过```inception set enable_fingerprint=1;```或配置文件开启全局配置
	- 也可以通过调用选项```--fingerprint=1;```开启单个配置
	- 两种配置取并集，即开启任一配置，则启用sql指纹功能，默认关闭。


## [v0.6-beta] - 2019-4-3
### Update
* 备份操作性能优化,备份信息改为批量写入
* 添加备份库连接超时检查
* explain函数性能优化
* 优化部分函数未指定架构名时的默认处理
* 优化默认值检查,添加计算列支持 (#12, #13, #14)
* 优化时间格式和范围检查,根据数据库sql_mode校验零值日期
* 升级到go 1.12

### Fix
* 修复index name校验逻辑,其可与列名一致
* 修复timestamp默认值校验不准确的问题

### New Features
* 添加kill功能支持,在审核和执行时可以kill,备份阶段无法kill (#10)
* 添加`check_timestamp_count`参数,可配置是否检查current_timestamp数量 (#11, #15)


## [v0.5.3-beta] - 2019-3-25
### Update
* 变更列名时使用逻辑校验,避免explain update失败
* 添加union子句校验
* 添加表名别名重复性校验

### Fix
* 修复update set子句指定表别名时校验问题
* 修复自增列校验问题
* 修复default value为表达式时的校验问题


## [v0.5.2-beta] - 2019-3-17
### Update
* 优化主键NULL列审核规则(审核`DEFAULT NULL`)
* 优化索引总长度校验,根据列字符集判断字节数长度
* 优化DDL备份对默认值的处理


## [v0.5.1-beta] - 2019-3-14
### Update
* 优化option解析规则,密码兼容特殊字符
* 优化语法解析失败时返回的sql语句
* 添加中文的异常和警告信息
* 添加新的参数
	- lang 设置返回的异常信息语言,可选值 `en-US`,`zh-CN`,默认`en-US`
### Fix
* 修复mariadb备份警告信息重复的问题


## [v0.5-beta] - 2019-3-10
### Update
* 兼容mariadb v10版本的备份兼容(高并发时回滚语句可能有误，须注意检查)
* 更新pt-osc部分参数名，使其与inception保持一致
	- osc_critical_running -> osc_critical_thread_running
	- osc_critical_connected -> osc_critical_thread_connected
	- osc_max_running -> osc_max_thread_running
	- osc_max_connected -> osc_max_thread_connected
* 隐藏gh-osc部分未使用参数
* 添加是否允许删除数据库参数`enable_drop_database`
* 优化系统变量variables显示和设置
* 调整部分参数默认值
	- ghost_ok_to_drop_table `true`
	- ghost_skip_foreign_key_checks `true`
	- osc_chunk_size `1000`
### Fix
* 修复json列校验异常问题 (#7)

## [v0.4.1-beta] - 2019-3-6
### Update
* 兼容mariadb数据库(v5.5.60)
	- 添加mariadb的binlog解析支持(测试版本**v5.5.60**,v10版本由于binlog格式改变,暂无法解析thread_id)
	- 优化备份失败时的返回信息


## [v0.4-beta] - 2019-3-5
### New Features
* 添加gh-ost工具支持
	- 无需安装gh-ost,功能内置(v1.0.48)
	- 进程列表 ```inception get osc processlist```
	- 指定进程信息 ```inception get osc_percent 'sqlsha1'```
	- 进程终止 ```inception stop alter 'sqlsha1'``` (同义词```inception kill osc 'sqlsha1'```)
	- 进程暂停 ```inception pause alter 'sqlsha1'``` (同义词```inception pause osc 'sqlsha1'```)
	- 进程恢复 ```inception resume alter 'sqlsha1'``` (同义词```inception resume osc 'sqlsha1'```)
	- 兼容gh-ost参数 ```inception show variables like 'ghost%'```


## [v0.3-beta] - 2019-2-13
### New Features
* 添加pt-osc工具支持
	- ```inception get osc processlist``` 查看osc进程列表
	- ```inception get osc_percent 'sqlsha1'``` 查看指定的osc进程
	- ```inception stop alter 'sqlsha1'``` (同义词```inception kill osc 'sqlsha1'```)中止指定的osc进程


## [v0.2-beta] - 2019-1-31
### Optimizer
* 优化二进制构建方式，压缩安装包大小
* 移除vendor依赖，优化GO111MODULE使用方式

* 跳过权限校验，以避免登陆goInception失败
* 移除root身份启动校验，以避免windows无法启动
* 优化inception set变量时的类型校验


## [v0.1-beta] - 2019-1-25
#### goInception正式发布


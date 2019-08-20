
#### 备份功能说明

goInception自带备份功能，首先服务启动时配置config.toml(放在 `[inc]` 段)

参数  |  默认值  |  可选范围 | 说明
------------ | ------------- | ------------ | ------------
backup_host   |  ""    |   string     |   备份数据库IP地址
backup_port   |  0    |   int     |     备份数据库端口
backup_user   |  ""    |   string     |   备份数据库用户名
backup_password   |  ""    |   string    |   备份数据库密码

并且在执行sql时，添加```--backup=true```选项

```sql
/*--user=root;--password=root;--host=127.0.0.1;--port=3306;--execute=1;--backup=1;*/
inception_magic_start;
use test;
create table t1(id int primary key);
inception_magic_commit;
```


#### 备份功能写入规则

- 在备份服务器上，备份库的命名格式为：```IP_PORT_库名```，例如```127_0_0_1_3306_test```
- 在备份库上创建备份信息表```$_$Inception_backup_information$_$```，用来保存该库的执行信息和回滚语句信息

    | 字段名             | 类型         | 说明
    --------------------|--------------|------
    opid_time         | varchar(50)  | 执行操作ID,格式为```时间戳_线程号_执行序号```
    start_binlog_file | varchar(512) | 起始binlog文件
    start_binlog_pos  | int(11)      | 起始binlog位置
    end_binlog_file   | varchar(512) | 终止binlog文件
    end_binlog_pos    | int(11)      | 终止binlog位置
    sql_statement     | text         | 执行SQL
    host              | varchar(64)  | 执行IP地址
    dbname            | varchar(64)  | 执行库名
    tablename         | varchar(64)  | 执行表名
    port              | int(11)      | 执行端口
    time              | timestamp    | 执行时间
    type              | varchar(20)  | 操作类型

- 在备份库有和操作表相同的表名，其表结构统一为：

    字段名  |  类型  | 说明
    ------------ | ------------- | ------------
    id   |  bigint     |   自增主键
    rollback_statement   |  mediumtext    |  回滚语句
    opid_time   |  varchar(50)    | 关联执行操作ID

#### 备份功能详细步骤

1. 配置备份数据库，并在执行SQl时开启备份功能
2. 在执行SQL前记录binlog位置和线程号(逐条执行逐条记录)
3. 执行SQL
4. 在执行SQL后记录binlog位置和线程号
5. 开始备份，解析远程服务器binlog
6. 在备份服务器创建备份库
7. 创建备份信息表，写入执行信息和binlog位置信息
8. 创建备份表，
9. 逐步解析binlog，并生成回滚语句，写入备份表

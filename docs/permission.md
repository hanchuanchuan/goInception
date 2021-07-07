# 权限说明

不同功能及阶段需要不同的权限，下面会列出各项功能可能涉及的权限要求，如有遗漏之处欢迎提出和补充。

建议的权限为：
`GRANT ALL PRIVILEGES ON *.* TO ...`

或者

`GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, PROCESS, REFERENCES, INDEX, ALTER, SUPER, REPLICATION SLAVE, REPLICATION CLIENT, TRIGGER ON *.* TO ...`


## 审核功能

* `information_schema库` 元数据查询权限，表结构，索引信息，约束等
* `mysql库` use权限，没有查询，默认连接该库，可通过调用选项的`--db`参数修改
* `DML操作` 审核时会对DML语句做explain操作，该操作需要实际的相应DML权限。
* `REFERENCES` 仅外键需要

## 执行

* 实际的SQL执行权限


### 使用pt-osc

* `PROCESS` 权限，查看processlist信息
* `TRIGGER` 创建和删除触发器
* `SUPER` 或 `REPLICATION CLIENT` 有主从时,查看主从延迟

### 使用gh-ost

* `SUPER|REPLICATION CLIENT, REPLICATION SLAVE` binlog解析
* `ALTER`, `CREATE`, `DELETE`, `DROP`, `INDEX`, `INSERT`, `LOCK TABLES`, `SELECT`, `TRIGGER`, `UPDATE`


## 备份

### 远端数据库

* `SUPER权限`，用以binlog格式不为row时执行`set session binlog_format='row'`

* `SUPER|REPLICATION CLIENT, REPLICATION SLAVE` binlog解析

### 备份库



* `建议授予备份库实例的所有权限`

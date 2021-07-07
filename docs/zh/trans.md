
# 事务

## 介绍

事务功能实现了DML语句按批次提交，以提高大批量DML语句的执行效率，以及保证事务一致性（`仅同一批次中`）。

以下详细说明配置方式以及涉及的回滚差异等。

## 配置

在调用goInception时添加参数`--trans=?`，其中参数值为数字，
* 默认为0，即不开启事务(逐行提交)
* 当大于1时，会按该参数分批进行提交，如500，则会按500条DML提交一次


### 示例
```py

import pymysql

sql = '''/*--host=127.0.0.1;--port=3306;--user=test;--password=test;\
--execute=1;--backup=1;--ignore-warnings=1;--trans=100;*/
inception_magic_start;
use test_inc;

-- drop table if exists t1;
create table t1 (id int primary key,c1 int ,c2 varchar(100));

insert into t1 values(1,2,'ccc');
insert into t1 values(2,2,'ccc');
insert into t1 values(3,3,'ccc');
insert into t1 values(4,2,'ccc');
insert into t1 values(5,2,'ccc');

inception_magic_commit;'''

conn = pymysql.connect(host='127.0.0.1', user='', passwd='',
                       db='', port=4000, charset="utf8mb4")
cur = conn.cursor()
ret = cur.execute(sql)
result = cur.fetchall()
cur.close()
conn.close()

for row in result:
    print(row)
```

## 执行

* 未开启事务前为逐行提交
* 开启事务后，按设置条数提交。如设为500，则会500条DML提交一次
* DDL执行无差异
* 当事务提交失败时，会`回滚该批次的SQL`，并立即中止(已执行SQL仍会生成回滚语句，以便有需要时快速回滚)

在事务中如果有DDL语句，会自动提交DML，因此`混合DDL和DML不会影响该功能`。


## 回滚

回滚无差异


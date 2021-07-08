
# Transaction

## Introdunction

The transaction function realizes that DML statements are submitted in batches to improve the execution efficiency of large batches of DML statements and ensure transaction consistency (`in the same batch`).

The following details the configuration method and the rollback differences involved.

## Config

Add the parameter `--trans=?` when calling goInception, where the parameter value is a number,
* The default is 0, that is, do not open the transaction (commit row by row)
* When it is greater than 1, it will be submitted in batches according to this parameter, such as 500, it will be submitted once according to 500 DML


### Demo
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

## Execute

* Commit row by row before opening the transaction
* After opening the transaction, submit according to the set number. If set to 500, 500 DMLs will be submitted once
* DDL execution is not affected by the transaction function, and like MySQL, it is independent of a transaction
* When the transaction fails to commit, it will `roll back the batch of SQL` and terminate it immediately (all executed SQL will generate a rollback statement, so that it can be rolled back quickly if necessary)

If there are DDL statements in the transaction, DML will be automatically submitted, so `mixing DDL and DML will not affect this function`.

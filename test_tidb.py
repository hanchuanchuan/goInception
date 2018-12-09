#!/usr/bin/env python
# -*- coding:utf-8 -*-

import pymysql
import prettytable as pt

tb = pt.PrettyTable()

sql = '''select * from test.t1;'''
sql = '''/*--user=admin;--password=han123;--host=127.0.0.1;\
--execute=1;--remote-backup=1;--port=3306;*/
inception_magic_start;
use `test`;
insert into test.t1(id) select c1,c2 from t1;
insert into test.t1(id,c1) select * from t1;
insert into test.t1(id,c1) select c1,c2 from t1;
inception_magic_commit;
'''

sql = '''/*--user=admin;--password=han123;--host=127.0.0.1;\
--execute=1;--remote-backup=1;--port=3306;*/
inception_magic_start;
use `test`;
 delete from t1 where id =1;
# delete t1 from t1,t2,t3 where t1.iid = t2.id and t2.id = t3.id and t3.id = 1;
inception_magic_commit;
'''

sql = '''/*--user=admin;--password=han123;--host=127.0.0.1;\
--execute=1;--remote-backup=1;--port=3306;*/
inception_magic_start;
use `test`;
update t1 set c1  = 1000 where id  =10001;
inception_magic_commit;
'''

# alter table t1 add index ix_1(c1,c2);
# ALTER TABLE t1
#     ADD CONSTRAINT fk FOREIGN KEY(c1) REFERENCES t2(ID);

# alter table t1 drop primary key;


# drop table if exists ttt1 ;

# create table ttt1(id int,c1 varchar(20) not null,c11 varchar(1000),primary key(id));

# insert into ttt1(id,c1) values(1,'test');
# insert into ttt1(id,c1) values(2,'test'),(3,'test');
# insert into ttt1(id,c1) select id+3,c1 from ttt1;
# insert into ttt1(id,c1) select 8,'123';

# drop table if exists ttt2;

# create table ttt2(id int primary key,pid int );

# insert into ttt2 select 1,1;
# insert into ttt2 select 2,2;

# update ttt1 set c11 = c1 ;

# delete from ttt1;

sql = '''/*--user=admin;--password=han123;--host=127.0.0.1;\
--execute=1;--backup=1;--port=3306;--ignore-warnings=1;*/
inception_magic_start;
use `test`;

# drop table if exists ttt1 ;

# create table ttt1(id int,c1 varchar(20) not null,c11 varchar(1000),primary key(id));

# insert into ttt1(id,c1) values(1,'test');
# insert into ttt1(id,c1) values(2,'test'),(3,'test');
# insert into ttt1(id,c1) select id+3,c1 from ttt1;
# insert into ttt1(id,c1) select 8,'123';

# drop table if exists ttt2;

# create table ttt2(id int primary key,pid int );

# insert into ttt2 select 1,1;
# insert into ttt2 select 2,2;

# update ttt1 set c11 = c1 ;

# delete from ttt1;

delete from t1 where id > 0;

inception_magic_commit;
'''

# alter table t1 add column c2 int;
# alter table t1 add column c3 int after c7,add column c4 int;

conn = pymysql.connect(host='127.0.0.1', user='root',
                       passwd='', db='', port=4000, charset="utf8mb4")
# cur = conn.cursor(pymysql.cursors.SSCursor)
cur = conn.cursor()
ret = cur.execute(sql)
result = cur.fetchall()
num_fields = len(cur.description)
field_names = [i[0] for i in cur.description]


tb.field_names = field_names
for row in result:
    tb.add_row(row)

print(tb)

# print(result)
# print(len(result))
# print(field_names)
# for row in result:
#     print(row)
cur.close()
conn.close()

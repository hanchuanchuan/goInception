

### python Demo


```
pip install pymysql prettytable
```

```python
#!/usr/bin/env python
# -*- coding:utf-8 -*-

import pymysql
import prettytable as pt
tb = pt.PrettyTable()

sql = '''/*--user=root;--password=root;--host=127.0.0.1;--check=1;--port=3306;*/
inception_magic_start;
use test_inc;
create table t1(id int primary key,c1 int);
insert into t1(id,c1,c2) values(1,1,1);
inception_magic_commit;'''

conn = pymysql.connect(host='127.0.0.1', user='', passwd='',
                       db='', port=4000, charset="utf8mb4")
cur = conn.cursor()
ret = cur.execute(sql)
result = cur.fetchall()
cur.close()
conn.close()

tb.field_names = [i[0] for i in cur.description]
for row in result:
    tb.add_row(row)
print(tb)
```

Results show：

order_id |  stage  | error_level |   stage_status   |         error_message        |                    sql                     | affected_rows |   sequence   | backup_dbname | execute_time | sqlsha1 | backup_time
----------|---------|----------|-----------------|-----------------------------|--------------------------------------------|---------------|--------------|---------------|--------------|-------------|---------
1     | CHECKED |    0     | Audit Completed |    None                         |                use test_inc                |       0       | 0_0_00000000 |      None     |      0       |   None |      0
2     | CHECKED |    0     | Audit Completed |    None                         | create table t1(id int primary key,c1 int) |       0       | 0_0_00000001 |      None     |      0       |   None |      0
3     | CHECKED |    2     | Audit Completed | Column 't1.c2' not existed. |   insert into t1(id,c1,c2) values(1,1,1)   |       1       | 0_0_00000002 |      None     |      0       |   None |      0


### system variables

connection
```bash
mysql -h127.0.0.1 -P4000
```

```sql
inception show variables;
```

![variables list](./images/variables.png)

### processlist

connections
```bash
mysql -h127.0.0.1 -P4000
```

```sql
inception show processlist;
```

![variables list](./images/processlist.png)




### pause process(`*new`)

** you can kill process at the stage of audit and execute. At backup stage, you can not use kill to pause process. ** `v0.6.2 new`

links： [kill support](https://github.com/hanchuanchuan/goInception/issues/10)

```bash
mysql -h127.0.0.1 -P4000
```

```sql
inception show processlist;
```

```sql
kill 2;
```




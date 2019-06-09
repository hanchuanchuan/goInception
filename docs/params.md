
#### 使用示例

goInception延用inception的使用方式，在审核的sql开始前添加注释来指定远端服务器，并在sql的前后添加特殊标识以区分待审核语句，示例如下：

```sql
/*--user=root;--password=root;--host=127.0.0.1;--check=1;--port=3306;*/
inception_magic_start;
use test;
create table t1(id int primary key);
inception_magic_commit;
```


### 选项列表

参数  |  默认值  |  数据类型 | 说明
------------ | ------------- | ------------ | ------------
host   |  ''    |   string     |   线上数据库IP地址
port | 0 | int | 线上数据库端口
user | '' | string | 线上数据库用户名
password | '' | string | 线上数据库密码
check | false | bool | 开启审核功能
execute | false | bool | 开启执行功能
backup | false | bool | 开启备份功能，仅在执行时生效
ignore_warnings | false | bool | 是否忽略警告，仅在执行时生效。该参数控制有警告时是继续执行还是中止
fingerprint `v0.6.2` | false | bool | 开启sql指纹功能。dml语句相似时，可以根据相同的指纹ID复用explain结果，以减少远端数据库explain操作，并提高审核速度
query-print `v0.7.1` | false | bool | 打印SQL语法树，返回JSON格式结果，详情请查看**[语法树打印](../tree)**
split `v0.9.1` | false | bool | 将一段SQL语句按互不影响原则分组DDL和DML语句，即相同表的DDL及DML语句分开两个语句块执行。指定后，其他选项(审核、执行、备份、打印语法树等)均不再生效。兼容老版inception，实际情况下 **可以不分组**，goInception记录有表结构快照，用以实现binlog解析。[更多信息](https://github.com/hanchuanchuan/goInception/pull/42)

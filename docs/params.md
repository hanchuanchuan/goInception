
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
db `v1.1.0` | "mysql" | string | 默认连接的数据库。该参数可忽略，即使用默认数据库`mysql`。可设置为空""。
check | false | bool | 开启审核功能。开启后，执行选项不再生效
execute | false | bool | 开启执行功能
backup | false | bool | 开启备份功能，仅在执行时生效
ignore_warnings | false | bool | 是否忽略警告，仅在执行时生效。该参数控制有警告时是继续执行还是中止
fingerprint `v0.6.2` | false | bool | 开启sql指纹功能。dml语句相似时，可以根据相同的指纹ID复用explain结果，以减少远端数据库explain操作，并提高审核速度
query-print `v0.7.1` | false | bool | 打印SQL语法树，返回JSON格式结果，详情请查看**[语法树打印](../tree)**
split `v0.9.1` | false | bool | 将一段SQL语句按互不影响原则分组DDL和DML语句，即相同表的DDL及DML语句分开两个语句块执行。指定后，其他选项(审核、执行、备份、打印语法树等)均不再生效。兼容老版inception，实际情况下 **可以不分组**，goInception记录有表结构快照，用以实现binlog解析。[更多信息](https://github.com/hanchuanchuan/goInception/pull/42)
sleep `v1.0-rc3` | 0 | int | 执行 `sleep_rows` 条SQL后休眠多少毫秒，用以降低对线上数据库的影响。单位为毫秒，最小值为 `0` ，即不设置，最大值为 `100000`，即100秒。默认值 `0`
sleep_rows `v1.0-rc3` | 1 | int | 执行多少条SQL后休眠一次。最小值为 `1`，默认值 `1`
real_row_count `v1.0.3` | false | bool | 设置是否通过count(*)获取真正受影响行数(DML操作).默认值 `false`

### mysql加密连接设置

参数  |  默认值  |  数据类型 | 说明
------------ | ------------- | ------------ | ------------
ssl   |  DISABLED    |   string     |   ssl-mode设置，参数和mysql的--ssl-mode一致
ssl-ca |   | string | 证书颁发机构（CA）证书文件的路径名（PEM格式）
ssl-cert |   | string | SSL公钥证书文件的路径名（PEM格式）
ssl-key |   | string | SSL私钥文件的路径名（PEM格式）

#### ssl类型说明
类型  |  说明
------------ | -------------
DISABLED `默认值` 		|	禁用TLS
PREFERRED 		|	由服务器发布时使用TLS
REQUIRED		|	使用TLS，但不检查CA证书
VERIFY_CA		|	验证CA证书，但忽略主机名不匹配
VERIFY_IDENTITY	| 	验证CA证书

##### CA证书认证示例

```python
# 通过ssl=verify_ca设置CA证书验证
# 需要把证书放在goInception服务同主机上
sql = '''/*--user=test;--password=xxx;--host=127.0.0.1;--port=3333;--check=1;\
--ignore-warnings=1;--ssl=verify_ca;\
--ssl-ca=/data/mysql/data/ca.pem;\
--ssl-cert=/data/mysql/data/client-cert.pem;\
--ssl-key=/data/mysql/data/client-key.pem;*/
inception_magic_start;
use test_inc;
...
inception_magic_commit;'''
```

##### ssl认证认证示例

```python
# 通过ssl=verify_ca设置CA证书验证
# 需要把证书放在goInception服务同主机上
sql = '''/*--user=test;--password=xxx;--host=127.0.0.1;--port=3333;--check=1;\
--ignore-warnings=1;--ssl=required;*/
inception_magic_start;
use test_inc;
...
inception_magic_commit;'''
```

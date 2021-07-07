
### Usage demo

goInception extension of the usage of Inception, to specify the remote server by adding annotations before the SQL review, and for distinguishing SQL and review adding special comments at the beginning and the end of SQL.

```sql
/*--user=root;--password=root;--host=127.0.0.1;--check=1;--port=3306;*/
inception_magic_start;
use test;
create table t1(id int primary key);
inception_magic_commit;
```

### Option List

|Command|Default|Type|Explanation|
|:----|:----|:----|:----|
|host|""|string|DB Server host|
|port|0|int|DB Service port|
|user|""|string|User name to connect to DB|
|password|""|string|password to connect to DB|
|db `v1.1.0`|mysql|string|Default connection DB, optional,can null.|
|check|FALSE|bool|Review only|
|execute|FALSE|bool|Execute SQL|
|backup|FALSE|bool|Backup when execute|
|ignore_warnings|FALSE|bool|If stop execute when warning or not.|
|trans `v1.1.6`|0|int|How many DML operations can be included in each transaction. `>1` means turn on transaction.|
|fingerprint `v0.6.2`|FALSE|bool|SQL fingerprint for similar DML. reuses the explanation, decreases explain operations, speed review.|
|query-print `v0.7.1`|FALSE|bool|Print the SQL syntax map|
|split `v0.9.1`|FALSE|bool|Split DDL and DML on the same table. But other option unavailable such as review, execute,backup,print sql syntax map. Normally, no need to split, GoInception records table structure snapshot which can be used for binlog translation.|
|sleep `v1.0-rc3`|0|int|For decrease the impact to DB service, set sleep time by sleep_rows. Default 0 millisecond, max value is 100000 millisecond, equally 100 second.|
|sleep_rows `v1.0-rc3`|1|int|Sleep after how many SQL execute. Default 1.|
|real_row_count `v1.0.3`|FALSE|bool|If get real DML effect rows by `count(*)`, default false, if ture, ignore fingerprintSQL, accurately first.|


### MySQL encrypted connection set

|Key|Default|Type|Explanation|
|:----|:----|:----|:----|
|ssl|DISABLED|string|Ssl-mode same as mysql --ssl-mode|
|ssl-ca| |string|CA file path, PEM format|
|ssl-cert| |string|SSL public key file path, PEM format|
|ssl-key| |string|SSL private key file path, PEM format|


### SSL type description

|type|description|
|:----|:----|
|DISABLED(default)|Disable TLS|
|PREFERRED|TLS when server publishing use.|
|REQUIRED|Use TLS, but do not check CA|
|VERIFY_CA|Identify CA, but ignore hostname mismatch|
|VERIFY_IDENTITY|Identify CA|


### CA Authentication Demo

```python
# use ssl=verify_ca setting CA identify
# need to put the CA with goInception service on the same server
sql = '''/*--user=test;--password=xxx;--host=127.0.0.1;--port=3333;--check=1;\
--ignore-warnings=1;--ssl=verify_ca;\
--ssl-ca=/data/mysql/data/ca.pem;\
--ssl-cert=/data/mysql/data/client-cert.pem;\
--ssl-key=/data/mysql/data/client-key.pem;*/
inception_magic_start;
use test_inc;
...
inception_magic_commit;
```

### SSL Authentication Demo

```python
sql = '''/*--user=test;--password=xxx;--host=127.0.0.1;--port=3333;--check=1;\
--ignore-warnings=1;--ssl=required;*/
inception_magic_start;
use test_inc;
...
inception_magic_commit;'''
```

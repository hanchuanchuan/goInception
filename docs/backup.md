
#### Backup Function

goInception support backup, at config.toml(at `[inc]` block)

parameters  |  default  |  type | description
------------ | ------------- | ------------ | ------------
backup_host   |  ""    |   string     |   Backup database ip address
backup_port   |  0    |   int     |     Backup database port
backup_user   |  ""    |   string     |   username to connect backup database
backup_password   |  ""    |   string    |   password to connect backup database
backup_tls `v1.4.0`   |  ""    |   string    |  Backup database ssl authentication method, please refer to the optional values (https://github.com/go-sql-driver/mysql/issues/899#issuecomment-443493840)

Add ```--backup=true``` option when execute SQL

```sql
/*--user=root;--password=root;--host=127.0.0.1;--port=3306;--execute=1;--backup=1;*/
inception_magic_start;
use test;
create table t1(id int primary key);
inception_magic_commit;
```


#### Backup Record Format

- backup database naming format: ```IP_PORT_{dbname}```, eg: ```127_0_0_1_3306_test```
- create backup information table on backup schema ```$_$Inception_backup_information$_$``` to save execute SQL and rollback SQL

    | Column             | Type         | Comment
    --------------------|--------------|------
    opid_time         | varchar(50)  | operation ID, formatting ```{timestamp}_{thread_id}_{operation_id}```
    start_binlog_file | varchar(512) | binlog start filename
    start_binlog_pos  | int(11)      | binlog start position
    end_binlog_file   | varchar(512) | binlog end filename
    end_binlog_pos    | int(11)      | binlog end position
    sql_statement     | text         | SQL execute
    host              | varchar(64)  | which IP address sql execute
    dbname            | varchar(64)  | which schema sql execute
    tablename         | varchar(64)  | which table sql execute
    port              | int(11)      | which port sql execute
    time              | timestamp    | when execute
    type              | varchar(20)  | execution type

- the table in the backup database has the same table as name of execution table, table structure as blow.

    Column  |  Type  | Comment
    ------------ | ------------- | ------------
    id   |  bigint     |   Auto_increment primary key
    rollback_statement   |  mediumtext    |  rollback SQL
    opid_time   |  varchar(50)    | operation ID related

#### Backup Process Details

1. Config backup database and turn on backup function before executing SQL.
2. Record binlog position and threadID before executing SQL one by one.
3. Execute SQL.
4. Record binlog position and threadID after executed SQL.
5. Start backup, decode binlog on remote server.
6. Create backup database on backup server.
7. Create backup information table, record execution information and binlog position.
8. Create backup table.
9. Decode binlog, build rollback SQL and then insert into backup information table.

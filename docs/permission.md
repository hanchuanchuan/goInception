# Permission Desc

Different functions and stages require different permissions. The permission requirements that may be involved in each function are listed below. If there are any omissions, please suggest and add.

The suggested permissions are:

`GRANT ALL PRIVILEGES ON *.* TO ...`

or

`GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, PROCESS, REFERENCES, INDEX, ALTER, SUPER, REPLICATION SLAVE, REPLICATION CLIENT, TRIGGER ON *.* TO ...`


## Audit function

* `information_schema db` Metadata query permissions, table structure, index information, constraints, etc.
* `mysql db` use permission, no query, the library is connected by default, and it can be modified by calling the option `--db` parameter
* `DML` During the audit, the explain operation will be performed on the DML statement, and this operation requires the actual corresponding DML authority.
* `REFERENCES` Only required for foreign keys

## Execute

* Actual SQL execution permissions


### Use pt-osc

* `PROCESS` permission, view processlist information
* `TRIGGER` create and delete triggers
* `SUPER` or `REPLICATION CLIENT` When there is a master-slave, check the master-slave delay

### Use gh-ost

* `SUPER|REPLICATION CLIENT, REPLICATION SLAVE` Simulate slave pull binlog events
* `ALTER`, `CREATE`, `DELETE`, `DROP`, `INDEX`, `INSERT`, `LOCK TABLES`, `SELECT`, `TRIGGER`, `UPDATE`


## Backup

### Remote database

* `SUPER` When the binlog format is not row, execute `set session binlog_format='row'`

* `SUPER|REPLICATION CLIENT, REPLICATION SLAVE` binlog解析

### Database used for backup

* `It is recommended to grant all permissions to the backup library instance`

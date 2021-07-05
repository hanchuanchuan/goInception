## Introdunction

goInception is a MySQL maintenance tool, which can be used to review, implement, backup, and generate SQL statements for rollback. It parses SQL syntax and returns the result of the review based on custom rules.


#### Architecture


![流程](./images/process.jpg)

#### Usage

GoInception extension of the usage of Inception, to specify the remote server by adding annotations before the SQL review, and for distinguishing SQL and review adding special comments at the beginning and the end of SQL.

Any MySQL protocol-driven can connect in the same way, but the syntax is slightly different. Support different parameters to set for review by specific formats.

```sql
/*--user=root;--password=root;--host=127.0.0.1;--check=1;--port=3306;*/
inception_magic_start;
use test;
create table t1(id int primary key);
inception_magic_commit;
```



#### Acknowledgments

GoInception reconstructs from the Inception which is a well-known MySQL auditing tool and uses TiDB SQL parser.

- [Inception - 审核工具](https://github.com/hanchuanchuan/inception)
- [TiDB](https://github.com/pingcap/tidb)

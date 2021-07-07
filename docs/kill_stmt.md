
### KILL Operation

**Please connect to GoInception and use `inception show processlist `to make sure the process current stage before use KILL.**

Process stage in tree:
- CHECKING
- EXECUTING
- BACKUP

#### Kill execution location
- Remote db server, the sql execute database.
- GoInception
- Goinception backup database


### goInception KILL

Current stage   | kill results
----------------|---------
|CHECKING  |	After the current statement audit is completed| the audit will be aborted. At this time, only the audit results from the beginning to the current statement will be returned and subsequent SQL will not be audited anymore.|
|EXECUTING  | After the execution of the current statement is completed, the execution will be aborted. If the backup is turned on the backup operation will be executed and it will return directly if it is not turned on.|
|BACKUP	 | After the current binlog event parsing is completed the backup is aborted but the generated rollback statement will continue to be written to the backup library and return after the writing is completed|


### Remote database kill（Not recommended）

Current stage   | kill results
----------------|--------
|CHECKING|kill operation does not affect the audit, the connection is automatically reconnect after kill (because of an audit failure audit ** ** do not abort, we need to reconnect and restore disconnected database in order to avoid subsequent SQL database access error)|
|EXECUTING (`after kill execution failed`)|When the statement takes too long, kill will directly stop the execution of the goInception statement. If the backup is turned on, the backup operation will be performed, and if it is not turned on, it will return directly.|
|EXECUTING (`after kill execution success，connection stop`)|When the statement is executed relatively quickly, it may have been executed successfully. At this time, it needs to be further verified according to the binlog backup, so it depends on the backup function|
|BACKUP|N/A|


### Goinception backup database kill (Absolutely not recommended)

Current stage   | kill results
----------------|--------
|CHECKING	Before starting the backup| it will automatically detect the connection and reconnect| so this operation is invalid.|
|EXECUTING	Before starting the backup| it will automatically detect the connection and reconnect| so this operation is invalid|
|BACKUP	Execution may succeed or fail| lead to uncertain results of the backup| so totally not recommended KILL operation in the backup repository|



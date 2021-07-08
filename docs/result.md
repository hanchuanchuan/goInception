
# Result set description


The information returned by goInception is described as follows:
* When there is an error in the basic information submitted to goInception, such as incomplete source information, a call exception will be returned, including error code and error information. The exception is the same as that of the MySQL server, and it can be handled in the same way as MySQL.
* During the audit, it will be returned to the client in the form of a query table. Same as mysql native result set. In the returned result set, each row of data is a submitted SQL statement. GoInception internally disassembles all submitted statement blocks one by one and returns them in the form of a result set. For each statement, what is the problem or status? The result is clear at a glance.


**Note:** If there is a grammatical error in the statement, it will not continue. Because goInception has been unable to parse the following statement correctly, the multiple lines that have been checked before will return normally, and the subsequent error statements will be merged into one line and returned.

Result set to return to normal following structure:

1. `order_id` sql sequence number, starting from 1
1. `stage` The stage that the current statement has reached, including NONE, CHECKED, EXECUTED, BACKUP
	- NONE means that no processing has been done.
	- CHECKED means this statement has been reviewed.
	- EXECUTED means that it has been executed. If the execution fails, it is also expressed in this state.
	- BACKUP means that the current is the backup phase.
1. `error_level` Error level. If the return value is not 0, it means there is an error. 0 means approved. 1 means warning, can be forced to skip to execute, 2 means serious error, unable to execute
1. `stage_status` Phase status description, used to indicate the success or failure of the phase
	- If the review is successful, return **Audit completed**
	- If the execution is successful, return **Execute Successfully**, otherwise return **Execute failed**
	- If the backup is successful, it returns **Backup successfully**, otherwise it returns **Backup failed**，
1. `error_message` Error message. Used to indicate error information, including all error information in a statement, separated by a newline character. If there is no error, NULL is returned. For execution errors, "execute: specific reasons for execution errors" will be appended. If it is a backup error, "backup: specific reasons for backup errors" will be appended.
1. `sql` Current sql statement
1. `affected_rows` Returns the estimated number of affected rows, and the actual number of affected rows is displayed during execution.
1. `sequence` This column is used for the backup function and corresponds to the value of the opid_time column in the **$$Inception_backup_information$$** table.
This is the entry point for the front-end application to roll back for a certain operation. Each statement will generate a sequence number when it is executed. If you want to roll back, use this value to find the corresponding rollback statement from the backup table and execute it. See [Backup Function](backup.html) for details
1. `backup_dbname` Returns which database of the backup server the backup information corresponding to the current statement is stored in. This is a string type value, only for the statement that needs to be backed up. The database name consists of the IP address, port, and source database name, connected by underscores. See [Backup Function](backup.html) for details
1. `execute_time` Indicates the execution time of the statement, in seconds, accurate to two decimal places. The column type is a string, which may need to be converted to a DOUBLE value when used. If it is only audited but not executed, the value returned by this column is 0.
1. `sqlsha1` Used to store the HASH value of the current statement. If there is a value in the returned information, it means that the statement will use OSC when it is executed, because there will be a separate audit operation before it is executed, and the upper layer can already get the value at this time After the review is passed, the statement will not change, of course this value will not change, then you can use this value to view the progress of OSC execution and other information during execution, for example: *D0210DFF35F0BC0A7C95CD98F5BCD4D9B0CA8154. For other information, please refer to [DDL Changes :pt-osc](osc.html) and [DDL changes: gh-ost](ghost.html)
1. `backup_time` The time it took to generate a backup of the current SQL.

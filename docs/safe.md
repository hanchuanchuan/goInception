# [Safe] User authentication

### Desc

Goinception itself is based on TiDB, so it has a complete user management module. For ease of use, this function is turned off by default.

**Open authentication method:**

Add the following parameters in the **config.toml** configuration file (file root node or [inc] node)

```
skip_grant_table = false
```

The supported syntax is as follows:

- CREATE USER
- DROP USER
- ALTER USER
- SET PASSWORD FOR
- GRANK/REVOKE `May not be used`
- SELECT * FROM MYSQL.USER `Query user list`

`The default initial user is root, and the password is empty`

If you forget the password, you can restart it by skipping authentication. After changing the password, turn on authentication and restart goinception (this method is similar to MySQL).

**Note**: The data directory (default is `/tmp/tidb`) may be damaged during abnormal shutdown. At this time, you need to delete the directory and restart, but the created users will be lost, so please pay attention to back up the directory or save the user creation script.

This function is the only one that needs attention to save the data directory, and no other functions are needed.


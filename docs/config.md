
#### config.toml配置文件说明

goInception通过`./goInception -config=config/config.toml`方式启动，接下来说明config.toml中的各项配置。

**由于goInception采用TiDB源码重构，所以部分参数可参考TiDB相关文档**

`config.toml`文件由几部分组成，分别为最外层配置如`host`,`port`等，以及各分组如`[inc]`,`[log]`等，接下来逐一说明。

示例(该示例仅为展示config.toml文件结构，[详细参数请参考](https://github.com/hanchuanchuan/goInception/blob/master/config/config.toml.default))：
```toml

host = "0.0.0.0"
port = 4000
path = "/tmp/tidb"

[log]
# 日志参数
level = "info"
format = "text"

[log.file]
# 日志文件参数
filename = ""
max-size = 300

[inc]
# 审核选项
enable_nullable = true
enable_drop_table = false
check_table_comment = false
check_column_comment = false
# 等等...

[osc]
# pt-osc参数
osc_on = false
osc_min_table_size = 16

[ghost]
# gh-ost参数
ghost_allow_on_master = true

```

### host
绑定的IP地址，默认值 `0.0.0.0`

### port
绑定的端口，默认值 `4000`

### path
TiDB数据库目录，默认值 `/tmp/tidb`，该参数会创建少量TiDB的系统表，如果设置为空时，则会在内存中创建。
建议指定实际目录，这样会加快启动的速度。



### [inc]

所有的 **[审核选项](../options)** 在此处设置

### [osc]

所有的 **[pt-osc选项](../osc)** 在此处设置

### [gh-ost]

所有的 **[gh-ost选项](../ghost)** 在此处设置



### [log]

##### level
日志级别，默认值 `info`
可选值： `debug`, `info`, `warn`, `error`.

##### format
日志格式，默认值 `text`
可选值： `json`, `text`, `console`

##### disable-timestamp
禁用时间戳输出，默认值 `false`


### [log.file]
##### filename
日志文件，默认值 `""`
建议指定日志文件，便于问题追溯

##### max-size
日志文件的最大上限(MB)，默认值 `300`

##### max-days
日志文件的保存天数，默认值 `0`，即不清理

##### max-backups
要保留的最大旧日志文件数，默认值 `0`，即不清理

##### log-rotate
日志轮询，默认值 `true`，即开启


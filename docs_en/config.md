
#### config.toml description

goInception run by `./goInception -config=config/config.toml`, this document show the detail of the configuration file.

**goInception based on some TiDB source code, so some config parameters can check TiDB documents**

`config.toml` contains general config such as `host` and `port`, and group config block such as `[inc]` and `[log]` etc.

Demo(the demo blow just shows the structure of config file, [see details](https://github.com/hanchuanchuan/goInception/blob/master/config/config.toml.default))

```toml

host = "0.0.0.0"
port = 4000
path = "/tmp/tidb"

[log]
# log setting
level = "info"
format = "text"

[log.file]
# log file setting
filename = ""
max-size = 300

[inc]
# audit option
enable_nullable = true
enable_drop_table = false
check_table_comment = false
check_column_comment = false


[osc]
# pt-osc options
osc_on = false
osc_min_table_size = 16

[ghost]
# gh-ost options
ghost_allow_on_master = true

```

### host
IP address, default `0.0.0.0`

### port
Service port, default `4000`

### path
TiDB date path, create some TiDB system table. If null, create in memory. Advice to set a specific data path for speed up start.


### [inc]

all **[audit options](../options)** in here

### [osc]

all **[pt-osc options](../osc)** in here

### [gh-ost]

all **[gh-ost options](../ghost)** in here


### [log]

##### level
log level,default `info`
option: `debug`, `info`, `warn`, `error`.

##### format
log format,default `text`
option: `json`, `text`, `console`

##### disable-timestamp
Diable timestamp, default `false`


### [log.file]
##### filename
log file name, default "", 
advice to set specific log file name for tracing.

##### max-size
Max size of log file, default `300MB`

##### max-days
Max days of log file keep. default `0`, it means keep all log files..

##### max-backups
Max numbers of log file keep. default `0`, it means keep all log files.

##### log-rotate
If turn on log rotate, default `true`, it means turn on.


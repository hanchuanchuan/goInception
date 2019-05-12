
### KILL操作

**使用前请连接goInception并使用`inception show processlist`确认当前阶段(`STATE`)**


阶段分为三种：

- CHECKING (`审核中`)
- EXECUTING (`执行中`)
- BACKUP (`备份中`)

####KILL操作执行位置

- 远端数据库,即执行实际sql的数据库
- goInception
- goInception配置的备份库


### goInception KILL

当前阶段   | kill后的结果
----------|---------
CHECKING | 在`当前语句` `审核完成` 后中止审核，此时仅返回从开始到当前语句的审核结果，后续SQL不再审核
EXECUTING | 在`当前语句` `执行完成` 后中止执行，如果开启了备份，会执行备份操作，未开启则直接返回
BACKUP | 在`当前binlog事件` 解析完成后中止备份，但已生成的回滚语句会继续写入备份库，待写入完成后返回


### 远端数据库KILL(不建议)

当前阶段   | kill后的结果
----------|---------
CHECKING | kill操作不会影响审核，连接被kill后会自动重连(`原因是审核失败**不会中止审核**，所以需要重连，并恢复断开的数据库，以避免后续SQL访问错数据库`)
EXECUTING (`语句kill后执行失败`) | 语句用时过长时，此时kill会直接停止goInception语句的执行，如果开启了备份，会执行备份操作，未开启则直接返回
EXECUTING (`语句kill后执行成功，连接断开`)| 语句执行比较快时，可能已经执行成功，此时需要根据binlog备份做进一步校验，所以`依赖备份功能`
BACKUP | `<无>`


### goInception备份库KILL (`完全不建议`)

当前阶段   | kill后的结果
----------|---------
CHECKING | 在开始备份前会自动检测连接并重连，所以该操作`无效`
EXECUTING | 在开始备份前会自动检测连接并重连，所以该操作`无效`
BACKUP | 执行可能成功也可能失败，会导致备份结果不确定，因此`完全不建议`在备份库执行KILL操作


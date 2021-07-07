# goinception用户鉴权

### 说明

goinception本身基于TiDB，所以拥有完整的用户管理模块，为了简单使用，默认是关闭该功能的。

**开启鉴权方法:**

在 config.toml 配置文件添加以下参数(文件根节点或者[inc]节点)

```
skip_grant_table = false
```

相应的语法支持如下:

- CREATE USER
- DROP USER
- ALTER USER
- SET PASSWORD FOR
- GRANK/REVOKE `可能用不到`
- SELECT * FROM MYSQL.USER `用户查询`

`默认初始用户为root, 密码为空`

忘记密码后可以通过跳过鉴权的方式重新启动，修改密码后开启鉴权并重启goinception。

在非正常关闭时数据目录(默认为`/tmp/tidb`)可能损坏，此时需要删除该目录并重启，但已创建用户会丢失，因此请注意备份该目录或保存用户创建脚本。

该功能是唯一需要注意保存数据目录的，其他功能均不需要。


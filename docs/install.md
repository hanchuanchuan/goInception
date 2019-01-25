

### 二进制安装


[goInception安装包](https://github.com/hanchuanchuan/goInceptionLFS)


### 源码安装

```sh

# 下载源码
git clone https://github.com/hanchuanchuan/goInception

cd goInception

# 构建二进制包
go build -o goInception tidb-server/main.go

# 启动服务
./goInception -config=config/config.toml
```

启动
```bash
goInception -config=config/config.toml
```





### 二进制安装


[goInception安装包](https://github.com/hanchuanchuan/goInception/releases)


### 源码安装

- *go版本v1.12及以上*

- *使用go mod做依赖管理*

```sh

# 下载源码
git clone https://github.com/hanchuanchuan/goInception

cd goInception

make parser

# 构建二进制包
go build -o goInception tidb-server/main.go

```

#### 启动(注意指定配置文件)

```sh
./goInception -config=config/config.toml
```



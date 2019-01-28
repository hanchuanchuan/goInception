# inception的go语言重构

[![GitHub release](https://img.shields.io/github/release-pre/hanchuanchuan/goInception.svg?style=brightgreen)](https://github.com/hanchuanchuan/goInception/releases)
![](https://img.shields.io/badge/go-1.11-brightgreen.svg) 
[![TiDB](https://img.shields.io/badge/TiDB-v2.1.1-brightgreen.svg)](https://github.com/pingcap/tidb)
![](https://img.shields.io/github/downloads/hanchuanchuan/goInception/total.svg)

**说明：使用go mod做依赖管理**


### 安装说明

#### 二进制免安装

[goInception下载](https://github.com/hanchuanchuan/goInception/releases)

#### 源码编译

***go version 1.11***

```bash
git clone https://github.com/hanchuanchuan/goInception
cd goInception
go build -o goInception tidb-server/main.go

./goInception -config=config/config.toml
```


### 引用

- **inception**：https://github.com/hanchuanchuan/inception
- **TiDB**：https://github.com/pingcap/tidb

### 联系方式
- 邮箱：chuanchuanhan@gmail.com

# goInception

[![travis-ci](https://img.shields.io/travis/hanchuanchuan/goInception.svg)](https://travis-ci.org/hanchuanchuan/goInception)
[![CircleCI Status](https://circleci.com/gh/hanchuanchuan/goInception.svg?style=shield)](https://circleci.com/gh/hanchuanchuan/goInception)
[![GitHub release](https://img.shields.io/github/release-pre/hanchuanchuan/goInception.svg?style=brightgreen)](https://github.com/hanchuanchuan/goInception/releases)
[![codecov](https://codecov.io/gh/hanchuanchuan/goInception/branch/master/graph/badge.svg)](https://codecov.io/gh/hanchuanchuan/goInception)
[![](https://img.shields.io/badge/go-1.11-brightgreen.svg)](https://golang.org/dl/)
[![TiDB](https://img.shields.io/badge/TiDB-v2.1.1-brightgreen.svg)](https://github.com/pingcap/tidb)
![](https://img.shields.io/github/downloads/hanchuanchuan/goInception/total.svg)
![](https://img.shields.io/github/license/hanchuanchuan/goInception.svg)


goInception是一个集审核、执行、备份及生成回滚语句于一身的MySQL运维工具， 通过对执行SQL的语法解析，返回基于自定义规则的审核结果，并提供执行和备份及生成回滚语句的功能

**[使用文档](https://hanchuanchuan.github.io/goInception/)**

**[更新日志](https://github.com/hanchuanchuan/goInception/blob/master/docs/changelog.md)**


#### 安装说明

##### 二进制免安装

[goInception下载](https://github.com/hanchuanchuan/goInception/releases)

##### 源码编译

***go version 1.11.3(go mod)***

```bash
git clone https://github.com/hanchuanchuan/goInception.git
cd goInception
make parser
go build -o goInception tidb-server/main.go

./goInception -config=config/config.toml
```

## 贡献

欢迎并非常感谢您的贡献。 有关提交PR的流程请参考 [CONTRIBUTING.md](CONTRIBUTING.md)。

[贡献者列表](CONTRIBUTORS.md)


#### Docker镜像
```
docker pull hanchuanchuan/goinception
```

#### 关联SQL审核平台 `已集成goInception`

* [Archery](https://github.com/hhyo/Archery) `查询支持(MySQL/MsSQL/Redis/PostgreSQL)、MySQL优化(SQLAdvisor|SOAR|SQLTuning)、慢日志管理、表结构对比、会话管理、阿里云RDS管理等`


#### 致谢
    goInception基于TiDB的语法解析器，和业内有名的inpcetion审核工具重构。
- [Inception - 审核工具](https://github.com/hanchuanchuan/inception)
- [TiDB](https://github.com/pingcap/tidb)

#### 交流

QQ群 **499262190**

*(通用问题建议提issue以便于记录及帮助他人)*


# goInception

[![travis-ci](https://img.shields.io/travis/hanchuanchuan/goInception.svg)](https://travis-ci.org/hanchuanchuan/goInception)
[![CircleCI Status](https://circleci.com/gh/hanchuanchuan/goInception.svg?style=shield)](https://circleci.com/gh/hanchuanchuan/goInception)
[![GitHub release](https://img.shields.io/github/release-pre/hanchuanchuan/goInception.svg?style=brightgreen)](https://github.com/hanchuanchuan/goInception/releases)
[![codecov](https://codecov.io/gh/hanchuanchuan/goInception/branch/master/graph/badge.svg)](https://codecov.io/gh/hanchuanchuan/goInception)
[![](https://img.shields.io/badge/go-1.12-brightgreen.svg)](https://golang.org/dl/)
[![TiDB](https://img.shields.io/badge/TiDB-v2.1.1-brightgreen.svg)](https://github.com/pingcap/tidb)
![](https://img.shields.io/github/downloads/hanchuanchuan/goInception/total.svg)
![](https://img.shields.io/github/license/hanchuanchuan/goInception.svg)


goInception是一个集审核、执行、备份及生成回滚语句于一身的MySQL运维工具， 通过对执行SQL的语法解析，返回基于自定义规则的审核结果，并提供执行和备份及生成回滚语句的功能

**[使用文档](https://hanchuanchuan.github.io/goInception/)**

**[更新日志](https://github.com/hanchuanchuan/goInception/blob/master/docs/changelog.md)**


### 安装说明

##### 二进制免安装

[goInception下载](https://github.com/hanchuanchuan/goInception/releases)

##### 源码编译

***go version 1.12 (go mod)***

```bash
git clone https://github.com/hanchuanchuan/goInception.git
cd goInception
make parser
go build -o goInception tidb-server/main.go

./goInception -config=config/config.toml
```

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

#### 赞助&定制
- [赞助与支持](https://hanchuanchuan.github.io/goInception/support/)

#### 交流

QQ群 **499262190**

*(通用问题建议提issue以便于记录及帮助他人)*

### 贡献

欢迎并非常感谢您的贡献。 有关提交PR的流程请参考 [CONTRIBUTING.md](CONTRIBUTING.md)。


## Contributors

### Code Contributors

This project exists thanks to all the people who contribute. [[Contribute](CONTRIBUTING.md)].
<a href="https://github.com/hanchuanchuan/goInception/graphs/contributors"><img src="https://opencollective.com/goInception/contributors.svg?width=890&button=false" /></a>

### Financial Contributors

Become a financial contributor and help us sustain our community. [[Contribute](https://opencollective.com/goInception/contribute)]

#### Individuals

<a href="https://opencollective.com/goInception"><img src="https://opencollective.com/goInception/individuals.svg?width=890"></a>

#### Organizations

Support this project with your organization. Your logo will show up here with a link to your website. [[Contribute](https://opencollective.com/goInception/contribute)]

<a href="https://opencollective.com/goInception/organization/0/website"><img src="https://opencollective.com/goInception/organization/0/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/1/website"><img src="https://opencollective.com/goInception/organization/1/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/2/website"><img src="https://opencollective.com/goInception/organization/2/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/3/website"><img src="https://opencollective.com/goInception/organization/3/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/4/website"><img src="https://opencollective.com/goInception/organization/4/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/5/website"><img src="https://opencollective.com/goInception/organization/5/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/6/website"><img src="https://opencollective.com/goInception/organization/6/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/7/website"><img src="https://opencollective.com/goInception/organization/7/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/8/website"><img src="https://opencollective.com/goInception/organization/8/avatar.svg"></a>
<a href="https://opencollective.com/goInception/organization/9/website"><img src="https://opencollective.com/goInception/organization/9/avatar.svg"></a>

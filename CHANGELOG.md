# goInception 更新日志


## [v0.4.1-beta] - 2019-3-6
### Update
* 兼容mariadb数据库(v5.5.60)
  - 添加mariadb的binlog解析支持(测试版本**v5.5.60**,v10版本由于binlog格式改变,暂无法解析thread_id)
  - 优化备份失败时的返回信息


## [v0.4-beta] - 2019-3-5
### New Features
* 添加gh-ost工具支持
  - 无需安装gh-ost,功能内置(v1.0.48)
  - 进程列表 ```inception get osc processlist```
  - 指定进程信息 ```inception get osc_percent 'sqlsha1'```
  - 进程终止 ```inception stop alter 'sqlsha1'``` (同义词```inception kill osc 'sqlsha1'```)
  - 进程暂停 ```inception pause alter 'sqlsha1'``` (同义词```inception pause osc 'sqlsha1'```)
  - 进程恢复 ```inception resume alter 'sqlsha1'``` (同义词```inception resume osc 'sqlsha1'```)
  - 兼容gh-ost参数 ```inception show variables like 'ghost%'```


## [v0.3-beta] - 2019-2-13
### New Features
* 添加pt-osc工具支持
  - ```inception get osc processlist``` 查看osc进程列表
  - ```inception get osc_percent 'sqlsha1'``` 查看指定的osc进程
  - ```inception stop alter 'sqlsha1'``` (同义词```inception kill osc 'sqlsha1'```)中止指定的osc进程


## [v0.2-beta] - 2019-1-31
### Optimizer
* 优化二进制构建方式，压缩安装包大小
* 移除vendor依赖，优化GO111MODULE使用方式

* 跳过权限校验，以避免登陆goInception失败
* 移除root身份启动校验，以避免windows无法启动
* 优化inception set变量时的类型校验


## [v0.1-beta] - 2019-1-25
#### goInception正式发布


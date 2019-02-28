# goInception 更新日志


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


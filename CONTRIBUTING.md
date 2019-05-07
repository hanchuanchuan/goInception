# 贡献指南

本文档约定了一些操作步骤，以方便快速提交PR，并易于合并。

在提交PR前，请先提交一个[Issue](https://github.com/hanchuanchuan/goInception/issues/new/choose)，以方便沟通。


### Go

`goInception`使用 [`Go`](http://golang.org) 语言编写，

GO版本应该在 **1.12** 及以上。

#### 依赖管理

*项目采用 [`Go Modules`](https://github.com/golang/go/wiki/Modules) 管理依赖。*

您仍须定义`GOPATH`，并修改`PATH`以访问您的Go二进制文件。

*由于单元测试使用gofail做模拟测试，所以仍须使用`GOPATH`*

例如：

```SH
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

## 工作流

### Step 1: Fork项目

1. 打开 https://github.com/hanchuanchuan/goInception
2. 点击右上角 `Fork` 按钮，将项目fork到自己的仓库中，等待Fork完成。

### Step 2: 克隆fork项目到本地


将代码放在你的`GOPATH`目录下，定义本地工作目录：

```sh
# 设置你的github用户名`user`:
user={your github profile name}

working_dir=$GOPATH/src/github.com/${user}
```

```sh
mkdir -p $working_dir
cd $working_dir

git clone https://github.com/${user}/goInception.git
# 或者: git clone git@github.com/${user}/goInception.git

cd $working_dir/goInception
git remote add upstream https://github.com/hanchuanchuan/goInception.git
# 或者: git remote add upstream git@github.com/hanchuanchuan/goInception.git

# 避免误推送
git remote set-url --push upstream no_push

# 配置检查
git remote -v
# 结果应该有四条:
# origin    git@github.com:$(user)/goInception.git (fetch)
# origin    git@github.com:$(user)/goInception.git (push)
# upstream  https://github.com/hanchuanchuan/goInception (fetch)
# upstream  no_push (push)
```

#### 定义提交前检查

*这个hook定义了提交前的格式检查*

```sh
cd $working_dir/goInception/.git/hooks
ln -s ../../hooks/pre-commit .
chmod +x pre-commit
cd ../..
```

### Step 3: 创建分支

获取最新版本的goInception

```sh
# 切换到项目目录
cd $working_dir/goInception/
# 获取最新更新
git fetch upstream

# 切换到本地master
git checkout master
# 版本合并
git rebase upstream/master
```

创建分支

```sh
# 创建并切换到新分支
# 新功能建议以feature开头,bug修改建议以fix开头
git checkout -b feature-test
```

### Step 4: 开发

#### 代码开发

现在可以在`feature-test`分支进行开发了.


#### 准备测试环境

运行本地测试需要一个mysql环境，用来测试审核、执行及备份功能。
* mysql建议版本: `5.7`
* mysql启动参数(可设置my.cnf) `mysqld --log-bin=on --server_id=111 --character-set-server=utf8mb4`
* 测试数据库 `mysql -e "create database if not exists test DEFAULT CHARACTER SET utf8;create database if not exists test_inc DEFAULT CHARACTER SET utf8;"`
* 测试用户  `mysql -e "grant all on *.* to test@'127.0.0.1' identified by 'test';FLUSH PRIVILEGES;"`

#### 运行测试

运行完整测试

*完整测试可能由于环境问题无法通过，此时需要保证`GO111MODULE=on go test session/session_inception_*.go`通过*

```sh
# 单元测试
make dev

# 检查checklist
make checklist
```

运行指定测试

```sh
GO111MODULE=on go test session/session_inception_*.go
# 或者:
GO111MODULE=on go test session/session_inception_test.go
```

### Step 5: 保持分支同步

*分支在提交前需要先合并goInception最新版本，以免PR无法合并*

```sh
# 在`feature-test`分支(`git checkout feature-test`)
git fetch upstream
git rebase upstream/master
```

### Step 6: 提交

提交变更

```sh
git commit
```

### Step 7: Push

推送分支到`github.com`
```sh
git push -f origin feature-test
```

### Step 8: 创建PR

1. 访问fork项目地址 `https://github.com/$user/goInception`
2. 点击 `feature-test` 分支旁边的 `Compare & pull request` 按钮

### Step 9: code review

在PR提交后，会自动进行travisci测试和circleci测试，
在review时,有什么变动可以直接在该分支上修改和提交，PR会使用最新的commit，不用再次提交PR,
如果一个PR涉及了多个功能或者修复,建议分成多个分支，以便于review以及合并。


## 代码风格

可参考Golang社区建议的编码风格 [style doc](https://github.com/golang/go/wiki/CodeReviewComments)

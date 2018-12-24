# -*- coding: utf-8 -*-
# 文件名要保存为 fabfile.py

from fabric.api import env, run, local, lcd, cd, put

# 登录用户和主机名：
env.user = 'root'
# 如果没有设置，在需要登录的时候，fabric 会提示输入
env.password = 'admin'
# 如果有多个主机，fabric会自动依次部署
env.hosts = ['localhost:5022']

TAR_FILE_NAME = "test.zip"


def pack():
    """
    定义一个pack任务, 打包
    """
    print('在当前目录创建一个打包文件: %s' % TAR_FILE_NAME)
    local('Bandizip.exe c %s *.py' % TAR_FILE_NAME)


def deploy():
    """
    定义一个部署任务
    """
    # 先进行打包
    pack()

    # 远程服务器的临时文件
    remote_tmp_tar = '/share/%s' % TAR_FILE_NAME

    print('上传文件: %s' % TAR_FILE_NAME)

    put(TAR_FILE_NAME, "/share/")

    # cd 命令将远程主机的工作目录切换到指定目录
    with cd("/share/"):
        # 解压
        run("unzip -l %s " % (TAR_FILE_NAME))
        run("unzip -o -d /share/github.com/hanchuanchuan/tidb/ppt/test %s  " %
            (TAR_FILE_NAME))
        run('ls -lt /share/github.com/hanchuanchuan/tidb/ppt/test | head -5')

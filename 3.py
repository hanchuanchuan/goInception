# -*- coding: utf-8 -*-

import os
import subprocess
import argparse
import sys

# print(os.system("whoami"))

# a = subprocess.call("whoami")
# print(a)

# stdout = subprocess.check_output("whoami",
#                                  stderr=subprocess.STDOUT)

# stdout = stdout.decode('utf-8').strip()

# print(stdout)

# https://python3-cookbook.readthedocs.io/zh_CN/latest/c13/p06_executing_external_command_and_get_its_output.html
if __name__ == '__main__':

    print(sys.stdout.encoding)
    a = input("请输入你的姓名:")
    print(type(a))
    print("欢迎:", a)

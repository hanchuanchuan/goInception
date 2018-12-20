# -*- coding: utf-8 -*-

import os
import subprocess
import argparse
import sys


os.system("whoami")


a = subprocess.call("whoami")
print(a, 123)

a = subprocess.check_output("whoami",
                            stderr=subprocess.STDOUT)
a = a.decode('utf-8').strip()
print(a)

# https://python3-cookbook.readthedocs.io/zh_CN/latest/c13/p06_executing_external_command_and_get_its_output.html
# if __name__ == '__main__':

#     print(sys.stdout.encoding)
#     a = input("请输入你的姓名:")
#     print(type(a))
#     print("欢迎:", a)

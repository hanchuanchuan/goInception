#!/usr/bin/env python
# -*- coding:utf-8 -*-

if __name__ == '__main__':
    print("这是程序入口")

if 2 > 1:
    print("2确实大于1")
else:
    print("这不科学")

i = 1
while True:
    if i > 5:
        break
    print("第%s次循环" % i)
    i = i + 1

# -*- coding: utf-8 -*-

import time
from flask import Flask
app = Flask(__name__)


@app.route('/')
def hello_world():
    print("操作开始")
    time.sleep(10)
    print("操作继续")
    return 'Hello, World!'


if __name__ == '__main__':
    app.run()

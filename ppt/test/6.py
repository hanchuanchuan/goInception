# -*- coding: utf-8 -*-

import requests

if __name__ == '__main__':
    res = requests.get("http://127.0.0.1:5000", timeout=1)
    print(res.text)
    # print(res.json())

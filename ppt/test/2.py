import pymysql
import json


conn = pymysql.connect(host='127.0.0.1', port=3306,
                       user='root', password='root',
                       charset='utf8mb4', db='mysql',
                       cursorclass=pymysql.cursors.DictCursor)

with conn.cursor() as cursor:
    sql = "show databases"
    cursor.execute(sql)
    result = cursor.fetchall()

conn.close()

for row in result:
    print(row)

with open("123", "w") as f:
    json.dump(result, f)

with open("123", "r") as f:
    new = json.load(f)

print(new)
print(type(new), type(new[0]))

import pymysql
import json


conn = pymysql.connect(host='127.0.0.1', port=3306,
                       user='root', password='root',
                       charset='utf8mb4', db='mysql',
                       cursorclass=pymysql.cursors.DictCursor)

with conn.cursor() as cursor:
    sql = "show variables like '%binlog%'"

    cursor.execute(sql)
    result = cursor.fetchall()

conn.close()

print(result)

for row in result:
    print(row)

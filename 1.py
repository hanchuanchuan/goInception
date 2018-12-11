import pymysql
import time

if __name__ == '__main__':
    print("这是程序入口")

# if 2 > 1:
#     print("true")
# else:
#     print("is wrong!")

# for i in range(5):
#     print("第%s次循环" % (i + 1))

# i = 0
# while i < 5:
#     i = i + 1
#     print("第%s次循环" % i)

with open("demo.txt", "w", encoding='utf-8') as f:
    f.write("中文")

try:
    with open("demo.txt", encoding='utf-8') as f:
        print(f.read())

    with open("demo.txt12") as f:
        print(f.read())
except FileNotFoundError as e:
    print("文件未找到")
except Exception as e:
    print(e)


conn = pymysql.connect(host='127.0.0.1', port=3306,
                       user='root', password='root',
                       charset='utf8mb4', db='mysql',
                       cursorclass=pymysql.cursors.DictCursor,
                       )


with conn.cursor() as cursor:
    sql = "show databases"
    cursor.execute(sql)
    result = cursor.fetchall()

for row in result:
    print(row)

with open("123", "w") as f:
    f.write(str(result))

with open("123", "r") as f:
    print(f.read())


config = {
    'host': '127.0.0.1',
    'port': 3306,
    'user': 'root',
    'password': 'root',
    'charset': 'utf8mb4',
    'cursorclass': pymysql.cursors.DictCursor,
}


# with pymysql.connect(**config) as cursor:
#     sql = "show databases"
#     cursor.execute(sql)
#     # result = cursor.fetchall()
#     result = cursor.fetchone()
#     while result:
#         print(result)
#         result = cursor.fetchone()

# time.sleep(5)

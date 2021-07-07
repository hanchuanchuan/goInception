# 审核规则

了解和熟悉下面的审核规则，可以知道goinception大概做了哪些操作。

**说明**：下面所列出来的规则，不一定能覆盖所有goinception当前已经实现的功能，具体包括什么规则，还需要在使用过程中总结，发现，同时可以结合配置参数来详细了解这些规则。

## 支持语法

语法  |  说明
------------ | -------------
use db | 会检查这个库是否存在，需要连接到线上服务器来判断。
set names charset | 仅支持这一个SET命令
create database | 建库
create table | 建表
alter table | 修改表
drop table | 删除表
truncate table | 清空表
insert | 插入语句(包括多值插入和查询插入)
update | 更新语句(包括多表关联更新)
delete | 删除语句(包括多表关联删除)
inception命令集 | 查看进程,osc进程,查看/设置变量及审核级别等
用户管理命令 | 开启安全登陆后,可以创建/修改/删除用户,以及授权/收回和设置密码


## DDL操作

### create table

#### 表属性

检查项  |  相关配置项
------ | ------
这个表不存在                            |
当前库存在                          |
对于`create table like`，会检查like的老表是不是存在。                           |
表名、列名、索引名的长度不大于64个字节                          |
对象名必须大写  | check_identifier_upper
对象名允许字符[a-zA-Z0-9_]  | check_identifier
对象名不能使用关键字  | enable_identifer_keyword
<s>如果建立的是临时表，则必须要以tmp为前缀</s> `不支持临时表!`            |
字符集限制              |   enable_set_charset,support_charset
排序规则限制              | enable_set_collation,support_collation
存储引擎限制              | enable_set_engine,support_engine
不能建立为分区表                | enable_partition_table
只能有一个自增列                |
只能有一个主键 |
表要有主键    | check_primary_key
表要有注释    | check_table_comment
至少有一个列    |
表必须包含某些列    | must_have_columns
表注释长度不能溢出  |
不允许create table as 语法  |
禁止使用Foreign key             |  enable_foreign_key


#### 列属性

检查项  |  相关配置项
------ | ------
不能设置列的字符集    | enable_column_charset
列的类型不能使用集合、枚举、位图类型    | enable_enum_set_bit
列必须要有注释    | check_column_comment
char长度大于20的时候需要改为varchar（长度可配置）   |  max_char_length
列的类型不能是BLOB/TEXT |  enable_blob_type
列的类型不能是JSON |  enable_json_type
不能有重复的列名    |
非数值列不能使用自增      |
不允许无效库名/表名前缀 |
每个列都使用not null  |  enable_nullable
是否允许timestamp类型  | enable_timestamp_type
如果是timestamp类型的，则要必须指定默认值。 | check_timestamp_default
如果是datetime类型的，则要必须指定默认值。 | check_datetime_default
不能同时有两个timestamp类型的列，如果是datetime类型，则不能有两个指定DEFAULT CURRENT_TIMESTAMP及ON UPDATE CURRENT_TIMESTAMP的列。  | check_timestamp_count,check_datetime_count
只有timestamp或datatime才能指定on update |
on update表达式只能为CURRENT_TIMESTAMP |
建议将 float/double 转成 decimal | check_float_double


#### 索引属性检查项

检查项  |  相关配置项
------ | ------
索引必须要有名字    | enable_null_index_name
Unique索引必须要以uniq_为前缀 |  check_index_prefix
普通索引必须要以idx_为前缀    | check_index_prefix
索引的列数不能超过5个   | max_key_parts
主键索引列数限制    | max_primary_key_parts
主键列必须使用int或bigint | enable_pk_columns_only_int
最多有5个索引 | max_keys
建索引时，指定的列必须存在。    |
索引中的列，不能重复    |
BLOB列不能建做KEY   |
索引长度不能超过767或3072,由实际mysql的innodb_large_prefix决定 |
索引名不能是PRIMARY                |
索引名不能重复  |

#### 默认值

检查项  |  相关配置项
------ | ------
BLOB/TEXT类型的列，不能有非NULL的默认值 | enable_blob_not_null
如果默认值为NULL，但列类型为NOT NULL，或者是主键列，或者定义为自增列，则报错。  |
JSON列不能设置默认值。  |
每个列都需要定义默认值，除了自增列/主键/JSON/计算列/以及大字段列之外    | check_column_default_value

#### 自增列
检查项  |  相关配置项
------ | ------
建表时，自增列初始值为1 | check_autoincrement_init_value
如果自增列的名字不为id，说明可能是有意义的，不建议 | check_autoincrement_name
自增列类型必须为int或bigint    |   check_autoincrement_datatype
自增列需要设置无符号 | enable_autoincrement_unsigned

### ALTER

检查项  |  相关配置项
------ | ------
创建索引 | 同建表
添加字段 | 同建表
默认值 | 同建表
检查字符集  | 同建表
检查排序规则    | 同建表
检查存储引擎    | 同建表
<center>-</center> | <center>-</center>
表是否存在  |
同一个表的多个ALTER建议合并 | merge_alter_table
列是否存在  |
表属性只支持对存储引擎、表注释、自增值及默认字符集的修改操作。  |
是否允许change column操作   | enable_change_column
是否允许列顺序变更  |   check_column_position_change
是否允许列类型变更  |   check_column_type_change



## DML


### INSERT


检查项  |  相关配置项
------ | ------
表是否存在  |
列必须存在    |
必须指定插入列表，也就是要写入哪些列，如insert into t (id,id2) values(...)   | check_insert_field
必须指定值列表。    |
插入列列表与值列表个数相同 |
不为null的列，如果插入的值是null，报错   |
插入指定的列列表中，同一个列不能出现多次。  |

### INSERT SELECT

检查项  |  相关配置项
------ | ------
涉及的所有库/表/字段必须存在   |
必须指定插入列表，也就是要写入哪些列，如insert into t (id,id2) select ...   | check_insert_field
是否允许select *  | enable_select_star
必须有where条件   | check_dml_where
不能有limit条件 |  check_dml_limit
不能有order by rand子句 |  enable_orderby_rand
使用explain获取预估行数或select count获取真实行数 | 调用选项`real_row_count`,explain_rule

### UPDATE/DELETE

检查项  |  相关配置项
------ | ------
表必须存在  |
必须有where条件   | check_dml_where
不能有limit条件 |  check_dml_limit
不能有order by语句    | check_dml_orderby
影响行数大于10000条，则报警（数目可配置）   |  max_update_rows
对WHERE条件这个表达式做简单检查，具体包括什么不一一指定 |
多表更新、删除时，每个表及涉及字段必须要存在  |
限制一条insert values的总行数 | max_insert_rows
update 多表关联时,如果set未指定表前缀,自动判断 |
多表时判断未指明表前缀的列是否有歧义 |
update多表关联时,如果set了多个表的字段,同样支持回滚语句生成 |
使用explain获取预估行数或select count获取真实行数 | 调用选项`realRowCount`,explain_rule
mysql版本在5.6之前时,自动将语句转换为select做explain |
设置数据库`sql_safe_updates`参数 | sql_safe_updates
多表关联时,审核join语句是否包含on子句  | check_dml_where
条件中的列是否存在隐式类型转换 | check_implicit_type_conversion
update set 判断set使用了逗号还是and分隔 |



# 说明
* SQL审核主要针对mysql 5.7版本，其他版本支持会有通用的支持，但细节处可能会有差异，如果有什么问题，欢迎提交[Issues](https://github.com/hanchuanchuan/goInception/issues)来共同建设。
### goInception Syntax tree print
#### result set

Setting `--query-print=1` or `--enable-query-print` to turn on print syntax tree.

Similarly, other options enable the beginning of it are mutually exclusive, can not be set at the same time, after switching, reconnection goInception, execution returns a result set included as follows:

- 1.`ID`: query number.
- 2.`STATEMENT`: query text.
- 3.`ERRLEVEL`: error level when print error.
- 4.`QUERY_TREE`: tree information.
- 5.`ERRMSG`: error message.

## Demo

SQL:
````
insert into t (sno,name)
        select sno, name from t alias_t
        where sno=(
                select sno+1 from my
                where
                name like "%zhufeng%" and
                sno > '10010' and
                name=alias_t.name
        )
        order by name
        limit 100, 10;
````

Return query_tree in Json:

````
{
  "text": "insert into t (sno,name)\n        select sno, name from t alias_t\n        where sno=(\n                select sno+1 from my\n                where\n                name like \"%zhufeng%\" and\n                sno > '10010' and\n                name=alias_t.name\n        )\n        order by name\n        limit 100, 10",
  "IsReplace": false,
  "IgnoreErr": false,
  "Table": {
    "text": "",
    "TableRefs": {
      "text": "",
      "resultFields": null,
      "Left": {
        "text": "",
        "Source": {
          "text": "",
          "resultFields": null,
          "Schema": {
            "O": "",
            "L": ""
          },
          "Name": {
            "O": "t",
            "L": "t"
          },
          "DBInfo": null,
          "TableInfo": null,
          "IndexHints": null
        },
        "AsName": {
          "O": "",
          "L": ""
        }
      },
      "Right": null,
      "Tp": 0,
      "On": null,
      "Using": null,
      "NaturalJoin": false,
      "StraightJoin": false
    }
  },
  "Columns": [
    {
      "text": "",
      "Schema": {
        "O": "",
        "L": ""
      },
      "Table": {
        "O": "",
        "L": ""
      },
      "Name": {
        "O": "sno",
        "L": "sno"
      }
    },
    {
      "text": "",
      "Schema": {
        "O": "",
        "L": ""
      },
      "Table": {
        "O": "",
        "L": ""
      },
      "Name": {
        "O": "name",
        "L": "name"
      }
    }
  ],
  "Lists": null,
  "Setlist": null,
  "Priority": 0,
  "OnDuplicate": null,
  "Select": {
    "text": "",
    "resultFields": null,
    "SQLCache": true,
    "CalcFoundRows": false,
    "StraightJoin": false,
    "Priority": 0,
    "Distinct": false,
    "From": {
      "text": "",
      "TableRefs": {
        "text": "",
        "resultFields": null,
        "Left": {
          "text": "",
          "Source": {
            "text": "",
            "resultFields": null,
            "Schema": {
              "O": "",
              "L": ""
            },
            "Name": {
              "O": "t",
              "L": "t"
            },
            "DBInfo": null,
            "TableInfo": null,
            "IndexHints": null
          },
          "AsName": {
            "O": "alias_t",
            "L": "alias_t"
          }
        },
        "Right": null,
        "Tp": 0,
        "On": null,
        "Using": null,
        "NaturalJoin": false,
        "StraightJoin": false
      }
    },
    "Where": {
      "text": "",
      "k": 0,
      "collation": 0,
      "decimal": 0,
      "length": 0,
      "i": 0,
      "b": null,
      "x": null,
      "Type": {
        "Tp": 0,
        "Flag": 0,
        "Flen": 0,
        "Decimal": 0,
        "Charset": "",
        "Collate": "",
        "Elems": null
      },
      "flag": 40,
      "Op": 7,
      "L": {
        "text": "",
        "k": 0,
        "collation": 0,
        "decimal": 0,
        "length": 0,
        "i": 0,
        "b": null,
        "x": null,
        "Type": {
          "Tp": 0,
          "Flag": 0,
          "Flen": 0,
          "Decimal": 0,
          "Charset": "",
          "Collate": "",
          "Elems": null
        },
        "flag": 8,
        "Name": {
          "text": "",
          "Schema": {
            "O": "",
            "L": ""
          },
          "Table": {
            "O": "",
            "L": ""
          },
          "Name": {
            "O": "sno",
            "L": "sno"
          }
        },
        "Refer": null
      },
      "R": {
        "text": "",
        "k": 0,
        "collation": 0,
        "decimal": 0,
        "length": 0,
        "i": 0,
        "b": null,
        "x": null,
        "Type": {
          "Tp": 0,
          "Flag": 0,
          "Flen": 0,
          "Decimal": 0,
          "Charset": "",
          "Collate": "",
          "Elems": null
        },
        "flag": 32,
        "Query": {
          "text": "select sno+1 from my\n                where\n                name like \"%zhufeng%\" and\n                sno > '10010' and\n                name=alias_t.name\n        ",
          "resultFields": null,
          "SQLCache": true,
          "CalcFoundRows": false,
          "StraightJoin": false,
          "Priority": 0,
          "Distinct": false,
          "From": {
            "text": "",
            "TableRefs": {
              "text": "",
              "resultFields": null,
              "Left": {
                "text": "",
                "Source": {
                  "text": "",
                  "resultFields": null,
                  "Schema": {
                    "O": "",
                    "L": ""
                  },
                  "Name": {
                    "O": "my",
                    "L": "my"
                  },
                  "DBInfo": null,
                  "TableInfo": null,
                  "IndexHints": null
                },
                "AsName": {
                  "O": "",
                  "L": ""
                }
              },
              "Right": null,
              "Tp": 0,
              "On": null,
              "Using": null,
              "NaturalJoin": false,
              "StraightJoin": false
            }
          },
          "Where": {
            "text": "",
            "k": 0,
            "collation": 0,
            "decimal": 0,
            "length": 0,
            "i": 0,
            "b": null,
            "x": null,
            "Type": {
              "Tp": 0,
              "Flag": 0,
              "Flen": 0,
              "Decimal": 0,
              "Charset": "",
              "Collate": "",
              "Elems": null
            },
            "flag": 8,
            "Op": 1,
            "L": {
              "text": "",
              "k": 0,
              "collation": 0,
              "decimal": 0,
              "length": 0,
              "i": 0,
              "b": null,
              "x": null,
              "Type": {
                "Tp": 0,
                "Flag": 0,
                "Flen": 0,
                "Decimal": 0,
                "Charset": "",
                "Collate": "",
                "Elems": null
              },
              "flag": 8,
              "Op": 1,
              "L": {
                "text": "",
                "k": 0,
                "collation": 0,
                "decimal": 0,
                "length": 0,
                "i": 0,
                "b": null,
                "x": null,
                "Type": {
                  "Tp": 0,
                  "Flag": 0,
                  "Flen": 0,
                  "Decimal": 0,
                  "Charset": "",
                  "Collate": "",
                  "Elems": null
                },
                "flag": 8,
                "Expr": {
                  "text": "",
                  "k": 0,
                  "collation": 0,
                  "decimal": 0,
                  "length": 0,
                  "i": 0,
                  "b": null,
                  "x": null,
                  "Type": {
                    "Tp": 0,
                    "Flag": 0,
                    "Flen": 0,
                    "Decimal": 0,
                    "Charset": "",
                    "Collate": "",
                    "Elems": null
                  },
                  "flag": 8,
                  "Name": {
                    "text": "",
                    "Schema": {
                      "O": "",
                      "L": ""
                    },
                    "Table": {
                      "O": "",
                      "L": ""
                    },
                    "Name": {
                      "O": "name",
                      "L": "name"
                    }
                  },
                  "Refer": null
                },
                "Pattern": {
                  "text": "",
                  "k": 5,
                  "collation": 0,
                  "decimal": 0,
                  "length": 0,
                  "i": 0,
                  "b": "JXpodWZlbmcl",
                  "x": null,
                  "Type": {
                    "Tp": 253,
                    "Flag": 0,
                    "Flen": 9,
                    "Decimal": -1,
                    "Charset": "utf8",
                    "Collate": "utf8_bin",
                    "Elems": null
                  },
                  "flag": 0,
                  "projectionOffset": -1
                },
                "Not": false,
                "Escape": 92,
                "PatChars": null,
                "PatTypes": null
              },
              "R": {
                "text": "",
                "k": 0,
                "collation": 0,
                "decimal": 0,
                "length": 0,
                "i": 0,
                "b": null,
                "x": null,
                "Type": {
                  "Tp": 0,
                  "Flag": 0,
                  "Flen": 0,
                  "Decimal": 0,
                  "Charset": "",
                  "Collate": "",
                  "Elems": null
                },
                "flag": 8,
                "Op": 10,
                "L": {
                  "text": "",
                  "k": 0,
                  "collation": 0,
                  "decimal": 0,
                  "length": 0,
                  "i": 0,
                  "b": null,
                  "x": null,
                  "Type": {
                    "Tp": 0,
                    "Flag": 0,
                    "Flen": 0,
                    "Decimal": 0,
                    "Charset": "",
                    "Collate": "",
                    "Elems": null
                  },
                  "flag": 8,
                  "Name": {
                    "text": "",
                    "Schema": {
                      "O": "",
                      "L": ""
                    },
                    "Table": {
                      "O": "",
                      "L": ""
                    },
                    "Name": {
                      "O": "sno",
                      "L": "sno"
                    }
                  },
                  "Refer": null
                },
                "R": {
                  "text": "",
                  "k": 5,
                  "collation": 0,
                  "decimal": 0,
                  "length": 0,
                  "i": 0,
                  "b": "MTAwMTA=",
                  "x": null,
                  "Type": {
                    "Tp": 253,
                    "Flag": 0,
                    "Flen": 5,
                    "Decimal": -1,
                    "Charset": "utf8",
                    "Collate": "utf8_bin",
                    "Elems": null
                  },
                  "flag": 0,
                  "projectionOffset": -1
                }
              }
            },
            "R": {
              "text": "",
              "k": 0,
              "collation": 0,
              "decimal": 0,
              "length": 0,
              "i": 0,
              "b": null,
              "x": null,
              "Type": {
                "Tp": 0,
                "Flag": 0,
                "Flen": 0,
                "Decimal": 0,
                "Charset": "",
                "Collate": "",
                "Elems": null
              },
              "flag": 8,
              "Op": 7,
              "L": {
                "text": "",
                "k": 0,
                "collation": 0,
                "decimal": 0,
                "length": 0,
                "i": 0,
                "b": null,
                "x": null,
                "Type": {
                  "Tp": 0,
                  "Flag": 0,
                  "Flen": 0,
                  "Decimal": 0,
                  "Charset": "",
                  "Collate": "",
                  "Elems": null
                },
                "flag": 8,
                "Name": {
                  "text": "",
                  "Schema": {
                    "O": "",
                    "L": ""
                  },
                  "Table": {
                    "O": "",
                    "L": ""
                  },
                  "Name": {
                    "O": "name",
                    "L": "name"
                  }
                },
                "Refer": null
              },
              "R": {
                "text": "",
                "k": 0,
                "collation": 0,
                "decimal": 0,
                "length": 0,
                "i": 0,
                "b": null,
                "x": null,
                "Type": {
                  "Tp": 0,
                  "Flag": 0,
                  "Flen": 0,
                  "Decimal": 0,
                  "Charset": "",
                  "Collate": "",
                  "Elems": null
                },
                "flag": 8,
                "Name": {
                  "text": "",
                  "Schema": {
                    "O": "",
                    "L": ""
                  },
                  "Table": {
                    "O": "alias_t",
                    "L": "alias_t"
                  },
                  "Name": {
                    "O": "name",
                    "L": "name"
                  }
                },
                "Refer": null
              }
            }
          },
          "Fields": {
            "text": "",
            "Fields": [
              {
                "text": "sno+1",
                "Offset": 108,
                "WildCard": null,
                "Expr": {
                  "text": "",
                  "k": 0,
                  "collation": 0,
                  "decimal": 0,
                  "length": 0,
                  "i": 0,
                  "b": null,
                  "x": null,
                  "Type": {
                    "Tp": 0,
                    "Flag": 0,
                    "Flen": 0,
                    "Decimal": 0,
                    "Charset": "",
                    "Collate": "",
                    "Elems": null
                  },
                  "flag": 8,
                  "Op": 11,
                  "L": {
                    "text": "",
                    "k": 0,
                    "collation": 0,
                    "decimal": 0,
                    "length": 0,
                    "i": 0,
                    "b": null,
                    "x": null,
                    "Type": {
                      "Tp": 0,
                      "Flag": 0,
                      "Flen": 0,
                      "Decimal": 0,
                      "Charset": "",
                      "Collate": "",
                      "Elems": null
                    },
                    "flag": 8,
                    "Name": {
                      "text": "",
                      "Schema": {
                        "O": "",
                        "L": ""
                      },
                      "Table": {
                        "O": "",
                        "L": ""
                      },
                      "Name": {
                        "O": "sno",
                        "L": "sno"
                      }
                    },
                    "Refer": null
                  },
                  "R": {
                    "text": "",
                    "k": 1,
                    "collation": 0,
                    "decimal": 0,
                    "length": 0,
                    "i": 1,
                    "b": null,
                    "x": null,
                    "Type": {
                      "Tp": 8,
                      "Flag": 128,
                      "Flen": 1,
                      "Decimal": 0,
                      "Charset": "binary",
                      "Collate": "binary",
                      "Elems": null
                    },
                    "flag": 0,
                    "projectionOffset": -1
                  }
                },
                "AsName": {
                  "O": "",
                  "L": ""
                },
                "Auxiliary": false
              }
            ]
          },
          "GroupBy": null,
          "Having": null,
          "OrderBy": null,
          "Limit": null,
          "LockTp": 0,
          "TableHints": null,
          "IsAfterUnionDistinct": false,
          "IsInBraces": false
        },
        "Evaluated": false,
        "Correlated": false,
        "MultiRows": false,
        "Exists": false
      }
    },
    "Fields": {
      "text": "",
      "Fields": [
        {
          "text": "sno",
          "Offset": 40,
          "WildCard": null,
          "Expr": {
            "text": "",
            "k": 0,
            "collation": 0,
            "decimal": 0,
            "length": 0,
            "i": 0,
            "b": null,
            "x": null,
            "Type": {
              "Tp": 0,
              "Flag": 0,
              "Flen": 0,
              "Decimal": 0,
              "Charset": "",
              "Collate": "",
              "Elems": null
            },
            "flag": 8,
            "Name": {
              "text": "",
              "Schema": {
                "O": "",
                "L": ""
              },
              "Table": {
                "O": "",
                "L": ""
              },
              "Name": {
                "O": "sno",
                "L": "sno"
              }
            },
            "Refer": null
          },
          "AsName": {
            "O": "",
            "L": ""
          },
          "Auxiliary": false
        },
        {
          "text": "name",
          "Offset": 45,
          "WildCard": null,
          "Expr": {
            "text": "",
            "k": 0,
            "collation": 0,
            "decimal": 0,
            "length": 0,
            "i": 0,
            "b": null,
            "x": null,
            "Type": {
              "Tp": 0,
              "Flag": 0,
              "Flen": 0,
              "Decimal": 0,
              "Charset": "",
              "Collate": "",
              "Elems": null
            },
            "flag": 8,
            "Name": {
              "text": "",
              "Schema": {
                "O": "",
                "L": ""
              },
              "Table": {
                "O": "",
                "L": ""
              },
              "Name": {
                "O": "name",
                "L": "name"
              }
            },
            "Refer": null
          },
          "AsName": {
            "O": "",
            "L": ""
          },
          "Auxiliary": false
        }
      ]
    },
    "GroupBy": null,
    "Having": null,
    "OrderBy": {
      "text": "",
      "Items": [
        {
          "text": "",
          "Expr": {
            "text": "",
            "k": 0,
            "collation": 0,
            "decimal": 0,
            "length": 0,
            "i": 0,
            "b": null,
            "x": null,
            "Type": {
              "Tp": 0,
              "Flag": 0,
              "Flen": 0,
              "Decimal": 0,
              "Charset": "",
              "Collate": "",
              "Elems": null
            },
            "flag": 8,
            "Name": {
              "text": "",
              "Schema": {
                "O": "",
                "L": ""
              },
              "Table": {
                "O": "",
                "L": ""
              },
              "Name": {
                "O": "name",
                "L": "name"
              }
            },
            "Refer": null
          },
          "Desc": false
        }
      ],
      "ForUnion": false
    },
    "Limit": {
      "text": "",
      "Count": {
        "text": "",
        "k": 2,
        "collation": 0,
        "decimal": 0,
        "length": 0,
        "i": 10,
        "b": null,
        "x": null,
        "Type": {
          "Tp": 8,
          "Flag": 160,
          "Flen": 2,
          "Decimal": 0,
          "Charset": "binary",
          "Collate": "binary",
          "Elems": null
        },
        "flag": 0,
        "projectionOffset": -1
      },
      "Offset": {
        "text": "",
        "k": 2,
        "collation": 0,
        "decimal": 0,
        "length": 0,
        "i": 100,
        "b": null,
        "x": null,
        "Type": {
          "Tp": 8,
          "Flag": 160,
          "Flen": 3,
          "Decimal": 0,
          "Charset": "binary",
          "Collate": "binary",
          "Elems": null
        },
        "flag": 0,
        "projectionOffset": -1
      }
    },
    "LockTp": 0,
    "TableHints": null,
    "IsAfterUnionDistinct": false,
    "IsInBraces": false
  }
}
````
## Others

The SQL statement above actually does not make sense, here just to be the best possible expression of each type and print out random structure.

You can see this JSON string is very long. But the entire statement is very clear after structuring. It shows which type of statement, which table to use, which column and no ORDER BY, etc. Statement analysis is no longer a difficult thing. Use the program to do this structured statement analysis should be very easy and accurate.

## tag definations
- 1.`command`: SQL type, support: `insert`, `update`, `delete`, `select`.
- 2.`table_object`: `insert/delete` which table. `update` can check by update columns.
- 3.`fields`: array type, `insert` fields.contains all column informations.each column is an expression which contains dbname,tablename and column name.the expression type is FIELD_ITEM.
- 4.`select_insert_values`: the select part of `insert`. contains all items in this query, columns, tables,where and order by, etc.
- 5.`select_list`: 表示当前查询语句（或者子查询）要查询的表达式信息，这里可以是列，也可以是其它计算出来的值，例子中就有select sno+1...这样的查询。
- 6.`table_ref`: array type. shows the reference table information.
- 7.`where`: where expresions.
- 8.`OrderBy`: array type.
- 9.`limit`: limit numbers.
- 10.`GroupBy`: array type.
- 11.`Having`: Having expression
- 12.`subselect`: `dic` type, show sub-query.
- 13.`many_values`: `insert` values in `select`.
- 14.`values`: array type, each item is a column expression. show `insert` rows. it belong to `many_values`. if `update`, the same.
- 15.`set_fields`: array type, `udpate` column informations.each item is a column expresion

The above statement is currently supported appearing label instructions, but lots of them are the expression, when you print an expression, every object has a public KEY, called type. For different type, KEY is not the same, you can try to find the details.


## Expression Type
The following is a list of all expressions currently supported are listed below only different values for type:
- 1.`STRING_ITEM`: string, storage at other keys.有其它KEY用来存储其具体值信息。
- 2.`FIELD_ITEM`: column information, storage at other keys.
- 3.`FUNC_ITEM,COND_ITEM`: logical caculation information, contains comparison operators, `AND`, `OR`, `ISNULL`, `ISNOTNULL`, `LIKE`,`BETWEEN`, `IN`, `NOT`, `NOW`, etc.
- 4.`INT_ITEM`: `integer` expression.
- 5.`REAL_ITEM`: `float` expression.
- 6.`NULL_ITEM`: `NULL` expression.
- 7.`SUBSELECT_ITEM`: `sub-query` expression.
- 8.`SUM_FUNC_ITEM`: `sum()`expression.
- 9.`ROW_ITEM`:  row expression, eg: `select * from t where (sno,name) = (select sno,name from t1)`
- 10.`DECIMAL_ITEM`: `DECIMAL` expression.

Above is the expression types currently supported, already covers all common expressions.

The last thing to note is that the information printed here is not exactly the information after the end of the grammatical analysis, but processed by Inception. For example, when a subquery is used in a query statement, it is also used when there are different contexts. In these cases, Inception has printed the library name table name corresponding to each column, so that the printed information does not exist without positioning (find its library name table) Name) is listed, it is more friendly and accurate in use. For example, in the above example, there is such a situation (the alias of the t table named alisa_t).

## Tips

This feature is a new development to achieve, without much verification (but there was no big problem), but also need to constantly improve and update, if you have any comments or suggestions, you can add QQ Group or contact me to discuss and resolve together.

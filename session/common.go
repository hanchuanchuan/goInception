// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package session

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"
	"strings"

	// "github.com/hanchuanchuan/goInception/types"
	"github.com/hanchuanchuan/goInception/ast"
	log "github.com/sirupsen/logrus"
)

// HTML escaping.
var (
	htmlQuot = []byte(`\"`) // shorter than "&quot;"
	htmlApos = []byte(`\'`) // shorter than "&apos;" and apos was not in HTML until HTML5
)

var charSets = map[string]int{
	"armscii8": 1,
	"ascii":    1,
	"big5":     2,
	"binary":   1,
	"cp1250":   1,
	"cp1251":   1,
	"cp1256":   1,
	"cp1257":   1,
	"cp850":    1,
	"cp852":    1,
	"cp866":    1,
	"cp932":    2,
	"dec8":     1,
	"eucjpms":  3,
	"euckr":    2,
	"gb18030":  4,
	"gb2312":   2,
	"gbk":      2,
	"geostd8":  1,
	"greek":    1,
	"hebrew":   1,
	"hp8":      1,
	"keybcs2":  1,
	"koi8r":    1,
	"koi8u":    1,
	"latin1":   1,
	"latin2":   1,
	"latin5":   1,
	"latin7":   1,
	"macce":    1,
	"macroman": 1,
	"sjis":     2,
	"swe7":     1,
	"tis620":   1,
	"ucs2":     2,
	"ujis":     3,
	"utf16":    4,
	"utf16le":  4,
	"utf32":    4,
	"utf8":     3,
	"utf8mb4":  4,
}

// HTMLEscape writes to w the escaped HTML equivalent of the plain text data b.
func HTMLEscape(w io.Writer, b []byte) {
	last := 0
	for i, c := range b {
		var html []byte
		switch c {
		case '"':
			html = htmlQuot
		case '\'':
			html = htmlApos
		default:
			continue
		}
		w.Write(b[last:i])
		w.Write(html)
		last = i + 1
	}
	w.Write(b[last:])
}

// HTMLEscapeString returns the escaped HTML equivalent of the plain text data s.
func HTMLEscapeString(s string) string {
	// Avoid allocation if we can.
	if !strings.ContainsAny(s, "'\"") {
		return s
	}
	var b bytes.Buffer
	HTMLEscape(&b, []byte(s))
	return b.String()
}

// GetDataBytes 计算数据类型字节数
// https://dev.mysql.com/doc/refman/8.0/en/storage-requirements.html
// return -1 表示该列无法计算数据大小
func (col *FieldInfo) GetDataBytes(dbVersion int, defaultCharset string) int {
	if col.Type == "" {
		log.Warnf("Can't get %s data type", col.Field)
		return -1
	}
	switch strings.ToLower(GetDataTypeBase(col.Type)) {
	case "tinyint", "smallint", "mediumint",
		"int", "integer", "bigint",
		"double", "real", "float", "decimal",
		"numeric", "bit":
		// numeric
		return numericStorageReq(col.Type)

	case "year", "date", "time", "datetime", "timestamp":
		// date & time
		return timeStorageReq(col.Type, dbVersion)

	case "char", "binary", "varchar", "varbinary", "enum", "set",
		"geometry", "point", "linestring", "polygon":
		// string
		// charset := "utf8mb4"
		if col.Collation != "" {
			defaultCharset = strings.SplitN(col.Collation, "_", 2)[0]
		}
		return StringStorageReq(col.Type, defaultCharset)
	case "tibyblob", "tinytext":
		return 1<<8 - 1
	case "blob", "text":
		return 1<<16 - 1
	case "mediumblob", "mediumtext":
		return 1<<24 - 1
	case "longblob", "longtext":
		return 1<<32 - 1
	// case "tinyblob", "tinytext", "blob", "text", "mediumblob", "mediumtext",
	// 	"longblob", "longtext":
	// 	// strings length depend on it's values
	// 	// 这些字段为不定长字段，添加索引时必须指定前缀，索引前缀与字符集相关
	// 	return MaxKeyLength + 1
	default:
		log.Warnf("Type %v not support:", col.Type)
		return -1
	}
}

func (col *FieldInfo) GetDataLength(dbVersion int, defaultCharset string) int {
	if col.Type == "" {
		log.Warnf("Can't get %s data type", col.Field)
		return -1
	}
	// get length
	typeLength := GetDataTypeLength(col.Type)
	if typeLength[0] == -1 {
		return 0
	}

	length := typeLength[0]
	if col.Collation != "" {
		defaultCharset = strings.SplitN(col.Collation, "_", 2)[0]
	}

	switch strings.ToLower(GetDataTypeBase(col.Type)) {
	case "tinyint", "smallint", "mediumint",
		"int", "integer", "bigint",
		"double", "real", "float", "decimal",
		"numeric", "bit":
		// numeric
		return numericStorageReq(col.Type)

	case "year", "date", "time", "datetime", "timestamp":
		// date & time
		return timeStorageReq(col.Type, dbVersion)

	case "char", "varchar", "enum", "set",
		"geometry", "point", "linestring", "polygon":
		return StringStorageReq(col.Type, defaultCharset)

	case "tinytext", "text", "mediumtext", "longtext":
		return StringStorageReq(col.Type, defaultCharset)
		// 二进制不限长度
	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob":
		return length

	// 	// strings length depend on it's values
	// 	// 这些字段为不定长字段，添加索引时必须指定前缀，索引前缀与字符集相关
	// 	return MaxKeyLength + 1
	default:
		log.Warnf("Type %v not support:", col.Type)
		return -1
	}
}

// StringStorageReq String Type Storage Requirements return bytes count
func StringStorageReq(dataType string, charset string) int {
	// get bytes per character, default 1
	bysPerChar := 1
	if _, ok := charSets[strings.ToLower(charset)]; ok {
		bysPerChar = charSets[strings.ToLower(charset)]
	}

	// get length
	typeLength := GetDataTypeLength(dataType)
	if typeLength[0] == -1 {
		return 0
	}

	// get type
	baseType := strings.ToLower(GetDataTypeBase(dataType))

	switch baseType {
	case "char":
		// Otherwise, M × w bytes, <= M <= 255,
		// where w is the number of bytes required for the maximum-length character in the character set.
		if typeLength[0] > 255 {
			typeLength[0] = 255
		}
		return typeLength[0] * bysPerChar
	case "binary":
		// M bytes, 0 <= M <= 255
		if typeLength[0] > 255 {
			typeLength[0] = 255
		}
		return typeLength[0]
	case "varchar", "tinytext", "text", "mediumtext", "longtext":
		if typeLength[0] <= 255 {
			return typeLength[0]*bysPerChar + 1
		}
		return typeLength[0]*bysPerChar + 2
	case "enum":
		// 1 or 2 bytes, depending on the number of enumeration values (65,535 values maximum)
		return typeLength[0]/(2^15) + 1
	case "set":
		// 1, 2, 3, 4, or 8 bytes, depending on the number of set members (64 members maximum)
		return typeLength[0]/8 + 1

	default:
		return 0
	}
}

// Date and Time Type Storage Requirements
// return bytes count
func timeStorageReq(dataType string, version int) int {
	/*
			https://dev.mysql.com/doc/refman/8.0/en/storage-requirements.html
			*   ============================================================================================
			*   |	Data Type |	Storage Required Before MySQL 5.6.4	| Storage Required as of MySQL 5.6.4   |
			*   | ---------------------------------------------------------------------------------------- |
			*   |	YEAR	  |	1 byte	                            | 1 byte                               |
			*   |	DATE	  | 3 bytes	                            | 3 bytes                              |
			*   |	TIME	  | 3 bytes	                            | 3 bytes + fractional seconds storage |
			*   |	DATETIME  | 8 bytes	                            | 5 bytes + fractional seconds storage |
			*   |	TIMESTAMP |	4 bytes	                            | 4 bytes + fractional seconds storage |
			*   ============================================================================================
			*	|  Fractional Seconds Precision |Storage Required  |
			*   | ------------------------------------------------ |
			*	|  0	    					|0 bytes		   |
			*	|  1, 2						    |1 byte            |
			*	|  3, 4						    |2 bytes           |
			*	|  5, 6						    |3 bytes           |
		    *   ====================================================
	*/

	typeLength := GetDataTypeLength(dataType)

	extr := func(length int) int {
		if length > 0 && length <= 2 {
			return 1
		} else if length > 2 && length <= 4 {
			return 2
		} else if length > 4 && length <= 6 || length > 6 {
			return 3
		}
		return 0
	}

	switch strings.ToLower(GetDataTypeBase(dataType)) {
	case "year":
		return 1
	case "date":
		return 3
	case "time":
		if version < 50604 {
			return 3
		}
		// 3 bytes + fractional seconds storage
		return 3 + extr(typeLength[0])
	case "datetime":
		if version < 50604 {
			return 8
		}
		// 5 bytes + fractional seconds storage
		return 5 + extr(typeLength[0])
	case "timestamp":
		if version < 50604 {
			return 4
		}
		// 4 bytes + fractional seconds storage
		return 4 + extr(typeLength[0])
	default:
		return 8
	}
}

// Numeric Type Storage Requirements
// return bytes count
func numericStorageReq(dataType string) int {
	typeLength := GetDataTypeLength(dataType)
	baseType := strings.ToLower(GetDataTypeBase(dataType))

	switch baseType {
	case "tinyint":
		return 1
	case "smallint":
		return 2
	case "mediumint":
		return 3
	case "int", "integer":
		return 4
	case "bigint", "double", "real":
		return 8
	case "float":
		if typeLength[0] == -1 || typeLength[0] >= 0 && typeLength[0] <= 24 {
			// 4 bytes if 0 <= p <= 24
			return 4
		}
		// 8 bytes if no p || 25 <= p <= 53
		return 8
	case "decimal", "numeric":
		// Values for DECIMAL (and NUMERIC) columns are represented using a binary format
		// that packs nine decimal (base 10) digits into four bytes. Storage for the integer
		// and fractional parts of each value are determined separately. Each multiple of nine
		// digits requires four bytes, and the “leftover” digits require some fraction of four bytes.

		if typeLength[0] == -1 {
			return 4
		}

		leftover := func(leftover int) int {
			if leftover > 0 && leftover <= 2 {
				return 1
			} else if leftover > 2 && leftover <= 4 {
				return 2
			} else if leftover > 4 && leftover <= 6 {
				return 3
			} else if leftover > 6 && leftover <= 8 {
				return 4
			} else {
				return 4
			}
		}

		integer := typeLength[0]/9*4 + leftover(typeLength[0]%9)
		fractional := typeLength[1]/9*4 + leftover(typeLength[1]%9)

		return integer + fractional

	case "bit":
		// approximately (M+7)/8 bytes
		if typeLength[0] == -1 {
			return 1
		}
		return (typeLength[0] + 7) / 8

	default:
		log.Errorf("No such numeric type: %s", baseType)
		return 8
	}
}

// GetDataTypeBase 获取dataType中的数据类型，忽略长度
func GetDataTypeBase(dataType string) string {
	if i := strings.Index(dataType, "("); i > 0 {
		return dataType[0:i]
	}

	return dataType
}

// GetDataTypeLength 获取dataType中的数据类型长度
func GetDataTypeLength(dataType string) []int {
	var length []int
	if si := strings.Index(dataType, "("); si > 0 {
		dataLength := dataType[si+1:]
		if ei := strings.Index(dataLength, ")"); ei > 0 {
			dataLength = dataLength[:ei]
			if strings.HasPrefix(dataType, "enum") ||
				strings.HasPrefix(dataType, "set") {
				// set('one', 'two'), enum('G','PG','PG-13','R','NC-17')
				length = []int{len(strings.Split(dataLength, ","))}
			} else {
				// char(10), varchar(10)
				for _, l := range strings.Split(dataLength, ",") {
					v, err := strconv.Atoi(l)
					if err != nil {
						log.Debugf("GetDataTypeLength() Error: %v", err)
						return []int{-1}
					}
					length = append(length, v)
				}
			}
		}
	}

	if len(length) == 0 {
		length = []int{-1}
	}

	return length
}

// func getDefaultValue(ctx sessionctx.Context, c *ast.ColumnOption, tp byte, fsp int) (interface{}, error) {
// 	if tp == mysql.TypeTimestamp || tp == mysql.TypeDatetime {
// 		vd, err := expression.GetTimeValue(ctx, c.Expr, tp, fsp)
// 		value := vd.GetValue()
// 		if err != nil {
// 			return nil, errors.Trace(err)
// 		}

// 		// Value is nil means `default null`.
// 		if value == nil {
// 			return nil, nil
// 		}

// 		// If value is types.Time, convert it to string.
// 		if vv, ok := value.(types.Time); ok {
// 			return vv.String(), nil
// 		}

// 		return value, nil
// 	}
// 	v, err := expression.EvalAstExpr(ctx, c.Expr)
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}

// 	if v.IsNull() {
// 		return nil, nil
// 	}

// 	if v.Kind() == types.KindBinaryLiteral || v.Kind() == types.KindMysqlBit {
// 		if tp == mysql.TypeBit ||
// 			tp == mysql.TypeString || tp == mysql.TypeVarchar || tp == mysql.TypeVarString ||
// 			tp == mysql.TypeBlob || tp == mysql.TypeLongBlob || tp == mysql.TypeMediumBlob || tp == mysql.TypeTinyBlob ||
// 			tp == mysql.TypeJSON {
// 			// For BinaryLiteral / string fields, when getting default value we cast the value into BinaryLiteral{}, thus we return
// 			// its raw string content here.
// 			return v.GetBinaryLiteral().ToString(), nil
// 		}
// 		// For other kind of fields (e.g. INT), we supply its integer value so that it acts as integers.
// 		return v.GetBinaryLiteral().ToInt(ctx.GetSessionVars().StmtCtx)
// 	}

// 	if tp == mysql.TypeBit {
// 		if v.Kind() == types.KindInt64 || v.Kind() == types.KindUint64 {
// 			// For BIT fields, convert int into BinaryLiteral.
// 			return types.NewBinaryLiteralFromUint(v.GetUint64(), -1).ToString(), nil
// 		}
// 	}

// 	return v.ToString()
// }

// func (b *BinlogSyncer) isClosed(ctx context.Context) bool {
// 	select {
// 	case <-ctx.Done():
// 		return true
// 	default:
// 		return false
// 	}
// }

func checkClose(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// if err := checkGoContext(ctx); err != nil {
// 	return nil, err
// }

func findColumn(c *ast.ColumnNameExpr, t *TableInfo) *FieldInfo {
	var tName string
	db := c.Name.Schema.L
	if t.AsName != "" {
		tName = t.AsName
	} else {
		tName = t.Name
	}
	if c.Name.Table.L != "" && (db == "" || strings.EqualFold(t.Schema, db)) &&
		(strings.EqualFold(tName, c.Name.Table.L)) ||
		c.Name.Table.L == "" {
		for i, field := range t.Fields {
			if strings.EqualFold(field.Field, c.Name.Name.L) && !field.IsDeleted {
				return &t.Fields[i]
			}
		}
	}
	return nil
}

func findColumnWithList(c *ast.ColumnNameExpr, tables []*TableInfo) *FieldInfo {
	for _, t := range tables {
		f := findColumn(c, t)
		if f != nil {
			return f
		}
	}
	return nil
}

func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

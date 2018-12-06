package main

import (
	// "database/sql/driver"
	"fmt"
	"github.com/hanchuanchuan/tidb/session"
	"time"
)

func main() {
	a := `insert into ttt1(id,c1) values(2,'test'),(3,'test');`

	sql := "insert into $_$inception_backup_information$_$ values("
	var sqls []string

	start := time.Now()
	i := 0
	for {
		if i > 10000 {
			break
		}
		sqls = append(sqls, sql)
		sqls = append(sqls, session.HTMLEscapeString(a))
		sqls = append(sqls, ")")
		sqls = nil
		// break
		i++
	}

	fmt.Println(time.Since(start))
}

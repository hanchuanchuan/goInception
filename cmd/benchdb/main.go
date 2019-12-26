// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/hanchuanchuan/goInception/ast"
	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/session"
	"github.com/hanchuanchuan/goInception/store/mockstore"
	"github.com/hanchuanchuan/goInception/store/tikv"
	"github.com/hanchuanchuan/goInception/terror"
	"github.com/hanchuanchuan/goInception/util/logutil"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var (
	addr      = flag.String("addr", "/tmp/tidb", "pd address")
	tableName = flag.String("table", "benchdb", "name of the table")
	batchSize = flag.Int("batch", 100, "number of statements in a transaction, used for insert and update-random only")
	blobSize  = flag.Int("blob", 1000, "size of the blob column in the row")
	logLevel  = flag.String("L", "warn", "log level")
	check     = flag.Bool("C", true, "check?")
	execute   = flag.Bool("E", false, "execute?")
	backup    = flag.Bool("B", false, "backup?")
	runJobs   = flag.String("run", strings.Join([]string{
		"create",
		"truncate",
		"insert:0_10000",
		"update-random:0_10000:100000",
		// "select:0_10000:10",
		"update-range:5000_5100:1000",
		// "select:0_10000:10",
		// "gc",
		// "select:0_10000:10",
	}, "|"), "jobs to run")
	sslCA   = flag.String("cacert", "", "path of file that contains list of trusted SSL CAs.")
	sslCert = flag.String("cert", "", "path of file that contains X509 certificate in PEM format.")
	sslKey  = flag.String("key", "", "path of file that contains X509 key in PEM format.")
)

func main() {
	flag.Parse()
	flag.PrintDefaults()
	err := logutil.InitLogger(&logutil.LogConfig{
		Level: *logLevel,
	})
	terror.MustNil(err)
	err = session.RegisterStore("tikv", tikv.Driver{})
	terror.MustNil(err)

	err = session.RegisterStore("mocktikv", mockstore.MockDriver{})
	terror.MustNil(err)

	ut := newBenchDB()
	works := strings.Split(*runJobs, "|")

	f, err := os.Create("/root/hcc/github.com/hanchuanchuan/goInception/profile_cpu")
	if err != nil {
		log.Error(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	for _, v := range works {
		work := strings.ToLower(strings.TrimSpace(v))
		name, spec := ut.mustParseWork(work)
		switch name {
		case "create":
			ut.createTable()
		case "truncate":
			ut.truncateTable()
		case "insert":
			ut.insertRows(spec)
		case "update-random", "update_random":
			ut.updateRandomRows(spec)
		case "update-range", "update_range":
			ut.updateRangeRows(spec)
		case "select":
			ut.selectRows(spec)
		case "query":
			ut.query(spec)
		default:
			cLog("Unknown job ", v)
			return
		}
	}
}

type benchDB struct {
	store   tikv.Storage
	session session.Session
}

func newBenchDB() *benchDB {
	// Create TiKV store and disable GC as we will trigger GC manually.
	store, err := session.NewStore("mocktikv://" + *addr + "?disableGC=true")
	terror.MustNil(err)
	_, err = session.BootstrapSession(store)
	terror.MustNil(err)
	se, err := session.CreateSession(store)
	terror.MustNil(err)
	_, err = se.Execute(context.Background(), "use test")
	terror.MustNil(err)

	inc := &config.GetGlobalConfig().Inc

	inc.Lang = "en-US"
	inc.EnableBlobType = true
	inc.EnableDropTable = true

	inc.BackupHost = "127.0.0.1"
	inc.BackupPort = 3306
	inc.BackupUser = "test"
	inc.BackupPassword = "test"

	return &benchDB{
		store:   store.(tikv.Storage),
		session: se,
	}
}

func (ut *benchDB) makeSQL(sql string) string {
	checkStr := "0"
	if *check && !(*execute || *backup) {
		checkStr = "1"
	}
	a := `/*--user=test;--password=test;--host=127.0.0.1;--check=` + checkStr + `;--execute=` + strconv.FormatBool(*execute) + `;--remote-backup=` + strconv.FormatBool(*backup) + `;--port=3306;--enable-ignore-warnings;*/
inception_magic_start;
use test_inc;
%s;
inception_magic_commit;`
	log.Info(fmt.Sprintf(a, sql))
	return fmt.Sprintf(a, sql)
}

func (ut *benchDB) mustExec(sql string) ([][]string, error) {
	sql = ut.makeSQL(sql)
	rss, err := ut.session.ExecuteInc(context.Background(), sql)
	if err != nil {
		log.Fatal(err)
	}

	if len(rss) > 0 {
		// ctx := context.Background()
		// rs := rss[0]
		// chk := rs.NewChunk()
		// for {
		// 	err := rs.Next(ctx, chk)
		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}
		// 	if chk.NumRows() == 0 {
		// 		break
		// 	}
		// }

		return ut.ResultSetToResult(rss[0]), nil
	}

	return nil, err
}

// ResultSetToResult converts ast.RecordSet to testkit.Result.
// It is used to check results of execute statement in binary mode.
func (ut *benchDB) ResultSetToResult(rs ast.RecordSet) [][]string {
	rows, _ := session.GetRows4Test(context.Background(), ut.session, rs)
	rs.Close()
	sRows := make([][]string, len(rows))
	for i := range rows {
		row := rows[i]
		iRow := make([]string, row.Len())
		for j := 0; j < row.Len(); j++ {
			if row.IsNull(j) {
				iRow[j] = "<nil>"
			} else {
				d := row.GetDatum(j, &rs.Fields()[j].Column.FieldType)
				iRow[j], _ = d.ToString()
			}
		}
		sRows[i] = iRow
	}
	return sRows
}

func (ut *benchDB) mustParseWork(work string) (name string, spec string) {
	strs := strings.Split(work, ":")
	if len(strs) == 1 {
		return strs[0], ""
	}
	return strs[0], strings.Join(strs[1:], ":")
}

func (ut *benchDB) mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal(err)
	}
	return i
}

func (ut *benchDB) mustParseRange(s string) (start, end int) {
	strs := strings.Split(s, "_")
	if len(strs) != 2 {
		log.Fatal("invalid range " + s)
	}
	startStr, endStr := strs[0], strs[1]
	start = ut.mustParseInt(startStr)
	end = ut.mustParseInt(endStr)
	if start < 0 || end < start {
		log.Fatal("invalid range " + s)
	}
	return
}

func (ut *benchDB) mustParseSpec(s string) (start, end, count int) {
	strs := strings.Split(s, ":")
	start, end = ut.mustParseRange(strs[0])
	if len(strs) == 1 {
		count = 1
		return
	}
	count = ut.mustParseInt(strs[1])
	return
}

func (ut *benchDB) createTable() {
	cLog("create table")
	createSQL := "CREATE TABLE IF NOT EXISTS " + *tableName + ` (
  id bigint(20) NOT NULL,
  name varchar(32) NOT NULL,
  exp bigint(20) NOT NULL DEFAULT '0',
  data blob,
  PRIMARY KEY (id),
  UNIQUE KEY name (name)
);`

	res, _ := ut.mustExec(createSQL)
	for _, row := range res {
		if row[2] == "2" {
			log.Fatal(res)
			log.Fatal(row)
		}
	}

	log.Info(res)
}

func (ut *benchDB) truncateTable() {
	cLog("truncate table")
	res, _ := ut.mustExec("truncate table " + *tableName)
	for _, row := range res {
		if row[2] == "2" {
			log.Warn(row)
		}
	}
}

func (ut *benchDB) runCountTimes(name string, count int, f func()) {
	var (
		sum, first, last time.Duration
		min              = time.Minute
		max              = time.Nanosecond
	)
	cLogf("%s started", name)
	for i := 0; i < count; i++ {
		before := time.Now()
		f()
		dur := time.Since(before)
		if first == 0 {
			first = dur
		}
		last = dur
		if dur < min {
			min = dur
		}
		if dur > max {
			max = dur
		}
		sum += dur
	}
	// cLogf("%s done, avg %s, count %d, sum %s, first %s, last %s, max %s, min %s\n\n",
	// 	name, (sum / time.Duration(count)), count, (sum),
	// 	(first), (last), (max), (min))

	cLogf("%s done, avg %s, count %d, sum %s, first %s, last %s, max %s, min %s\n\n",
		name, format(sum/time.Duration(count)), count, format(sum),
		format(first), format(last), format(max), format(min))
}

func format(t time.Duration) string {
	str := ""

	// str += fmt.Sprintf("%.3fs", t.Seconds())

	if t.Minutes() > 1 {
		str += fmt.Sprintf("%dm", int(t.Minutes()))
	}
	if t.Seconds() > 1 {
		str += fmt.Sprintf("%ds", int(t.Seconds()))
	}

	str += fmt.Sprintf("%.0fms", float64(t.Nanoseconds()%1000000000)/1000000.0)
	return str
}

func (ut *benchDB) insertRows(spec string) {
	start, end, _ := ut.mustParseSpec(spec)
	loopCount := (end - start + *batchSize - 1) / *batchSize
	id := start
	ut.runCountTimes("insert", loopCount, func() {
		// ut.mustExec("begin")
		bufSQL := bytes.NewBufferString("")
		buf := make([]byte, *blobSize/2)
		for i := 0; i < *batchSize; i++ {
			if id == end {
				break
			}
			rand.Read(buf)
			bufSQL.WriteString(fmt.Sprintf("insert %s (id, name, data) values (%d, '%d', '%x');",
				*tableName, id, id, buf))
			id++
		}
		res, _ := ut.mustExec(bufSQL.String())
		for _, row := range res {
			if row[2] == "2" {
				log.Fatal(res)
				log.Fatal(row)
			}
		}
	})
}

func (ut *benchDB) updateRandomRows(spec string) {
	start, end, totalCount := ut.mustParseSpec(spec)
	loopCount := (totalCount + *batchSize - 1) / *batchSize
	var runCount = 0
	ut.runCountTimes("update-random", loopCount, func() {
		bufSQL := bytes.NewBufferString("")
		for i := 0; i < *batchSize; i++ {
			if runCount == totalCount {
				break
			}
			id := rand.Intn(end-start) + start
			bufSQL.WriteString(fmt.Sprintf("update %s set exp = exp + 1 where id = %d;", *tableName, id))
			runCount++
		}
		res, _ := ut.mustExec(bufSQL.String())
		for _, row := range res {
			if row[2] == "2" {
				log.Fatal(res)
				log.Fatal(row)
			}
		}
	})
}

func (ut *benchDB) updateRangeRows(spec string) {
	start, end, count := ut.mustParseSpec(spec)
	ut.runCountTimes("update-range", count, func() {
		// ut.mustExec("begin")
		bufSQL := bytes.NewBufferString("")
		bufSQL.WriteString(fmt.Sprintf("update %s set exp = exp + 1 where id >= %d and id < %d;", *tableName, start, end))
		res, _ := ut.mustExec(bufSQL.String())
		for _, row := range res {
			if row[2] == "2" {
				log.Fatal(res)
				log.Fatal(row)
			}
		}
	})
}

func (ut *benchDB) selectRows(spec string) {
	start, end, count := ut.mustParseSpec(spec)
	ut.runCountTimes("select", count, func() {
		selectQuery := fmt.Sprintf("select * from %s where id >= %d and id < %d;", *tableName, start, end)
		ut.mustExec(selectQuery)
	})
}

func (ut *benchDB) query(spec string) {
	strs := strings.Split(spec, ":")
	sql := strs[0]
	count, err := strconv.Atoi(strs[1])
	terror.MustNil(err)
	ut.runCountTimes("query", count, func() {
		ut.mustExec(sql)
	})
}

func cLogf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	fmt.Println("\033[0;32m" + str + "\033[0m\n")
}

func cLog(args ...interface{}) {
	str := fmt.Sprint(args...)
	fmt.Println("\033[0;32m" + str + "\033[0m\n")
}

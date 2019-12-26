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
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/hanchuanchuan/goInception/kv"
	"github.com/hanchuanchuan/goInception/store/tikv"
	"github.com/hanchuanchuan/goInception/terror"
	"github.com/pingcap/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var (
	store     kv.Storage
	dataCnt   = flag.Int("N", 1000000, "data num")
	workerCnt = flag.Int("C", 400, "concurrent num")
	pdAddr    = flag.String("pd", "localhost:2379", "pd address:localhost:2379")
	valueSize = flag.Int("V", 5, "value size in byte")
	sslCA     = flag.String("cacert", "", "path of file that contains list of trusted SSL CAs.")
	sslCert   = flag.String("cert", "", "path of file that contains X509 certificate in PEM format.")
	sslKey    = flag.String("key", "", "path of file that contains X509 key in PEM format.")
)

// Init initializes information.
func Init() {
	driver := tikv.Driver{}
	var err error
	store, err = driver.Open(fmt.Sprintf("tikv://%s?cluster=1", *pdAddr))
	terror.MustNil(err)

	go func() {
		err1 := http.ListenAndServe(":9191", nil)
		terror.Log(errors.Trace(err1))
	}()
}

// batchRW makes sure conflict free.
func batchRW(value []byte) {
	wg := sync.WaitGroup{}
	base := *dataCnt / *workerCnt
	wg.Add(*workerCnt)
	for i := 0; i < *workerCnt; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < base; j++ {
				k := base*i + j
				txn, err := store.Begin()
				if err != nil {
					log.Fatal(err)
				}
				key := fmt.Sprintf("key_%d", k)
				err = txn.Set([]byte(key), value)
				terror.Log(errors.Trace(err))
				err = txn.Commit(context.Background())
				if err != nil {
					terror.Call(txn.Rollback)
				}
			}
		}(i)
	}
	wg.Wait()
}

func main() {
	flag.Parse()
	log.SetLevel(log.ErrorLevel)
	Init()

	value := make([]byte, *valueSize)
	t := time.Now()
	batchRW(value)
	resp, err := http.Get("http://localhost:9191/metrics")
	terror.MustNil(err)

	defer terror.Call(resp.Body.Close)
	text, err1 := ioutil.ReadAll(resp.Body)
	terror.Log(errors.Trace(err1))

	fmt.Println(string(text))

	fmt.Printf("\nelapse:%v, total %v\n", time.Since(t), *dataCnt)
}

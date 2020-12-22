// Copyright 2015 PingCAP, Inc.
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

package session_test

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hanchuanchuan/goInception/util"
	. "github.com/pingcap/check"
)

var _ = Suite(&testSessionParallelSuite{})

func TestParallel(t *testing.T) {
	TestingT(t)
}

type testSessionParallelSuite struct {
	testCommon
}

func (s *testSessionParallelSuite) SetUpSuite(c *C) {
	s.initSetUp(c)
}

func (s *testSessionParallelSuite) TearDownSuite(c *C) {
	s.tearDownSuite(c)
}

func (s *testSessionParallelSuite) TearDownTest(c *C) {
	s.reset()
	s.tearDownTest(c)
}

func (s *testSessionParallelSuite) TestParallel(c *C) {
	// 测试一个对象或者函数在多线程的场景下面是否安全
	var oscConnID uint32
	sm := s.tk.Se.GetSessionManager()
	go func() {
		for i := 0; i < 1000; i++ {
			sm.AddOscProcess(&util.OscProcessInfo{
				ID:         uint64(atomic.AddUint32(&oscConnID, 1)),
				ConnID:     1024,
				Schema:     "test",
				Table:      "test",
				Command:    "select 1",
				Percent:    0,
				RemainTime: "",
				Sqlsha1:    fmt.Sprintf("%v", oscConnID),
				Info:       "",
				IsGhost:    true,
				PanicAbort: make(chan util.ProcessOperation),
				RW:         &sync.RWMutex{},
			})
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// go func() {
	for i := 0; i < 1000; i++ {
		time.Sleep(1 * time.Millisecond)
		sm.OscLock()
		pl := sm.ShowOscProcessListWithWrite()
		// log.Errorf("osc process length: %v", len(pl))
		for _, pi := range pl {
			// log.Errorf("%#v", pi)
			c.Assert(pi.Table, Equals, "test")
		}
		sm.OscUnLock()
	}
	// }()
}

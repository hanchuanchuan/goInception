// Copyright 2018 PingCAP, Inc.
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

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hanchuanchuan/goInception/config"
	"github.com/hanchuanchuan/goInception/domain"
	"github.com/hanchuanchuan/goInception/kv"
	"github.com/hanchuanchuan/goInception/session"
	"github.com/hanchuanchuan/goInception/sessionctx/stmtctx"
	"github.com/hanchuanchuan/goInception/store/mockstore"
	"github.com/hanchuanchuan/goInception/store/mockstore/mocktikv"
	"github.com/hanchuanchuan/goInception/store/tikv"
	"github.com/hanchuanchuan/goInception/tablecodec"
	"github.com/hanchuanchuan/goInception/types"
	"github.com/hanchuanchuan/goInception/util/codec"
	. "github.com/pingcap/check"
)

type HTTPHandlerTestSuite struct {
	server  *Server
	store   kv.Storage
	domain  *domain.Domain
	tidbdrv *TiDBDriver
}

var _ = Suite(new(HTTPHandlerTestSuite))

func (ts *HTTPHandlerTestSuite) TestRegionIndexRange(c *C) {
	sTableID := int64(3)
	sIndex := int64(11)
	eTableID := int64(9)
	recordID := int64(133)
	indexValues := []types.Datum{
		types.NewIntDatum(100),
		types.NewBytesDatum([]byte("foobar")),
		types.NewFloat64Datum(-100.25),
	}
	var expectIndexValues []string
	for _, v := range indexValues {
		str, err := v.ToString()
		if err != nil {
			str = fmt.Sprintf("%d-%v", v.Kind(), v.GetValue())
		}
		expectIndexValues = append(expectIndexValues, str)
	}
	encodedValue, err := codec.EncodeKey(&stmtctx.StatementContext{TimeZone: time.Local}, nil, indexValues...)
	c.Assert(err, IsNil)

	startKey := tablecodec.EncodeIndexSeekKey(sTableID, sIndex, encodedValue)
	recordPrefix := tablecodec.GenTableRecordPrefix(eTableID)
	endKey := tablecodec.EncodeRecordKey(recordPrefix, recordID)

	region := &tikv.KeyLocation{
		Region:   tikv.RegionVerID{},
		StartKey: startKey,
		EndKey:   endKey,
	}
	r, err := NewRegionFrameRange(region)
	c.Assert(err, IsNil)
	c.Assert(r.first.IndexID, Equals, sIndex)
	c.Assert(r.first.IsRecord, IsFalse)
	c.Assert(r.first.RecordID, Equals, int64(0))
	c.Assert(r.first.IndexValues, DeepEquals, expectIndexValues)
	c.Assert(r.last.IsRecord, IsTrue)
	c.Assert(r.last.RecordID, Equals, recordID)
	c.Assert(r.last.IndexValues, IsNil)

	testCases := []struct {
		tableID int64
		indexID int64
		isCover bool
	}{
		{2, 0, false},
		{3, 0, true},
		{9, 0, true},
		{10, 0, false},
		{2, 10, false},
		{3, 10, false},
		{3, 11, true},
		{3, 20, true},
		{9, 10, true},
		{10, 1, false},
	}
	for _, t := range testCases {
		var f *FrameItem
		if t.indexID == 0 {
			f = r.getRecordFrame(t.tableID, "", "")
		} else {
			f = r.getIndexFrame(t.tableID, t.indexID, "", "", "")
		}
		if t.isCover {
			c.Assert(f, NotNil)
		} else {
			c.Assert(f, IsNil)
		}
	}
}

func (ts *HTTPHandlerTestSuite) TestRegionIndexRangeWithEndNoLimit(c *C) {
	sTableID := int64(15)
	startKey := tablecodec.GenTableRecordPrefix(sTableID)
	endKey := []byte("z_aaaaafdfd")
	region := &tikv.KeyLocation{
		Region:   tikv.RegionVerID{},
		StartKey: startKey,
		EndKey:   endKey,
	}
	r, err := NewRegionFrameRange(region)
	c.Assert(err, IsNil)
	c.Assert(r.first.IsRecord, IsTrue)
	c.Assert(r.last.IsRecord, IsTrue)
	c.Assert(r.getRecordFrame(300, "", ""), NotNil)
	c.Assert(r.getIndexFrame(200, 100, "", "", ""), NotNil)
}

func (ts *HTTPHandlerTestSuite) TestRegionIndexRangeWithStartNoLimit(c *C) {
	eTableID := int64(9)
	startKey := []byte("m_aaaaafdfd")
	endKey := tablecodec.GenTableRecordPrefix(eTableID)
	region := &tikv.KeyLocation{
		Region:   tikv.RegionVerID{},
		StartKey: startKey,
		EndKey:   endKey,
	}
	r, err := NewRegionFrameRange(region)
	c.Assert(err, IsNil)
	c.Assert(r.first.IsRecord, IsFalse)
	c.Assert(r.last.IsRecord, IsTrue)
	c.Assert(r.getRecordFrame(3, "", ""), NotNil)
	c.Assert(r.getIndexFrame(8, 1, "", "", ""), NotNil)
}

func (ts *HTTPHandlerTestSuite) TestRegionsAPI(c *C) {
	ts.startServer(c)
	defer ts.stopServer(c)
	resp, err := http.Get("http://127.0.0.1:10090/tables/information_schema/SCHEMATA/regions")
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var data TableRegions
	err = decoder.Decode(&data)
	c.Assert(err, IsNil)
	c.Assert(len(data.RecordRegions) > 0, IsTrue)

	// list region
	for _, region := range data.RecordRegions {
		c.Assert(regionContainsTable(c, region.ID, data.TableID), IsTrue)
	}
}

func regionContainsTable(c *C, regionID uint64, tableID int64) bool {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:10090/regions/%d", regionID))
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	var data RegionDetail
	err = decoder.Decode(&data)
	c.Assert(err, IsNil)
	for _, index := range data.Frames {
		if index.TableID == tableID {
			return true
		}
	}
	return false
}

func (ts *HTTPHandlerTestSuite) TestListTableRegionsWithError(c *C) {
	ts.startServer(c)
	defer ts.stopServer(c)
	resp, err := http.Get("http://127.0.0.1:10090/tables/fdsfds/aaa/regions")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusBadRequest)
}

func (ts *HTTPHandlerTestSuite) TestGetRegionByIDWithError(c *C) {
	ts.startServer(c)
	defer ts.stopServer(c)
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:10090/regions/xxx"))
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusBadRequest)
	defer resp.Body.Close()
}

func (ts *HTTPHandlerTestSuite) TestRegionsFromMeta(c *C) {
	ts.startServer(c)
	defer ts.stopServer(c)
	resp, err := http.Get("http://127.0.0.1:10090/regions/meta")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	// Verify the resp body.
	decoder := json.NewDecoder(resp.Body)
	metas := make([]RegionMeta, 0)
	err = decoder.Decode(&metas)
	c.Assert(err, IsNil)
	for _, meta := range metas {
		c.Assert(meta.ID != 0, IsTrue)
	}
}

func (ts *HTTPHandlerTestSuite) startServer(c *C) {
	mvccStore := mocktikv.MustNewMVCCStore()
	var err error
	ts.store, err = mockstore.NewMockTikvStore(mockstore.WithMVCCStore(mvccStore))
	c.Assert(err, IsNil)
	ts.domain, err = session.BootstrapSession(ts.store)
	c.Assert(err, IsNil)
	ts.tidbdrv = NewTiDBDriver(ts.store)

	cfg := config.NewConfig()
	cfg.Port = 4001
	cfg.Store = "tikv"
	cfg.Status.StatusPort = 10090
	cfg.Status.ReportStatus = true

	server, err := NewServer(cfg, ts.tidbdrv)
	c.Assert(err, IsNil)
	ts.server = server
	go server.Run()
	waitUntilServerOnline(cfg.Status.StatusPort)
}

func (ts *HTTPHandlerTestSuite) stopServer(c *C) {
	if ts.domain != nil {
		ts.domain.Close()
	}
	if ts.store != nil {
		ts.store.Close()
	}
	if ts.server != nil {
		ts.server.Close()
	}
}

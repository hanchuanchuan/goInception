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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hanchuanchuan/goInception/domain"
	"github.com/hanchuanchuan/goInception/infoschema"
	"github.com/hanchuanchuan/goInception/kv"
	"github.com/hanchuanchuan/goInception/meta"
	"github.com/hanchuanchuan/goInception/model"
	"github.com/hanchuanchuan/goInception/session"
	"github.com/hanchuanchuan/goInception/sessionctx"
	"github.com/hanchuanchuan/goInception/sessionctx/stmtctx"
	"github.com/hanchuanchuan/goInception/store/tikv"
	"github.com/hanchuanchuan/goInception/store/tikv/tikvrpc"
	"github.com/hanchuanchuan/goInception/table"
	"github.com/hanchuanchuan/goInception/tablecodec"
	"github.com/hanchuanchuan/goInception/terror"
	"github.com/hanchuanchuan/goInception/types"
	"github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	pDBName     = "db"
	pHexKey     = "hexKey"
	pIndexName  = "index"
	pHandle     = "handle"
	pRegionID   = "regionID"
	pStartTS    = "startTS"
	pTableName  = "table"
	pColumnID   = "colID"
	pColumnTp   = "colTp"
	pColumnFlag = "colFlag"
	pColumnLen  = "colLen"
	pRowBin     = "rowBin"
)

// For query string
const qTableID = "table_id"
const qLimit = "limit"

const (
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

type kvStore interface {
	GetRegionCache() *tikv.RegionCache
	SendReq(bo *tikv.Backoffer, req *tikvrpc.Request, regionID tikv.RegionVerID, timeout time.Duration) (*tikvrpc.Response, error)
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	_, err = w.Write([]byte(err.Error()))
	terror.Log(errors.Trace(err))
}

func writeData(w http.ResponseWriter, data interface{}) {
	js, err := json.Marshal(data)
	if err != nil {
		writeError(w, err)
		return
	}
	log.Info(string(js))
	// write response
	w.Header().Set(headerContentType, contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(js)
	terror.Log(errors.Trace(err))
}

type tikvHandlerTool struct {
	regionCache *tikv.RegionCache
	store       kvStore
}

func (t *tikvHandlerTool) getMvccByEncodedKey(encodedKey kv.Key) (*kvrpcpb.MvccGetByKeyResponse, error) {
	keyLocation, err := t.regionCache.LocateKey(tikv.NewBackoffer(context.Background(), 500), encodedKey)
	if err != nil {
		return nil, errors.Trace(err)
	}

	tikvReq := &tikvrpc.Request{
		Type: tikvrpc.CmdMvccGetByKey,
		MvccGetByKey: &kvrpcpb.MvccGetByKeyRequest{
			Key: encodedKey,
		},
	}
	kvResp, err := t.store.SendReq(tikv.NewBackoffer(context.Background(), 500), tikvReq, keyLocation.Region, time.Minute)
	log.Info(string(encodedKey), keyLocation.Region, string(keyLocation.StartKey), string(keyLocation.EndKey), kvResp, err)

	if err != nil {
		return nil, errors.Trace(err)
	}
	return kvResp.MvccGetByKey, nil
}

func (t *tikvHandlerTool) getMvccByHandle(tableID, handle int64) (*kvrpcpb.MvccGetByKeyResponse, error) {
	encodedKey := tablecodec.EncodeRowKeyWithHandle(tableID, handle)
	return t.getMvccByEncodedKey(encodedKey)
}

func (t *tikvHandlerTool) getMvccByStartTs(startTS uint64, startKey, endKey []byte) (*kvrpcpb.MvccGetByStartTsResponse, error) {
	bo := tikv.NewBackoffer(context.Background(), 5000)
	for {
		curRegion, err := t.regionCache.LocateKey(bo, startKey)
		if err != nil {
			log.Error(startTS, startKey, err)
			return nil, errors.Trace(err)
		}

		tikvReq := &tikvrpc.Request{
			Type: tikvrpc.CmdMvccGetByStartTs,
			MvccGetByStartTs: &kvrpcpb.MvccGetByStartTsRequest{
				StartTs: startTS,
			},
		}
		tikvReq.Context.Priority = kvrpcpb.CommandPri_Low
		kvResp, err := t.store.SendReq(bo, tikvReq, curRegion.Region, time.Hour)
		log.Info(startTS, string(startKey), curRegion.Region, string(curRegion.StartKey), string(curRegion.EndKey), kvResp)
		if err != nil {
			log.Error(startTS, string(startKey), curRegion.Region, string(curRegion.StartKey), string(curRegion.EndKey), err)
			return nil, errors.Trace(err)
		}
		data := kvResp.MvccGetByStartTS
		if err := data.GetRegionError(); err != nil {
			log.Warn(startTS, string(startKey), curRegion.Region, string(curRegion.StartKey), string(curRegion.EndKey), err)
			continue
		}

		if len(data.GetError()) > 0 {
			log.Error(startTS, string(startKey), curRegion.Region, string(curRegion.StartKey), string(curRegion.EndKey), data.GetError())
			return nil, errors.New(data.GetError())
		}

		key := data.GetKey()
		if len(key) > 0 {
			return data, nil
		}

		if len(endKey) > 0 && curRegion.Contains(endKey) {
			return nil, nil
		}
		if len(curRegion.EndKey) == 0 {
			return nil, nil
		}
		startKey = curRegion.EndKey
	}
}

func (t *tikvHandlerTool) getMvccByIdxValue(idx table.Index, values url.Values, idxCols []*model.ColumnInfo, handleStr string) (*kvrpcpb.MvccGetByKeyResponse, error) {
	sc := new(stmtctx.StatementContext)
	// HTTP request is not a database session, set timezone to UTC directly here.
	// See https://github.com/hanchuanchuan/goInception/blob/master/docs/tidb_http_api.md for more details.
	sc.TimeZone = time.UTC
	idxRow, err := t.formValue2DatumRow(sc, values, idxCols)
	if err != nil {
		return nil, errors.Trace(err)
	}
	handle, err := strconv.ParseInt(handleStr, 10, 64)
	if err != nil {
		return nil, errors.Trace(err)
	}
	encodedKey, _, err := idx.GenIndexKey(sc, idxRow, handle, nil)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return t.getMvccByEncodedKey(encodedKey)
}

// formValue2DatumRow converts URL query string to a Datum Row.
func (t *tikvHandlerTool) formValue2DatumRow(sc *stmtctx.StatementContext, values url.Values, idxCols []*model.ColumnInfo) ([]types.Datum, error) {
	data := make([]types.Datum, len(idxCols))
	for i, col := range idxCols {
		colName := col.Name.String()
		vals, ok := values[colName]
		if !ok {
			return nil, errors.BadRequestf("Missing value for index column %s.", colName)
		}

		switch len(vals) {
		case 0:
			data[i].SetNull()
		case 1:
			bDatum := types.NewStringDatum(vals[0])
			cDatum, err := bDatum.ConvertTo(sc, &col.FieldType)
			if err != nil {
				return nil, errors.Trace(err)
			}
			data[i] = cDatum
		default:
			return nil, errors.BadRequestf("Invalid query form for column '%s', it's values are %v."+
				" Column value should be unique for one index record.", colName, vals)
		}
	}
	return data, nil
}

func (t *tikvHandlerTool) getTableID(dbName, tableName string) (int64, error) {
	schema, err := t.schema()
	if err != nil {
		return 0, errors.Trace(err)
	}
	tableVal, err := schema.TableByName(model.NewCIStr(dbName), model.NewCIStr(tableName))
	if err != nil {
		return 0, errors.Trace(err)
	}
	return tableVal.Meta().ID, nil
}

func (t *tikvHandlerTool) schema() (infoschema.InfoSchema, error) {
	session, err := session.CreateSession(t.store.(kv.Storage))
	if err != nil {
		return nil, errors.Trace(err)
	}
	return domain.GetDomain(session.(sessionctx.Context)).InfoSchema(), nil
}

func (t *tikvHandlerTool) handleMvccGetByHex(params map[string]string) (interface{}, error) {
	encodedKey, err := hex.DecodeString(params[pHexKey])
	if err != nil {
		return nil, errors.Trace(err)
	}
	return t.getMvccByEncodedKey(encodedKey)
}

func (t *tikvHandlerTool) getAllHistoryDDL() ([]*model.Job, error) {
	s, err := session.CreateSession(t.store.(kv.Storage))
	if err != nil {
		return nil, errors.Trace(err)
	}

	if s != nil {
		defer s.Close()
	}

	store := domain.GetDomain(s.(sessionctx.Context)).Store()
	txn, err := store.Begin()

	if err != nil {
		return nil, errors.Trace(err)
	}
	txnMeta := meta.NewMeta(txn)

	jobs, err := txnMeta.GetAllHistoryDDLJobs()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return jobs, nil
}

const (
	opMvccGetByHex = "hex"
	opMvccGetByKey = "key"
	opMvccGetByIdx = "idx"
	opMvccGetByTxn = "txn"
)

// TableRegions is the response data for list table's regions.
// It contains regions list for record and indices.
type TableRegions struct {
	TableName     string         `json:"name"`
	TableID       int64          `json:"id"`
	RecordRegions []RegionMeta   `json:"record_regions"`
	Indices       []IndexRegions `json:"indices"`
}

// RegionMeta contains a region's peer detail
type RegionMeta struct {
	ID          uint64              `json:"region_id"`
	Leader      *metapb.Peer        `json:"leader"`
	Peers       []*metapb.Peer      `json:"peers"`
	RegionEpoch *metapb.RegionEpoch `json:"region_epoch"`
}

// IndexRegions is the region info for one index.
type IndexRegions struct {
	Name    string       `json:"name"`
	ID      int64        `json:"id"`
	Regions []RegionMeta `json:"regions"`
}

// RegionDetail is the response data for get region by ID
// it includes indices and records detail in current region.
type RegionDetail struct {
	RegionID uint64       `json:"region_id"`
	StartKey []byte       `json:"start_key"`
	EndKey   []byte       `json:"end_key"`
	Frames   []*FrameItem `json:"frames"`
}

// addTableInRange insert a table into RegionDetail
// with index's id or record in the range if r.
func (rt *RegionDetail) addTableInRange(dbName string, curTable *model.TableInfo, r *RegionFrameRange) {
	tName := curTable.Name.String()
	tID := curTable.ID

	for _, index := range curTable.Indices {
		if f := r.getIndexFrame(tID, index.ID, dbName, tName, index.Name.String()); f != nil {
			rt.Frames = append(rt.Frames, f)
		}
	}
	if f := r.getRecordFrame(tID, dbName, tName); f != nil {
		rt.Frames = append(rt.Frames, f)
	}
}

// FrameItem includes a index's or record's meta data with table's info.
type FrameItem struct {
	DBName      string   `json:"db_name"`
	TableName   string   `json:"table_name"`
	TableID     int64    `json:"table_id"`
	IsRecord    bool     `json:"is_record"`
	RecordID    int64    `json:"record_id,omitempty"`
	IndexName   string   `json:"index_name,omitempty"`
	IndexID     int64    `json:"index_id,omitempty"`
	IndexValues []string `json:"index_values,omitempty"`
}

// RegionFrameRange contains a frame range info which the region covered.
type RegionFrameRange struct {
	first  *FrameItem        // start frame of the region
	last   *FrameItem        // end frame of the region
	region *tikv.KeyLocation // the region
}

func (t *tikvHandlerTool) getRegionsMeta(regionIDs []uint64) ([]RegionMeta, error) {
	regions := make([]RegionMeta, len(regionIDs))
	for i, regionID := range regionIDs {
		meta, leader, err := t.regionCache.PDClient().GetRegionByID(context.TODO(), regionID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		regions[i] = RegionMeta{
			ID:          regionID,
			Leader:      leader,
			Peers:       meta.Peers,
			RegionEpoch: meta.RegionEpoch,
		}

	}
	return regions, nil
}

// pdRegionStats is the json response from PD.
type pdRegionStats struct {
	Count            int              `json:"count"`
	EmptyCount       int              `json:"empty_count"`
	StorageSize      int64            `json:"storage_size"`
	StoreLeaderCount map[uint64]int   `json:"store_leader_count"`
	StorePeerCount   map[uint64]int   `json:"store_peer_count"`
	StoreLeaderSize  map[uint64]int64 `json:"store_leader_size"`
	StorePeerSize    map[uint64]int64 `json:"store_peer_size"`
}

// NewFrameItemFromRegionKey creates a FrameItem with region's startKey or endKey,
// returns err when key is illegal.
func NewFrameItemFromRegionKey(key []byte) (frame *FrameItem, err error) {
	frame = &FrameItem{}
	frame.TableID, frame.IndexID, frame.IsRecord, err = tablecodec.DecodeKeyHead(key)
	if err == nil {
		if frame.IsRecord {
			_, frame.RecordID, err = tablecodec.DecodeRecordKey(key)
		} else {
			_, _, frame.IndexValues, err = tablecodec.DecodeIndexKey(key)
		}
		log.Warnf("decode region key %q fail: %v", key, err)
		// Ignore decode errors.
		err = nil
		return
	}
	if bytes.HasPrefix(key, tablecodec.TablePrefix()) {
		// If SplitTable is enabled, the key may be `t{id}`.
		if len(key) == tablecodec.TableSplitKeyLen {
			frame.TableID = tablecodec.DecodeTableID(key)
			return frame, nil
		}
		return nil, errors.Trace(err)
	}

	// key start with tablePrefix must be either record key or index key
	// That's means table's record key and index key are always together
	// in the continuous interval. And for key with prefix smaller than
	// tablePrefix, is smaller than all tables. While for key with prefix
	// bigger than tablePrefix, means is bigger than all tables.
	err = nil
	if bytes.Compare(key, tablecodec.TablePrefix()) < 0 {
		frame.TableID = math.MinInt64
		frame.IndexID = math.MinInt64
		frame.IsRecord = false
		return
	}
	// bigger than tablePrefix, means is bigger than all tables.
	frame.TableID = math.MaxInt64
	frame.TableID = math.MaxInt64
	frame.IsRecord = true
	return
}

// NewRegionFrameRange init a NewRegionFrameRange with region info.
func NewRegionFrameRange(region *tikv.KeyLocation) (idxRange *RegionFrameRange, err error) {
	var first, last *FrameItem
	// check and init first frame
	if len(region.StartKey) > 0 {
		first, err = NewFrameItemFromRegionKey(region.StartKey)
		if err != nil {
			return
		}
	} else { // empty startKey means start with -infinite
		first = &FrameItem{
			IndexID:  int64(math.MinInt64),
			IsRecord: false,
			TableID:  int64(math.MinInt64),
		}
	}

	// check and init last frame
	if len(region.EndKey) > 0 {
		last, err = NewFrameItemFromRegionKey(region.EndKey)
		if err != nil {
			return
		}
	} else { // empty endKey means end with +infinite
		last = &FrameItem{
			TableID:  int64(math.MaxInt64),
			IndexID:  int64(math.MaxInt64),
			IsRecord: true,
		}
	}

	idxRange = &RegionFrameRange{
		region: region,
		first:  first,
		last:   last,
	}
	return idxRange, nil
}

// getRecordFrame returns the record frame of a table. If the table's records
// are not covered by this frame range, it returns nil.
func (r *RegionFrameRange) getRecordFrame(tableID int64, dbName, tableName string) *FrameItem {
	if tableID == r.first.TableID && r.first.IsRecord {
		r.first.DBName, r.first.TableName = dbName, tableName
		return r.first
	}
	if tableID == r.last.TableID && r.last.IsRecord {
		r.last.DBName, r.last.TableName = dbName, tableName
		return r.last
	}

	if tableID >= r.first.TableID && tableID < r.last.TableID {
		return &FrameItem{
			DBName:    dbName,
			TableName: tableName,
			TableID:   tableID,
			IsRecord:  true,
		}
	}
	return nil
}

// getIndexFrame returns the indnex frame of a table. If the table's indices are
// not covered by this frame range, it returns nil.
func (r *RegionFrameRange) getIndexFrame(tableID, indexID int64, dbName, tableName, indexName string) *FrameItem {
	if tableID == r.first.TableID && !r.first.IsRecord && indexID == r.first.IndexID {
		r.first.DBName, r.first.TableName, r.first.IndexName = dbName, tableName, indexName
		return r.first
	}
	if tableID == r.last.TableID && indexID == r.last.IndexID {
		r.last.DBName, r.last.TableName, r.last.IndexName = dbName, tableName, indexName
		return r.last
	}

	greaterThanFirst := tableID > r.first.TableID || (tableID == r.first.TableID && !r.first.IsRecord && indexID > r.first.IndexID)
	lessThanLast := tableID < r.last.TableID || (tableID == r.last.TableID && (r.last.IsRecord || indexID < r.last.IndexID))
	if greaterThanFirst && lessThanLast {
		return &FrameItem{
			DBName:    dbName,
			TableName: tableName,
			TableID:   tableID,
			IsRecord:  false,
			IndexName: indexName,
			IndexID:   indexID,
		}
	}
	return nil
}

// parseQuery is used to parse query string in URL with shouldUnescape, due to golang http package can not distinguish
// query like "?a=" and "?a". We rewrite it to separate these two queries. e.g.
// "?a=" which means that a is an empty string "";
// "?a"  which means that a is null.
// If shouldUnescape is true, we use QueryUnescape to handle keys and values that will be put in m.
// If shouldUnescape is false, we don't use QueryUnescap to handle.
func parseQuery(query string, m url.Values, shouldUnescape bool) error {
	var err error
	for query != "" {
		key := query
		if i := strings.IndexAny(key, "&;"); i >= 0 {
			key, query = key[:i], key[i+1:]
		} else {
			query = ""
		}
		if key == "" {
			continue
		}
		if i := strings.Index(key, "="); i >= 0 {
			value := ""
			key, value = key[:i], key[i+1:]
			if shouldUnescape {
				key, err = url.QueryUnescape(key)
				if err != nil {
					return errors.Trace(err)
				}
				value, err = url.QueryUnescape(value)
				if err != nil {
					return errors.Trace(err)
				}
			}
			m[key] = append(m[key], value)
		} else {
			if shouldUnescape {
				key, err = url.QueryUnescape(key)
				if err != nil {
					return errors.Trace(err)
				}
			}
			if _, ok := m[key]; !ok {
				m[key] = nil
			}
		}
	}
	return errors.Trace(err)
}

// serverInfo is used to report the servers info when do http request.
type serverInfo struct {
	IsOwner bool `json:"is_owner"`
	*domain.ServerInfo
}

// clusterServerInfo is used to report cluster servers info when do http request.
type clusterServerInfo struct {
	ServersNum                   int                           `json:"servers_num,omitempty"`
	OwnerID                      string                        `json:"owner_id"`
	IsAllServerVersionConsistent bool                          `json:"is_all_server_version_consistent,omitempty"`
	AllServersDiffVersions       []domain.ServerVersionInfo    `json:"all_servers_diff_versions,omitempty"`
	AllServersInfo               map[string]*domain.ServerInfo `json:"all_servers_info,omitempty"`
}

module github.com/hanchuanchuan/goInception

go 1.22.1

replace gopkg.in/gcfg.v1 => github.com/hanchuanchuan/gcfg.v1 v0.0.0-20190302111942-77c0f3dcc0b3

// replace go.etcd.io/gofail => github.com/etcd-io/gofail v0.0.0-20180808172546-51ce9a71510a
replace github.com/etcd-io/gofail => go.etcd.io/gofail v0.0.0-20180808172546-51ce9a71510a

// replace vitess.io/vitess => github.com/vitessio/vitess v3.0.0-rc.3+incompatible
replace vitess.io/vitess => github.com/vitessio/vitess v0.19.1

replace google.golang.org/grpc => google.golang.org/grpc v1.29.1

replace github.com/codahale/hdrhistogram => github.com/HdrHistogram/hdrhistogram-go v1.1.2

// replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.9

replace github.com/go-sql-driver/mysql => github.com/go-sql-driver/mysql v1.4.1-0.20191022112324-6ea7374bc1b0

// replace  go.etcd.io/etcd => github.com/etcd-io/etcd a4f7c65
// replace  github.com/coreos/etcd => github.com/etcd-io/etcd a4f7c65
// replace go.etcd.io/etcd => github.com/etcd-io/etcd v0.5.0-alpha.5.0.20231122225832-2c8e2e933f77

// replace github.com/coreos/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20231122225832-2c8e2e933f77
// replace inet.af/netaddr => github.com/inetaf/netaddr
// replace gopkg.in/DataDog/dd-trace-go.v1 => github.com/DataDog/datadog-go v0.0.0-2d091ec
replace gopkg.in/DataDog/dd-trace-go.v1 => gopkg.in/DataDog/dd-trace-go.v1 v1.29.0-alpha.1.0.20210216140755-2d091eca40bb

require (
	github.com/BurntSushi/toml v1.3.2
	github.com/CorgiMan/json2 v0.0.0-20150213135156-e72957aba209
	github.com/blacktear23/go-proxyprotocol v0.0.0-20171102103907-62e368e1c470
	github.com/coreos/etcd v3.3.13+incompatible
	github.com/cznic/parser v0.0.0-20181122101858-d773202d5b1f
	github.com/cznic/sortutil v0.0.0-20181122101858-f5f958428db8
	github.com/cznic/strutil v0.0.0-20181122101858-275e90344537
	github.com/cznic/y v0.0.0-20181122101901-b05e8c2e8d7b
	github.com/go-mysql-org/go-mysql v1.9.1
	github.com/go-sql-driver/mysql v1.7.1
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/google/btree v1.0.1
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hanchuanchuan/gh-ost v1.0.49-0.20210117111015-ca873c0b5ca6
	github.com/imroc/req v0.2.3
	github.com/jinzhu/copier v0.3.5
	github.com/jinzhu/gorm v1.9.16
	github.com/juju/errors v0.0.0-20181118221551-089d3ea4e4d5
	github.com/klauspost/cpuid v0.0.0-20170728055534-ae7887de9fa5
	github.com/ngaut/pools v0.0.0-20180318154953-b7bc8c42aac7
	github.com/ngaut/sync2 v0.0.0-20141008032647-7a24ed77b2ef
	github.com/percona/go-mysql v0.0.0-20190307200310-f5cfaf6a5e55
	github.com/pingcap/check v0.0.0-20190102082844-67f458068fc8
	github.com/pingcap/errors v0.11.5-0.20221009092201-b66cddb77c32
	github.com/pingcap/goleveldb v0.0.0-20171020122428-b9ff6c35079e
	github.com/pingcap/kvproto v0.0.0-20181206061346-54cf0a0dfe55
	github.com/pingcap/pd v2.1.0+incompatible
	github.com/pingcap/tipb v0.0.0-20190428032612-535e1abaa330
	github.com/shopspring/decimal v1.2.0
	github.com/siddontang/go-log v0.0.0-20180807004314-8d05993dda07
	github.com/sirupsen/logrus v1.9.3
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/viper v1.18.2
	golang.org/x/net v0.23.0
	golang.org/x/text v0.14.0
	google.golang.org/grpc v1.62.1
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	modernc.org/mathutil v1.6.0
	vitess.io/vitess v2.1.1+incompatible
)

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/coreos/bbolt v1.3.2 // indirect
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/cznic/golex v0.0.0-20181122101858-9c343928389c // indirect
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548 // indirect
	github.com/denisenkom/go-mssqldb v0.11.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang/glog v1.2.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hanchuanchuan/go-mysql v0.0.0-20200114082439-6d0d8d3a982e // indirect
	github.com/hanchuanchuan/golib v0.0.0-20200113085747-47643bc243f1 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/juju/testing v0.0.0-20180920084828-472a3e8b2073 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/lib/pq v1.10.2 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-sqlite3 v1.14.16 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/gomega v1.24.2 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/outbrain/golib v0.0.0-20180830062331-ab954725f502 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/pingcap/log v1.1.1-0.20230317032135-a0d097d16e22 // indirect
	github.com/pingcap/tidb/pkg/parser v0.0.0-20231103042308-035ad5ccbe67 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.19.0 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/prometheus/common v0.49.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/siddontang/go v0.0.0-20180604090527-bdc77568d726 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20201229170055-e5319fda7802 // indirect
	github.com/unrolled/render v0.0.0-20180914162206-b9786414de4d // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225 // indirect
	golang.org/x/sys v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240304212257-790db918fca8 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/gcfg.v1 v1.2.3 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

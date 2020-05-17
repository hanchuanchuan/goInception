module github.com/hanchuanchuan/goInception

go 1.12

replace gopkg.in/gcfg.v1 => github.com/hanchuanchuan/gcfg.v1 v0.0.0-20190302111942-77c0f3dcc0b3

replace vitess.io/vitess => github.com/vitessio/vitess v3.0.0-rc.3+incompatible

replace github.com/go-sql-driver/mysql => github.com/go-sql-driver/mysql v1.4.1-0.20191022112324-6ea7374bc1b0

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/CorgiMan/json2 v0.0.0-20150213135156-e72957aba209
	github.com/blacktear23/go-proxyprotocol v0.0.0-20171102103907-62e368e1c470
	github.com/coreos/etcd v3.3.10+incompatible
	github.com/cznic/mathutil v0.0.0-20181021201202-eba54fb065b7
	github.com/cznic/parser v0.0.0-20181122101858-d773202d5b1f
	github.com/cznic/sortutil v0.0.0-20150617083342-4c7342852e65
	github.com/cznic/strutil v0.0.0-20181122101858-275e90344537
	github.com/cznic/y v0.0.0-20181122101901-b05e8c2e8d7b
	github.com/etcd-io/gofail v0.0.0-20180808172546-51ce9a71510a
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/protobuf v1.3.1
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c
	github.com/gorilla/mux v1.6.2
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hanchuanchuan/gh-ost v1.0.49-0.20200114083508-62a578b91654
	github.com/hanchuanchuan/go-mysql v0.0.0-20200114082439-6d0d8d3a982e
	github.com/hanchuanchuan/inception-core v0.0.0-20200517095426-9e30f7d26537
	github.com/imroc/req v0.2.3
	github.com/jinzhu/gorm v1.9.2
	github.com/juju/errors v0.0.0-20181118221551-089d3ea4e4d5
	github.com/klauspost/cpuid v0.0.0-20170728055534-ae7887de9fa5
	github.com/ngaut/pools v0.0.0-20180318154953-b7bc8c42aac7
	github.com/ngaut/sync2 v0.0.0-20141008032647-7a24ed77b2ef
	github.com/percona/go-mysql v0.0.0-20190307200310-f5cfaf6a5e55
	github.com/pingcap/check v0.0.0-20190102082844-67f458068fc8
	github.com/pingcap/errors v0.11.0
	github.com/pingcap/goleveldb v0.0.0-20171020122428-b9ff6c35079e
	github.com/pingcap/kvproto v0.0.0-20181206061346-54cf0a0dfe55
	github.com/pingcap/pd v2.1.0+incompatible
	github.com/pingcap/tipb v0.0.0-20190428032612-535e1abaa330
	github.com/shopspring/decimal v0.0.0-20180709203117-cd690d0c9e24
	github.com/siddontang/go-log v0.0.0-20180807004314-8d05993dda07
	github.com/sirupsen/logrus v1.4.2
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72
	github.com/spf13/viper v1.3.1
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/text v0.3.2
	google.golang.org/grpc v1.16.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	vitess.io/vitess v2.1.1+incompatible
)

module github.com/hanchuanchuan/goInception

go 1.14

replace gopkg.in/gcfg.v1 => github.com/hanchuanchuan/gcfg.v1 v0.0.0-20190302111942-77c0f3dcc0b3

replace vitess.io/vitess => github.com/vitessio/vitess v3.0.0-rc.3+incompatible

replace github.com/go-sql-driver/mysql => github.com/go-sql-driver/mysql v1.4.1-0.20191022112324-6ea7374bc1b0

// replace  go.etcd.io/etcd => github.com/etcd-io/etcd a4f7c65
// replace  github.com/coreos/etcd => github.com/etcd-io/etcd a4f7c65
replace go.etcd.io/etcd => github.com/etcd-io/etcd v0.5.0-alpha.5.0.20190829210359-a4f7c65ef846

replace github.com/coreos/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20190829210359-a4f7c65ef846

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/CorgiMan/json2 v0.0.0-20150213135156-e72957aba209
	github.com/blacktear23/go-proxyprotocol v0.0.0-20171102103907-62e368e1c470
	github.com/coreos/etcd v3.3.10+incompatible
	github.com/coreos/go-systemd v0.0.0-20181031085051-9002847aa142 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/cznic/golex v0.0.0-20181122101858-9c343928389c // indirect
	github.com/cznic/mathutil v0.0.0-20181122101859-297441e03548 // indirect
	github.com/cznic/parser v0.0.0-20181122101858-d773202d5b1f
	github.com/cznic/sortutil v0.0.0-20150617083342-4c7342852e65
	github.com/cznic/strutil v0.0.0-20181122101858-275e90344537
	github.com/cznic/y v0.0.0-20181122101901-b05e8c2e8d7b
	github.com/denisenkom/go-mssqldb v0.0.0-20190121005146-b04fd42d9952 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/eknkc/amber v0.0.0-20171010120322-cdade1c07385 // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/etcd-io/gofail v0.0.0-20180808172546-51ce9a71510a // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20181024230925-c65c006176ff // indirect
	github.com/golang/protobuf v1.3.2
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/google/btree v1.0.0
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hanchuanchuan/gh-ost v1.0.49-0.20210117111015-ca873c0b5ca6
	github.com/hanchuanchuan/go-mysql v0.0.0-20200114082439-6d0d8d3a982e
	github.com/imroc/req v0.2.3
	github.com/jinzhu/copier v0.3.5
	github.com/jinzhu/gorm v1.9.2
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v0.0.0-20181116074157-8ec929ed50c3 // indirect
	github.com/juju/errors v0.0.0-20181118221551-089d3ea4e4d5
	github.com/juju/loggo v0.0.0-20180524022052-584905176618 // indirect
	github.com/juju/testing v0.0.0-20180920084828-472a3e8b2073 // indirect
	github.com/klauspost/cpuid v0.0.0-20170728055534-ae7887de9fa5
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/montanaflynn/stats v0.0.0-20180911141734-db72e6cae808 // indirect
	github.com/ngaut/pools v0.0.0-20180318154953-b7bc8c42aac7
	github.com/ngaut/sync2 v0.0.0-20141008032647-7a24ed77b2ef
	github.com/onsi/ginkgo v1.7.0 // indirect
	github.com/onsi/gomega v1.4.3 // indirect
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/percona/go-mysql v0.0.0-20190307200310-f5cfaf6a5e55
	github.com/pingcap/check v0.0.0-20190102082844-67f458068fc8
	github.com/pingcap/errors v0.11.0
	github.com/pingcap/goleveldb v0.0.0-20171020122428-b9ff6c35079e
	github.com/pingcap/kvproto v0.0.0-20181206061346-54cf0a0dfe55
	github.com/pingcap/pd v2.1.0+incompatible
	github.com/pingcap/tipb v0.0.0-20190428032612-535e1abaa330
	github.com/pkg/errors v0.9.0 // indirect
	github.com/shopspring/decimal v0.0.0-20180709203117-cd690d0c9e24
	github.com/siddontang/go-log v0.0.0-20180807004314-8d05993dda07
	github.com/sirupsen/logrus v1.4.2
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72
	github.com/spf13/viper v1.3.1
	github.com/stretchr/testify v1.4.0 // indirect
	github.com/tmc/grpc-websocket-proxy v0.0.0-20171017195756-830351dc03c6 // indirect
	github.com/unrolled/render v0.0.0-20180914162206-b9786414de4d // indirect
	go.etcd.io/etcd v0.0.0-00010101000000-000000000000 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974
	golang.org/x/sys v0.0.0-20220627191245-f75cf1eec38b // indirect
	golang.org/x/text v0.3.3
	golang.org/x/time v0.0.0-20181108054448-85acf8d2951c // indirect
	google.golang.org/grpc v1.23.0
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	modernc.org/mathutil v1.4.1
	vitess.io/vitess v2.1.1+incompatible
)

PROJECT=goInception
GOPATH ?= $(shell go env GOPATH)

# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
endif
FAIL_ON_STDOUT := awk '{ print } END { if (NR > 0) { exit 1 } }'

CURDIR := $(shell pwd)
UNAME_S := $(shell uname -s)
path_to_add := $(addsuffix /bin,$(subst :,/bin:,$(GOPATH)))
export PATH := $(path_to_add):$(PATH)

GO        := GO111MODULE=on go
GOBUILD   := CGO_ENABLED=0 $(GO) build $(BUILD_FLAG)

VERSION := $(shell git describe --tags --dirty)

VERSION_EASY := $(shell git describe --tags)

# 指定部分单元测试跳过
ifeq ("$(SHORT)", "1")
	GOTEST    := CGO_ENABLED=1 $(GO) test -p 3 -short
else
	GOTEST    := CGO_ENABLED=1 $(GO) test -p 3
endif

ifeq "$(GOVERALLS_SERVICE)" ""
	GOVERALLS_SERVICE    := "circle-ci"
	GOVERALLS_PROJECT	 := "/home/circleci/go/src/github.com/hanchuanchuan/goInception"
else
	GOVERALLS_SERVICE    := "github"
	GOVERALLS_PROJECT	 := "/home/runner/work/goInception/goInception"
endif

OVERALLS  := CGO_ENABLED=1 GO111MODULE=on overalls
GOVERALLS := goveralls

ARCH      := "`uname -s`"
LINUX     := "Linux"
MAC       := "Darwin"
PACKAGE_LIST  := go list ./...| grep -vE "vendor"
PACKAGES  := $$($(PACKAGE_LIST))
PACKAGE_DIRECTORIES := $(PACKAGE_LIST) | sed 's|github.com/hanchuanchuan/$(PROJECT)/||'
FILES     := $$(find $$($(PACKAGE_DIRECTORIES)) -name "*.go")

GOFAIL_ENABLE  := $$(find $$PWD/ -type d | grep -vE "(\.git|vendor)" | xargs gofail enable)
GOFAIL_DISABLE := $$(find $$PWD/ -type d | grep -vE "(\.git|vendor)" | xargs gofail disable)

LDFLAGS += -X "github.com/hanchuanchuan/goInception/mysql.TiDBReleaseVersion=$(shell git describe --tags --dirty)"
LDFLAGS += -X "github.com/hanchuanchuan/goInception/util/printer.TiDBBuildTS=$(shell date '+%Y-%m-%d %H:%M:%S')"
LDFLAGS += -X "github.com/hanchuanchuan/goInception/util/printer.TiDBGitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "github.com/hanchuanchuan/goInception/util/printer.TiDBGitBranch=$(shell git rev-parse --abbrev-ref HEAD)"
LDFLAGS += -X "github.com/hanchuanchuan/goInception/util/printer.GoVersion=$(shell go version)"

TEST_LDFLAGS =  -X "github.com/hanchuanchuan/goInception/config.checkBeforeDropLDFlag=1"

CHECK_LDFLAGS += $(LDFLAGS) ${TEST_LDFLAGS}

TARGET = ""

.PHONY: all build update parser clean todo test gotest interpreter server dev benchkv benchraw check parserlib checklist testapi docs level

default: server buildsucc

server-admin-check: server_check buildsucc

buildsucc:
	@echo Build TiDB Server successfully!

all: dev server benchkv

# dev: checklist parserlib test check
dev: checklist parserlib test

build:
	$(GOBUILD)

goyacc:
	@$(GOBUILD) -o bin/goyacc parser/goyacc/main.go

bin/goyacc: parser/goyacc/main.go parser/goyacc/format_yacc.go
	$(GO) mod download
	$(GO) build -o bin/goyacc parser/goyacc/main.go parser/goyacc/format_yacc.go

parser: parser/parser.go parser/hintparser.go

parser/parser.go: parser/parser.y bin/goyacc
	@echo "bin/goyacc -o $@ -p yy -t Parser $<"
	@bin/goyacc -o $@ -p yy -t Parser $< && echo 'SUCCESS!' || ( rm -f $@ && echo 'Please check y.output for more information' && exit 1 )
	@rm -f y.output
	# Clean invalid UTF-8 encoding at the end

	echo "os: ${UNAME_S}"
ifeq ($(UNAME_S),Darwin)
	sed -i '' '$$d' $@;
else
	sed -i '$$d' $@;
endif
	gofmt -s -w $@

parser/hintparser.go: parser/hintparser.y bin/goyacc
	@echo "bin/goyacc -o $@ -p yyhint -t hintParser $<"
	@bin/goyacc -o $@ -p yyhint -t hintParser $< && echo 'SUCCESS!' || ( rm -f $@ && echo 'Please check y.output for more information' && exit 1 )
	@rm -f y.output
	# Clean invalid UTF-8 encoding at the end

	echo "os: ${UNAME_S}"
ifeq ($(UNAME_S),Darwin)
	sed -i '' '$$d' $@;
else
	sed -i '$$d' $@;
endif
	gofmt -s -w $@

# %arser.go: prefix = $(@:parser.go=)
# %arser.go: %arser.y bin/goyacc
# 	@echo "bin/goyacc -o $@ -p yy$(prefix) -t $(prefix)Parser $<"
# 	@bin/goyacc -o $@ -p yy$(prefix) -t $(prefix)Parser $< && echo 'SUCCESS!' || ( rm -f $@ && echo 'Please check y.output for more information' && exit 1 )
# 	@rm -f y.output

%arser.go: prefix = $(@:parser.go=)
%arser.go: %arser.y bin/goyacc
	@echo "bin/goyacc -o $@ -p yy$(prefix) -t $(prefix)Parser $<"
	@bin/goyacc -o $@ -p yy$(prefix) -t $(prefix)Parser $< && echo 'SUCCESS!' || ( rm -f $@ && echo 'Please check y.output for more information' && exit 1 )
	@rm -f y.output


%arser_golden.y: %arser.y
	@bin/goyacc -fmt -fmtout $@ $<
	@(git diff --no-index --exit-code $< $@ && rm $@) || (mv $@ $< && >&2 echo "formatted $<" && exit 1)


# parser: goyacc
# 	bin/goyacc -o /dev/null parser/parser.y
# 	@bin/goyacc -o parser/parser.go parser/parser.y 2>&1 | egrep "(shift|reduce)/reduce" | awk '{print} END {if (NR > 0) {print "Find conflict in parser.y. Please check y.output for more information."; exit 1;} else {print "SUCCESS!"}}'
# 	@rm -f y.output

	# @if [ $(ARCH) = $(LINUX) ]; \
	# then \
	# 	sed -i -e 's|//line.*||' -e 's/yyEofCode/yyEOFCode/' parser/parser.go; \
	# elif [ $(ARCH) = $(MAC) ]; \
	# then \
	# 	/usr/bin/sed -i "" 's|//line.*||' parser/parser.go; \
	# 	/usr/bin/sed -i "" 's/yyEofCode/yyEOFCode/' parser/parser.go; \
	# fi

	# @awk 'BEGIN{print "// Code generated by goyacc"} {print $0}' parser/parser.go > tmp_parser.go && mv tmp_parser.go parser/parser.go;

parserlib: parser/parser.go

# parser/parser.go: parser/parser.y
# 	make parser

# Install the check tools.
check-setup:tools/bin/revive tools/bin/goword tools/bin/gometalinter tools/bin/gosec
# @which retool >/dev/null 2>&1 || go get github.com/twitchtv/retool
# @retool sync

check: check-setup fmt lint vet fmt-parser

# These need to be fixed before they can be ran regularly
check-fail: goword check-static check-slow

fmt:
	@echo "gofmt (simplify)"
	@gofmt -s -l -w $(FILES) 2>&1 | $(FAIL_ON_STDOUT)

# hint_golden.y
fmt-parser: bin/goyacc parser_golden.y
	@echo "gofmt (simplify)"
	@gofmt -s -l -w . 2>&1 | awk '{print} END{if(NR>0) {exit 1}}'

%arser_golden.y: parser/%arser.y
	@bin/goyacc -fmt -fmtout $@ $<
	@(git diff --no-index --exit-code $< $@ && rm $@) || (mv $@ $< && >&2 echo "formatted $<" && exit 1)

goword:tools/bin/goword
	tools/bin/goword $(FILES) 2>&1 | $(FAIL_ON_STDOUT)

check-static:
	@ # vet and fmt have problems with vendor when ran through metalinter
	CGO_ENABLED=0 retool do gometalinter.v2 --disable-all --deadline 120s \
	  --enable misspell \
	  --enable megacheck \
	  --enable ineffassign \
	  $$($(PACKAGE_DIRECTORIES))

check-slow:
	CGO_ENABLED=0 retool do gometalinter.v2 --disable-all \
	  --enable errcheck \
	  $$($(PACKAGE_DIRECTORIES))
	CGO_ENABLED=0 retool do gosec $$($(PACKAGE_DIRECTORIES))

lint:tools/bin/revive
	@echo "linting"
	@tools/bin/revive -formatter friendly -config tools/check/revive.toml ./...

vet:
	@echo "vet"
	$(GO) vet -all $(PACKAGES) 2>&1 | $(FAIL_ON_STDOUT)

clean:
	$(GO) clean -i ./...
	rm -rf *.out

todo:
	@grep -n ^[[:space:]]*_[[:space:]]*=[[:space:]][[:alpha:]][[:alnum:]]* */*.go parser/parser.y || true
	@grep -n TODO */*.go parser/parser.y || true
	@grep -n BUG */*.go parser/parser.y || true
	@grep -n println */*.go parser/parser.y || true

test: checklist gotest explaintest

explaintest: server
	@cd cmd/explaintest && ./run-tests.sh -s ../../bin/goInception

gotest: parserlib
	$(GO) install github.com/etcd-io/gofail@v0.0.0-20180808172546-51ce9a71510a
	@$(GOFAIL_ENABLE)
ifeq ("$(TRAVIS_COVERAGE)", "1")
	@echo "Running in TRAVIS_COVERAGE mode."
	@export log_level=error; \
	go install github.com/go-playground/overalls@7df9f728c018
	# go get github.com/mattn/goveralls
	# $(OVERALLS) -project=github.com/hanchuanchuan/goInception -covermode=count -ignore='.git,vendor,cmd,docs,LICENSES' || { $(GOFAIL_DISABLE); exit 1; }
	# $(GOVERALLS) -service=$(GOVERALLS_SERVICE) -coverprofile=overalls.coverprofile || { $(GOFAIL_DISABLE); exit 1; }

	$(OVERALLS) -project=$(GOVERALLS_PROJECT) -covermode=count -ignore='.git,vendor,cmd,docs,LICENSES' -concurrency=1 -- -short || { $(GOFAIL_DISABLE); exit 1; }
else

ifeq ("$(API)", "1")
	@echo "Running in native mode (API)."
	@export log_level=error;
	$(GOTEST) -timeout 30m -ldflags '$(TEST_LDFLAGS)' github.com/hanchuanchuan/goInception/session -api
else
	@echo "Running in native mode."
	@export log_level=error;
	$(GO) mod tidy;
	$(GOTEST) -timeout 30m -ldflags '$(TEST_LDFLAGS)' -cover $(PACKAGES) || { $(GOFAIL_DISABLE); exit 1; }
endif

endif
	@$(GOFAIL_DISABLE)

testapi: parserlib
	@echo "Running in native mode (API)."
	@export log_level=error;
	$(GOTEST) -timeout 30m -ldflags '$(TEST_LDFLAGS)' github.com/hanchuanchuan/goInception/session -api


race: parserlib
	$(GO) install github.com/etcd-io/gofail@v0.0.0-20180808172546-51ce9a71510a
	@$(GOFAIL_ENABLE)
	@export log_level=debug; \
	$(GOTEST) -timeout 30m -race $(PACKAGES) || { $(GOFAIL_DISABLE); exit 1; }
	@$(GOFAIL_DISABLE)

leak: parserlib
	$(GO) install github.com/etcd-io/gofail@v0.0.0-20180808172546-51ce9a71510a
	@$(GOFAIL_ENABLE)
	@export log_level=debug; \
	$(GOTEST) -tags leak $(PACKAGES) || { $(GOFAIL_DISABLE); exit 1; }
	@$(GOFAIL_DISABLE)

tikv_integration_test: parserlib
	$(GO) install github.com/etcd-io/gofail@v0.0.0-20180808172546-51ce9a71510a
	@$(GOFAIL_ENABLE)
	$(GOTEST) ./store/tikv/. -with-tikv=true || { $(GOFAIL_DISABLE); exit 1; }
	@$(GOFAIL_DISABLE)

RACE_FLAG =
ifeq ("$(WITH_RACE)", "1")
	RACE_FLAG = -race
	GOBUILD   = GOPATH=$(GOPATH) CGO_ENABLED=1 $(GO) build
endif

CHECK_FLAG =
ifeq ("$(WITH_CHECK)", "1")
	CHECK_FLAG = $(TEST_LDFLAGS)
endif

server: parserlib
ifeq ($(TARGET), "")
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/goInception tidb-server/main.go
else
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o '$(TARGET)' tidb-server/main.go
endif

server_check: parserlib
ifeq ($(TARGET), "")
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(CHECK_LDFLAGS)' -o bin/goInception tidb-server/main.go
else
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(CHECK_LDFLAGS)' -o '$(TARGET)' tidb-server/main.go
endif

benchkv:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/benchkv cmd/benchkv/main.go

benchraw:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/benchraw cmd/benchraw/main.go

benchdb:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/benchdb cmd/benchdb/main.go

importer:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o bin/importer ./cmd/importer

update:
	which dep 2>/dev/null || go get -u github.com/golang/dep/cmd/dep
ifdef PKG
	dep ensure -add ${PKG}
else
	dep ensure -update
endif
	@echo "removing test files"
	dep prune
	bash ./hack/clean_vendor.sh

checklist:
	cat checklist.md

gofail-enable:
# Converting gofail failpoints...
	@$(GOFAIL_ENABLE)

gofail-disable:
# Restoring gofail failpoints...
	@$(GOFAIL_DISABLE)

upload-coverage: SHELL:=/bin/bash
upload-coverage:
ifeq ("$(TRAVIS_COVERAGE)", "1")
	mv overalls.coverprofile coverage.txt
	bash <(curl -s https://codecov.io/bash)
endif

tools/bin/revive: tools/check/go.mod
	cd tools/check; \
	$(GO) build -o ../bin/revive github.com/mgechev/revive

tools/bin/goword: tools/check/go.mod
	cd tools/check; \
	$(GO) build -o ../bin/goword github.com/chzchzchz/goword

tools/bin/gometalinter: tools/check/go.mod
	cd tools/check; \
	$(GO) build -o ../bin/gometalinter gopkg.in/alecthomas/gometalinter.v3

tools/bin/gosec: tools/check/go.mod
	cd tools/check; \
	$(GO) build -o ../bin/gosec github.com/securego/gosec/cmd/gosec

# 	windows无法build,github.com/outbrain/golib有引用syslog.Writer,其在windows未实现.
.PHONY: release
release:
	@echo "$(CGREEN)Cross platform building for release ...$(CEND)"
	@mkdir -p release
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags '-s -w $(LDFLAGS)'  -o goInception tidb-server/main.go
	@tar -czf release/goInception-macOS-${VERSION}.tar.gz goInception config/config.toml.default
	@rm -f goInception

	@GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags '-s -w $(LDFLAGS)'  -o goInception tidb-server/main.go
	@tar -czf release/goInception-linux-${VERSION}.tar.gz goInception config/config.toml.default
	@rm -f goInception

.PHONY: release
release2:
	@echo "$(CGREEN)Cross platform building for release ...$(CEND)"
	@mkdir -p release

	@GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags '-s -w $(LDFLAGS)'  -o goInception tidb-server/main.go
	@tar -czf release/goInception-macOS-${VERSION_EASY}.tar.gz goInception config/config.toml.default
	@rm -f goInception

	@GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags '-s -w $(LDFLAGS)'  -o goInception tidb-server/main.go
	@tar -czf release/goInception-linux-${VERSION_EASY}.tar.gz goInception config/config.toml.default
	@rm -f goInception


docker:
	@if [ ! -f bin/percona-toolkit.tar.gz ];then \
		wget -O bin/percona-toolkit.tar.gz https://www.percona.com/downloads/percona-toolkit/3.0.4/source/tarball/percona-toolkit-3.0.4.tar.gz; \
	fi
	@if [ ! -f bin/pt-online-schema-change ];then \
		wget -O bin/pt-online-schema-change percona.com/get/pt-online-schema-change; \
	fi
	@if [ ! -f bin/gh-ost ];then \
		wget -O bin/gh-ost.tar.gz https://github.com/github/gh-ost/releases/download/v1.1.0/gh-ost-binary-linux-20200828140552.tar.gz; \
		tar -zxvf bin/gh-ost.tar.gz -C bin/; \
	fi
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags '-s -w $(LDFLAGS)' -o bin/goInception tidb-server/main.go
	v1=$(shell git tag | awk -F'-' '{print $1}' |tail -1) && docker build -t hanchuanchuan/goinception:$${v1} . \
	&& docker tag hanchuanchuan/goinception:$${v1} hanchuanchuan/goinception:latest

docker-push:
	v1=$(shell git tag|tail -1) && docker push hanchuanchuan/goinception:$${v1} \
	&& docker push hanchuanchuan/goinception:latest

docs:
	$(shell bash docs/deploy.sh)

level:
	$(GO) run config/generate_levels/main.go
	gofmt -w config/error_level.go

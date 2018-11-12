BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
HARNESS      := $(TST)harness.go
MAIN         := radiucal.go 
SRC          := $(shell find {core,plugins,tests,server} -type f -name "*.go") $(MAIN)
PLUGINS      := log stats debug usermac naswhitelist
VENDOR_LOCAL := $(PWD)/vendor/github.com/epiphyte/radiucal
VERSION      ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
CMN_FLAGS    :=  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
FLAGS        := $(CMN_FLAGS)pie
PLUGIN_FLAGS := $(CMN_FLAGS)plugin
GO_TESTS     := go test -v
TEST_CONFS   := normal norjct
COMPONENTS   := core server

.PHONY: $(COMPONENTS) tools

all: clean modules radiucal integrate format

modules: $(PLUGINS)
components: $(COMPONENTS)

$(PLUGINS):
	@echo $@
	go build $(PLUGIN_FLAGS) -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go
	cd $(PLUGIN)$@ && $(GO_TESTS)

integrate: harness $(TEST_CONFS)

harness:
	mkdir -p $(TST)plugins/
	cp $(BIN)*.rd $(TST)plugins/
	go build -o $(BIN)harness $(HARNESS)

$(TEST_CONFS):
	mkdir -p $(TST)log/
	rm -f $(TST)log/*
	./tests/run.sh $@

$(COMPONENTS):
	rm -f $(VENDOR_LOCAL)/$@
	ln -s $(PWD)/$@ $(VENDOR_LOCAL)/$@
	cd $@ && $(GO_TESTS)

radiucal: components radiucalbin
	$(GO_TESTS)

radiucalbin:
	go build -o $(BIN)radiucal $(FLAGS) $(MAIN)

format:
	@echo $(SRC)
	exit $(shell gofmt -l $(SRC) | wc -l)

setup:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	rm -rf $(VENDOR_LOCAL)
	mkdir -p $(VENDOR_LOCAL)

clean: setup $(COMPONENTS)

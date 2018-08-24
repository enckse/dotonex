BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
HARNESS      := $(TST)harness.go
GENERATED    := generated.go
MAIN         := radiucal.go 
SRC          := $(shell find . -type f -name "*.go" | grep -v "vendor/")
PLUGINS      := log stats debug usermac naswhitelist
VENDOR_LOCAL := $(PWD)/vendor/github.com/epiphyte/radiucal
VERSION      ?= $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS        := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
PLUGIN_FLAGS := --buildmode=plugin -ldflags '-s -w'
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
	exit $(shell echo $(SRC) | grep "\.go$$" | goimports -l $(SRC) | wc -l)

setup:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	rm -rf $(VENDOR_LOCAL)
	mkdir -p $(VENDOR_LOCAL)

clean: setup $(COMPONENTS)

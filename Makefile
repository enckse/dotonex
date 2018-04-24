BIN=bin/
TST=tests/
PLUGIN=plugins/
HARNESS=$(TST)harness.go
MAIN=radiucal.go context.go
SRC=$(MAIN) $(shell find $(PLUGIN) -type f | grep "\.go$$") $(HARNESS)
PLUGINS=$(shell ls $(PLUGIN) | grep -v "common.go")

VERSION=
ifeq ($(VERSION),)
	VERSION=master
endif
export GOPATH := $(PWD)/vendor
.PHONY: tools plugins

all: clean plugins radiucal integrate tools format

deps:
	git submodule update --init --recursive

plugins: $(PLUGINS)

$(PLUGINS):
	@echo $@
	go build --buildmode=plugin -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go
	cd $(PLUGIN)$@ && go test -v

integrate:
	mkdir -p $(TST)plugins/
	mkdir -p $(TST)log/
	rm -f $(TST)log/*
	cp $(BIN)*.rd $(TST)plugins/
	go build -o $(BIN)harness $(HARNESS)
	./tests/run.sh

radiucal:
	go test -v
	go build -o $(BIN)radiucal -ldflags '-X main.vers=$(VERSION)' $(MAIN)

format:
	@echo $(SRC)
	exit $(shell echo $(SRC) | grep "\.go$$" | gofmt -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)

tools:
	cd tools && make -C .

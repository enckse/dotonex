BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
HARNESS      := $(TST)harness.go
MAIN         := radiucal.go context.go
SRC          := $(shell find . -type f -name "*.go" | grep -v "vendor/")
PLUGINS      := log stats trace usermac
VENDOR_LOCAL := $(PWD)/vendor/github.com/epiphyte/radiucal
VERSION      := master
FLAGS        := -ldflags '-s -w -X main.vers=$(VERSION)'
PLUGIN_FLAGS := --buildmode=plugin -ldflags '-s -w'
GO_TESTS     := go test -v

.PHONY: tools plugins

all: clean plugins radiucal integrate tools format

plugins: $(PLUGINS)

$(PLUGINS):
	@echo $@
	go build $(PLUGIN_FLAGS) -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go
	cd $(PLUGIN)$@ && $(GO_TESTS)

integrate:
	mkdir -p $(TST)plugins/
	mkdir -p $(TST)log/
	rm -f $(TST)log/*
	cp $(BIN)*.rd $(TST)plugins/
	go build -o $(BIN)harness $(HARNESS)
	./tests/run.sh

radiucal:
	$(GO_TESTS)
	go build -o $(BIN)radiucal $(FLAGS) $(MAIN)

format:
	@echo $(SRC)
	exit $(shell echo $(SRC) | grep "\.go$$" | goimports -l $(SRC) | wc -l)

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	mkdir -p $(VENDOR_LOCAL)
	rm -f $(VENDOR_LOCAL)/plugins
	ln -s $(PWD)/$(PLUGIN) $(VENDOR_LOCAL)/plugins

tools:
	cd tools && make -C .

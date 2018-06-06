BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
HARNESS      := $(TST)harness.go
MAIN         := radiucal.go context.go
SRC          := $(shell find . -type f -name "*.go" | grep -v "vendor/")
PLUGINS      := log stats debug usermac
VENDOR_LOCAL := $(PWD)/vendor/github.com/epiphyte/radiucal
VERSION      := $(shell git describe --long | sed "s/\([^-]*-g\)/r\1/;s/-/./g")
FLAGS        := -ldflags '-s -w -X main.vers=$(VERSION)' -buildmode=pie
PLUGIN_FLAGS := --buildmode=plugin -ldflags '-s -w'
GO_TESTS     := go test -v
PY           := $(shell find . -type f -name "*.py" | grep -v "\_\_init\_\_.py")
TEST_CONFS   := normal norjct

.PHONY: tools plugins

all: clean plugins radiucal scripts integrate tools format

plugins: $(PLUGINS)

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
	pycodestyle $(PY)
	pep257 $(PY)
	cd tools/tests && ./check.sh

scripts:
	m4 -DVERSION='"$(VERSION)"' tools/configure.sh.in > $(BIN)configure

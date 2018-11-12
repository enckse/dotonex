BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
SRC          := $(shell find . -type f -name "*.go" | grep -v "vendor/")
PLUGINS      := log stats debug usermac naswhitelist
VERSION      := DEVELOP
CMN_FLAGS    :=  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
FLAGS        := $(CMN_FLAGS)pie
PLUGIN_FLAGS := $(CMN_FLAGS)plugin
TEST_CONFS   := normal norjct
COMPONENTS   := core server

.PHONY: $(COMPONENTS) tools

all: clean modules radiucal test format

modules: $(PLUGINS)
components: $(COMPONENTS)

$(PLUGINS):
	go build $(PLUGIN_FLAGS) -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go

test: utests integrate

utests:
	for f in $(shell find . -type f -name "*_test.go" -exec dirname {} \;); do go test -v $$f; done

integrate: harness $(TEST_CONFS)

harness:
	rm -f $(TST)/plugins/*
	cp $(BIN)*.rd $(TST)plugins/
	go build -o $(BIN)harness $(TST)harness.go

$(TEST_CONFS):
	rm -f $(TST)log/*
	./tests/run.sh $@

$(COMPONENTS):
	ln -sf $(PWD)/$@ $(PWD)/vendor/github.com/epiphyte/radiucal/

radiucal:
	go build -o $(BIN)radiucal $(FLAGS) radiucal.go

format:
	@echo $(SRC)
	exit $(shell gofmt -l $(SRC) | wc -l)

setup:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	mkdir -p $(TST)plugins/
	mkdir -p $(TST)log/

clean: setup components

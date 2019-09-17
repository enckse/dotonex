BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
PLUGINS      := $(shell ls $(PLUGIN))
VERSION      := $(BUILD_VERSION)
ifeq ($(VERSION),)
	VERSION  := DEVELOP
	CHECK_RUST := $(VERSION)
else
	CHECK_RUST := $(shell cat Cargo.toml | grep "version = " | grep $(VERSION) | cut -d "=" -f 2 | sed 's/"//g' | sed "s/ //g")
endif
CMN_FLAGS    :=  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
FLAGS        := $(CMN_FLAGS)pie
PLUGIN_FLAGS := $(CMN_FLAGS)plugin
TEST_CONFS   := normal norjct
ADMIN        := admin
UTESTS       := $(shell find . -type f -name "*_test.go")

.PHONY: $(UTESTS)

all: clean modules radiucal $(ADMIN) test format

modules: $(PLUGINS)

$(PLUGINS):
	go build $(PLUGIN_FLAGS) -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go

test: utests integrate

$(ADMIN):
ifneq ($(CHECK_RUST),$(VERSION))
	$(error "administrative version mismatch $(CHECK_RUST) != $(VERSION)")
endif
	go build -o $(BIN)radiucal-lua-bridge $(FLAGS) admin.go
	cargo build --release
	cp target/release/radiucal-admin $(BIN)
	cd $(TST)$(ADMIN) && ./run.sh

utests: $(UTESTS)

$(UTESTS):
	go test -v $(shell dirname $@)/*.go

integrate: harness $(TEST_CONFS)

harness:
	rm -f $(TST)/plugins/*
	cp $(BIN)*.rd $(TST)plugins/
	go build -o $(BIN)harness $(TST)harness.go

$(TEST_CONFS):
	rm -f $(TST)log/*
	./tests/run.sh $@

radiucal:
	go build -o $(BIN)radiucal $(FLAGS) radiucal.go

format:
	goformatter
	cargo clippy

setup:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	mkdir -p $(TST)plugins/
	mkdir -p $(TST)log/

clean: setup

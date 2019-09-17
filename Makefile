BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
PLUGINS      := $(shell ls $(PLUGIN))
VERSION      ?= master
ifeq ($(VERSION),master)
	CHECK_RUST := $(VERSION)
else
	CHECK_RUST := $(shell cat Cargo.toml | grep "version = " | grep $(VERSION) | cut -d "=" -f 2 | sed 's/"//g' | sed "s/ //g")
endif
FLAGS    :=  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
TEST_CONFS   := normal norjct
UTESTS       := $(shell find . -type f -name "*_test.go")
EXES         := $(BIN)radiucal $(BIN)radiucal-lua-bridge

.PHONY: $(UTESTS)

all: clean modules $(EXES) admin test format

modules: $(PLUGINS)

$(PLUGINS):
	go build $(FLAGS)plugin -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go

test: utests $(TEST_CONFS)

admin: $(LUA_BRIDGE)
ifneq ($(CHECK_RUST),$(VERSION))
	$(error "administrative version mismatch $(CHECK_RUST) != $(VERSION)")
endif
	cargo build --release
	cp target/release/radiucal-admin $(BIN)
	cd $(TST)admin && ./run.sh

utests: $(UTESTS)

$(UTESTS):
	go test -v $(shell dirname $@)/*.go

harness:
	rm -f $(TST)/plugins/*
	cp $(BIN)*.rd $(TST)plugins/
	go build -o $(BIN)harness $(TST)harness.go

$(TEST_CONFS): harness
	rm -f $(TST)log/*
	./tests/run.sh $@

$(EXES): radiucal.go radiucal-lua-bridge.go
	go build -o $@ $(FLAGS)pie $(shell echo $@ | sed "s|$(BIN)||g").go

format:
	goformatter
	cargo clippy

clean:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	mkdir -p $(TST)plugins/
	mkdir -p $(TST)log/

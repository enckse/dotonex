BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
PLUGINS      := $(shell ls $(PLUGIN))
VERSION      ?= master
ifneq ($(VERSION),master)
	CHECK_RUST := $(shell cat Cargo.toml | grep "version = " | grep $(VERSION) | cut -d "=" -f 2 | sed 's/"//g' | sed "s/ //g")
endif
CHECK_RUST   ?= $(VERSION)
FLAGS        := -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
TEST_CONFS   := normal norjct
UTESTS       := $(shell find . -type f -name "*_test.go")
EXES         := $(BIN)radiucal $(BIN)radiucal-lua-bridge
RADIUCAL_ADM := $(BIN)radiucal-admin

.PHONY: $(UTESTS)

all: clean $(PLUGINS) $(EXES) admin test format

modules: $(PLUGINS)

$(PLUGINS):
	go build $(FLAGS)plugin -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go

test: $(UTESTS) $(TEST_CONFS)

admin: $(EXES) $(RADIUCAL_ADM)
	cd $(TST)admin && ./run.sh

$(RADIUCAL_ADM): $(shell find src/ -type f -name "*.rs")
ifneq ($(CHECK_RUST),$(VERSION))
	$(error "administrative version mismatch $(CHECK_RUST) != $(VERSION)")
endif
	cargo build --release
	cp target/release/radiucal-admin $@

$(UTESTS):
	go test -v $(shell dirname $@)/*.go

harness:
	go build -o $(BIN)harness $(TST)harness.go

$(TEST_CONFS): harness
	./tests/run.sh $@

$(EXES): radiucal.go radiucal-lua-bridge.go
	go build -o $@ $(FLAGS)pie $(shell echo $@ | sed "s|$(BIN)||g").go

format:
	goformatter
	cargo clippy

clean:
	rm -rf $(BIN)

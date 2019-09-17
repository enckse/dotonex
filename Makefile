BIN          := bin/
PLUGIN       := plugins/
PLUGINS      := $(shell ls $(PLUGIN))
VERSION      ?= master
ifneq ($(VERSION),master)
	CHECK_RUST := $(shell cat Cargo.toml | grep "version = " | grep $(VERSION) | cut -d "=" -f 2 | sed 's/"//g' | sed "s/ //g")
endif
CHECK_RUST   ?= $(VERSION)
FLAGS        := -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
UTESTS       := $(shell find . -type f -name "*_test.go")
EXES         := $(BIN)radiucal $(BIN)radiucal-lua-bridge $(BIN)harness
RADIUCAL_ADM := $(BIN)radiucal-admin

.PHONY: $(UTESTS)

all: clean $(PLUGINS) $(EXES) admin test format

modules: $(PLUGINS)

$(PLUGINS):
	go build $(FLAGS)plugin -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go

test: $(UTESTS)
	./tests/run.sh normal
	./tests/run.sh norjct
	cd tests/admin && ./run.sh

admin: $(EXES) $(RADIUCAL_ADM)

$(RADIUCAL_ADM): $(shell find src/ -type f -name "*.rs")
ifneq ($(CHECK_RUST),$(VERSION))
	$(error "administrative version mismatch $(CHECK_RUST) != $(VERSION)")
endif
	cargo build --release
	cp target/release/radiucal-admin $@

$(UTESTS):
	go test -v $(shell dirname $@)/*.go

$(EXES): cmd/*.go
	go build -o $@ $(FLAGS)pie cmd/$(shell echo $@ | sed "s|$(BIN)||g").go

format:
	goformatter
	cargo clippy

clean:
	rm -rf $(BIN)

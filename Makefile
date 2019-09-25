BIN          := bin/
PLUGIN       := plugins/
PLUGINS      := $(shell ls $(PLUGIN) | grep -v "test" | cut -d "." -f 1)
VERSION      ?= master
ifneq ($(VERSION),master)
	CHECK_RUST := $(shell cat Cargo.toml | grep "version = " | grep $(VERSION) | cut -d "=" -f 2 | sed 's/"//g' | sed "s/ //g")
endif
CHECK_RUST   ?= $(VERSION)
FLAGS        := -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
EXES         := $(BIN)radiucal $(BIN)radiucal-lua-bridge $(BIN)harness
RADIUCAL_ADM := $(BIN)radiucal-admin
UTESTS       := $(shell find core -type f -name "*_test.go")

.PHONY: $(UTESTS) all build modules test admin lint clean

all: clean build test lint

build: modules admin

modules: $(PLUGINS)

$(PLUGINS):
	go build $(FLAGS)plugin -o $(BIN)$@.rd $(PLUGIN)$@.go
	test ! -s $(PLUGIN)$@_test.go || go test -v $(PLUGIN)$@_test.go $(PLUGIN)$@.go

$(UTESTS):
	go test -v $@ $(shell ls core/*.go | grep -v test)

test: $(UTESTS)
	make -C tests

admin: $(EXES) $(RADIUCAL_ADM)

$(RADIUCAL_ADM): $(shell find src/ -type f -name "*.rs")
ifneq ($(CHECK_RUST),$(VERSION))
	$(error "administrative version mismatch $(CHECK_RUST) != $(VERSION)")
endif
	cargo build --release
	cp target/release/radiucal-admin $@

$(EXES): $(shell find . -type f -name "*.go")
	go build -o $@ $(FLAGS)pie cmd/$(shell echo $@ | sed "s|$(BIN)||g").go

lint:
	@golinter
	cargo clippy

clean:
	rm -rf $(BIN)

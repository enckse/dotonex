PLUGINS      := access.rd debug.rd log.rd naswhitelist.rd stats.rd usermac.rd
VERSION      ?= master
ifneq ($(VERSION),master)
	CHECK_RUST := $(shell cat Cargo.toml | grep "version = " | grep $(VERSION) | cut -d "=" -f 2 | sed 's/"//g' | sed "s/ //g")
endif
CHECK_RUST   ?= $(VERSION)
FLAGS        := -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
EXES         := radiucal radiucal-lua-bridge harness
UTESTS       := $(shell find core/ -type f -name "*_test.go")
SRC          := $(shell find . -type f -name "*.go" | grep -v "test")

.PHONY: $(UTESTS) build test lint clean

build: $(PLUGINS) $(EXES) radiucal-admin test lint

$(PLUGINS): $(SRC)
	go build $(FLAGS)plugin -o $@ plugins/$@.go
	test ! -s plugins/$@_test.go || go test -v plugins/$@_test.go plugins/$@.go

$(UTESTS):
	go test -v $@ $(shell ls core/*.go | grep -v test)

test: $(UTESTS)
	make -C tests

radiucal-admin: $(shell find src/ -type f -name "*.rs")
ifneq ($(CHECK_RUST),$(VERSION))
	$(error "administrative version mismatch $(CHECK_RUST) != $(VERSION)")
endif
	cargo build --release

$(EXES): $(SRC)
	go build -o $@ $(FLAGS)pie cmd/$@.go

lint:
	@golinter
	cargo clippy

clean:
	rm -rf $(EXECS) radiucal-admin $(PLUGINS)

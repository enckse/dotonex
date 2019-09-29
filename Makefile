VERSION      ?= master
ifneq ($(VERSION),master)
	CHECK_RUST := $(shell cat Cargo.toml | grep "version = " | grep $(VERSION) | cut -d "=" -f 2 | sed 's/"//g' | sed "s/ //g")
endif
CHECK_RUST   ?= $(VERSION)
FLAGS        := -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=pie
EXES         := radiucal radiucal-lua-bridge
UTESTS       := $(shell find internal/ -type f -name "*_test.go" | xargs dirname | sort -u)
SRC          := $(shell find . -type f -name "*.go" | grep -v "test")

.PHONY: $(UTESTS) build test lint clean

build: $(EXES) radiucal-admin test lint

$(UTESTS):
	cd $@ && go test -v

test: $(UTESTS)
	make -C tests

radiucal-admin: $(shell find src/ -type f -name "*.rs")
ifneq ($(CHECK_RUST),$(VERSION))
	$(error "administrative version mismatch $(CHECK_RUST) != $(VERSION)")
endif
	cargo build --release

$(EXES): $(SRC)
	go build -o $@ $(FLAGS) cmd/$@/main.go

lint:
	@golinter
	cargo clippy

clean:
	rm -rf $(EXES) radiucal-admin

VERSION      ?= master
FLAGS        := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -gcflags=all=-trimpath=$(PWD) -asmflags=all=-trimpath=$(PWD) -ldflags=-extldflags=-zrelro -ldflags=-extldflags=-znow  -buildmode=pie
EXES         := radiucal
UTESTS       := $(shell find internal/ -type f -name "*_test.go" | xargs dirname | sort -u)
SRC          := $(shell find . -type f -name "*.go" | grep -v "test")

.PHONY: $(UTESTS) build test lint clean

build: $(EXES) test lint

$(UTESTS):
	cd $@ && go test -v

test: $(UTESTS)
	make -C tests

$(EXES): $(SRC)
	go build -o $@ $(FLAGS) cmd/$@/main.go

lint:
	@golinter

clean:
	rm -rf $(EXES) radiucal-admin

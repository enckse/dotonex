BIN          := bin/
TST          := tests/
PLUGIN       := plugins/
SRC          := $(shell find . -type f -name "*.go")
PLUGINS      := log stats debug usermac naswhitelist
VERSION      := DEVELOP
CMN_FLAGS    :=  -gcflags=all=-trimpath=$(GOPATH) -asmflags=all=-trimpath=$(GOPATH) -ldflags '-linkmode external -extldflags '$(LDFLAGS)' -s -w -X main.vers=$(VERSION)' -buildmode=
FLAGS        := $(CMN_FLAGS)pie
PLUGIN_FLAGS := $(CMN_FLAGS)plugin
TEST_CONFS   := normal norjct
VAR_LIB      := /var/lib/radiucal/
VAR_PLUG     := $(VAR_LIB)plugins/
ETC          := /etc/radiucal/
ETC_CERTS    := $(ETC)certs/
SUPPORT      := supporting/
SYSD         := /lib/systemd/system/
TMPD         := /usr/lib/tmpfiles.d/

.PHONY: admin

all: clean modules radiucal admin test format

modules: $(PLUGINS)

$(PLUGINS):
	go build $(PLUGIN_FLAGS) -o $(BIN)$@.rd $(PLUGIN)$@/plugin.go

test: utests integrate

admin:
	cd admin && make FLAGS="$(FLAGS)"

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

radiucal:
	go build -o $(BIN)radiucal $(FLAGS) radiucal.go

format:
	@echo $(SRC)
	exit $(shell goimports -l $(SRC) | wc -l)

setup:
	rm -rf $(BIN)
	mkdir -p $(BIN)
	mkdir -p $(TST)plugins/
	mkdir -p $(TST)log/

clean: setup

install:
	install -Dm755 $(BIN)radiucal $(DESTDIR)/usr/bin/radiucal
	for f in $(VAR_LIB) $(VAR_PLUG) $(ETC) $(ETC_CERTS) $(SYSD) $(TMPD); do install -Dm755 -d $(DESTDIR)$$f; done
	for f in ca.cnf Makefile server.cnf xpextensions; do install -Dm644 certs/$$f $(DESTDIR)$(ETC_CERTS); done
	for f in renew.sh bootstrap; do install -Dm755 certs/$$f $(DESTDIR)$(ETC_CERTS); done
	for f in $(shell find $(BIN) -type f -name "*.rd"); do install -Dm644 $$f $(DESTDIR)$(VAR_PLUG); done
	for f in $(shell find $(SUPPORT) -type f -name "*.conf" | cut -d "/" -f 2); do install -Dm644 $(SUPPORT)$$f $(DESTDIR)$(ETC)$$f.ex; done
	for f in $(shell find $(SUPPORT) -type f -name "*.service"); do install -Dm644 $$f $(DESTDIR)$(SYSD); done
	install -Dm644 $(SUPPORT)tmpfiles.d $(DESTDIR)$(TMPD)radiucal.conf

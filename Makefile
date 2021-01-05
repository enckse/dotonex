VERSION     ?= master
FLAGS       := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -trimpath -buildmode=pie -mod=readonly -modcacherw
EXES        := $(shell ls cmd/)
UTESTS      := $(shell find . -type f -name "*_test.go" | xargs dirname | sort -u)
SRC         := $(shell find . -type f -name "*.go" | grep -v "test")
HOSTAP_VERS := hostap_2_9
HOSTAPD     := hostap/hostap/hostapd/hostapd


.PHONY: $(UTESTS) build test lint clean

build: $(EXES) $(HOSTAPD) test lint

$(UTESTS):
	cd $@ && go test -v

$(EXES): $(SRC)
	go build -o $@ $(FLAGS) cmd/$@/main.go

lint:
	@golinter

clean:
	rm -rf $(EXES) radiucal-admin
	rm -rf hostap/hostap

$(HOSTAPD):
	cd hostap && ./configure $(HOSTAP_VERSION)

VERSION     ?= master
FLAGS       := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -trimpath -buildmode=pie -mod=readonly -modcacherw
EXES        := $(shell ls cmd/)
UTESTS      := $(shell find . -type f -name "*_test.go" | xargs dirname | sort -u)
SRC         := $(shell find . -type f -name "*.go" | grep -v "test")
HOSTAP_VERS := hostap_2_9
HOSTAPD     := hostap/hostap/hostapd/hostapd
CONFIG_IN   := grad-daemon.sh hostap/hostapd.conf
LIBRARY     := /var/lib/grad
TEMPLATE    := /etc/grad/hostapd
LIB_HOSTAPD := $(LIBRARY)/hostapd
ACCTPORT    := 1815
AUTHPORT    := 1814

.PHONY: $(UTESTS) $(CONFIG_IN) build test lint clean

build: $(EXES) $(CONFIG_IN) $(HOSTAPD) test lint

$(CONFIG_IN):
	m4 -DGRAD=$(LIBRARY) \
	   -DHOSTAPD=$(LIB_HOSTAPD) \
	   -DTEMPLATE=$(TEMPLATE) \
	   -DCLIENTS=$(LIB_HOSTAPD)/clients \
	   -DGRADKEYS=$(LIBRARY)key \
	   -DAUTHPORT=$(AUTHPORT) \
	   -DACCTPORT=$(ACCTPOR) $@.in > $@

$(UTESTS):
	cd $@ && go test -v

test: $(UTESTS)

$(EXES): $(SRC)
	go build -o $@ $(FLAGS) cmd/$@/main.go

lint:
	@golinter

clean:
	rm -rf $(EXES) radiucal-admin
	rm -rf hostap/hostap

$(HOSTAPD):
	cd hostap && ./configure $(HOSTAP_VERS)

VERSION     ?= master
FLAGS       := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w -X main.vers=$(VERSION)' -trimpath -buildmode=pie -mod=readonly -modcacherw
UTESTS      := $(shell find . -type f -name "*_test.go" | xargs dirname | sort -u)
SRC         := $(shell find . -type f -name "*.go" | grep -v "test")
HOSTAP_VERS := hostap_2_9
HOSTAPD     := hostap/hostap/hostapd/hostapd
CONFIG_IN   := $(shell find . -type f -name "*.in" | cut -d "/" -f 2- | sed "s/\.in//g")
LIBRARY     := /var/lib/grad
ETC         := /etc/grad
TEMPLATE    := $(ETC)/hostapd
LIB_HOSTAPD := $(LIBRARY)/hostapd
SRCAUTHPORT := 1812
SRCACCTPORT := 1813
DSTACCTPORT := 1815
DSTAUTHPORT := 1814
LOGS        := /var/log/grad

.PHONY: $(UTESTS) $(CONFIG_IN) build test lint clean

build: grad $(CONFIG_IN) $(HOSTAPD) test lint

$(CONFIG_IN):
	m4 -DGRAD=$(LIBRARY) \
	   -DHOSTAPD=$(LIB_HOSTAPD) \
	   -DTEMPLATE=$(TEMPLATE) \
	   -DCLIENTS=$(LIB_HOSTAPD)/clients \
	   -DGRADKEYS=$(LIBRARY)key \
	   -DSRCAUTHPORT=$(SRCAUTHPORT) \
	   -DDSTAUTHPORT=$(DSTAUTHPORT) \
	   -DSRCACCTPORT=$(SRCACCTPORT) \
	   -DETCGRAD=$(ETC) \
	   -DLOGS=$(LOGS) \
	   -DLIBRARY=$(LIBRARY) \
	   -DDSTACCTPORT=$(DSTACCTPORT) $@.in > $@

$(UTESTS):
	cd $@ && go test -v

test: $(UTESTS)

grd: $(SRC)
	go build -o grad $(FLAGS) cmd/grad/main.go

lint:
	@golinter

clean:
	rm -rf $(EXES) radiucal-admin
	rm -rf hostap/hostap

$(HOSTAPD):
	cd hostap && ./configure $(HOSTAP_VERS)

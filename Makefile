FLAGS        := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w' -trimpath -buildmode=pie -mod=readonly -modcacherw
EXES         := dotonex dotonex-runner dotonex-compose
UTESTS       := $(shell find . -type f -name "*_test.go" | xargs dirname | sort -u)
SRC          := $(shell find . -type f -name "*.go" | grep -v "test")
HOSTAPD      := hostap/hostap/hostapd/hostapd
HOSTAP_VERS  := hostap_2_9
DESTDIR      :=
SERVER_REPO  :=
RADIUS_KEY   :=
SHARED_KEY   :=
GITLAB_TLD   :=

.PHONY: $(UTESTS)

all: build test

build: $(HOSTAPD) $(EXES) configuration

$(UTESTS):
	cd $@ && go test -v

test: $(UTESTS)
	make -C tests

$(EXES): $(SRC)
	go build -o $@ $(FLAGS) cmd/$@/main.go

$(HOSTAPD):
	cd hostap && ./configure $(HOSTAP_VERS)

clean:
	rm -rf $(EXES)
	rm -rf hostap/hostap

configuration:
	go run tools/generator.go

install:
ifeq ($(SERVER_REPO),)
	$(error "please set SERVER_REPO for server installion")
endif
ifeq ($(SHARED_KEY),)
	$(error "please set SHARED_KEY for server installation")
endif
ifeq ($(GITLAB_TLD),)
	$(error "please set GITLAB_TLD for server installion")
endif
ifeq ($(RADIUS_KEY),)
	$(error "please set RADIUS_KEY for server installion")
endif
	install -dm700 $(DESTDIR)/var/lib/dotonex
	install -dm700 $(DESTDIR)/etc/dotonex/hostapd
	install -dm700 $(DESTDIR)/var/cache/dotonex
	install -dm700 $(DESTDIR)/var/log/dotonex
	echo "127.0.0.1 $(RADIUS_KEY)" > $(DESTDIR)/var/lib/dotonex/clients
	echo "127.0.0.1 $(RADIUS_KEY)" > $(DESTDIR)/var/lib/dotonex/secrets
	echo "0.0.0.0 $(RADIUS_KEY)" >> $(DESTDIR)/var/lib/dotonex/secrets
	echo "export SERVER_REPO=$(SERVER_REPO)" > $(DESTDIR)/etc/dotonex/env
	install -d $(DESTDIR)/usr/lib/dotonex
	install -Dm755 $(HOSTAPD) $(DESTDIR)/usr/lib/dotonex/hostapd
	install -Dm644 hostap/hostapd.conf $(DESTDIR)/etc/dotonex/hostapd/
	install -Dm755 dotonex $(DESTDIR)/usr/bin/
	install -Dm755 dotonex-runner $(DESTDIR)/usr/bin/
	install -Dm755 dotonex-compose $(DESTDIR)/usr/bin/
	install -Dm755 tools/dotonex-daemon $(DESTDIR)/usr/bin/
	install -Dm644 configs/accounting.conf $(DESTDIR)/etc/dotonex/accounting.conf
	install -Dm644 configs/proxy.conf $(DESTDIR)/etc/dotonex/proxy.conf
	install -Dm644 systemd/dotonex.conf $(DESTDIR)/usr/lib/tmpfiles.d/
	install -Dm644 systemd/dotonex.service $(DESTDIR)/usr/lib/systemd/system/
	cp -r hostap/certs $(DESTDIR)/etc/dotonex/hostapd/
	sed -i "s/serverkey: secretkey/serverkey: $(SHARED_KEY)/g" $(DESTDIR)/etc/dotonex/*.conf
	sed -i "s/gitlab.url/$(GITLAB_TLD)/g" $(DESTDIR)/etc/dotonex/*.conf

FLAGS        := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w' -trimpath -buildmode=pie -mod=readonly -modcacherw
EXES         := dotonex dotonex-runner
UTESTS       := $(shell find . -type f -name "*_test.go" | xargs dirname | sort -u)
SRC          := $(shell find . -type f -name "*.go" | grep -v "test")
HOSTAPD      := hostap/hostap/hostapd/hostapd
HOSTAP_VERS  := hostap_2_9
DESTDIR      :=
SERVER_REPO  :=
RADIUS_KEY   :=

.PHONY: $(UTESTS)

build: $(HOSTAPD) $(EXES) test

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

install:
ifeq ($(SERVER_REPO),)
	$(error "please set SERVER_REPO for server installion")
endif
ifeq ($(RADIUS_KEY),)
	$(error "please set RADIUS_KEY for server installation")
endif
	install -d $(DESTDIR)/var/lib/dotonex
	install -d $(DESTDIR)/etc/dotonex/hostapd
	install -d $(DESTDIR)/usr/lib/dotonex
	install -d $(DESTDIR)/var/cache/dotonex
	install -d $(DESTDIR)/var/log/dotonex
	echo "127.0.0.1 $(RADIUS_KEY)" > $(DESTDIR)/var/lib/dotonex/clients
	echo "127.0.0.1 $(RADIUS_KEY)" > $(DESTDIR)/var/lib/dotonex/secrets
	echo "0.0.0.0 $(RADIUS_KEY)" >> $(DESTDIR)/var/lib/dotonex/secrets
	git clone $(SERVER_REPO) $(DESTDIR)/var/cache/dotonex/config
	install -Dm755 $(HOSTAPD) $(DESTDIR)/usr/lib/dotonex/hostapd
	install -Dm644 hostap/hostapd.conf $(DESTDIR)/etc/dotonex/hostapd/
	install -Dm755 dotonex $(DESTDIR)/usr/bin/
	install -Dm755 dotonex-runner $(DESTDIR)/usr/bin/
	install -Dm755 tools/dotonex-config $(DESTDIR)/usr/bin/
	install -Dm755 tools/dotonex-daemon $(DESTDIR)/usr/bin/
	install -Dm644 configs/accounting.conf $(DESTDIR)/etc/dotonex/accounting.conf
	install -Dm644 configs/proxy.conf $(DESTDIR)/etc/dotonex/proxy.conf
	install -Dm644 systemd/dotonex.conf $(DESTDIR)/usr/lib/tmpfiles.d/
	install -Dm644 systemd/dotonex.service $(DESTDIR)/usr/lib/systemd/system/
	cp -r hostap/certs $(DESTDIR)/etc/dotonex/hostapd/

FLAGS        := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w' -trimpath -buildmode=pie -mod=readonly -modcacherw
CLIENT       := authem-configurator authem-passwd
SERVER       := $(CLIENT) radiucal radiucal-runner
EXES         := $(SERVER)
UTESTS       := $(shell find . -type f -name "*_test.go" | xargs dirname | sort -u)
SRC          := $(shell find . -type f -name "*.go" | grep -v "test")
HOSTAPD      := hostap/hostap/hostapd/hostapd
HOSTAP_VERS  := hostap_2_9
SERVER_FILE  := server.txt
DESTDIR      :=
SERVER_REPO  :=
RADIUS_KEY   :=
AUTHEM_KEY   :=

.PHONY: $(UTESTS)

client: $(CLIENT)

install-client:
	install -Dm755 authem-configurator $(DESTDIR)/usr/bin/
	install -Dm755 authem-passwd $(DESTDIR)/usr/bin/

install:
	make install-client
	test -e $(SERVER_FILE) && make install-server

server: client $(HOSTAPD) $(SERVER)
	touch $(SERVER_FILE)

install-server:
ifeq ($(SERVER_REPO),)
	$(error "please set SERVER_REPO for server installion")
endif
ifeq ($(RADIUS_KEY),)
	$(error "please set RADIUS_KEY for server installation")
endif
ifeq ($(AUTHEM_KEY),)
	$(error "please set AUTHEM_KEY for server installation")
endif
	install -d $(DESTDIR)/var/lib/radiucal
	install -d $(DESTDIR)/etc/radiucal/hostapd
	install -d $(DESTDIR)/usr/lib/radiucal
	install -d $(DESTDIR)/etc/authem
	echo "127.0.0.1 $(SECRET_KEY)" > $(DESTDIR)/var/lib/radiucal/clients
	echo "RADIUCAL_REPO=$(SERVER_REPO)" > $(DESTDIR)/etc/radiucal/env
	install -Dm755 $(HOSTAPD) $(DESTDIR)/usr/lib/radiucal/hostapd
	install -Dm755 radiucal $(DESTDIR)/usr/bin/
	install -Dm755 radiucal-runner $(DESTDIR)/usr/bin/
	install -Dm755 radiucal-daemon.sh $(DESTDIR)/usr/bin/radiucal-daemon
	install -Dm644 configs/accounting.conf.example $(DESTDIR)/etc/radiucal/accounting.conf
	install -Dm644 configs/proxy.conf.example $(DESTDIR)/etc/radiucal/proxy.conf
	install -Dm644 configs/systemd/radiucal.conf $(DESTDIR)/usr/lib/tmpfiles.d/
	install -Dm644 configs/systemd/radiucal.service $(DESTDIR)/usr/lib/systemd/system/
	install -Dm644 configs/configurator.yaml.example $(DESTDIR)/etc/authem/configurator.yaml
	sed -i "s/{PASSWORD}/$(AUTHEM_KEY)/g" $(DESTDIR)/etc/authem/configurator.yaml
	cp -r hostapd/certs $(DESTDIR)/etc/radiucal/hostapd/

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

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
SECRET_KEY   :=

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
ifeq ($(SECRET_KEY),)
	$(error "please set SECRET_KEY for server installation")
endif
	mkdir -p $(DESTDIR)/var/lib/radiucal
	mkdir -p $(DESTDIR)/etc/radiucal/
	echo "127.0.0.1 $(SECRET_KEY)" > $(DESTDIR)/var/lib/radiucal/clients
	echo "RADIUCAL_REPO=$(SERVER_REPO)" > $(DESTDIR)/etc/radiucal/env

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

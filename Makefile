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

#build: $(EXES) $(HOSTAPD) test

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

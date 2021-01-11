FLAGS        := -ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w' -trimpath -buildmode=pie -mod=readonly -modcacherw
EXES         := dotonex dotonex-runner
UTESTS       := $(shell find . -type f -name "*_test.go" | xargs dirname | sort -u)
SRC          := $(shell find . -type f -name "*.go" | grep -v "test")
HOSTAPD      := hostap/hostap/hostapd/hostapd
HOSTAP_VERS  := hostap_2_9

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

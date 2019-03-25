BIN      := ../bin/
FLAGS    :=
RELEASE  := release
BUILD    := --locked --$(RELEASE)
ifneq ($(RELEASE),release)
	BUILD   =
	RELEASE = debug
endif

all: clean app test

app:
	go build -o $(BIN)radiucal-admin-legacy $(FLAGS) src/*.go
	cargo build $(BUILD)
	cp target/$(RELEASE)/radiucal-admin $(BIN)

test:
	cd tests && make

clean:
	mkdir -p $(BIN)

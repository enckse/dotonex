BIN      := ../bin/
FLAGS    := 
RELEASE  := release

all: clean app test

app:
	go build -o $(BIN)radiucal-legacy $(FLAGS) src/*.go
	cargo build --$(RELEASE)
	cp target/$(RELEASE)/radiucal-admin $(BIN)

test:
	cd tests && make

clean:
	mkdir -p $(BIN)

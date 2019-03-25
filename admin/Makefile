BIN      := ../bin/
FLAGS    := 

all: clean app test

app:
	go build -o $(BIN)radiucal-legacy $(FLAGS) src/*.go
	cargo build --release
	cp target/release/radiucal-admin $(BIN)

test:
	cd tests && make

clean:
	mkdir -p $(BIN)

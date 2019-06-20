BIN      := ../bin/
FLAGS    := 

all: clean app test

app:
	go build -o $(BIN)radiucal-admin $(FLAGS) src/*.go

test:
	cd tests && make

clean:
	mkdir -p $(BIN)

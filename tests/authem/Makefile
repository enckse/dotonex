AUTHEM_KEY := aaaaaaaabbbbbbbbccccccccdddddddd
BIN        := bin/
EXPECT     := expect/

all: clean run

clean:
	rm -rf $(BIN)
	mkdir $(BIN)

run:
	cp $(EXPECT)* $(BIN)
	AUTHEM_KEY=$(AUTHEM_KEY) ../../grad configurate --run-scripts=true test.sh
	diff -u $(EXPECT)test.stdout $(BIN)script.stdout
	AUTHEM_KEY=$(AUTHEM_KEY) ../../grad configurate

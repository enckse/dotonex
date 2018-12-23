OUT   := ../../bin/
FILES := $(OUT)bin/
USRS  := $(OUT)users/
DIFFS := audit.csv manifest eap_users sysinfo.csv segment-diagram.dot segments.md

.PHONY: $(DIFFS)

all: clean run compare

clean:
	mkdir -p $(USRS)
	mkdir -p $(FILES)
	cp passwords $(USRS)
	cp *.lua $(USRS)

run:
	@cd $(OUT) && ./radiucal-admin --command netconf

$(DIFFS):
	diff -u $@ $(FILES)$@

compare: $(DIFFS)

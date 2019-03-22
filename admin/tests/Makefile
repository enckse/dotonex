OUT   := ../../bin/
FILES := $(OUT)bin/
USRS  := $(OUT)config/
DIFFS := audit.csv manifest eap_users sysinfo.csv segment-diagram.dot segments.md
PATH  := $(OUT):$(shell echo $$PATH)
export PATH

.PHONY: $(DIFFS)

all: clean run compare

clean:
	mkdir -p $(USRS)
	mkdir -p $(FILES)
	cp passwords $(USRS)
	cp *.lua $(USRS)

run:
	@cd $(OUT) && PATH=.; ./radiucal-admin netconf

$(DIFFS):
	diff -u $@ $(FILES)$@

compare: $(DIFFS)

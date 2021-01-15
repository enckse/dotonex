package modules

import (
	"testing"
)

func TestKeyValueString(t *testing.T) {
	c := keyValue{key: "k", value: "v"}
	if c.str() != "k = v" {
		t.Error("should collapse")
	}
}

func TestKeyValueStrings(t *testing.T) {
	c := keyValueStore{}
	c.keyValues = append(c.keyValues, keyValue{key: "key", value: "val"})
	c.add("key2", "val2")
	c.add("key2", "val3")
	res := c.strings()
	if len(res) != 3 {
		t.Error("invalid results")
	}
	if res[0] != "key = val" {
		t.Error("invalid first")
	}
	if res[1] != "  key2 = val2" {
		t.Error("invalid mid")
	}
	if res[2] != "  key2 = val3" {
		t.Error("invalid last")
	}
}

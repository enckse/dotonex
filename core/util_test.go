package core

import (
	"testing"

	"voidedtech.com/radiucal/core"
)

func TestKeyValueStrings(t *testing.T) {
	c := core.KeyValueStore{}
	c.KeyValues = append(c.KeyValues, core.KeyValue{Key: "key", Value: "val"})
	c.Add("key2", "val2")
	c.Add("key2", "val3")
	res := c.Strings()
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

func TestKeyValueString(t *testing.T) {
	c := core.KeyValue{Key: "k", Value: "v"}
	if c.String() != "k = v" {
		t.Error("should collapse")
	}
}

func TestKeyValueEmpty(t *testing.T) {
	c := core.KeyValueStore{}
	c.KeyValues = append(c.KeyValues, core.KeyValue{Key: "key", Value: "val"})
	c.Add("key2", "val2")
	c.Add("key2", "")
	c.DropEmpty = true
	res := c.Strings()
	if len(res) != 2 {
		t.Error("invalid results")
	}
	if res[0] != "key = val" {
		t.Error("invalid first")
	}
	if res[1] != "  key2 = val2" {
		t.Error("invalid mid")
	}
}

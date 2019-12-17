package core

import (
	"testing"
)

func TestKeyValueStrings(t *testing.T) {
	c := KeyValueStore{}
	c.KeyValues = append(c.KeyValues, KeyValue{Key: "key", Value: "val"})
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
	c := KeyValue{Key: "k", Value: "v"}
	if c.String() != "k = v" {
		t.Error("should collapse")
	}
}

func TestKeyValueEmpty(t *testing.T) {
	c := KeyValueStore{}
	c.KeyValues = append(c.KeyValues, KeyValue{Key: "key", Value: "val"})
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

func TestCompare(t *testing.T) {
	diff := Compare([]byte(""), []byte("a"), false)
	if diff {
		t.Error("different")
	}
	diff = Compare([]byte("a"), []byte("a"), false)
	if !diff {
		t.Error("different")
	}
}

func TestIn(t *testing.T) {
	if IntegerIn(1, []int{0, 2}) {
		t.Error("in was wrong")
	}
	if !IntegerIn(3, []int{1, 2, 3}) {
		t.Error("in should be right...")
	}
}

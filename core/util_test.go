package core

import (
	"fmt"
	"testing"

	"voidedtech.com/radiucal/core"
)

func TestKeyValueString(t *testing.T) {
	c := core.KeyValueStore{}
	c.KeyValues = append(c.KeyValues, core.KeyValue{Key: "key", Value: "val"})
	c.KeyValues = append(c.KeyValues, core.KeyValue{Key: "key2", Value: "val2"})
	expected := `key = val
  key2 = val2`
	if expected != c.String() {
		fmt.Println(expected)
		fmt.Println(c.String())
		t.Error("keyvalue output does not match")
	}
}

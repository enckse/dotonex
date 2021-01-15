package modules

import (
	"testing"
)

func TestKeyValueString(t *testing.T) {
	c := KeyValue{Key: "k", Value: "v"}
	if c.String() != "k = v" {
		t.Error("should collapse")
	}
}

package internal

import (
	"testing"
)

func TestNewPluginContext(t *testing.T) {
	c := NewPluginContext(&Configuration{Dir: "test"})
	if c.Lib != "test" {
		t.Error("invalid context")
	}
}

func TestKeyValueString(t *testing.T) {
	c := KeyValue{Key: "k", Value: "v"}
	if c.String() != "k = v" {
		t.Error("should collapse")
	}
}

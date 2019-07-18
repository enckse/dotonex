package core

import (
	"testing"

	"voidedtech.com/radiucal/core"
)

func TestDefaults(t *testing.T) {
	c := &core.Configuration{}
	c.Defaults([]byte{})
	if c.Host != "localhost" {
		t.Error("invalid default host")
	}
	if c.Dir != "/var/lib/radiucal/" {
		t.Error("invalid lib dir")
	}
	if c.Log != "/var/log/radiucal/" {
		t.Error("invalid log dir")
	}
	if c.Accounting {
		t.Error("wrong type")
	}
	if c.Bind != 1812 {
		t.Error("invalid port")
	}
	if c.LogBuffer != 10 {
		t.Error("invalid log buffer")
	}
}

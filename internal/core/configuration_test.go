package core

import (
	"testing"
)

func TestDefaults(t *testing.T) {
	c := &Configuration{}
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
	if c.Internals.Logs != 10 {
		t.Error("invalid log buffer")
	}
	if c.Internals.SpanCheck != 1 {
		t.Error("invalid span check")
	}
	if c.Internals.Lifespan != 24 {
		t.Error("invalid lifespan")
	}
}

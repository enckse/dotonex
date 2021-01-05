package server

import (
	"testing"
)

func TestDefaults(t *testing.T) {
	c := &Configuration{}
	c.Defaults()
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
	if c.Logs != 10 {
		t.Error("invalid log buffer")
	}
}
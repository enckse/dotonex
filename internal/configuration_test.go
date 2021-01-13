package internal

import (
	"testing"
)

func TestDefaults(t *testing.T) {
	c := &Configuration{}
	c.Defaults([]byte{})
	if c.Host != "localhost" {
		t.Error("invalid default host")
	}
	if c.Dir != "/var/lib/dotonex/" {
		t.Error("invalid lib dir")
	}
	if c.Log != "/var/log/dotonex/" {
		t.Error("invalid log dir")
	}
	if c.Configurator.Timeout != 60 {
		t.Error("invalid timeout")
	}
	if c.Configurator.Refresh != 5 {
		t.Error("invalid refresh")
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
	if c.Internals.Lifespan != 12 {
		t.Error("invalid lifespan")
	}
	l := c.Internals.LifeHours
	for _, o := range []int{22, 23, 0, 1, 2, 3, 4, 5} {
		if !IntegerIn(o, l) {
			t.Error("invalid hour defaults")
		}
	}
}

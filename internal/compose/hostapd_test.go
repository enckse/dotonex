package compose

import (
	"testing"
)

func TestMABString(t *testing.T) {
	h := NewHostapd("test", "test", "123")
	if h.String() != `"TEST" MD5 "TEST"
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:123` {
		t.Error("invalid MAB string")
	}
}

func TestUserString(t *testing.T) {
	h := NewHostapd("test", "atest", "123")
	if h.String() != `"test" PEAP

"test" MSCHAPV2 hash:atest [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:123` {
		t.Error("invalid user string")
	}
}

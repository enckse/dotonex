package runner

import (
	"fmt"
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func TestKeyValueString(t *testing.T) {
	c := keyValue{key: "k", value: "v"}
	if c.str() != "k = v" {
		t.Error("should collapse")
	}
}

func TestKeyValueStrings(t *testing.T) {
	c := keyValueStore{}
	c.keyValues = append(c.keyValues, keyValue{key: "key", value: "val"})
	c.add("key2", "val2")
	c.add("key2", "val3")
	res := c.strings()
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

func TestUserMacBasics(t *testing.T) {
	newTestSet(t, "user:test", "11-22-33-44-55-66", true, "")
	newTestSet(t, "user:test@vlan.test", "11-22-33-44-55-66", true, "")
	newTestSet(t, "test", "11-22-33-44-55-66", false, "INVALIDTOKEN")
	newTestSet(t, "user:test", "12-22-33-44-55-66", false, "TOKENMACFAIL")
	newTestSet(t, "11-22-33-11-22-33", "11-22-33-11-22-33", false, "NOMACFOUND")
}

func ErrorIfNotPre(t *testing.T, p *ClientPacket, message string) {
	err := checkUserMac(p)
	if err == nil {
		if message != "" {
			t.Errorf("expected to fail with: %s", message)
		}
	} else {
		if err.Error() != message {
			t.Errorf("'%s' != '%s'", err.Error(), message)
		}
	}
}

func newTestSet(t *testing.T, user, mac string, valid bool, reason string) *ClientPacket {
	SetAllowed([]string{"test/112233445566"})
	var secret = []byte("secret")
	p := NewClientPacket(nil, nil)
	p.Packet = radius.New(radius.CodeAccessRequest, secret)
	ErrorIfNotPre(t, p, "radius: attribute not found")
	if err := rfc2865.UserName_AddString(p.Packet, user); err != nil {
		t.Error("unable to add user name")
	}
	ErrorIfNotPre(t, p, "radius: attribute not found")
	if err := rfc2865.CallingStationID_AddString(p.Packet, mac); err != nil {
		t.Error("unable to add calling station")
	}
	if valid {
		ErrorIfNotPre(t, p, "")
	}
	if !valid {
		ErrorIfNotPre(t, p, fmt.Sprintf("failed preauth: %s %s (%s)", user, clean(mac), reason))
	}
	return p
}

func TestUserMacCache(t *testing.T) {
	pg := newTestSet(t, "user:test", "11-22-33-44-55-66", true, "")
	pb := newTestSet(t, "user:test", "11-22-33-44-55-68", false, "TOKENMACFAIL")
	first := "failed preauth: user:test 112233445568 (TOKENMACFAIL)"
	ErrorIfNotPre(t, pg, "")
	ErrorIfNotPre(t, pb, first)
}

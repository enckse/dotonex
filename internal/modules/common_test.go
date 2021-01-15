package modules

import (
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"voidedtech.com/dotonex/internal/op"
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
	newTestSet(t, "test", "11-22-33-44-55-66", true)
	newTestSet(t, "test", "12-22-33-44-55-66", false)
}

func ErrorIfNotPre(t *testing.T, p *op.ClientPacket, message string) {
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

func newTestSet(t *testing.T, user, mac string, valid bool) *op.ClientPacket {
	op.SetAllowed([]string{"test/112233445566"})
	var secret = []byte("secret")
	p := op.NewClientPacket(nil, nil)
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
		ErrorIfNotPre(t, p, "failed preauth: test "+clean(mac))
	}
	return p
}

func TestUserMacCache(t *testing.T) {
	pg := newTestSet(t, "test", "11-22-33-44-55-66", true)
	pb := newTestSet(t, "test", "11-22-33-44-55-68", false)
	first := "failed preauth: test 112233445568"
	ErrorIfNotPre(t, pg, "")
	ErrorIfNotPre(t, pb, first)
}

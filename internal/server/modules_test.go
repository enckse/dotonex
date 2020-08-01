package server

import (
	"log"
	"net"
	"strings"
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func TestDebug(t *testing.T) {
	testDebug(t, ":1234")
}

func testDebug(t *testing.T, hostAddr string) {
	var addr *net.UDPAddr
	addr = nil
	if len(hostAddr) > 0 {
		taddr, aerr := net.ResolveUDPAddr("udp", hostAddr)
		if aerr != nil {
			t.Error("invalid address")
		}
		addr = taddr
	}
	p := NewClientPacket(nil, addr)
	var secret = []byte("secret")
	p.Packet = radius.New(radius.CodeAccessRequest, secret)
	p.Packet.Identifier = 100
	if err := rfc2865.UserName_AddString(p.Packet, "test"); err != nil {
		t.Error("unable to add user name")
	}
	b := &logTrace{}
	tm := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	writeTrace(b, "testmode", TraceRequest, p, tm)
	expect := &logTrace{}
	expect.Write([]byte("TraceType = 1\n"))
	expect.Write([]byte("Mode = testmode\n"))
	if len(hostAddr) > 0 {
		expect.Write([]byte("UDPAddr = " + hostAddr + "\n"))
	}
	expect.Write([]byte(`Access-Request Id 100
  User-Name = "test"`))
	expected := strings.TrimSpace(expect.data.String())
	actual := strings.TrimSpace(b.data.String())
	if actual != expected {
		log.Println("actual:")
		log.Println(actual)
		log.Println("expect:")
		log.Println(expected)
		t.Error("actual != expected dump")
	}
}

func TestUserMacBasics(t *testing.T) {
	newTestSet(t, "test", "11-22-33-44-55-66", true)
	newTestSet(t, "test", "12-22-33-44-55-66", false)
}

func ErrorIfNotPre(t *testing.T, m *umac, p *ClientPacket, message string) {
	err := m.checkUserMac(p)
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

func newTestSet(t *testing.T, user, mac string, valid bool) (*ClientPacket, *umac) {
	m := setupUserMac()
	if m.Name() != "usermac" {
		t.Error("invalid/wrong name")
	}
	var secret = []byte("secret")
	p := NewClientPacket(nil, nil)
	p.Packet = radius.New(radius.CodeAccessRequest, secret)
	ErrorIfNotPre(t, m, p, "radius: attribute not found")
	if err := rfc2865.UserName_AddString(p.Packet, user); err != nil {
		t.Error("unable to add user name")
	}
	ErrorIfNotPre(t, m, p, "radius: attribute not found")
	if err := rfc2865.CallingStationID_AddString(p.Packet, mac); err != nil {
		t.Error("unable to add calling station")
	}
	if valid {
		ErrorIfNotPre(t, m, p, "")
	}
	if !valid {
		ErrorIfNotPre(t, m, p, "failed preauth: test "+clean(mac))
	}
	return p, m
}

func setupUserMac() *umac {
	fileMAC = "./test/manifest"
	m := &umac{}
	m.load()
	return m
}

func TestUserMacCache(t *testing.T) {
	pg, m := newTestSet(t, "test", "11-22-33-44-55-66", true)
	pb, _ := newTestSet(t, "test", "11-22-33-44-55-68", false)
	first := "failed preauth: test 112233445568"
	ErrorIfNotPre(t, m, pg, "")
	ErrorIfNotPre(t, m, pb, first)
}

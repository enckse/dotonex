package main

import (
	"log"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/epiphyte/radiucal/plugins"
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
	p := plugins.NewClientPacket(nil, addr)
	var secret = []byte("secret")
	p.Packet = radius.New(radius.CodeAccessRequest, secret)
	p.Packet.Identifier = 100
	if err := rfc2865.UserName_AddString(p.Packet, "test"); err != nil {
		t.Error("unable to add user name")
	}
	b := &logTrace{}
	tm := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	write(b, "testmode", plugins.TraceRequest, p, tm)
	expect := &logTrace{}
	expect.Write([]byte("tracetype: 1\n"))
	expect.Write([]byte("Mode = testmode (2009-11-10 23:00:00 +0000 UTC)\n"))
	if len(hostAddr) > 0 {
		expect.Write([]byte("UDPAddr = " + hostAddr + "\n"))
	}
	expect.Write([]byte(`Access-Request Id 100
  User-Name = "test"`))
	expected := strings.TrimSpace(expect.data.String())
	actual := strings.TrimSpace(b.data.String())
	if actual != expected {
		log.Println(actual)
		log.Println(expected)
		t.Error("actual != expected dump")
	}
}

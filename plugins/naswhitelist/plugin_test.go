package main

import (
	"net"
	"testing"

	"github.com/epiphyte/radiucal/core"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func TestFallThrough(t *testing.T) {
	p, n := newTestSet(t, net.IPv4(192, 168, 5, 10))
	if !n.Pre(p) {
		t.Error("should be considered valid")
	}
	p, n = newTestSet(t, net.IPv4(192, 168, 100, 10))
	if n.Pre(p) {
		t.Error("should be considered invalid")
	}
	p, n = newTestSet(t, net.IPv4(192, 168, 100, 5))
	if !n.Pre(p) {
		t.Error("should be considered valid")
	}
}

func TestBasic(t *testing.T) {
	p, n := newTestSet(t, net.IPv4(10, 10, 10, 10))
	if !n.Pre(p) {
		t.Error("should be considered valid")
	}
	p, n = newTestSet(t, net.IPv4(172, 10, 10, 5))
	if n.Pre(p) {
		t.Error("should be considered invalid")
	}
}

func newTestSet(t *testing.T, nasip net.IP) (*core.ClientPacket, *nwl) {
	m := setupModule(t)
	if m.Name() != "naswhitelist" {
		t.Error("invalid/wrong name")
	}
	var secret = []byte("secret")
	p := core.NewClientPacket(nil, nil)
	p.Packet = radius.New(radius.CodeAccessRequest, secret)
	if err := rfc2865.NASIPAddress_Add(p.Packet, nasip); err != nil {
		t.Error("unable to add ip")
	}
	return p, m
}

func setupModule(t *testing.T) *nwl {
	var array []string
	array = append(array, "!192.168.")
	array = append(array, "192.168.")
	array = append(array, "!192.168.100.")
	array = append(array, "192.168.100.5")
	array = append(array, "10.10.10.")
	array = append(array, "!172.10.10.5")
	array = append(array, "10.10.10.100.100")
	m := &nwl{}
	m.startup(array)
	if !enabled {
		t.Error("invalid setup")
	}
	return m
}

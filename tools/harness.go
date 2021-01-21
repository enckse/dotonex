package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func newPacket(user, mac string, ip net.IP) []byte {
	var secret = []byte("secret")
	p := radius.New(radius.CodeAccessRequest, secret)
	if err := rfc2865.UserName_AddString(p, user); err != nil {
		panic("unable to set attribute: user-name")
	}
	if err := rfc2865.CallingStationID_AddString(p, mac); err != nil {
		panic("unable to set attribute: calling-station-id")
	}
	if ip != nil {
		if err := rfc2865.NASIPAddress_Add(p, ip); err != nil {
			panic("unable to set attribute: nas-ip")
		}
	}
	b, err := p.Encode()
	if err != nil {
		panic("unable to encode packet")
	}
	return b
}

func runEndpoint() {
	addr, err := net.ResolveUDPAddr("udp", ":1814")
	if err != nil {
		panic("unable to get address")
	}
	srv, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic("unable to listen")
	}
	count := 0
	for {
		var buffer []byte
		_, c, _ := srv.ReadFromUDP(buffer)
		count++
		ioutil.WriteFile("./bin/count", []byte(fmt.Sprintf("count:%d", count)), 0644)
		b := newPacket("", "", nil)
		srv.WriteToUDP(b, c)
	}
}

func write(user, mac string, conn *net.UDPConn, nasip net.IP) {
	time.Sleep(1 * time.Second)
	p := newPacket(user, mac, nasip)
	_, err := conn.Write(p)
	if err != nil {
		panic("unable to write")
	}
}

func test(accounting bool) {
	bind := "1812"
	if accounting {
		bind = "1813"
	}
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%s", bind))
	if err != nil {
		panic("unable to get address")
	}
	srv, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic("unable to dial")
	}
	for _, ip := range []net.IP{net.IPv4(127, 0, 0, 1)} {
		write("user:test", "11-22-33-44-55-66", srv, ip)
		write("user:test@vlan.id", "11-22-33-44-55-67", srv, ip)
		write("user:test", "11-22-33-44-55-66", srv, ip)
	}
}

func main() {
	endpoint := flag.Bool("endpoint", false, "indicates if running as a fake endpoint")
	flag.Parse()
	if *endpoint {
		runEndpoint()
	} else {
		test(false)
		test(true)
	}
}

package core

import (
	"net"

	"layeh.com/radius"
)

type ClientPacket struct {
	ClientAddr *net.UDPAddr
	Buffer     []byte
	Packet     *radius.Packet
	Error      error
}

func NewClientPacket(buffer []byte, addr *net.UDPAddr) *ClientPacket {
	return &ClientPacket{Buffer: buffer, ClientAddr: addr}
}

package core

import (
	"net"

	"layeh.com/radius"
)

// ClientPacket represents the radius packet from the client
type ClientPacket struct {
	ClientAddr *net.UDPAddr
	Buffer     []byte
	Packet     *radius.Packet
	Error      error
}

// NewClientPacket creates a client packet from an input data packet
func NewClientPacket(buffer []byte, addr *net.UDPAddr) *ClientPacket {
	return &ClientPacket{Buffer: buffer, ClientAddr: addr}
}

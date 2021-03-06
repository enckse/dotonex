package runner

import (
	"bytes"
	"fmt"
	"net"

	"layeh.com/radius"
	"voidedtech.com/dotonex/internal/core"
)

const (
	// failure of auth reasons
	successCode   ReasonCode = 0
	badSecretCode ReasonCode = 1
	preAuthCode   ReasonCode = 2
	// NoTrace indicates no tracing to occur
	NoTrace TraceType = 0
	// TraceRequest indicate to trace the request
	TraceRequest TraceType = 1
)

type (
	writeBack func([]byte)

	// ReasonCode for authorization state
	ReasonCode int

	// Context is the server's operating context
	Context struct {
		Debug    bool
		secret   []byte
		pre      PreAuth
		acct     Account
		trace    Trace
		noReject bool
		// shortcuts
		hasPre   bool
		hasAcct  bool
		hasTrace bool
	}

	// TraceType indicates how to trace a request
	TraceType int

	// PreAuth represents the function required to pre-authorize a packet
	PreAuth func(*ClientPacket) bool

	// Trace represents the function required to trace requests
	Trace func(TraceType, *ClientPacket)

	// Account represents the function required to handle accounting
	Account func(*ClientPacket)

	// ClientPacket represents the radius packet from the client
	ClientPacket struct {
		ClientAddr *net.UDPAddr
		Buffer     []byte
		Packet     *radius.Packet
		Error      error
	}
)

// NewClientPacket creates a client packet from an input data packet
func NewClientPacket(buffer []byte, addr *net.UDPAddr) *ClientPacket {
	return &ClientPacket{Buffer: buffer, ClientAddr: addr}
}

func (ctx *Context) authorize(packet *ClientPacket) ReasonCode {
	if packet == nil {
		return successCode
	}
	valid := successCode
	ctx.packet(packet)
	// we may not be able to always read a packet during conversation
	// especially during initial EAP phases
	// we let that go
	if packet.Error != nil {
		return valid
	}
	if err := ctx.checkSecret(packet); err != nil {
		core.WriteError("invalid radius secret", err)
		valid = badSecretCode
	}
	if ctx.hasPre {
		failure := !ctx.pre(packet)
		if failure {
			core.WriteDebug("unauthorized (failed preauth)")
			if valid == successCode {
				valid = preAuthCode
			}
		}
	}
	if ctx.hasTrace {
		ctx.trace(TraceRequest, packet)
	}
	return valid
}

// FromConfig parses config data into a Context object
func (ctx *Context) FromConfig(c *core.Configuration) {
	ctx.noReject = c.NoReject
	ctx.secret = []byte(c.PacketKey)
	if len(c.PacketKey) == 0 {
		core.Fatal("invalid packet key", fmt.Errorf("packet key must be set to process packets"))
	}
	if c.Accounting {
		ctx.hasAcct = true
		ctx.acct = AccountPacket
	} else {
		ctx.hasPre = true
		ctx.pre = PrePacket
	}
	ctx.hasTrace = !c.NoTrace
	if ctx.hasTrace {
		ctx.trace = TracePacket
	}
}

// DebugDump dumps context information for debugging
func (ctx *Context) DebugDump() {
	if ctx.Debug {
		core.WriteDebug("secret", string(ctx.secret))
	}
}

func (ctx *Context) checkSecret(p *ClientPacket) error {
	var inSecret []byte
	if p == nil || p.Packet == nil {
		return fmt.Errorf("no packet information")
	}
	inSecret = p.Packet.Secret
	if inSecret == nil {
		return fmt.Errorf("no secret input")
	}
	if !bytes.Equal(ctx.secret, inSecret) {
		return fmt.Errorf("does not match shared secret")
	}
	return nil
}

func (ctx *Context) packet(p *ClientPacket) {
	if p.Error == nil && p.Packet == nil {
		packet, err := radius.Parse(p.Buffer, ctx.secret)
		p.Error = err
		p.Packet = packet
	}
}

// Account is responsible for performing all accounting module operations
func (ctx *Context) Account(packet *ClientPacket) {
	if !ctx.hasAcct {
		return
	}
	ctx.packet(packet)
	if packet.Error != nil {
		// unable to parse, exit early
		return
	}
	ctx.acct(packet)
}

// HandlePreAuth handles the actual pre-authorization checks
func HandlePreAuth(ctx *Context, b []byte, addr *net.UDPAddr, write writeBack) bool {
	packet := NewClientPacket(b, addr)
	authCode := ctx.authorize(packet)
	authed := authCode == successCode
	if !authed {
		if !ctx.noReject && write != nil && authCode != badSecretCode {
			if packet.Error == nil {
				p := packet.Packet
				p = p.Response(radius.CodeAccessReject)
				rej, err := p.Encode()
				if err == nil {
					core.WriteDebug("rejecting client")
					write(rej)
				} else {
					if ctx.Debug {
						core.WriteError("unable to encode rejection", err)
					}
				}
			} else {
				if ctx.Debug && packet.Error != nil {
					core.WriteError("unable to parse packets", packet.Error)
				}
			}
		}
	}
	return authed
}

package server

import (
	"fmt"
	"net"

	"layeh.com/radius"
	"voidedtech.com/radiucal/internal/core"
)

const (
	preMode  authingMode = 0
	postMode authingMode = 1
	// failure of auth reasons
	successCode  ReasonCode = 0
	preAuthCode  ReasonCode = 1
	postAuthCode ReasonCode = 2
)

type (
	writeBack func([]byte)

	authingMode int

	// ReasonCode for authorization state
	ReasonCode int

	// AuthorizePacket handles determining whether a packet is authorized to continue
	AuthorizePacket func(*Context, []byte, *net.UDPAddr) ReasonCode

	authCheck func(Module, *ClientPacket) bool

	// Context is the server's operating context
	Context struct {
		Debug   bool
		secret  []byte
		modules []Module
		// shortcuts
		postauth bool
		preauth  bool
		acct     bool
		module   bool
	}
)

// AddModule adds a general model to the context
func (ctx *Context) AddModule(m Module) {
	ctx.module = true
	ctx.modules = append(ctx.modules, m)
}

// PostAuthorize performs packet post-authorization (after radius check)
func PostAuthorize(ctx *Context, b []byte, addr *net.UDPAddr) ReasonCode {
	return ctx.doAuthing(b, addr, postMode)
}

// PreAuthorize performs a packet pre-check (before radius check)
func PreAuthorize(ctx *Context, b []byte, addr *net.UDPAddr) ReasonCode {
	return ctx.doAuthing(b, addr, preMode)
}

func (ctx *Context) doAuthing(b []byte, addr *net.UDPAddr, mode authingMode) ReasonCode {
	p := NewClientPacket(b, addr)
	return ctx.authorize(p, mode)
}

func (ctx *Context) authorize(packet *ClientPacket, mode authingMode) ReasonCode {
	if packet == nil {
		return successCode
	}
	valid := successCode
	preauthing := false
	postauthing := false
	switch mode {
	case preMode:
		preauthing = ctx.preauth
		break
	case postMode:
		postauthing = ctx.postauth
	}
	if preauthing || postauthing {
		ctx.packet(packet)
		// we may not be able to always read a packet during conversation
		// especially during initial EAP phases
		// we let that go
		if packet.Error == nil {
			var checks []Module
			var checking authCheck
			var code ReasonCode
			if preauthing {
				checking = getAuthChecker(true)
				for _, m := range ctx.modules {
					checks = append(checks, m)
				}
				code = preAuthCode
			}
			if postauthing {
				checking = getAuthChecker(false)
				for _, m := range ctx.modules {
					checks = append(checks, m)
				}
				code = postAuthCode
			}
			if len(checks) > 0 {
				failure := checkAuthMods(checks, packet, checking)
				if failure {
					if valid == successCode {
						valid = code
					}
				}
			}
		}
	}
	return valid
}

func getAuthChecker(preauthing bool) authCheck {
	return func(m Module, p *ClientPacket) bool {
		if preauthing {
			return m.Process(p, PreProcess)
		}
		return m.Process(p, PostProcess)
	}
}

func checkAuthMods(modules []Module, packet *ClientPacket, fxn authCheck) bool {
	failure := false
	for _, mod := range modules {
		if fxn(mod, packet) {
			continue
		}
		failure = true
		core.WriteDebug(fmt.Sprintf("unauthorized (failed: %s)", mod.Name()))
	}
	return failure
}

// DebugDump dumps context information for debugging
func (ctx *Context) DebugDump() {
	if ctx.Debug {
		core.WriteDebug("secret", string(ctx.secret))
	}
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
	ctx.packet(packet)
	if packet.Error != nil {
		// unable to parse, exit early
		return
	}
	if ctx.acct {
		for _, mod := range ctx.modules {
			mod.Process(packet, AccountingProcess)
		}
	}
}

// HandleAuth supports checking if a packet if allowed to continue on
func HandleAuth(fxn AuthorizePacket, ctx *Context, b []byte, addr *net.UDPAddr) bool {
	authCode := fxn(ctx, b, addr)
	authed := authCode == successCode
	return authed
}

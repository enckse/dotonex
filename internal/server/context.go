package server

import (
	"fmt"

	"layeh.com/radius"
	"voidedtech.com/radiucal/internal/core"
)

type (
	writeBack func([]byte)

	// AuthorizePacket handles determining whether a packet is authorized to continue
	AuthorizePacket func(*Context, []byte, string) bool

	authCheck func(Module, *ClientPacket) bool

	// Context is the server's operating context
	Context struct {
		Config *Configuration
		secret []byte
	}
)

// PostAuthorize performs packet post-authorization (after radius check)
func PostAuthorize(ctx *Context, b []byte, nas string) bool {
	return ctx.doAuthing(b, nas, PostProcess)
}

// PreAuthorize performs a packet pre-check (before radius check)
func PreAuthorize(ctx *Context, b []byte, nas string) bool {
	return ctx.doAuthing(b, nas, PreProcess)
}

func (ctx *Context) doAuthing(b []byte, nas string, mode ModuleMode) bool {
	p := NewClientPacket(b, nas)
	return ctx.authorize(p, mode)
}

func (ctx *Context) authorize(packet *ClientPacket, mode ModuleMode) bool {
	if packet == nil {
		return true
	}
	pre := mode == PreProcess
	post := mode == PostProcess
	valid := true
	if pre || post {
		ctx.packet(packet)
		// we may not be able to always read a packet during conversation
		// especially during initial EAP phases
		// we let that go
		if packet.Error == nil {
			if pre {
				Access(PreProcess, packet)
				if ctx.Config.Gitlab.Enable {
					valid = AuthorizeUserMAC(packet)
				} else {
					core.WriteWarn("Gitlab integration required for user MAC control")
				}
			}
			if post {
				Access(PostProcess, packet)
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
	if ctx.Config.Debug {
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
	LogPacket(AccountingProcess, packet)
}

// HandleAuth supports checking if a packet if allowed to continue on
func HandleAuth(fxn AuthorizePacket, ctx *Context, b []byte, nas string) bool {
	return fxn(ctx, b, nas)
}

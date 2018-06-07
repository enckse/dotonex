package core

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/epiphyte/goutils"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
)

type Context struct {
	Debug    bool
	secret   []byte
	preauths []plugins.PreAuth
	accts    []plugins.Accounting
	traces   []plugins.Tracing
	modules  []plugins.Module
	noReject bool
	// shortcuts
	preauth bool
	acct    bool
	trace   bool
	module  bool
}

func (ctx *Context) AddTrace(t plugins.Tracing) {
	ctx.trace = true
	ctx.traces = append(ctx.traces, t)
}

func (ctx *Context) AddPreAuth(p plugins.PreAuth) {
	ctx.preauth = true
	ctx.preauths = append(ctx.preauths, p)
}

func (ctx *Context) AddModule(m plugins.Module) {
	ctx.module = true
	ctx.modules = append(ctx.modules, m)
}

func (ctx *Context) AddAccounting(a plugins.Accounting) {
	ctx.acct = true
	ctx.accts = append(ctx.accts, a)
}

type writeBack func([]byte)

type authingMode int

const (
	preMode authingMode = 0
)

type packetAuthorize func(*Context, []byte, *net.UDPAddr) (*plugins.ClientPacket, bool)

func PreAuthorize(ctx *Context, b []byte, addr *net.UDPAddr) (*plugins.ClientPacket, bool) {
	return ctx.doAuthing(b, addr, preMode)
}

func (ctx *Context) doAuthing(b []byte, addr *net.UDPAddr, mode authingMode) (*plugins.ClientPacket, bool) {
	p := plugins.NewClientPacket(b, addr)
	return p, ctx.authorize(p, mode)
}

func (ctx *Context) authorize(packet *plugins.ClientPacket, mode authingMode) bool {
	if packet == nil {
		return true
	}
	valid := true
	traceMode := plugins.NoTrace
	preauthing := false
	switch mode {
	case preMode:
		preauthing = true
		traceMode = plugins.TraceRequest
		break
	}
	tracing := ctx.trace && traceMode != plugins.NoTrace
	if preauthing || tracing {
		ctx.packet(packet)
		// we may not be able to always read a packet during conversation
		// especially during initial EAP phases
		// we let that go
		if packet.Error == nil {
			if preauthing {
				for _, mod := range ctx.preauths {
					if mod.Pre(packet) {
						continue
					}
					valid = false
					goutils.WriteDebug(fmt.Sprintf("unauthorized (failed: %s)", mod.Name()))
				}
			}
			if tracing {
				for _, mod := range ctx.traces {
					mod.Trace(traceMode, packet)
				}
			}
		}
	}
	return valid
}

func (ctx *Context) FromConfig(libPath string, c *goutils.Config) {
	ctx.noReject = c.GetTrue("noreject")
	secrets := filepath.Join(libPath, "secrets")
	ctx.parseSecrets(secrets)
}

func (ctx *Context) parseSecrets(secretFile string) {
	s, err := parseSecretFile(secretFile)
	if LogError("unable to read secrets", err) {
		panic("unable to read secrets")
	}
	ctx.secret = []byte(s)
}

func parseSecretFile(secretFile string) (string, error) {
	if goutils.PathNotExists(secretFile) {
		return "", errors.New("no secrets file")
	}
	f, err := os.Open(secretFile)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "127.0.0.1") {
			parts := strings.Split(l, " ")
			secret := strings.TrimSpace(strings.Join(parts[1:], " "))
			if len(secret) > 0 {
				return strings.TrimSpace(strings.Join(parts[1:], " ")), nil
			}
		}
	}
	return "", errors.New("no secret found")
}

func (ctx *Context) DebugDump() {
	if ctx.Debug {
		goutils.WriteDebug("secret", string(ctx.secret))
	}
}

func (ctx *Context) Reload() {
	if ctx.module {
		goutils.WriteInfo("reloading")
		for _, m := range ctx.modules {
			goutils.WriteDebug("reloading module", m.Name())
			m.Reload()
		}
	}
}

func (ctx *Context) checkSecret(p *radius.Packet) error {
	valid := true
	var inSecret []byte
	if p == nil {
		valid = false
	} else {
		inSecret = p.Secret
	}
	if valid && inSecret == nil {
		valid = false
	}
	if valid && bytes.Compare(ctx.secret, inSecret) != 0 {
		valid = false
	}
	if valid {
		return nil
	}
	return errors.New("invalid secret")
}

func (ctx *Context) packet(p *plugins.ClientPacket) {
	packet, err := radius.Parse(p.Buffer, ctx.secret)
	p.Error = err
	p.Packet = packet
	if err != nil {
		p.Error = ctx.checkSecret(packet)
	}
}

func (ctx *Context) Account(packet *plugins.ClientPacket) {
	ctx.packet(packet)
	if packet.Error != nil {
		// unable to parse, exit early
		return
	}
	if ctx.acct {
		for _, mod := range ctx.accts {
			mod.Account(packet)
		}
	}
}

func HandleAuth(fxn packetAuthorize, ctx *Context, b []byte, addr *net.UDPAddr, write writeBack) bool {
	packet, authed := fxn(ctx, b, addr)
	if !authed {
		if !ctx.noReject && write != nil {
			if packet.Error == nil {
				p := packet.Packet
				p = p.Response(radius.CodeAccessReject)
				rej, err := p.Encode()
				if err == nil {
					goutils.WriteDebug("rejecting client")
					write(rej)
				} else {
					if ctx.Debug {
						goutils.WriteError("unable to encode rejection", err)
					}
				}
			} else {
				if ctx.Debug && packet.Error != nil {
					goutils.WriteError("unable to parse packets", packet.Error)
				}
			}
		}
	}
	return authed
}

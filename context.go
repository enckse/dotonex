package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/epiphyte/goutils"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
)

type context struct {
	debug    bool
	secret   []byte
	preauths []plugins.PreAuth
	accts    []plugins.Accounting
	traces   []plugins.Tracing
	modules  []plugins.Module
	noreject bool
	// shortcuts
	preauth bool
	acct    bool
	trace   bool
	module  bool
}

type writeBack func([]byte)

type authingMode int

const (
	preMode authingMode = 0
)

type packetAuthorize func(*context, []byte, *net.UDPAddr) (*plugins.ClientPacket, bool)

func preauthorize(ctx *context, b []byte, addr *net.UDPAddr) (*plugins.ClientPacket, bool) {
	return ctx.doAuthing(b, addr, preMode)
}

func (ctx *context) doAuthing(b []byte, addr *net.UDPAddr, mode authingMode) (*plugins.ClientPacket, bool) {
	p := plugins.NewClientPacket(b, addr)
	return p, ctx.authorize(p, mode)
}

func (ctx *context) authorize(packet *plugins.ClientPacket, mode authingMode) bool {
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

func parseSecrets(secretFile string) string {
	s, err := parseSecretFile(secretFile)
	if logError("unable to read secrets", err) {
		panic("unable to read secrets")
	}
	return s
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

func (ctx *context) reload() {
	if ctx.module {
		goutils.WriteInfo("reloading")
		for _, m := range ctx.modules {
			goutils.WriteDebug("reloading module", m.Name())
			m.Reload()
		}
	}
}

func (ctx *context) packet(p *plugins.ClientPacket) {
	packet, err := radius.Parse(p.Buffer, ctx.secret)
	p.Error = err
	p.Packet = packet
}

func (ctx *context) account(packet *plugins.ClientPacket) {
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

func handleAuth(fxn packetAuthorize, ctx *context, b []byte, addr *net.UDPAddr, write writeBack) bool {
	packet, authed := fxn(ctx, b, addr)
	if !authed {
		if !ctx.noreject && write != nil {
			if packet.Error == nil {
				p := packet.Packet
				p = p.Response(radius.CodeAccessReject)
				rej, err := p.Encode()
				if err == nil {
					goutils.WriteDebug("rejecting client")
					write(rej)
				} else {
					if ctx.debug {
						goutils.WriteError("unable to encode rejection", err)
					}
				}
			} else {
				if ctx.debug && packet.Error != nil {
					goutils.WriteError("unable to parse packets", packet.Error)
				}
			}
		}
	}
	return authed
}

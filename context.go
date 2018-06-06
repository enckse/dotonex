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

type authingMode int

const (
	preMode authingMode = 0
)

func (ctx *context) preauthorize(b []byte, addr *net.UDPAddr) bool {
	return ctx.doAuthing(b, addr, preMode)
}

func (ctx *context) doAuthing(b []byte, addr *net.UDPAddr, mode authingMode) bool {
	p := plugins.NewClientPacket(b, addr)
	return ctx.authorize(p, mode)
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
		err := ctx.packet(packet)
		// we may not be able to always read a packet during conversation
		// especially during initial EAP phases
		// we let that go
		if err == nil {
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

func (ctx *context) packet(p *plugins.ClientPacket) error {
	packet, err := radius.Parse(p.Buffer, []byte(ctx.secret))
	if err != nil {
		return err
	}
	p.Packet = packet
	return nil
}

func (ctx *context) account(packet *plugins.ClientPacket) {
	e := ctx.packet(packet)
	if e != nil {
		// unable to parse, exit early
		return
	}
	if ctx.acct {
		for _, mod := range ctx.accts {
			mod.Account(packet)
		}
	}
}

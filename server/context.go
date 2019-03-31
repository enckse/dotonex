package server

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"layeh.com/radius"
	"voidedtech.com/goutils/logger"
	"voidedtech.com/goutils/opsys"
	"voidedtech.com/radiucal/core"
)

const (
	preMode  authingMode = 0
	postMode authingMode = 1
	localKey             = "127.0.0.1"
	allKey               = "0.0.0.0"
	// failure of auth reasons
	successCode   ReasonCode = 0
	badSecretCode ReasonCode = 1
	preAuthCode   ReasonCode = 2
	postAuthCode  ReasonCode = 3
)

type writeBack func([]byte)

type authingMode int

type ReasonCode int

type AuthorizePacket func(*Context, []byte, *net.UDPAddr) (*core.ClientPacket, ReasonCode)

type authCheck func(core.Module, *core.ClientPacket) bool

type Context struct {
	Debug     bool
	secret    []byte
	preauths  []core.PreAuth
	postauths []core.PostAuth
	accts     []core.Accounting
	traces    []core.Tracing
	modules   []core.Module
	secrets   map[string][]byte
	noReject  bool
	// shortcuts
	postauth bool
	preauth  bool
	acct     bool
	trace    bool
	module   bool
}

func (ctx *Context) AddTrace(t core.Tracing) {
	ctx.trace = true
	ctx.traces = append(ctx.traces, t)
}

func (ctx *Context) AddPreAuth(p core.PreAuth) {
	ctx.preauth = true
	ctx.preauths = append(ctx.preauths, p)
}

func (ctx *Context) AddPostAuth(p core.PostAuth) {
	ctx.postauth = true
	ctx.postauths = append(ctx.postauths, p)
}

func (ctx *Context) AddModule(m core.Module) {
	ctx.module = true
	ctx.modules = append(ctx.modules, m)
}

func (ctx *Context) AddAccounting(a core.Accounting) {
	ctx.acct = true
	ctx.accts = append(ctx.accts, a)
}

func PostAuthorize(ctx *Context, b []byte, addr *net.UDPAddr) (*core.ClientPacket, ReasonCode) {
	return ctx.doAuthing(b, addr, postMode)
}

func PreAuthorize(ctx *Context, b []byte, addr *net.UDPAddr) (*core.ClientPacket, ReasonCode) {
	return ctx.doAuthing(b, addr, preMode)
}

func (ctx *Context) doAuthing(b []byte, addr *net.UDPAddr, mode authingMode) (*core.ClientPacket, ReasonCode) {
	p := core.NewClientPacket(b, addr)
	return p, ctx.authorize(p, mode)
}

func (ctx *Context) authorize(packet *core.ClientPacket, mode authingMode) ReasonCode {
	if packet == nil {
		return successCode
	}
	valid := successCode
	traceMode := core.NoTrace
	preauthing := false
	receiving := false
	postauthing := false
	switch mode {
	case preMode:
		receiving = true
		preauthing = ctx.preauth
		traceMode = core.TraceRequest
		break
	case postMode:
		postauthing = ctx.postauth
		traceMode = core.TraceRequest
	}
	tracing := ctx.trace && traceMode != core.NoTrace
	if preauthing || postauthing || tracing || receiving {
		ctx.packet(packet)
		// we may not be able to always read a packet during conversation
		// especially during initial EAP phases
		// we let that go
		if packet.Error == nil {
			if receiving {
				err := ctx.checkSecret(packet)
				if err != nil {
					logger.WriteError("invalid radius secret", err)
					valid = badSecretCode
				}
			}
			var checks []core.Module
			var checking authCheck
			var code ReasonCode
			if preauthing {
				checking = getAuthChecker(true)
				for _, m := range ctx.preauths {
					checks = append(checks, m)
				}
				code = preAuthCode
			}
			if postauthing {
				checking = getAuthChecker(false)
				for _, m := range ctx.postauths {
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
			if tracing {
				for _, mod := range ctx.traces {
					mod.Trace(traceMode, packet)
				}
			}
		}
	}
	return valid
}

func getAuthChecker(preauthing bool) authCheck {
	return func(m core.Module, p *core.ClientPacket) bool {
		if preauthing {
			return m.(core.PreAuth).Pre(p)
		} else {
			return m.(core.PostAuth).Post(p)
		}
	}
}

func checkAuthMods(modules []core.Module, packet *core.ClientPacket, fxn authCheck) bool {
	failure := false
	for _, mod := range modules {
		if fxn(mod, packet) {
			continue
		}
		failure = true
		logger.WriteDebug(fmt.Sprintf("unauthorized (failed: %s)", mod.Name()))
	}
	return failure
}

func (ctx *Context) FromConfig(libPath string, c *core.Configuration) {
	ctx.noReject = c.NoReject
	secrets := filepath.Join(libPath, "secrets")
	ctx.parseSecrets(secrets)
	ctx.secrets = make(map[string][]byte)
	secrets = filepath.Join(libPath, "clients")
	if opsys.PathExists(secrets) {
		mappings, err := parseSecretMappings(secrets)
		if err != nil {
			logger.Fatal("invalid client secret mappings", err)
		}
		for k, v := range mappings {
			ctx.secrets[k] = []byte(v)
		}
	}
}

func parseSecretMappings(filename string) (map[string][]byte, error) {
	mappings, err := parseSecretFromFile(filename, true)
	if err != nil {
		return nil, err
	}
	m := make(map[string][]byte)
	for k, v := range mappings {
		m[k] = []byte(v)
	}
	return m, nil
}

func (ctx *Context) parseSecrets(secretFile string) {
	s, err := parseSecretFile(secretFile)
	if core.LogError(fmt.Sprintf("unable to read secrets: %s", secretFile), err) {
		panic("unable to read secrets")
	}
	ctx.secret = []byte(s)
}

func parseSecretFile(secretFile string) (string, error) {
	s, err := parseSecretFromFile(secretFile, false)
	if err != nil {
		return "", err
	} else {
		return s[localKey], nil
	}
}

func parseSecretFromFile(secretFile string, mapping bool) (map[string]string, error) {
	if opsys.PathNotExists(secretFile) {
		return nil, errors.New("no secrets file")
	}
	f, err := os.Open(secretFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	lines := make(map[string]string)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "#") {
			continue
		}
		if mapping || strings.HasPrefix(l, localKey) {
			parts := strings.Split(l, " ")
			secret := strings.TrimSpace(strings.Join(parts[1:], " "))
			if len(secret) > 0 {
				if mapping {
					lines[parts[0]] = secret
				} else {
					lines[localKey] = secret
					break
				}
			}
		}
	}
	if len(lines) == 0 && !mapping {
		return nil, errors.New("no secrets found")
	}
	return lines, nil
}

func (ctx *Context) DebugDump() {
	if ctx.Debug {
		logger.WriteDebug("secret", string(ctx.secret))
		if len(ctx.secrets) > 0 {
			logger.WriteDebug("client mappings")
			for k, v := range ctx.secrets {
				logger.WriteDebug(k, string(v))
			}
		}
	}
}

func (ctx *Context) Reload() {
	if ctx.module {
		logger.WriteInfo("reloading")
		for _, m := range ctx.modules {
			logger.WriteDebug("reloading module", m.Name())
			m.Reload()
		}
	}
}

func (ctx *Context) checkSecret(p *core.ClientPacket) error {
	var inSecret []byte
	if p == nil || p.Packet == nil {
		return errors.New("no packet information")
	} else {
		inSecret = p.Packet.Secret
	}
	if inSecret == nil {
		return errors.New("no secret input")
	}
	if len(ctx.secrets) > 0 {
		if p.ClientAddr == nil {
			return errors.New("no client addr")
		}
		ip := p.ClientAddr.String()
		h, _, err := net.SplitHostPort(ip)
		if err != nil {
			return err
		}
		ip = h
		good := false
		logger.WriteInfo(ip)
		for k, v := range ctx.secrets {
			if strings.HasPrefix(ip, k) || k == allKey {
				if bytes.Equal(v, inSecret) {
					good = true
					break
				}
			}
		}
		if !good {
			return errors.New("matches no secrets")
		}
	} else {
		if !bytes.Equal(ctx.secret, inSecret) {
			return errors.New("does not match shared secret")
		}
	}
	return nil
}

func (ctx *Context) packet(p *core.ClientPacket) {
	if p.Error == nil && p.Packet == nil {
		packet, err := radius.Parse(p.Buffer, ctx.secret)
		p.Error = err
		p.Packet = packet
	}
}

func (ctx *Context) Account(packet *core.ClientPacket) {
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

func HandleAuth(fxn AuthorizePacket, ctx *Context, b []byte, addr *net.UDPAddr, write writeBack) bool {
	packet, authCode := fxn(ctx, b, addr)
	authed := authCode == successCode
	if !authed {
		if !ctx.noReject && write != nil && authCode != badSecretCode {
			if packet.Error == nil {
				p := packet.Packet
				p = p.Response(radius.CodeAccessReject)
				rej, err := p.Encode()
				if err == nil {
					logger.WriteDebug("rejecting client")
					write(rej)
				} else {
					if ctx.Debug {
						logger.WriteError("unable to encode rejection", err)
					}
				}
			} else {
				if ctx.Debug && packet.Error != nil {
					logger.WriteError("unable to parse packets", packet.Error)
				}
			}
		}
	}
	return authed
}

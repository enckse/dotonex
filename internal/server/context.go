package server

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"layeh.com/radius"
	"voidedtech.com/radiucal/internal/core"
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
		secrets map[string][]byte
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
			if preauthing {
				if err := ctx.checkSecret(packet); err != nil {
					core.WriteError("invalid radius secret", err)
					valid = badSecretCode
				}
			}
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

// FromConfig parses config data into a Context object
func (ctx *Context) FromConfig(libPath string, c *Configuration) {
	secrets := filepath.Join(libPath, "secrets")
	ctx.parseSecrets(secrets)
	ctx.secrets = make(map[string][]byte)
	secrets = filepath.Join(libPath, "clients")
	if core.PathExists(secrets) {
		mappings, err := parseSecretMappings(secrets)
		if err != nil {
			core.Fatal("invalid client secret mappings", err)
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
	if err != nil {
		core.Fatal(fmt.Sprintf("unable to read secrets: %s", secretFile), err)
	}
	ctx.secret = []byte(s)
}

func parseSecretFile(secretFile string) (string, error) {
	s, err := parseSecretFromFile(secretFile, false)
	if err != nil {
		return "", err
	}
	return s[localKey], nil
}

func parseSecretFromFile(secretFile string, mapping bool) (map[string]string, error) {
	if !core.PathExists(secretFile) {
		return nil, fmt.Errorf("no secrets file")
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
		return nil, fmt.Errorf("no secrets found")
	}
	return lines, nil
}

// DebugDump dumps context information for debugging
func (ctx *Context) DebugDump() {
	if ctx.Debug {
		core.WriteDebug("secret", string(ctx.secret))
		if len(ctx.secrets) > 0 {
			core.WriteDebug("client mappings")
			for k, v := range ctx.secrets {
				core.WriteDebug(k, string(v))
			}
		}
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
	if len(ctx.secrets) > 0 {
		if p.ClientAddr == nil {
			return fmt.Errorf("no client addr")
		}
		ip := p.ClientAddr.String()
		h, _, err := net.SplitHostPort(ip)
		if err != nil {
			return err
		}
		ip = h
		good := false
		core.WriteInfo(ip)
		for k, v := range ctx.secrets {
			if strings.HasPrefix(ip, k) || k == allKey {
				if bytes.Equal(v, inSecret) {
					good = true
					break
				}
			}
		}
		if !good {
			return fmt.Errorf("matches no secrets")
		}
	} else {
		if !bytes.Equal(ctx.secret, inSecret) {
			return fmt.Errorf("does not match shared secret")
		}
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

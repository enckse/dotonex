package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"layeh.com/radius"
)

const (
	localKey = "127.0.0.1"
	allKey   = "0.0.0.0"
	// failure of auth reasons
	successCode   ReasonCode = 0
	badSecretCode ReasonCode = 1
	preAuthCode   ReasonCode = 2
)

type (
	writeBack func([]byte)

	// ReasonCode for authorization state
	ReasonCode int

	authCheck func(Module, *ClientPacket) bool

	// Context is the server's operating context
	Context struct {
		Debug    bool
		secret   []byte
		preauths []PreAuth
		accts    []Accounting
		traces   []Tracing
		modules  []Module
		secrets  map[string][]byte
		noReject bool
		// shortcuts
		preauth bool
		acct    bool
		trace   bool
		module  bool
	}
)

// AddTrace adds a tracing check to the context
func (ctx *Context) AddTrace(t Tracing) {
	ctx.trace = true
	ctx.traces = append(ctx.traces, t)
}

// AddPreAuth adds a pre-authorization check to the context
func (ctx *Context) AddPreAuth(p PreAuth) {
	ctx.preauth = true
	ctx.preauths = append(ctx.preauths, p)
}

// AddModule adds a general model to the context
func (ctx *Context) AddModule(m Module) {
	ctx.module = true
	ctx.modules = append(ctx.modules, m)
}

// AddAccounting adds an accounting check to the context
func (ctx *Context) AddAccounting(a Accounting) {
	ctx.acct = true
	ctx.accts = append(ctx.accts, a)
}

func (ctx *Context) authorize(packet *ClientPacket) ReasonCode {
	if packet == nil {
		return successCode
	}
	valid := successCode
	preauthing := ctx.preauth
	tracing := ctx.trace
	ctx.packet(packet)
	// we may not be able to always read a packet during conversation
	// especially during initial EAP phases
	// we let that go
	if packet.Error == nil {
		if err := ctx.checkSecret(packet); err != nil {
			WriteError("invalid radius secret", err)
			valid = badSecretCode
		}
		var checks []Module
		var checking authCheck
		var code ReasonCode
		if preauthing {
			checking = func(m Module, p *ClientPacket) bool {
				return m.(PreAuth).Pre(p)
			}
			for _, m := range ctx.preauths {
				checks = append(checks, m)
			}
			code = preAuthCode
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
				mod.Trace(TraceRequest, packet)
			}
		}
	}
	return valid
}

func checkAuthMods(modules []Module, packet *ClientPacket, fxn authCheck) bool {
	failure := false
	for _, mod := range modules {
		if fxn(mod, packet) {
			continue
		}
		failure = true
		WriteDebug(fmt.Sprintf("unauthorized (failed: %s)", mod.Name()))
	}
	return failure
}

// FromConfig parses config data into a Context object
func (ctx *Context) FromConfig(libPath string, c *Configuration) {
	ctx.noReject = c.NoReject
	secrets := filepath.Join(libPath, "secrets")
	ctx.parseSecrets(secrets)
	ctx.secrets = make(map[string][]byte)
	secrets = filepath.Join(libPath, "clients")
	if PathExists(secrets) {
		mappings, err := parseSecretMappings(secrets)
		if err != nil {
			Fatal("invalid client secret mappings", err)
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
		Fatal(fmt.Sprintf("unable to read secrets: %s", secretFile), err)
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
	if !PathExists(secretFile) {
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
		WriteDebug("secret", string(ctx.secret))
		if len(ctx.secrets) > 0 {
			WriteDebug("client mappings")
			for k, v := range ctx.secrets {
				WriteDebug(k, string(v))
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
		WriteInfo(ip)
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
		for _, mod := range ctx.accts {
			mod.Account(packet)
		}
	}
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
					WriteDebug("rejecting client")
					write(rej)
				} else {
					if ctx.Debug {
						WriteError("unable to encode rejection", err)
					}
				}
			} else {
				if ctx.Debug && packet.Error != nil {
					WriteError("unable to parse packets", packet.Error)
				}
			}
		}
	}
	return authed
}

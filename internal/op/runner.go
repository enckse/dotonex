package op

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"layeh.com/radius"
	"voidedtech.com/dotonex/internal/core"
)

const (
	localKey = "127.0.0.1"
	allKey   = "0.0.0.0"
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
		acct     Accounting
		trace    Tracing
		secrets  map[string][]byte
		noReject bool
		// shortcuts
		hasPre   bool
		hasAcct  bool
		hasTrace bool
	}

	// TraceType indicates how to trace a request
	TraceType int

	// PreAuth represents the interface required to pre-authorize a packet
	PreAuth interface {
		Pre(*ClientPacket) bool
	}

	// Tracing represents the interface required to trace requests
	Tracing interface {
		Trace(TraceType, *ClientPacket)
	}

	// Accounting represents the interface required to handle accounting
	Accounting interface {
		Account(*ClientPacket)
	}

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

// SetTrace adds a tracing check to the context
func (ctx *Context) SetTrace(t Tracing) {
	ctx.hasTrace = true
	ctx.trace = t
}

// SetPreAuth adds a pre-authorization check to the context
func (ctx *Context) SetPreAuth(p PreAuth) {
	ctx.hasPre = true
	ctx.pre = p
}

// SetAccounting adds an accounting check to the context
func (ctx *Context) SetAccounting(a Accounting) {
	ctx.hasAcct = true
	ctx.acct = a
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
		failure := !ctx.pre.Pre(packet)
		if failure {
			core.WriteDebug("unauthorized (failed preauth)")
			if valid == successCode {
				valid = preAuthCode
			}
		}
	}
	if ctx.hasTrace {
		ctx.trace.Trace(TraceRequest, packet)
	}
	return valid
}

// FromConfig parses config data into a Context object
func (ctx *Context) FromConfig(c *core.Configuration) {
	ctx.noReject = c.NoReject
	secrets := filepath.Join(c.Dir, "secrets")
	ctx.parseSecrets(secrets)
	ctx.secrets = make(map[string][]byte)
	secrets = filepath.Join(c.Dir, "clients")
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
	if !ctx.hasAcct {
		return
	}
	ctx.packet(packet)
	if packet.Error != nil {
		// unable to parse, exit early
		return
	}
	ctx.acct.Account(packet)
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

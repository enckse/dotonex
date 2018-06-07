package core

import (
	"net"
	"testing"

	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

type MockModule struct {
	acct   int
	trace  int
	pre    int
	fail   bool
	reload int
	// TraceType
	preAuth int
}

func (m *MockModule) Name() string {
	return "mock"
}

func (m *MockModule) Reload() {
	m.reload++
}

func (m *MockModule) Setup(c *plugins.PluginContext) {
}

func (m *MockModule) Pre(p *plugins.ClientPacket) bool {
	m.pre++
	return !m.fail
}

func (m *MockModule) Trace(t plugins.TraceType, p *plugins.ClientPacket) {
	m.trace++
	switch t {
	case plugins.TraceRequest:
		m.preAuth++
		break
	}
}

func (m *MockModule) Account(p *plugins.ClientPacket) {
	m.acct++
}

func TestPreAuthNoMods(t *testing.T) {
	ctx := &Context{}
	if !ctx.authorize(nil, preMode) {
		t.Error("should have passed, nothing to do")
	}
}

func TestSecrets(t *testing.T) {
	ctx, p := getPacket(t)
	ctx.packet(p)
	p.Packet.Secret = []byte("test")
	if ctx.authorize(p, preMode) {
		t.Error("different secrets")
	}
	ctx, p = getPacket(t)
	if !ctx.authorize(p, preMode) {
		t.Error("same secrets")
	}
	ctx.secrets = make(map[string][]byte)
	ctx.secrets["10."] = []byte("invalid")
	ctx.secrets["10.100."] = p.Packet.Secret
	ctx.secrets["10.10.1."] = []byte("invalid")
	if ctx.authorize(p, preMode) {
		t.Error("no addr but secrets")
	}
	addr, err := net.ResolveUDPAddr("udp", "10.10.1.100:1234")
	if err != nil {
		t.Error("invalid udp test addr")
	}
	p.ClientAddr = addr
	if ctx.authorize(p, preMode) {
		t.Error("no matching secrets")
	}
	ctx.secrets["10.10.1.10"] = p.Packet.Secret
	if !ctx.authorize(p, preMode) {
		t.Error("matching secrets")
	}
}

func TestPreAuth(t *testing.T) {
	ctx, p := getPacket(t)
	m := &MockModule{}
	ctx.AddTrace(m)
	// invalid packet
	if !ctx.authorize(plugins.NewClientPacket(nil, nil), preMode) {
		t.Error("didn't authorize")
	}
	if m.trace != 0 {
		t.Error("did auth")
	}
	if !ctx.authorize(p, preMode) {
		t.Error("didn't authorize")
	}
	if m.trace != 1 {
		t.Error("didn't auth")
	}
	ctx.AddPreAuth(m)
	if !ctx.authorize(p, preMode) {
		t.Error("didn't authorize")
	}
	if m.trace != 2 {
		t.Error("didn't auth again")
	}
	if m.pre != 1 {
		t.Error("didn't preauth")
	}
	m.fail = true
	if ctx.authorize(p, preMode) {
		t.Error("did authorize")
	}
	if m.trace != 3 {
		t.Error("didn't auth again")
	}
	if m.pre != 2 {
		t.Error("didn't preauth")
	}
	ctx.trace = false
	if ctx.authorize(p, preMode) {
		t.Error("did authorize")
	}
	if m.trace != 3 {
		t.Error("didn't auth again")
	}
	if m.pre != 3 {
		t.Error("didn't preauth")
	}
	if m.preAuth != 3 {
		t.Error("not enough preauth types")
	}
}

func getPacket(t *testing.T) (*Context, *plugins.ClientPacket) {
	c := &Context{}
	c.secret = []byte("secret")
	p := radius.New(radius.CodeAccessRequest, c.secret)
	if err := rfc2865.UserName_AddString(p, "user"); err != nil {
		t.Error("unable to add user name")
	}
	if err := rfc2865.CallingStationID_AddString(p, "11-22-33-44-55-66"); err != nil {
		t.Error("unable to add calling statiron")
	}
	b, err := p.Encode()
	if err != nil {
		t.Error("unable to encode")
	}
	return c, plugins.NewClientPacket(b, nil)
}

func TestSecretParsing(t *testing.T) {
	dir := "../tests/"
	_, err := parseSecretFile(dir + "nofile")
	if err.Error() != "no secrets file" {
		t.Error("file does not exist")
	}
	_, err = parseSecretFile(dir + "emptysecrets")
	if err.Error() != "no secret found" {
		t.Error("file is empty")
	}
	_, err = parseSecretFile(dir + "nosecrets")
	if err.Error() != "no secret found" {
		t.Error("file is empty")
	}
	s, _ := parseSecretFile(dir + "onesecret")
	if s != "mysecretkey" {
		t.Error("wrong parsed key")
	}
	s, _ = parseSecretFile(dir + "multisecret")
	if s != "test" {
		t.Error("wrong parsed key")
	}
	_, err = parseSecretFile(dir + "noopsecret")
	if err.Error() != "no secret found" {
		t.Error("empty key")
	}
}

func TestReload(t *testing.T) {
	ctx, _ := getPacket(t)
	m := &MockModule{}
	ctx.Reload()
	ctx.AddModule(m)
	ctx.AddModule(m)
	ctx.Reload()
	if m.reload != 2 {
		t.Error("should have reloaded each module once")
	}
}

func TestAcctNoMods(t *testing.T) {
	ctx := &Context{}
	ctx.Account(plugins.NewClientPacket(nil, nil))
}

func TestAcct(t *testing.T) {
	ctx, p := getPacket(t)
	m := &MockModule{}
	ctx.Account(plugins.NewClientPacket(nil, nil))
	if m.acct != 0 {
		t.Error("didn't account")
	}
	ctx.AddAccounting(m)
	ctx.Account(p)
	if m.acct != 1 {
		t.Error("didn't account")
	}
	ctx.Account(p)
	if m.acct != 2 {
		t.Error("didn't account")
	}
}

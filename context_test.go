package main

import (
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

func TestAuthNoMods(t *testing.T) {
	ctx := &context{}
	if !ctx.authorize(nil) {
		t.Error("should have passed, nothing to do")
	}
}

func TestAuth(t *testing.T) {
	ctx, p := getPacket(t)
	m := &MockModule{}
	ctx.traces = append(ctx.traces, m)
	ctx.trace = true
	// invalid packet
	if !ctx.authorize(plugins.NewClientPacket(nil, nil)) {
		t.Error("didn't authorize")
	}
	if m.trace != 0 {
		t.Error("did auth")
	}
	if !ctx.authorize(p) {
		t.Error("didn't authorize")
	}
	if m.trace != 1 {
		t.Error("didn't auth")
	}
	ctx.preauth = true
	ctx.preauths = append(ctx.preauths, m)
	if !ctx.authorize(p) {
		t.Error("didn't authorize")
	}
	if m.trace != 2 {
		t.Error("didn't auth again")
	}
	if m.pre != 1 {
		t.Error("didn't preauth")
	}
	m.fail = true
	if ctx.authorize(p) {
		t.Error("did authorize")
	}
	if m.trace != 3 {
		t.Error("didn't auth again")
	}
	if m.pre != 2 {
		t.Error("didn't preauth")
	}
	ctx.trace = false
	if ctx.authorize(p) {
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

func getPacket(t *testing.T) (*context, *plugins.ClientPacket) {
	c := &context{}
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
	dir := "./tests/"
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
	ctx.reload()
	ctx.modules = append(ctx.modules, m)
	ctx.modules = append(ctx.modules, m)
	ctx.module = true
	ctx.reload()
	if m.reload != 2 {
		t.Error("should have reloaded each module once")
	}
}

func TestAcctNoMods(t *testing.T) {
	ctx := &context{}
	ctx.account(plugins.NewClientPacket(nil, nil))
}

func TestAcct(t *testing.T) {
	ctx, p := getPacket(t)
	m := &MockModule{}
	ctx.account(plugins.NewClientPacket(nil, nil))
	if m.acct != 0 {
		t.Error("didn't account")
	}
	ctx.acct = true
	ctx.accts = append(ctx.accts, m)
	ctx.account(p)
	if m.acct != 1 {
		t.Error("didn't account")
	}
	ctx.account(p)
	if m.acct != 2 {
		t.Error("didn't account")
	}
}

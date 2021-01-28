package runner

import (
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

const (
	testDir = "../../tests/unittests/"
)

type MockModule struct {
	acct   int
	trace  int
	pre    int
	fail   bool
	reload int
	unload int
	// TraceType
	preAuth int
}

func (m *MockModule) Name() string {
	return "mock"
}

func (m *MockModule) Pre(p *ClientPacket) bool {
	m.pre++
	return !m.fail
}

func (m *MockModule) Trace(t TraceType, p *ClientPacket) {
	m.trace++
	switch t {
	case TraceRequest:
		m.preAuth++
		break
	}
}

func (m *MockModule) Account(p *ClientPacket) {
	m.acct++
}

func TestPreAuthNoMods(t *testing.T) {
	ctx := &Context{}
	if ctx.authorize(nil) != successCode {
		t.Error("should have passed, nothing to do")
	}
}

func checkAuthMode(t *testing.T) {
	ctx, p := getPacket(t)
	m := &MockModule{}
	ctx.SetTrace(m.Trace)
	// invalid packet
	if ctx.authorize(NewClientPacket(nil, nil)) != successCode {
		t.Error("didn't authorize")
	}
	if m.trace != 0 {
		t.Error("did auth")
	}
	if ctx.authorize(p) != successCode {
		t.Error("didn't authorize")
	}
	if m.trace != 1 {
		t.Error("didn't auth")
	}
	var getCounts func() (int, int)
	var reasonCode ReasonCode
	getCounts = func() (int, int) {
		return m.pre, m.preAuth
	}
	reasonCode = preAuthCode
	ctx.SetPreAuth(m.Pre)
	if ctx.authorize(p) != successCode {
		t.Error("didn't authorize")
	}
	if m.trace != 2 {
		t.Error("didn't auth again")
	}
	cnt, sum := getCounts()
	if cnt != 1 {
		t.Error("didn't mod auth")
	}
	m.fail = true
	if ctx.authorize(p) != reasonCode {
		t.Error("did authorize")
	}
	if m.trace != 3 {
		t.Error("didn't auth again")
	}
	cnt, sum = getCounts()
	if cnt != 2 {
		t.Error("didn't mod auth")
	}
	ctx.hasTrace = false
	if ctx.authorize(p) != reasonCode {
		t.Error("did authorize")
	}
	if m.trace != 3 {
		t.Error("didn't auth again")
	}
	cnt, sum = getCounts()
	if cnt != 3 {
		t.Error("didn't mod auth")
	}
	if sum != 3 {
		t.Error("not enough mod auth types")
	}
}

func TestPreAuth(t *testing.T) {
	checkAuthMode(t)
}

func getPacket(t *testing.T) (*Context, *ClientPacket) {
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
	return c, NewClientPacket(b, nil)
}

func TestAcctNoMods(t *testing.T) {
	ctx := &Context{}
	ctx.Account(NewClientPacket(nil, nil))
}

func TestAcct(t *testing.T) {
	ctx, p := getPacket(t)
	m := &MockModule{}
	ctx.Account(NewClientPacket(nil, nil))
	if m.acct != 0 {
		t.Error("didn't account")
	}
	ctx.SetAccounting(m.Account)
	ctx.Account(p)
	if m.acct != 1 {
		t.Error("didn't account")
	}
	ctx.Account(p)
	if m.acct != 2 {
		t.Error("didn't account")
	}
}

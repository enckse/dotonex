package server

import (
	"testing"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

type MockModule struct {
	acct   int
	pre    int
	fail   bool
	reload int
	post   int
	unload int
	// TraceType
	preAuth  int
	postAuth int
}

func (m *MockModule) Name() string {
	return "mock"
}

func (m *MockModule) Setup(c *ModuleContext) error {
	return nil
}

func (m *MockModule) Process(p *ClientPacket, mode ModuleMode) bool {
	switch mode {
	case PreProcess:
		m.pre++
	case AccountingProcess:
		m.acct++
	case PostProcess:
		m.post++
	}
	return !m.fail
}

func TestPreAuthNoMods(t *testing.T) {
	ctx := &Context{}
	if ctx.authorize(nil, preMode) != successCode {
		t.Error("should have passed, nothing to do")
	}
}

func checkAuthMode(t *testing.T, mode authingMode) {
	ctx, p := getPacket(t)
	m := &MockModule{}
	ctx.AddModule(m)
	ctx.preauth = mode == preMode
	ctx.postauth = mode == postMode
	// invalid packet
	if ctx.authorize(NewClientPacket(nil, nil), mode) != successCode {
		t.Error("didn't authorize")
	}
	if ctx.authorize(p, mode) != successCode {
		t.Error("didn't authorize")
	}
	var getCounts func() int
	var reasonCode ReasonCode
	if mode == preMode {
		getCounts = func() int {
			return m.pre
		}
		reasonCode = preAuthCode
		ctx.AddModule(m)
	} else {
		getCounts = func() int {
			return m.post
		}
		reasonCode = postAuthCode
		ctx.AddModule(m)
	}
	if ctx.authorize(p, mode) != successCode {
		t.Error("didn't authorize")
	}
	cnt := getCounts()
	if cnt != 3 {
		t.Error("didn't mod auth")
	}
	m.fail = true
	if ctx.authorize(p, mode) != reasonCode {
		t.Error("did authorize")
	}
	cnt = getCounts()
	if cnt != 5 {
		t.Error("didn't mod auth")
	}
	if ctx.authorize(p, mode) != reasonCode {
		t.Error("did authorize")
	}
	cnt = getCounts()
	if cnt != 7 {
		t.Error("didn't mod auth")
	}
}

func TestPostAuth(t *testing.T) {
	checkAuthMode(t, postMode)
}

func TestPreAuth(t *testing.T) {
	checkAuthMode(t, preMode)
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
	ctx.acct = true
	m := &MockModule{}
	ctx.Account(NewClientPacket(nil, nil))
	if m.acct != 0 {
		t.Error("didn't account")
	}
	ctx.AddModule(m)
	ctx.Account(p)
	if m.acct != 1 {
		t.Error("didn't account")
	}
	ctx.Account(p)
	if m.acct != 2 {
		t.Error("didn't account")
	}
}

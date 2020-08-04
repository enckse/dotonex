package server

import (
	"fmt"
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func TestPostAuth(t *testing.T) {
	moduleLogs = []string{}
	ctx, b, _ := getPacket(t)
	if !PostAuthorize(ctx, b, "127.0.0.1") {
		t.Error("Should have post authorized")
	}
	time.Sleep(100 * time.Millisecond)
	if len(moduleLogs) != 5 {
		t.Error("should have logged a packet")
	}
}

func TestPreAuth(t *testing.T) {
	SetUserAuths([]string{})
	moduleLogs = []string{}
	ctx, b, _ := getPacket(t)
	ctx.Config = &Configuration{}
	if !PreAuthorize(ctx, b, "127.0.0.1") {
		t.Error("Should have pre authorized")
	}
	time.Sleep(100 * time.Millisecond)
	if len(moduleLogs) != 5 {
		t.Error("should have logged a packet")
	}
	ctx.Config.Gitlab.Enable = true
	if PreAuthorize(ctx, b, "127.0.0.1") {
		t.Error("Should have pre authorized")
	}
	time.Sleep(100 * time.Millisecond)
	if len(moduleLogs) != 17 {
		t.Error("should have logged a packet")
	}
	SetUserAuths([]string{"user.112233445566"})
	if !PreAuthorize(ctx, b, "127.0.0.1") {
		t.Error("Should have pre authorized")
	}
	time.Sleep(100 * time.Millisecond)
	if len(moduleLogs) != 29 {
		t.Error("should have logged a packet")
	}
}

func getPacket(t *testing.T) (*Context, []byte, *ClientPacket) {
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
	return c, b, NewClientPacket(b, "127.0.0.1")
}

func TestAcct(t *testing.T) {
	moduleLogs = []string{}
	ctx, _, packet := getPacket(t)
	packet.Error = fmt.Errorf("test")
	ctx.Account(packet)
	if len(moduleLogs) != 0 {
		t.Error("should not have logged")
	}
	packet.Error = nil
	ctx.Account(packet)
	time.Sleep(100 * time.Millisecond)
	if len(moduleLogs) != 6 {
		t.Error("should have logged a packet")
	}
}

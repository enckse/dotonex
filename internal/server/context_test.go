package server

import (
	"fmt"
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/internal/core"
	"voidedtech.com/radiucal/internal/server/modules"
	"voidedtech.com/radiucal/internal/server/processing"
)

func TestPostAuth(t *testing.T) {
	processing.WriteModuleMessages("")
	ctx, b, _ := getPacket(t)
	if !PostAuthorize(ctx, b, "127.0.0.1") {
		t.Error("Should have post authorized")
	}
	time.Sleep(100 * time.Millisecond)
	if processing.WriteModuleMessages("") != 5 {
		t.Error("should have logged a packet")
	}
}

func TestPreAuth(t *testing.T) {
	modules.SetUserAuths([]string{})
	processing.WriteModuleMessages("")
	ctx, b, _ := getPacket(t)
	ctx.Config = &core.Configuration{}
	if !PreAuthorize(ctx, b, "127.0.0.1") {
		t.Error("Should have pre authorized")
	}
	time.Sleep(100 * time.Millisecond)
	if processing.WriteModuleMessages("") != 5 {
		t.Error("should have logged a packet")
	}
	ctx.Config.Gitlab.Enable = true
	if PreAuthorize(ctx, b, "127.0.0.1") {
		t.Error("Should have pre authorized")
	}
	time.Sleep(100 * time.Millisecond)
	if processing.WriteModuleMessages("") != 12 {
		t.Error("should have logged a packet")
	}
	modules.SetUserAuths([]string{"user.112233445566"})
	if !PreAuthorize(ctx, b, "127.0.0.1") {
		t.Error("Should have pre authorized")
	}
	time.Sleep(100 * time.Millisecond)
	if processing.WriteModuleMessages("") != 12 {
		t.Error("should have logged a packet")
	}
}

func getPacket(t *testing.T) (*Context, []byte, *processing.ClientPacket) {
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
	return c, b, processing.NewClientPacket(b, "127.0.0.1")
}

func TestAcct(t *testing.T) {
	processing.WriteModuleMessages("")
	ctx, _, packet := getPacket(t)
	packet.Error = fmt.Errorf("test")
	ctx.Account(packet)
	if processing.WriteModuleMessages("") != 0 {
		t.Error("should not have logged")
	}
	packet.Error = nil
	ctx.Account(packet)
	time.Sleep(100 * time.Millisecond)
	if processing.WriteModuleMessages("") != 6 {
		t.Error("should have logged a packet")
	}
}

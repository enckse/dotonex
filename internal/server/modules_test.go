package server

import (
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func TestAccess(t *testing.T) {
	moduleLogs = []string{}
	packet := getModulePacket(t)
	Access(AccountingProcess, packet)
	time.Sleep(100 * time.Millisecond)
	if len(moduleLogs) != 5 {
		t.Error("should have logged a packet")
	}
}

func TestLogPacket(t *testing.T) {
	moduleLogs = []string{}
	packet := getModulePacket(t)
	LogPacket(AccountingProcess, packet)
	time.Sleep(100 * time.Millisecond)
	if len(moduleLogs) != 6 {
		t.Error("should have logged a packet")
	}
}

func getModulePacket(t *testing.T) *ClientPacket {
	p := radius.New(radius.CodeAccessRequest, []byte("secret"))
	if err := rfc2865.UserName_AddString(p, "user"); err != nil {
		t.Error("unable to add user name")
	}
	if err := rfc2865.CallingStationID_AddString(p, "11-22-33-44-55-66"); err != nil {
		t.Error("unable to add calling statiron")
	}
	packet := NewClientPacket([]byte{}, "127.0.0.1")
	packet.Packet = p
	return packet
}

func TestUserMAC(t *testing.T) {
	SetUserAuths([]string{})
	if AuthorizeUserMAC(getModulePacket(t)) {
		t.Error("should not have authorized")
	}

	SetUserAuths([]string{"use2r.112233445566"})
	if AuthorizeUserMAC(getModulePacket(t)) {
		t.Error("should not have authorized")
	}

	SetUserAuths([]string{"user.112233445566"})
	if !AuthorizeUserMAC(getModulePacket(t)) {
		t.Error("should have authorized")
	}
}

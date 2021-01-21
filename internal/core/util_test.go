package core

import (
	"testing"
)

func TestIn(t *testing.T) {
	if IntegerIn(1, []int{0, 2}) {
		t.Error("in was wrong")
	}
	if !IntegerIn(3, []int{1, 2, 3}) {
		t.Error("in should be right...")
	}
}

func TestUserVLANLogin(t *testing.T) {
	if NewUserVLANLogin("user", "test") != "user@vlan.test" {
		t.Error("invalid user/vlan login")
	}
}

func TestGetTokenFrom(t *testing.T) {
	if GetTokenFromLogin("user") != "" {
		t.Error("not valid")
	}
	if GetTokenFromLogin("user@vlan.t") != "" {
		t.Error("no user")
	}
	if GetTokenFromLogin("user:token") != "token" {
		t.Error("user token was valid")
	}
	if GetTokenFromLogin("user:token:test") != "token:test" {
		t.Error("still valid with another delimiter")
	}
	if GetTokenFromLogin("user:token:test@vlan.id") != "token:test" {
		t.Error("still valid with another delimiter")
	}
}

func TestNewUserLogin(t *testing.T) {
	if NewUserLogin("abc", "xyz") != "abc:xyz" {
		t.Error("invalid login")
	}
}

func TestCleanMAC(t *testing.T) {
	mac, ok := CleanMAC("aba")
	if ok {
		t.Errorf("mac is invalid")
	}
	mac, ok = CleanMAC("aabb11:22:33:FF")
	if !ok || mac != "aabb112233ff" {
		t.Errorf("invalid mac")
	}
}

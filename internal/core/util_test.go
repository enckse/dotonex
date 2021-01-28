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
	u, tok := GetTokenFromLogin("user")
	if u != "" || tok != "" {
		t.Error("not valid")
	}
	u, tok = GetTokenFromLogin("user@vlan.t")
	if u != "" || tok != "" {
		t.Error("no user")
	}
	u, tok = GetTokenFromLogin("user:token")
	if u != "user" || tok != "token" {
		t.Error("user token was valid")
	}
	u, tok = GetTokenFromLogin("user:token:test")
	if u != "user" || tok != "token:test" {
		t.Error("still valid with another delimiter")
	}
	u, tok = GetTokenFromLogin("user:token:test@vlan.id")
	if u != "user" || tok != "token:test" {
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

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

func TestGetUserVLAN(t *testing.T) {
	if GetUserFromVLANLogin("user") != "user" {
		t.Error("no vlan indicator")
	}
	if GetUserFromVLANLogin("user@vlan.t") != "user" {
		t.Error("invalid parse")
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

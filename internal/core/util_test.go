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

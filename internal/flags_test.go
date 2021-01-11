package internal

import (
	"testing"
)

func TestArgs(t *testing.T) {
	p := ProcessFlags{}
	a := p.Args()
	if len(a) != 0 {
		t.Error("no args")
	}
	p.Config = "cfg"
	a = p.Args()
	if len(a) != 2 {
		t.Error("config not set")
	}
	if a[0] != "--config" || a[1] != "cfg" {
		t.Error("config not set")
	}
	p.Instance = "inst"
	a = p.Args()
	if len(a) != 4 {
		t.Error("instance not set")
	}
	if a[2] != "--instance" || a[3] != "inst" {
		t.Error("instance not set")
	}
	p.Debug = true
	a = p.Args()
	if len(a) != 5 {
		t.Error("debug not on")
	}
	if a[4] != "--debug" {
		t.Error("no debug flag")
	}
}

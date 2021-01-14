package internal

import (
	"testing"
)

func TestConfigLocalFile(t *testing.T) {
	c := ConfigFlags{Repo: "test"}
	if "test/bin/file.db" != c.LocalFile("file") {
		t.Error("invalid file result")
	}
}

func TestConfigValid(t *testing.T) {
	c := ConfigFlags{}
	if c.Valid() {
		t.Error("is invalid")
	}
	c.Repo = "test"
	if c.Valid() {
		t.Error("is invalid")
	}
	c.Repo = ""
	c.Mode = "test"
	if c.Valid() {
		t.Error("is invalid")
	}
	c.Repo = "test"
	if !c.Valid() {
		t.Error("is valid")
	}
}

func TestArgs(t *testing.T) {
	p := ProcessFlags{}
	a := p.Args("")
	if len(a) != 2 {
		t.Error("no args")
	}
	p.Directory = "cfg"
	a = p.Args("i")
	if len(a) != 4 {
		t.Error("config not set")
	}
	if a[0] != "--config" || a[1] != "cfg" {
		t.Error("config not set")
	}
	p.Instance = "inst"
	a = p.Args("inst")
	if len(a) != 4 {
		t.Error("instance not set")
	}
	if a[2] != "--instance" || a[3] != "inst" {
		t.Error("instance not set")
	}
	p.Debug = true
	a = p.Args("")
	if len(a) != 5 {
		t.Error("debug not on")
	}
	if a[4] != "--debug" {
		t.Error("no debug flag")
	}
}

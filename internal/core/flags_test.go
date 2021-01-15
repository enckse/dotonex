package core

import (
	"testing"
)

func TestComposeLocalFile(t *testing.T) {
	c := ComposeFlags{Repo: "test"}
	if "test/bin/file.db" != c.LocalFile("file") {
		t.Error("invalid file result")
	}
}

func TestComposeArgs(t *testing.T) {
	c := ComposeFlags{}
	if len(c.Args()) != 0 {
		t.Error("no args")
	}
	c.Command = []string{"TEST", "XYZ"}
	if len(c.Args()) != 2 {
		t.Error("command args")
	}
	c.Repo = "TEST"
	c.Token = "test"
	args := c.Args()
	if len(args) != 6 {
		t.Error("invalid args")
	}
	if args[0] != "--repository" {
		t.Error("invalid arg 1")
	}
	if args[1] != "TEST" {
		t.Error("invalid arg 2")
	}
	if args[2] != "--token" {
		t.Error("invalid arg 3")
	}
	if args[3] != "test" {
		t.Error("invalid arg 4")
	}
	if args[4] != "TEST" {
		t.Error("invalid arg 5")
	}
	if args[5] != "XYZ" {
		t.Error("invalid arg 6")
	}
}

func TestComposeValid(t *testing.T) {
	c := ComposeFlags{}
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

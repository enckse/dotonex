package core

import (
	"testing"
)

func TestNewPluginContext(t *testing.T) {
	c := NewPluginContext(&Configuration{Dir: "test"})
	if c.Lib != "test" {
		t.Error("invalid context")
	}
}

func TestCloneContext(t *testing.T) {
	c := NewPluginContext(&Configuration{Dir: "test"}).CloneContext()
	if c.Lib != "test" {
		t.Error("invalid context")
	}
}

func TestDisabled(t *testing.T) {
	if !Disabled("mode", []string{"mode"}) {
		t.Error("mode should be disabled")
	}
	if Disabled("modes", []string{"mode"}) {
		t.Error("mode should be enabled")
	}
}

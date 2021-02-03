package core

import (
	"fmt"
	"strings"
	"testing"
)

func envContains(env []string, key string) string {
	keyVal := fmt.Sprintf("%s=", key)
	for _, k := range env {
		if strings.HasPrefix(k, keyVal) {
			return strings.Split(k, "=")[1]
		}
	}
	return ""
}

func TestToEnv(t *testing.T) {
	c := &Composition{}
	c.Debug = false
	env := c.ToEnv([]string{"TEST"})
	if len(env) != 0 {
		t.Error("invalid env")
	}
	if envContains(env, "DOTONEX_DEBUG") != "" {
		t.Error("no debugging")
	}
	c.Debug = true
	env = c.ToEnv([]string{"TEST"})
	if len(env) != 2 {
		t.Error("invalid env")
	}
	if envContains(env, "DOTONEX_DEBUG") != "true" {
		t.Error("debugging")
	}
	c.Search = []string{"TEST", "XYZ"}
	env = c.ToEnv([]string{"TEST"})
	if len(env) != 3 {
		t.Error("invalid env")
	}
	if envContains(env, "DOTONEX_DEBUG") != "true" {
		t.Error("debugging")
	}
	if envContains(env, "DOTONEX_SEARCH") != "TEST XYZ" {
		t.Error("searching")
	}
	c.Debug = false
	c.Search = []string{"TEST", "XYZ"}
	env = c.ToEnv([]string{"TEST"})
	if len(env) != 2 {
		t.Error("invalid env")
	}
	if envContains(env, "DOTONEX_DEBUG") != "" {
		t.Error("debugging")
	}
	if envContains(env, "DOTONEX_SEARCH") != "TEST XYZ" {
		t.Error("searching")
	}
}

func TestDefaults(t *testing.T) {
	c := &Configuration{}
	c.Defaults([]byte{})
	if c.Host != "localhost" {
		t.Error("invalid default host")
	}
	if c.Log != "/var/log/dotonex/" {
		t.Error("invalid log dir")
	}
	if c.Compose.Timeout != 30 {
		t.Error("invalid timeout")
	}
	if c.Compose.Refresh != 5 {
		t.Error("invalid refresh")
	}
	if c.Accounting {
		t.Error("wrong type")
	}
	if c.Bind != 1812 {
		t.Error("invalid port")
	}
	if c.Internals.Logs != 10 {
		t.Error("invalid log buffer")
	}
	if c.Internals.MaxConnections != 100000 {
		t.Error("invalid max connect check")
	}
	if c.Internals.Lifespan != 12 {
		t.Error("invalid lifespan")
	}
	l := c.Internals.LifeHours
	for _, o := range []int{22, 23, 0, 1, 2, 3, 4, 5} {
		if !IntegerIn(o, l) {
			t.Error("invalid hour defaults")
		}
	}
}

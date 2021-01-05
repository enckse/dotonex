package server

import (
	yaml "gopkg.in/yaml.v2"
	"voidedtech.com/radiucal/internal/core"
)

type (
	// Endpoint is a hosted endpoint (e.g. auth/acct)
	Endpoint struct {
		Host   string
		Port   int
		To     int
		Mods   []string
		Enable bool
	}
	// Configuration is the configuration definition
	Configuration struct {
		Auth    Endpoint
		Acct    Endpoint
		Dir     string
		Logging struct {
			Dir   string
			Flush int
		}
		Secret string
		Users  string
		Debug  bool
		Gitlab struct {
			URL    string
			Token  string
			Enable bool
		}
	}
)

// Dump writes debug information about the configuration
func (c *Configuration) Dump() {
	config, err := yaml.Marshal(c)
	if err == nil {
		core.WriteDebug("configuration", string(config))
	} else {
		core.WriteError("unable to read yaml configuration", err)
	}
}

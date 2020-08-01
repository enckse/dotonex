package server

import (
	yaml "gopkg.in/yaml.v2"
	"voidedtech.com/radiucal/internal/core"
)

type (
	// Configuration is the configuration definition
	Configuration struct {
		Cache      bool
		Host       string
		Accounting bool
		To         int
		Bind       int
		Dir        string
		Log        string
		Logs       int
		Plugins    []string
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

func defaultString(given, dflt string) string {
	if len(given) == 0 {
		return dflt
	}
	return given
}

// Defaults will set uninitialized values to default values
func (c *Configuration) Defaults() {
	c.Host = defaultString(c.Host, "localhost")
	c.Dir = defaultString(c.Dir, "/var/lib/radiucal/")
	c.Log = defaultString(c.Log, "/var/log/radiucal/")
	if c.Bind <= 0 {
		if c.Accounting {
			c.Bind = 1813
		} else {
			c.Bind = 1812
		}
	}
	if c.Logs <= 0 {
		c.Logs = 10
	}
}

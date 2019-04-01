package core

import (
	"voidedtech.com/goutils/logger"
	"voidedtech.com/goutils/yaml"
)

// Configuration is the configuration definition
type Configuration struct {
	Debug      bool
	Cache      bool
	Host       string
	Accounting bool
	To         int
	Bind       int
	Dir        string
	NoReject   bool
	Log        string
	Plugins    []string
	Disable    struct {
		Accounting []string
		Preauth    []string
		Trace      []string
		Postauth   []string
	}
	backing []byte
}

// Dump writes debug information about the configuration
func (c *Configuration) Dump() {
	config, err := yaml.MarshalToBytes(c)
	if err == nil {
		logger.WriteDebug("configuration (mem/raw)", string(config), string(c.backing))
	} else {
		logger.WriteError("unable to read yaml configuration", err)
	}
}

func defaultString(given, dflt string) string {
	if len(given) == 0 {
		return dflt
	} else {
		return given
	}
}

// Defaults will set uninitialized values to default values
func (c *Configuration) Defaults(backing []byte) {
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
	c.backing = backing
}

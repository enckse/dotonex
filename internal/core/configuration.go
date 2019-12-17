package core

import yaml "gopkg.in/yaml.v2"

type (
	// Configuration is the configuration definition
	Configuration struct {
		Cache      bool
		Host       string
		Accounting bool
		To         int
		Bind       int
		Dir        string
		NoReject   bool
		Log        string
		Plugins    []string
		Internals  struct {
			NoInterrupt bool
			NoLogs      bool
			Logs        int
			Lifespan    int
			SpanCheck   int
			LifeHours   []int
		}
		Disable struct {
			Accounting []string
			Preauth    []string
			Trace      []string
			Postauth   []string
		}
	}
)

// Dump writes debug information about the configuration
func (c *Configuration) Dump() {
	config, err := yaml.Marshal(c)
	if err == nil {
		WriteDebug("configuration", string(config))
	} else {
		WriteError("unable to read yaml configuration", err)
	}
}

func defaultString(given, dflt string) string {
	if len(given) == 0 {
		return dflt
	}
	return given
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
	if c.Internals.Logs <= 0 {
		c.Internals.Logs = 10
	}
	if c.Internals.Lifespan <= 0 {
		c.Internals.Lifespan = 12
	}
	if c.Internals.SpanCheck <= 0 {
		c.Internals.SpanCheck = 1
	}
	if len(c.Internals.LifeHours) == 0 {
		c.Internals.LifeHours = []int{22, 23, 0, 1, 2, 3, 4, 5}
	}
}

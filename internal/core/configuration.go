package core

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type (
	// Composition represents compose configurations
	Composition struct {
		Debug      bool
		Static     bool
		Repository string
		Payload    []string
		ServerKey  string
		Refresh    int
		Polling    bool
		Timeout    int
		Binary     string
		UserRegex  string
		Search     []string
	}

	// MonitorState is for configuration of state monitoring of internals
	MonitorState struct {
		Check int
		Count int
	}

	// Configuration is the configuration definition
	Configuration struct {
		Preload    []string
		Host       string
		Accounting bool
		To         int
		Bind       int
		NoReject   bool
		Log        string
		NoTrace    bool
		PacketKey  string
		Compose    Composition
		Internals  struct {
			NoInterrupt    bool
			NoLogs         bool
			Logs           int
			Lifespan       int
			LifeCheck      int
			LifeHours      []int
			MaxConnections MonitorState
			ClientFailures MonitorState
		}
		Quit struct {
			Wait    bool
			Timeout int
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
	c.Log = defaultString(c.Log, "/var/log/dotonex/")
	if c.Bind <= 0 {
		if c.Accounting {
			c.Bind = 1813
		} else {
			c.Bind = 1812
		}
	}
	c.Compose.Repository = defaultString(c.Compose.Repository, "/var/lib/dotonex/config")
	if c.Compose.Refresh <= 0 {
		c.Compose.Refresh = 5
	}
	if c.Compose.Timeout <= 0 {
		c.Compose.Timeout = 30
	}
	if c.Internals.Logs <= 0 {
		c.Internals.Logs = 10
	}
	if c.Internals.Lifespan <= 0 {
		c.Internals.Lifespan = 12
	}
	if c.Internals.MaxConnections.Count <= 0 {
		c.Internals.MaxConnections.Count = 100000
	}
	if len(c.Internals.LifeHours) == 0 {
		c.Internals.LifeHours = []int{22, 23, 0, 1, 2, 3, 4, 5}
	}
	if c.Internals.ClientFailures.Count <= 0 {
		c.Internals.ClientFailures.Count = 100
	}
}

// ToEnv will convert composition options to actual command environments
func (c Composition) ToEnv(rootEnv []string) []string {
	var env []string
	if c.Debug {
		env = newEnv(DebugEnvVariable, DebugEnvOn, env, rootEnv)
	}
	if len(c.Search) > 0 {
		env = newEnv(SearchEnvVariable, strings.Join(c.Search, " "), env, rootEnv)
	}
	return env
}

func newEnv(key, value string, env, root []string) []string {
	keyVal := fmt.Sprintf("%s=%s", key, value)
	if len(env) == 0 {
		return append(root, keyVal)
	}
	return append(env, keyVal)
}

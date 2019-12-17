package core

import (
	"flag"
)

type (
	// ProcessFlags represent command line arguments to the process
	ProcessFlags struct {
		Config   string
		Instance string
		Debug    bool
	}
)

const (
	configFlag   = "config"
	instanceFlag = "instance"
	debugFlag    = "debug"
)

// Args converts the process flags back to callable arguments
func (p ProcessFlags) Args() []string {
	args := []string{"--" + configFlag, p.Config, "--" + instanceFlag, p.Instance}
	if p.Debug {
		args = append(args, "--"+debugFlag)
	}
	return args
}

// Flags parses CLI flags for radiucal
func Flags() ProcessFlags {
	var cfg = flag.String(configFlag, "/etc/radiucal/radiucal.conf", "Configuration file")
	var instance = flag.String(instanceFlag, "", "Instance name")
	var debugging = flag.Bool(debugFlag, false, "Enable debugging")
	flag.Parse()
	return ProcessFlags{
		Config:   *cfg,
		Instance: *instance,
		Debug:    *debugging,
	}
}

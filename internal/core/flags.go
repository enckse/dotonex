package core

import (
	"flag"
)

type (
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

func (p ProcessFlags) Args() []string {
	args := []string{"--" + configFlag, p.Config, "--" + instanceFlag, p.Instance}
	if p.Debug {
		args = append(args, "--"+debugFlag)
	}
	return args
}

func Flags() ProcessFlags {
	var cfg = flag.String(configFlag, "/etc/radiucal/radiucal.conf", "Configuration file")
	var instance = flag.String(instanceFlag, "", "Instance name")
	var debugging = flag.Bool(debugFlag, false, "debugging")
	flag.Parse()
	return ProcessFlags{
		Config:   *cfg,
		Instance: *instance,
		Debug:    *debugging,
	}
}

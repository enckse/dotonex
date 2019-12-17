package server

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
	dash         = "--"
)

// Args converts the process flags back to callable arguments
func (p ProcessFlags) Args() []string {
	var args []string
	for k, v := range map[string]string{
		dash + configFlag:   p.Config,
		dash + instanceFlag: p.Instance,
	} {
		if len(v) > 0 {
			args = append(args, []string{k, v}...)
		}
	}
	if p.Debug {
		args = append(args, dash+debugFlag)
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

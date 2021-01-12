package internal

import (
	"flag"
)

type (
	// ProcessFlags represent command line arguments to the process
	ProcessFlags struct {
		Directory string
		Instance  string
		Debug     bool
	}
)

const (
	configFlag   = "config"
	instanceFlag = "instance"
	debugFlag    = "debug"
	dash         = "--"
	// InstanceConfig indicates a configuration file of instance type
	InstanceConfig = ".conf"
)

// Args converts the process flags back to callable arguments
func (p ProcessFlags) Args() []string {
	var args []string
	if len(p.Directory) > 0 {
		args = append(args, dash+configFlag)
		args = append(args, p.Directory)
	}
	if len(p.Instance) > 0 {
		args = append(args, dash+instanceFlag)
		args = append(args, p.Instance)
	}
	if p.Debug {
		args = append(args, dash+debugFlag)
	}
	return args
}

// Flags parses CLI flags for dotonex
func Flags() ProcessFlags {
	var dir = flag.String(configFlag, "/etc/dotonex/", "Configuration file")
	var instance = flag.String(instanceFlag, "", "Instance name")
	var debugging = flag.Bool(debugFlag, false, "Enable debugging")
	flag.Parse()
	return ProcessFlags{
		Directory: *dir,
		Instance:  *instance,
		Debug:     *debugging,
	}
}

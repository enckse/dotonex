package internal

import (
	"flag"
	"path/filepath"
)

type (
	// ProcessFlags represent command line arguments to the process
	ProcessFlags struct {
		Directory string
		Instance  string
		Debug     bool
	}
	ConfigFlags struct {
		Mode    string
		Repo    string
		Hash    string
		MAC     string
		Token   string
		Command []string
	}
)

const (
	configFlag   = "config"
	instanceFlag = "instance"
	debugFlag    = "debug"
	dash         = "--"
	modeFlag     = dash + "mode"
	repoFlag     = dash + "repository"
	macFlag      = dash + "mac"
	tokenFlag    = dash + "token"
	hashFlag     = dash + "hash"
	configTarget = "bin"
	configData   = ".db"
	// InstanceConfig indicates a configuration file of instance type
	InstanceConfig = ".conf"
	// Config modes
	ModeValidate = "validate"
	ModeServer   = "server"
	ModeFetch    = "fetch"
	ModeBuild    = "build"
	ModeRebuild  = "rebuild"
)

func (c ConfigFlags) LocalFile(name string) string {
	return filepath.Join(c.Repo, configTarget, name+configData)
}

func GetConfigFlags() ConfigFlags {
	mode := flag.String(modeFlag, "", "operating mode")
	repo := flag.String(repoFlag, "", "repository to work on")
	mac := flag.String(macFlag, "", "MAC address")
	hash := flag.String(hashFlag, "", "server hash")
	token := flag.String(tokenFlag, "", "token to validate")
	flag.Parse()
	args := flag.Args()
	return ConfigFlags{Mode: *mode,
		Repo:    *repo,
		MAC:     *mac,
		Token:   *token,
		Hash:    *hash,
		Command: args}
}

func (c ConfigFlags) Valid() bool {
	return len(c.Mode) > 0 && len(c.Repo) > 0
}

// Args converts the process flags back to callable arguments
func (p ProcessFlags) Args(instance string) []string {
	var args []string
	if len(p.Directory) > 0 {
		args = append(args, dash+configFlag)
		args = append(args, p.Directory)
	}
	args = append(args, dash+instanceFlag)
	args = append(args, instance)
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

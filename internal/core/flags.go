package core

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type (
	// ProcessFlags represent command line arguments to the process
	ProcessFlags struct {
		Directory string
		Instance  string
		Debug     bool
	}

	// ComposeFlags are config backend arguments
	ComposeFlags struct {
		Mode    string
		Repo    string
		Hash    string
		MAC     string
		Token   string
		Search  []string
		Debug   bool
		Command []string
	}
)

const (
	configFlag   = "config"
	instanceFlag = "instance"
	debugFlag    = "debug"
	dash         = "--"
	modeFlag     = "mode"
	repoFlag     = "repository"
	macFlag      = "mac"
	tokenFlag    = "token"
	hashFlag     = "hash"
	// InstanceConfig indicates a configuration file of instance type
	InstanceConfig = ".conf"
	// ModeValidate tells configuration to validate a user+mac
	ModeValidate = "validate"
	// ModeServer will configure the baseline server requirements
	ModeServer = "server"
	// ModeFetch will indicate changes should be fetched remotely
	ModeFetch = "fetch"
	// ModeBuild will indicate optional rebuild
	ModeBuild = "build"
	// ModeRebuild will force rebuild
	ModeRebuild = "rebuild"
	// ModeMAC will check for MAC validity
	ModeMAC = "mac"
	// DebugEnvOn indicates environment variable debugging is on for processes
	DebugEnvOn = "true"
	// DebugEnvVariable is the environment variable to indicate debug state
	DebugEnvVariable = "DOTONEX_DEBUG"
	// SearchEnvVariable is an underlying method to set how the configurator search for keys
	SearchEnvVariable = "DOTONEX_SEARCH"
)

// Debugging writes potential information from composition if debugging is one
func (c ComposeFlags) Debugging(message string) {
	if c.Debug {
		WriteInfo(message)
	}
}

func argIfSet(flag, value string, appendTo []string) []string {
	if len(value) > 0 {
		appendTo = append(appendTo, fmt.Sprintf("%s%s", dash, flag))
		appendTo = append(appendTo, value)
	}
	return appendTo
}

// Args compose flags into callable arguments
func (c ComposeFlags) Args() []string {
	var flags []string
	flags = argIfSet(modeFlag, c.Mode, flags)
	flags = argIfSet(repoFlag, c.Repo, flags)
	flags = argIfSet(tokenFlag, c.Token, flags)
	flags = argIfSet(hashFlag, c.Hash, flags)
	flags = argIfSet(macFlag, c.MAC, flags)
	if len(c.Command) > 0 {
		flags = append(flags, c.Command...)
	}
	return flags
}

// GetComposeFlags will get the arguments for configuration backend needs
func GetComposeFlags() ComposeFlags {
	mode := flag.String(modeFlag, "", "operating mode")
	repo := flag.String(repoFlag, "", "repository to work on")
	mac := flag.String(macFlag, "", "MAC address")
	hash := flag.String(hashFlag, "", "server hash")
	token := flag.String(tokenFlag, "", "token to validate")
	flag.Parse()
	args := flag.Args()
	debug := os.Getenv(DebugEnvVariable) == DebugEnvOn
	search := []string{"username"}
	searchEnv := strings.TrimSpace(os.Getenv(SearchEnvVariable))
	if searchEnv != "" {
		search = strings.Split(searchEnv, " ")
	}
	return ComposeFlags{Mode: *mode,
		Repo:    *repo,
		MAC:     *mac,
		Token:   *token,
		Hash:    *hash,
		Search:  search,
		Debug:   debug,
		Command: args}
}

// Valid will check the basics for valid config backend flags
func (c ComposeFlags) Valid() bool {
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

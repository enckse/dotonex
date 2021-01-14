package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"voidedtech.com/dotonex/internal"
)

const (
	serverHash = "server"
)

func main() {
	if err := run(); err != nil {
		internal.WriteError("config failure", err)
		os.Exit(1)
	}
}

func validate(flags internal.ConfigFlags) error {
	return nil
}

func fetch(flags internal.ConfigFlags) error {
	for _, cmd := range []string{"fetch", "pull"} {
		command := exec.Command("git", "-C", flags.Repo, cmd)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return err
		}
	}
	return nil
}

func server(flags internal.ConfigFlags) error {
	hash := flags.LocalFile(serverHash)
	if internal.PathExists(hash) {
		curr, err := ioutil.ReadFile(hash)
		if err != nil {
			return err
		}
		if string(curr) == flags.Hash {
			return nil
		}
	}
	internal.WriteInfo("hash update")
	if err := ioutil.WriteFile(hash, []byte(flags.Hash), 0644); err != nil {
		return err
	}
	return build(flags, true)
}

func build(flags internal.ConfigFlags, force bool) error {
	return nil
}

func run() error {
	flags := internal.GetConfigFlags()
	if !flags.Valid() {
		return fmt.Errorf("invalid arguments")
	}
	if !internal.PathExists(flags.Repo) {
		return fmt.Errorf("repository invalid/does not exist")
	}

	target := filepath.Dir(flags.LocalFile(""))
	if !internal.PathExists(target) {
		internal.WriteInfo("creating target")
		if err := os.Mkdir(target, 0600); err != nil {
			return err
		}
	}
	switch flags.Mode {
	case internal.ModeValidate:
		if len(flags.Command) == 0 || len(flags.Token) == 0 || len(flags.MAC) == 0 {
			return fmt.Errorf("missing flags for validation")
		}
		return validate(flags)
	case internal.ModeServer:
		if len(flags.Hash) == 0 {
			return fmt.Errorf("missing flags for server")
		}
		return server(flags)
	case internal.ModeFetch:
		return fetch(flags)
	case internal.ModeBuild:
		return build(flags, false)
	case internal.ModeRebuild:
		return build(flags, true)
	default:
		return fmt.Errorf("unknown mode")
	}
	return nil
}

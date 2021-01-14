package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"voidedtech.com/dotonex/internal"
)

const (
	serverHash = "server"
	perms      = 0600
	vlanConfig = "vlans.cfg"
)

func main() {
	if err := run(); err != nil {
		internal.WriteError("config failure", err)
		os.Exit(1)
	}
}

func piped(args []string) (string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	o := strings.TrimSpace(stdout.String())
	if len(o) > 0 {
		internal.WriteInfo("stdout")
		internal.WriteInfo(o)
	}
	e := stderr.String()
	if len(e) > 0 {
		internal.WriteInfo("stderr")
		internal.WriteInfo(e)
		return "", fmt.Errorf("command errored")
	}
	return o, nil
}

func validate(flags internal.ConfigFlags) error {
	internal.WriteInfo("validating inputs")
	mac, ok := internal.CleanMAC(flags.MAC)
	if !ok {
		return fmt.Errorf("invalid MAC")
	}
	hash := internal.MD4(flags.Token)
	tokenFile := flags.LocalFile(hash)
	user := ""
	if internal.PathExists(tokenFile) {
		internal.WriteInfo("token is known")
		b, err := ioutil.ReadFile(tokenFile)
		if err != nil {
			return err
		}
		user = string(b)
	}
	change := false
	if user == "" {
		command := []string{}
		for _, c := range flags.Command {
			command = append(command, fmt.Sprintf(c, flags.Token))
		}
		output, err := piped(command)
		if err != nil {
			return err
		}
		m := make(map[string]string)
		if err := json.Unmarshal([]byte(output), &m); err != nil {
			return err
		}
		if _, ok := m["username"]; !ok {
			return fmt.Errorf("invalid json, required key missing")
		}
		user = m["username"]
		internal.WriteInfo(fmt.Sprintf("%s token changed", user))
		if err := ioutil.WriteFile(tokenFile, []byte(user), perms); err != nil {
			return err
		}
		change = true
		internal.WriteInfo("token validated")
	}
	if user == "" {
		return fmt.Errorf("empty user found")
	}
	internal.WriteInfo(fmt.Sprintf("user found: %s", user))
	if change {
		internal.WriteInfo("user is new")
		userFile := flags.LocalFile(user)
		if err := ioutil.WriteFile(userFile, []byte(flags.Token), perms); err != nil {
			return err
		}
		if err := build(flags, true); err != nil {
			return err
		}
	}
	userDir := filepath.Join(flags.Repo, user)
	for _, file := range []string{mac, vlanConfig} {
		if !internal.PathExists(filepath.Join(userDir, file)) {
			return fmt.Errorf("%s file not found", file)
		}
	}
	internal.WriteInfo("validated")
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
	if err := ioutil.WriteFile(hash, []byte(flags.Hash), perms); err != nil {
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
		if err := os.Mkdir(target, 0700); err != nil {
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

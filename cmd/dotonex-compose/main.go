package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
	"voidedtech.com/dotonex/internal/compose"
	"voidedtech.com/dotonex/internal/core"
)

const (
	serverHash = "server"
	perms      = 0600
	vlanConfig = "vlans.cfg"
)

func main() {
	if err := run(); err != nil {
		core.WriteError("config failure", err)
		os.Exit(1)
	}
}

func piped(flags core.ComposeFlags, args []string) (string, error) {
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
		flags.Debugging("stdout")
		flags.Debugging(o)
	}
	e := stderr.String()
	if len(e) > 0 {
		flags.Debugging("stderr")
		flags.Debugging(e)
		return "", fmt.Errorf("command errored")
	}
	return o, nil
}

func validate(flags core.ComposeFlags) error {
	flags.Debugging("validating inputs")
	mac, ok := core.CleanMAC(flags.MAC)
	if !ok {
		return fmt.Errorf("invalid MAC")
	}
	hash := core.MD4(flags.Token)
	tokenFile := flags.LocalFile(hash)
	user := ""
	if core.PathExists(tokenFile) {
		flags.Debugging("token is known")
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
			text := c
			if strings.Contains(c, "%s") {
				text = fmt.Sprintf(c, flags.Token)
			}
			command = append(command, text)
		}
		output, err := piped(flags, command)
		if err != nil {
			return err
		}
		user, err = compose.TryGetUser([]byte(output), func(possibleUser string) bool {
			return core.PathExists(filepath.Join(flags.Repo, possibleUser))
		})
		if err != nil {
			return err
		}
		flags.Debugging(fmt.Sprintf("%s token changed", user))
		if err := ioutil.WriteFile(tokenFile, []byte(user), perms); err != nil {
			return err
		}
		change = true
		flags.Debugging("token validated")
	}
	if user == "" {
		return fmt.Errorf("empty user found")
	}
	flags.Debugging(fmt.Sprintf("user found: %s", user))
	if change {
		flags.Debugging("user is new")
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
		if !core.PathExists(filepath.Join(userDir, file)) {
			return fmt.Errorf("%s file not found", file)
		}
	}
	flags.Debugging("validated")
	return nil
}

func fetch(flags core.ComposeFlags) error {
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

func compareFileToText(file, text string) (bool, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return false, err
	}
	return text == string(b), nil
}

func server(flags core.ComposeFlags) error {
	hash := flags.LocalFile(serverHash)
	if core.PathExists(hash) {
		same, err := compareFileToText(hash, flags.Hash)
		if err != nil {
			return err
		}
		if same {
			return nil
		}
	}
	flags.Debugging("hash update")
	if err := ioutil.WriteFile(hash, []byte(flags.Hash), perms); err != nil {
		return err
	}
	return build(flags, true)
}

func getHostapd(flags core.ComposeFlags, def compose.Definition) ([]compose.Hostapd, error) {
	hashFile := flags.LocalFile(serverHash)
	if !core.PathExists(hashFile) {
		return nil, fmt.Errorf("no server hash found")
	}
	b, err := ioutil.ReadFile(hashFile)
	if err != nil {
		return nil, err
	}
	hash := string(b)
	dirs, err := ioutil.ReadDir(flags.Repo)
	if err != nil {
		return nil, err
	}
	if len(hash) == 0 {
		return nil, fmt.Errorf("empty hash")
	}
	var result []compose.Hostapd
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		name := dir.Name()
		if name == core.ComposeTarget {
			continue
		}
		path := filepath.Join(flags.Repo, name)
		if id, ok := def.IsVLAN(name); ok {
			flags.Debugging(fmt.Sprintf("%s (MAB)", name))
			sub, err := ioutil.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, mac := range sub {
				cleaned, ok := core.CleanMAC(mac.Name())
				if !ok {
					continue
				}
				flags.Debugging(fmt.Sprintf(" -> %s", cleaned))
				result = append(result, compose.NewHostapd(cleaned, cleaned, id))
			}
			continue
		}
		possible := filepath.Join(path, vlanConfig)
		if !core.PathExists(possible) {
			continue
		}
		secret := flags.LocalFile(name)
		if !core.PathExists(secret) {
			continue
		}
		flags.Debugging(fmt.Sprintf("%s (USER)", name))
		b, err := ioutil.ReadFile(secret)
		if err != nil {
			return nil, err
		}
		loginName := string(b)
		if len(loginName) == 0 {
			core.WriteWarn("empty login name")
			continue
		}
		loginName = core.NewUserLogin(name, loginName)
		b, err = ioutil.ReadFile(possible)
		if err != nil {
			return nil, err
		}
		d := compose.Definition{}
		if err := yaml.Unmarshal(b, &d); err != nil {
			core.WriteError("unable to read user yaml", err)
			continue
		}
		if err := d.ValidateMembership(); err != nil {
			core.WriteError("invalid memberships found", err)
			continue
		}
		first := true
		for _, member := range d.Membership {
			id, ok := def.IsVLAN(member.VLAN)
			if !ok {
				core.WriteWarn(fmt.Sprintf("invalid VLAN %s", member.VLAN))
				continue
			}
			if first {
				result = append(result, compose.NewHostapd(loginName, hash, id))
				first = false
			}
			result = append(result, compose.NewHostapd(core.NewUserVLANLogin(loginName, member.VLAN), hash, id))
		}
	}
	return result, nil
}

func getVLANs(flags core.ComposeFlags) (compose.Definition, error) {
	cfg := filepath.Join(flags.Repo, vlanConfig)
	d := compose.Definition{}
	if !core.PathExists(cfg) {
		return d, fmt.Errorf("no root vlan config found")
	}
	b, err := ioutil.ReadFile(cfg)
	if err != nil {
		return d, err
	}
	if err := yaml.Unmarshal(b, &d); err != nil {
		return d, err
	}

	if err := d.ValidateVLANs(); err != nil {
		return d, err
	}
	return d, nil
}

func configure(flags core.ComposeFlags) error {
	flags.Debugging("configuring")
	vlans, err := getVLANs(flags)
	if err != nil {
		return err
	}
	hostapd, err := getHostapd(flags, vlans)
	if err != nil {
		return err
	}
	var eapUsers []string
	for _, h := range hostapd {
		eapUsers = append(eapUsers, h.String())
	}
	if len(eapUsers) == 0 {
		return fmt.Errorf("no hostapd configurations found")
	}
	sort.Strings(eapUsers)
	hostapdFile := filepath.Join(flags.Repo, core.ComposeTarget, "eap_users")
	hostapdText := strings.Join(eapUsers, "\n\n") + "\n"
	if core.PathExists(hostapdFile) {
		same, err := compareFileToText(hostapdFile, hostapdText)
		if err != nil {
			return err
		}
		if same {
			flags.Debugging("no hostapd changes")
			return nil
		}
	}
	if err := ioutil.WriteFile(hostapdFile, []byte(hostapdText), perms); err != nil {
		return err
	}
	return resetHostapd(flags)
}

func resetHostapd(flags core.ComposeFlags) error {
	flags.Debugging("hostapd reset")
	pids, err := piped(flags, []string{"pidof", "hostapd"})
	if err != nil {
		core.WriteWarn(fmt.Sprintf("unable to get hostapd pids: %v", err))
		return nil
	}
	for _, pid := range strings.Split(pids, " ") {
		p := strings.TrimSpace(pid)
		if len(p) == 0 {
			continue
		}
		cmd := exec.Command("kill", "-HUP", p)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func build(flags core.ComposeFlags, force bool) error {
	if !force {
		last, err := piped(flags, []string{"git", "-C", flags.Repo, "log", "-n", "1", "--format=%h"})
		if err != nil {
			return err
		}
		last = strings.TrimSpace(last)
		if len(last) == 0 {
			return fmt.Errorf("no commit retrieved")
		}
		lastFile := flags.LocalFile("commit")
		if core.PathExists(lastFile) {
			same, err := compareFileToText(lastFile, last)
			if err != nil {
				return err
			}
			if same {
				flags.Debugging("no config changes found")
				return nil
			}
		}
		if err := ioutil.WriteFile(lastFile, []byte(last), perms); err != nil {
			return err
		}
	}
	return configure(flags)
}

func run() error {
	flags := core.GetComposeFlags()
	if !flags.Valid() {
		return fmt.Errorf("invalid arguments")
	}
	if !core.PathExists(flags.Repo) {
		return fmt.Errorf("repository invalid/does not exist")
	}

	target := filepath.Dir(flags.LocalFile(""))
	if !core.PathExists(target) {
		flags.Debugging("creating target")
		if err := os.Mkdir(target, 0700); err != nil {
			return err
		}
	}
	switch flags.Mode {
	case core.ModeValidate:
		if len(flags.Command) == 0 || len(flags.Token) == 0 || len(flags.MAC) == 0 {
			return fmt.Errorf("missing flags for validation")
		}
		return validate(flags)
	case core.ModeServer:
		if len(flags.Hash) == 0 {
			return fmt.Errorf("missing flags for server")
		}
		return server(flags)
	case core.ModeFetch:
		return fetch(flags)
	case core.ModeBuild:
		return build(flags, false)
	case core.ModeRebuild:
		return build(flags, true)
	default:
		return fmt.Errorf("unknown mode")
	}
	return nil
}

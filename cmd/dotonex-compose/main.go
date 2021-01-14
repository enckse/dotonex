package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
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
			text := c
			if strings.Contains(c, "%s") {
				text = fmt.Sprintf(c, flags.Token)
			}
			command = append(command, text)
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

func compareFileToText(file, text string) (bool, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return false, err
	}
	return text == string(b), nil
}

func server(flags internal.ConfigFlags) error {
	hash := flags.LocalFile(serverHash)
	if internal.PathExists(hash) {
		same, err := compareFileToText(hash, flags.Hash)
		if err != nil {
			return err
		}
		if same {
			return nil
		}
	}
	internal.WriteInfo("hash update")
	if err := ioutil.WriteFile(hash, []byte(flags.Hash), perms); err != nil {
		return err
	}
	return build(flags, true)
}

func getHostapd(flags internal.ConfigFlags, def internal.Definition) ([]internal.Hostapd, error) {
	hashFile := flags.LocalFile(serverHash)
	if !internal.PathExists(hashFile) {
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
	var result []internal.Hostapd
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		name := dir.Name()
		if name == internal.ConfigTarget {
			continue
		}
		path := filepath.Join(flags.Repo, name)
		if id, ok := def.IsVLAN(name); ok {
			internal.WriteInfo(fmt.Sprintf("%s (MAB)", name))
			sub, err := ioutil.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, mac := range sub {
				cleaned, ok := internal.CleanMAC(mac.Name())
				if !ok {
					continue
				}
				internal.WriteInfo(fmt.Sprintf(" -> %s", cleaned))
				result = append(result, internal.NewHostapd(cleaned, cleaned, id))
			}
			continue
		}
		possible := filepath.Join(path, vlanConfig)
		if !internal.PathExists(possible) {
			continue
		}
		secret := flags.LocalFile(name)
		if !internal.PathExists(secret) {
			continue
		}
		internal.WriteInfo(fmt.Sprintf("%s (USER)", name))
		b, err := ioutil.ReadFile(secret)
		if err != nil {
			return nil, err
		}
		loginName := string(b)
		if len(loginName) == 0 {
			internal.WriteWarn("empty login name")
			continue
		}
		b, err = ioutil.ReadFile(possible)
		if err != nil {
			return nil, err
		}
		d := internal.Definition{}
		if err := yaml.Unmarshal(b, &d); err != nil {
			internal.WriteError("unable to read user yaml", err)
			continue
		}
		if err := d.ValidateMembership(); err != nil {
			internal.WriteError("invalid memberships found", err)
			continue
		}
		first := true
		for _, member := range d.Membership {
			id, ok := def.IsVLAN(member.VLAN)
			if !ok {
				internal.WriteWarn(fmt.Sprintf("invalid VLAN %s", member.VLAN))
				continue
			}
			if first {
				result = append(result, internal.NewHostapd(loginName, hash, id))
				first = false
			}
			result = append(result, internal.NewHostapd(fmt.Sprintf("%s:%s", member.VLAN, loginName), hash, id))
		}
	}
	return result, nil
}

func getVLANs(flags internal.ConfigFlags) (internal.Definition, error) {
	cfg := filepath.Join(flags.Repo, vlanConfig)
	d := internal.Definition{}
	if !internal.PathExists(cfg) {
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

func configure(flags internal.ConfigFlags) error {
	internal.WriteInfo("configuring")
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
	hostapdFile := filepath.Join(flags.Repo, internal.ConfigTarget, "eap_users")
	hostapdText := strings.Join(eapUsers, "\n\n") + "\n"
	if internal.PathExists(hostapdFile) {
		same, err := compareFileToText(hostapdFile, hostapdText)
		if err != nil {
			return err
		}
		if same {
			internal.WriteInfo("no hostapd changes")
			return nil
		}
	}
	if err := ioutil.WriteFile(hostapdFile, []byte(hostapdText), perms); err != nil {
		return err
	}
	return resetHostapd()
}

func resetHostapd() error {
	internal.WriteInfo("hostapd reset")
	pids, err := piped([]string{"pidof", "hostapd"})
	if err != nil {
		return err
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

func build(flags internal.ConfigFlags, force bool) error {
	if !force {
		last, err := piped([]string{"git", "-C", flags.Repo, "log", "-n", "1", "--format=%h"})
		if err != nil {
			return err
		}
		last = strings.TrimSpace(last)
		if len(last) == 0 {
			return fmt.Errorf("no commit retrieved")
		}
		lastFile := flags.LocalFile("commit")
		if internal.PathExists(lastFile) {
			same, err := compareFileToText(lastFile, last)
			if err != nil {
				return err
			}
			if same {
				internal.WriteInfo("no config changes found")
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

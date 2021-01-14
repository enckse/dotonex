package internal

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	lock    = &sync.Mutex{}
	backend *script
)

type (
	script struct {
		repo    string
		command []string
		hash    string
		static  bool
		timeout time.Duration
		payload []string
		debug   bool
	}
)

func (s script) execute(flags ComposeFlags) bool {
	flags.Repo = s.repo
	arguments := flags.Args()

	if s.debug {
		WriteInfo(fmt.Sprintf("running: %v", arguments))
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "dotonex-compose", arguments...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		WriteWarn("script timeout")
		return false
	}
	str := stderr.String()
	if len(str) > 0 {
		WriteInfo("stderr")
		WriteInfo(str)
	}
	if s.debug {
		str = strings.TrimSpace(string(out))
		if len(str) > 0 {
			WriteInfo("stdout")
			WriteInfo(str)
		}
	}
	if err != nil {
		WriteError("script result", err)
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode() == 0
		}
	}
	return true
}

func (s script) Validate(token, mac string) bool {
	if s.static {
		key := fmt.Sprintf("%s/%s", token, mac)
		for _, p := range s.payload {
			if p == key {
				return true
			}
		}
		return false
	}
	c := ComposeFlags{Mode: ModeValidate, MAC: mac, Token: token, Command: s.command}
	return s.execute(c)
}

func (s script) Server() bool {
	c := ComposeFlags{Mode: ModeServer, Hash: s.hash}
	return s.execute(c)
}

func (s script) Fetch() bool {
	return s.execute(ComposeFlags{Mode: ModeFetch})
}

func (s script) Build() bool {
	return s.execute(ComposeFlags{Mode: ModeBuild})
}

// SetAllowed hard sets which token+mac combos are allowed
func SetAllowed(payload []string) {
	objects := []string{}
	for _, obj := range payload {
		str := strings.TrimSpace(obj)
		if len(str) > 0 {
			objects = append(objects, str)
		}
	}
	lock.Lock()
	defer lock.Unlock()
	backend = &script{payload: objects, static: true}
}

// Manage configures the backend for access checks
func Manage(cfg *Configuration) error {
	if len(cfg.Configurator.Payload) == 0 {
		return fmt.Errorf("no command configured for management")
	}
	if len(cfg.Configurator.ServerKey) == 0 {
		return fmt.Errorf("no server key/passphrase found")
	}
	backend = &script{debug: cfg.Configurator.Debug, timeout: time.Duration(cfg.Configurator.Timeout) * time.Second, repo: cfg.Configurator.Repository, command: cfg.Configurator.Payload, hash: MD4(cfg.Configurator.ServerKey)}
	lock.Lock()
	result := backend.Server()
	lock.Unlock()
	if !result {
		return fmt.Errorf("server command failed")
	}
	go run(time.Duration(cfg.Configurator.Refresh) * time.Minute)
	return nil
}

// CheckTokenMAC validates a token+mac combination as valid
func CheckTokenMAC(token, mac string) bool {
	lock.Lock()
	defer lock.Unlock()
	return backend.Validate(token, mac)
}

func run(sleep time.Duration) {
	for {
		time.Sleep(sleep)
		WriteInfo("running fetch and update")
		if !backend.Fetch() {
			WriteWarn("fetch failed")
			continue
		}
		lock.Lock()
		result := backend.Build()
		lock.Unlock()
		if !result {
			WriteWarn("config backend update failed")
		}
	}
}

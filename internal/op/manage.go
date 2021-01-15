package op

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"voidedtech.com/dotonex/internal/core"
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

func (s script) execute(flags core.ComposeFlags) bool {
	flags.Repo = s.repo
	arguments := flags.Args()

	if s.debug {
		core.WriteInfo(fmt.Sprintf("running: %v", arguments))
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "dotonex-compose", arguments...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		core.WriteWarn("script timeout")
		return false
	}
	str := stderr.String()
	if len(str) > 0 {
		core.WriteInfo("stderr")
		core.WriteInfo(str)
	}
	if s.debug {
		str = strings.TrimSpace(string(out))
		if len(str) > 0 {
			core.WriteInfo("stdout")
			core.WriteInfo(str)
		}
	}
	if err != nil {
		core.WriteError("script result", err)
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
	c := core.ComposeFlags{Mode: core.ModeValidate, MAC: mac, Token: token, Command: s.command}
	return s.execute(c)
}

func (s script) Server() bool {
	c := core.ComposeFlags{Mode: core.ModeServer, Hash: s.hash}
	return s.execute(c)
}

func (s script) Fetch() bool {
	return s.execute(core.ComposeFlags{Mode: core.ModeFetch})
}

func (s script) Build() bool {
	return s.execute(core.ComposeFlags{Mode: core.ModeBuild})
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
func Manage(cfg *core.Configuration) error {
	if len(cfg.Configurator.Payload) == 0 {
		return fmt.Errorf("no command configured for management")
	}
	if len(cfg.Configurator.ServerKey) == 0 {
		return fmt.Errorf("no server key/passphrase found")
	}
	backend = &script{debug: cfg.Configurator.Debug, timeout: time.Duration(cfg.Configurator.Timeout) * time.Second, repo: cfg.Configurator.Repository, command: cfg.Configurator.Payload, hash: core.MD4(cfg.Configurator.ServerKey)}
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
		core.WriteInfo("running fetch and update")
		if !backend.Fetch() {
			core.WriteWarn("fetch failed")
			continue
		}
		lock.Lock()
		result := backend.Build()
		lock.Unlock()
		if !result {
			core.WriteWarn("config backend update failed")
		}
	}
}

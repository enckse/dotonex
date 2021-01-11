package internal

import (
	"fmt"
	"os"
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
		command string
		hash    string
		static  bool
		payload []string
	}
)

func (s script) execute(sub string, args []string) bool {
	arguments := []string{sub, s.repo}
	arguments = append(arguments, args...)
	cmd := exec.Command("dotonex-config", arguments...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		WriteError("script result", err)
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode() != 0
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
	return s.execute("validate", []string{fmt.Sprintf("--command='%s'", s.command), fmt.Sprintf("--mac=%s", mac), fmt.Sprintf("--token=%s", token)})
}

func (s script) Server() bool {
	return s.execute("server", []string{fmt.Sprintf("--hash=%s", s.hash)})
}

func (s script) Update() bool {
	return s.execute("update", []string{})
}

// SetAllowed hard sets which token+mac combos are allowed
func SetAllowed(payload string) {
	list := strings.Split(payload, ",")
	objects := []string{}
	for _, obj := range list {
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
	backend = &script{repo: cfg.Configurator.Repository, command: cfg.Configurator.Payload, hash: MD4(cfg.Configurator.ServerKey)}
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
		lock.Lock()
		result := backend.Update()
		lock.Unlock()
		if !result {
			WriteWarn("config backend update failed")
		}
	}
}

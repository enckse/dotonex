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
	backend manager
)

type (
	manager interface {
		Validate(string, string) bool
		Server() bool
		Update() bool
	}
	script struct {
		repo    string
		command string
		hash    string
	}
	static struct {
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
	return s.execute("validate", []string{fmt.Sprintf("--command='%s'", s.command), fmt.Sprintf("--mac=%s", mac), fmt.Sprintf("--token=%s", token)})
}

func (s static) Validate(token, mac string) bool {
	key := fmt.Sprintf("%s.%s", token, mac)
	for _, p := range s.payload {
		if p == key {
			return true
		}
	}
	return false
}

func (s script) Server() bool {
	return s.execute("server", []string{fmt.Sprintf("--hash=%s", s.hash)})
}

func (s static) Server() bool {
	return true
}

func (s script) Update() bool {
	return s.execute("update", []string{})
}

func (s static) Update() bool {
	return true
}

func setBackend(mgr manager) {
	lock.Lock()
	defer lock.Unlock()
	backend = mgr
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
	setBackend(static{payload: objects})
}

// Manage configures the backend for access checks
func Manage(cfg *Configuration) error {
	if cfg.Configurator.Static {
		SetAllowed(cfg.Configurator.Payload)
	} else {
		if len(cfg.Configurator.Payload) == 0 {
			return fmt.Errorf("no command configured for management")
		}
		if len(cfg.Configurator.ServerKey) == 0 {
			return fmt.Errorf("no server key/passphrase found")
		}
		setBackend(&script{repo: cfg.Configurator.Repository, command: cfg.Configurator.Payload, hash: MD4(cfg.Configurator.ServerKey)})
	}
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

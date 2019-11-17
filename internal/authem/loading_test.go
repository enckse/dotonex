package authem

import (
	"os"
	"testing"

	"voidedtech.com/radiucal/internal/core"
)

func setup(t *testing.T) LoadingOptions {
	for _, d := range []string{TempDir} {
		if !core.PathExists(d) {
			if err := os.Mkdir(d, 0755); err != nil {
				t.Error("unable to make test dir")
			}
		}
	}
	return LoadingOptions{
		Verbose: false,
		Sync:    true,
		Key:     "aaaaaaaabbbbbbbbccccccccdddddddd",
	}
}

func TestLoadVLANs(t *testing.T) {
	opts := setup(t)
	vlans, err := opts.LoadVLANs()
	if len(vlans) != 2 || err != nil {
		t.Error("invalid vlan count")
	}
}

func TestLoadSecrets(t *testing.T) {
	opts := setup(t)
	secret, err := opts.LoadSecrets()
	if len(secret) != 6 || err != nil {
		t.Error("invalid secret count")
	}
}

func TestLoadSystems(t *testing.T) {
	opts := setup(t)
	sys, err := opts.LoadSystems()
	if len(sys) != 1 || err != nil {
		t.Error("invalid sys count")
	}
}

func TestLoadusers(t *testing.T) {
	opts := setup(t)
	vlans, _ := opts.LoadVLANs()
	secret, _ := opts.LoadSecrets()
	sys, _ := opts.LoadSystems()
	users, radius, err := opts.LoadUsers(vlans, sys, secret)
	if err != nil {
		t.Error("load failed")
	}
	if len(users) != 5 {
		t.Error("user count wrong")
	}
	if len(radius) != 2 {
		t.Error("radius count wrong")
	}
	opts = LoadingOptions{
		Verbose: false,
		Sync:    false,
		Key:     "aaaaaaaabbbbbbbbccccccccdddddddd",
	}
	users, radius, err = opts.LoadUsers(vlans, sys, secret)
	if err != nil {
		t.Error("load failed")
	}
	if len(users) != 5 {
		t.Error("user count wrong")
	}
	if len(radius) != 2 {
		t.Error("radius count wrong")
	}
}

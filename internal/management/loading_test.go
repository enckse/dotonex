package management

import (
	"fmt"
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
		Key: "aaaaaaaabbbbbbbbccccccccdddddddd",
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

func TestBuildTrust(t *testing.T) {
	var users []*User
	i := 0
	for i < 6 {
		u := &User{}
		u.UserName = fmt.Sprintf("user%d", i)
		if i == 0 {
			u.Perms.IsRoot = true
		}
		users = append(users, u)
		i = i + 1
	}
	users[0].Perms.Trusts = append(users[0].Perms.Trusts, "user1")
	users[1].Perms.Trusts = append(users[1].Perms.Trusts, "user2")
	users[1].Perms.Trusts = append(users[1].Perms.Trusts, "user3")
	users[2].Perms.Trusts = append(users[2].Perms.Trusts, "user4")
	users[3].Perms.Trusts = append(users[3].Perms.Trusts, "user5")
	opts := setup(t)
	err := opts.BuildTrust(users)
	if err != nil {
		t.Error("no trust")
	}
	u := &User{}
	u.UserName = "none"
	users = append(users, u)
	err = opts.BuildTrust(users)
	if err == nil {
		t.Error("missing trust")
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
		Key: "aaaaaaaabbbbbbbbccccccccdddddddd",
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

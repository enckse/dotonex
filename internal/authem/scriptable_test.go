package authem

import (
	"fmt"
	"sort"
	"testing"
)

var (
	sVlans = []*VLAN{
		&VLAN{
			ID:   1,
			Name: "test1",
		},
		&VLAN{
			ID:   2,
			Name: "test2",
		},
	}
	sSystems = []*System{
		&System{
			Type: "sys1",
		},
		&System{
			Type: "sys2",
		},
	}
)

func TestToScriptable(t *testing.T) {
	u := &User{}
	u.UserName = "test"
	u.VLANs = []string{"test1", "test2", "test3"}
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{VLAN: "test1", MACs: []string{"aabbccddeeff", "112233445566"}}, MACMap{VLAN: "test2", MACs: []string{"aabbccddeeff"}}}}}
	u2 := &User{}
	u2.LoginAs = "test1"
	secrets := []*Secret{}
	secrets = append(secrets, &Secret{UserName: "test", Password: "pass"})
	scriptable := ToScriptable(UserConfig{[]*User{u, u2}}, sVlans, sSystems, secrets)
	if len(scriptable.VLANs) != 2 || len(scriptable.Systems) != 2 || len(scriptable.Users) != 2 {
		t.Error("invalid conversion")
	}
	user := scriptable.Users[1]
	if user.UserName != "" || user.LoginName != "test1" || user.Password != "" {
		t.Error("invalid user conversion - login")
	}
	user = scriptable.Users[0]
	if user.UserName != "test" || user.LoginName != "test" || user.Password != "pass" {
		t.Error("invalid user conversion")
	}
	if len(user.VLANs) != 3 || len(user.Systems) != 1 {
		t.Error("invalid user composite")
	}
	sys := user.Systems[0]
	if len(sys.MACs) != 2 {
		t.Error("invalid MACs")
	}
	macs := sys.MACs
	sort.Strings(macs)
	if fmt.Sprintf("%v", macs) != "[112233445566 aabbccddeeff]" {
		t.Error("invalid MAC set")
	}
	if sys.ID != "test" {
		t.Error("invalid system id")
	}
}

func TestRunner(t *testing.T) {
	b := BashRunner{}
	if err := b.Execute(); err == nil || err.Error() != "no data" {
		t.Error("no data...")
	}
	b.Data = []byte("echo hello")
	if err := b.Execute(); err != nil {
		t.Error("should run")
	}
}

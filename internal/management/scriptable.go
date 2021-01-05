package management

import (
	"fmt"
	"os"
	"os/exec"
)

type (
	// ScriptableSystem is an exported system for script execution
	ScriptableSystem struct {
		System
		ID   string
		MACs []string
	}

	// ScriptableUser represents a user for scripting
	ScriptableUser struct {
		UserName  string
		FullName  string
		LoginName string
		VLANs     []string
		Trusts    []string
		Systems   []*ScriptableSystem
	}

	// Scriptable represents the combined system for script usage
	Scriptable struct {
		Systems []*System
		VLANs   []*VLAN
		Users   []*ScriptableUser
	}

	// BashRunner handles running an actual script through bash
	BashRunner struct {
		Data []byte
		Name string
	}
)

// ToScriptable converts a set of objects for script usage
func ToScriptable(users UserConfig, vlans []*VLAN, systems []*System) *Scriptable {
	scriptable := &Scriptable{
		Systems: systems,
		VLANs:   vlans,
	}
	for _, u := range users.Users {
		user := &ScriptableUser{}
		user.UserName = u.UserName
		user.FullName = u.FullName
		user.LoginName = u.LoginName()
		user.VLANs = u.VLANs
		user.Trusts = u.Perms.Trusts
		for _, s := range systems {
			for _, userSys := range u.Systems {
				if s.Type == userSys.Type {
					sys := &ScriptableSystem{}
					sys.Type = s.Type
					sys.ID = userSys.ID
					sys.Revision = s.Revision
					sys.Make = s.Make
					sys.Model = s.Model
					track := make(map[string]bool)
					for _, macs := range userSys.MACs {
						for _, mac := range macs.MACs {
							if _, ok := track[mac]; ok {
								continue
							}
							track[mac] = true
							sys.MACs = append(sys.MACs, mac)
						}
					}
					user.Systems = append(user.Systems, sys)
				}
			}
		}
		scriptable.Users = append(scriptable.Users, user)
	}
	return scriptable
}

// Execute will perform execution of the script via piping into bash
func (p BashRunner) Execute() error {
	if len(p.Data) == 0 {
		return fmt.Errorf("no data")
	}
	cmd := exec.Command("bash")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		return err
	}
	stdin.Write(p.Data)
	stdin.Close()
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

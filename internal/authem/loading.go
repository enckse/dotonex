package authem

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"sync"

	yaml "gopkg.in/yaml.v2"
	"voidedtech.com/radiucal/internal/core"
)

const (
	// UserDir is the location of user yaml files
	UserDir = "users"
	// SecretsDir is where user secrets are stored
	SecretsDir = "secrets"
	// VLANsDir are where vlan configurations live
	VLANsDir = "vlans"
	// SystemsDir is where hardware information is stored
	SystemsDir = "hardware"
	// TempDir is a locally working directory
	TempDir = "bin"
)

type (
	onLoad func(string, []byte) error

	// LoadingOptions control how objects are loaded
	LoadingOptions struct {
		Verbose bool
		Key     string
		Sync    bool
		NoKey   bool
	}
)

// LoadVLANs loads vlans from disk
func (l LoadingOptions) LoadVLANs() ([]*VLAN, error) {
	tracked := make(map[int]string)
	var vlans []*VLAN
	err := loadDirectory(VLANsDir, func(n string, b []byte) error {
		v := &VLAN{}
		if err := yaml.Unmarshal(b, &v); err != nil {
			return err
		}
		if err := v.Check(); err != nil {
			return err
		}
		if _, ok := tracked[v.ID]; ok {
			return fmt.Errorf("%d redefined in %s", v.ID, v.Name)
		}
		tracked[v.ID] = v.Name
		vlans = append(vlans, v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return vlans, nil
}

// LoadSecrets load secrets from disk
func (l LoadingOptions) LoadSecrets() ([]*Secret, error) {
	var secrets []*Secret
	if l.NoKey {
		return secrets, nil
	}
	err := loadDirectory(SecretsDir, func(n string, b []byte) error {
		dec, err := Decrypt(l.Key, string(b))
		if err != nil {
			return err
		}
		s := &Secret{}
		if err := yaml.Unmarshal([]byte(dec), &s); err != nil {
			return err
		}
		s.Fake = false
		secrets = append(secrets, s)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// LoadSystems load system hardware from disk
func (l LoadingOptions) LoadSystems() ([]*System, error) {
	var systems []*System
	err := loadDirectory(SystemsDir, func(n string, b []byte) error {
		s := &System{}
		if err := yaml.Unmarshal(b, &s); err != nil {
			return err
		}
		if err := s.Check(); err != nil {
			return err
		}
		systems = append(systems, s)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return systems, nil
}

func loadDirectory(dir string, load onLoad) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	core.WriteInfo(fmt.Sprintf("[%s]", dir))
	for _, f := range files {
		n := f.Name()
		core.WriteInfoDetail(n)
		b, err := ioutil.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return err
		}
		if err := load(n, b); err != nil {
			return err
		}
	}
	return nil
}

func loadUser(b []byte, opts LoadingOptions, vlan []*VLAN, sys []*System, secret []*Secret) (*User, *UserRADIUS, error) {
	u := &User{}
	var r *UserRADIUS
	r = nil
	if err := yaml.Unmarshal(b, &u); err != nil {
		return nil, nil, err
	}
	secrets := secret
	if opts.NoKey {
		secrets = []*Secret{&Secret{UserName: u.UserName, Fake: true}}
	}
	if err := u.Inflate(opts.Key, secrets); err != nil {
		return nil, nil, err
	}
	if u.Perms.IsRADIUS {
		radiusUser, err := u.ForRADIUS(vlan, sys, RADIUSOptions{})
		if err != nil {
			return nil, nil, err
		}
		r = radiusUser
	}
	return u, r, nil
}

func asyncLoadUser(comm chan bool, users *sync.Map, radius *sync.Map, file string, b []byte, opts LoadingOptions, vlan []*VLAN, sys []*System, secret []*Secret) {
	u, r, err := loadUser(b, opts, vlan, sys, secret)
	if err != nil {
		core.WriteError(file, err)
		if comm != nil {
			comm <- false
		}
		return
	}
	if _, ok := users.LoadOrStore(u.UserName, u); ok {
		core.WriteError(fmt.Sprintf("%s is already defined", u.UserName), nil)
		if comm != nil {
			comm <- false
		}
		return
	}
	if r != nil {
		if _, ok := radius.LoadOrStore(u.UserName, r); ok {
			core.WriteError(fmt.Sprintf("%s is already defined", u.UserName), nil)
			if comm != nil {
				comm <- false
			}
			return
		}
	}
	if comm != nil {
		comm <- true
	}
}

// LoadUsers builds users from disk and other necessary objects)
func (l LoadingOptions) LoadUsers(vlan []*VLAN, sys []*System, secret []*Secret) ([]*User, []*UserRADIUS, error) {
	users := &sync.Map{}
	radius := &sync.Map{}
	var chans []chan bool
	err := loadDirectory(UserDir, func(n string, b []byte) error {
		if l.Sync {
			asyncLoadUser(nil, users, radius, n, b, l, vlan, sys, secret)
		} else {
			c := make(chan bool)
			go asyncLoadUser(c, users, radius, n, b, l, vlan, sys, secret)
			chans = append(chans, c)
		}
		return nil
	})
	for _, c := range chans {
		result := <-c
		if !result {
			return nil, nil, fmt.Errorf("user load failure")
		}
	}
	if err != nil {
		return nil, nil, err
	}
	var userNames []string
	userMap := make(map[string]*User)
	radiusMap := make(map[string]*UserRADIUS)
	users.Range(func(k, v interface{}) bool {
		key := k.(string)
		userNames = append(userNames, key)
		userMap[key] = v.(*User)
		return true
	})
	radius.Range(func(k, v interface{}) bool {
		radiusMap[k.(string)] = v.(*UserRADIUS)
		return true
	})
	sort.Strings(userNames)
	var userSet []*User
	var radiusSet []*UserRADIUS
	for _, n := range userNames {
		userSet = append(userSet, userMap[n])
		val, ok := radiusMap[n]
		if ok {
			radiusSet = append(radiusSet, val)
		}
	}
	return userSet, radiusSet, nil
}

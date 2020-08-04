package external

import (
	"fmt"
	"strings"
	"sync"

	"voidedtech.com/grad/internal/core"
)

const (
	attr = `
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:%s`

	loginUser = `"%s" PEAP

"%s" MSCHAPV2 hash:%s [2]` + attr

	md5System = `"%s" MD5 "%s"` + attr
)

var (
	lockGitlab        = &sync.Mutex{}
	gitlabUsers       = make(map[string]Auth)
	gitlabTokenLogins = make(map[string]Auth)
	// ExternalRequestor is the requestor used to make calls to get user/group info
	ExternalRequestor Requestor
)

type (
	// Auth is a representation of metadata for user+mac auths
	Auth struct {
		User           string
		CallingStation string
	}

	// VLAN represents a name, id, and MACs that are bypassed on the VLAN
	VLAN struct {
		MACs []string
		ID   string
		Name string
	}

	// User represents a user, the vlans (first is default), and associated MACs
	User struct {
		MACs  []string
		Name  string
		VLANs []string
	}

	// Requestor defines how to do token lookups
	Requestor interface {
		CurrentUser(key string) (string, error)
		Groups() ([]VLAN, error)
		Users() ([]User, error)
	}

	emptyRequestor struct {
	}
)

func newUserLogin(username, hash, vlan string) string {
	return fmt.Sprintf(loginUser, username, username, hash, vlan)
}

// Hostapd converts a VLAN configuration of MACs for MAB
func (v VLAN) Hostapd() string {
	var result []string
	for _, m := range v.MACs {
		result = append(result, fmt.Sprintf(md5System, m, m, v.ID))
	}
	return strings.Join(result, "\n")
}

// Hostapd converts a user configuration into a hostapd-friendly setting
func (u User) Hostapd(hash string, vlans []VLAN) string {
	var result []string
	first := true
	for _, v := range u.VLANs {
		matched := false
		for _, vlan := range vlans {
			if vlan.Name == v {
				matched = true
				// first vlan is default vlan
				if first {
					result = append(result, newUserLogin(u.Name, hash, vlan.ID))
				}
				result = append(result, newUserLogin(fmt.Sprintf("%s:%s", v, u.Name), hash, vlan.ID))
				first = false
			}
		}
		if !matched {
			core.WriteWarn(fmt.Sprintf("user %s has '%s' which is an invalid vlan", u.Name, v))
		}
	}
	return strings.Join(result, "\n")
}

func (r *emptyRequestor) CurrentUser(key string) (string, error) {
	return "", fmt.Errorf("no user available")
}

func (r *emptyRequestor) Groups() ([]VLAN, error) {
	return []VLAN{}, fmt.Errorf("no groups available")
}

func (r *emptyRequestor) Users() ([]User, error) {
	return []User{}, fmt.Errorf("no users available")
}

func init() {
	ExternalRequestor = &emptyRequestor{}
}

func (a Auth) String() string {
	return fmt.Sprintf("%s:%s", a.User, a.CallingStation)
}

// Equals confirms an auth entity is equivalent to another
func (a Auth) Equals(other Auth) bool {
	return a.User == other.User && a.CallingStation == other.CallingStation
}

func tokenAuth(object Auth) bool {
	user, err := ExternalRequestor.CurrentUser(object.User)
	if err != nil {
		core.WriteError("unable to get user from key", err)
	}
	valid := user != ""
	newUser := NewAuth("(NONE)", "(NONE)")
	if valid {
		newUser = object
	}
	lockGitlab.Lock()
	gitlabTokenLogins[object.String()] = newUser
	lockGitlab.Unlock()
	return valid
}

// SetUserAuths does a lock-safe update the set of user+mac combinations
func SetUserAuths(set []Auth) {
	lockGitlab.Lock()
	defer lockGitlab.Unlock()
	gitlabUsers = make(map[string]Auth)
	for _, u := range set {
		gitlabUsers[u.String()] = u
	}
}

// NewAuth creates a newly define user+MAC combo
func NewAuth(user, callingStation string) Auth {
	return Auth{User: user, CallingStation: callingStation}
}

// AuthorizeUser checks if a user is valid in gitlab
func AuthorizeUser(object Auth) bool {
	lockGitlab.Lock()
	s := object.String()
	valid := false
	useToken := true
	compare, ok := gitlabUsers[s]
	if ok {
		valid = compare.Equals(object)
	}
	if !valid {
		compare, ok = gitlabTokenLogins[s]
		if ok {
			valid = compare.Equals(object)
			useToken = false
		}
	}
	lockGitlab.Unlock()
	if valid {
		return true
	}
	if !useToken {
		return false
	}
	return tokenAuth(object)
}

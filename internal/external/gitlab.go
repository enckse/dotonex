package external

import (
	"fmt"
	"sync"
)

var (
	lockGitlab        = &sync.Mutex{}
	gitlabUsers       = make(map[string]Auth)
	gitlabTokenLogins = make(map[string]Auth)
)

type (
	// Auth is a representation of metadata for user+mac auths
	Auth struct {
		User           string
		CallingStation string
	}
)

func (a Auth) String() string {
	return fmt.Sprintf("%s:%s", a.User, a.CallingStation)
}

// Equals confirms an auth entity is equivalent to another
func (a Auth) Equals(other Auth) bool {
	return a.User == other.User && a.CallingStation == other.CallingStation
}

func tryToken(object Auth) bool {
	return false
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
	defer lockGitlab.Unlock()
	s := object.String()
	compare, ok := gitlabUsers[s]
	if ok {
		return compare.Equals(object)
	}
	compare, ok = gitlabTokenLogins[s]
	if ok {
		return compare.Equals(object)
	}
	return tryToken(object)
}

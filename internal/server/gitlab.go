package server

import (
	"sync"
)

var (
	lockGitlab        = &sync.Mutex{}
	gitlabUsers       = make(map[string]bool)
	gitlabTokenLogins = make(map[string]bool)
)

func tryToken(user string) bool {
	return true
}

// AuthorizeUser checks if a user is valid in gitlab
func AuthorizeUser(user string) bool {
	lockGitlab.Lock()
	defer lockGitlab.Unlock()
	if _, ok := gitlabUsers[user]; ok {
		return true
	}
	if _, ok := gitlabTokenLogins[user]; ok {
		return true
	}
	return tryToken(user)
}

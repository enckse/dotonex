package core

import (
	"fmt"
	"os"
	"strings"
)

const (
	userVLANLogin = "@vlan."
)

// PathExists reports if a path exists or does not exist
func PathExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// IntegerIn will check if an integer is in a list
func IntegerIn(i int, list []int) bool {
	for _, obj := range list {
		if obj == i {
			return true
		}
	}
	return false
}

// NewUserVLANLogin creates a new user+vlan login name
func NewUserVLANLogin(user, vlan string) string {
	return fmt.Sprintf("%s%s%s", user, userVLANLogin, vlan)
}

// GetUserFromVLANLogin gets the user part from a FQDN user+vlan login
func GetUserFromVLANLogin(input string) string {
	if !strings.Contains(input, userVLANLogin) {
		return input
	}
	parts := strings.Split(input, userVLANLogin)
	return parts[0]
}

// CleanMAC will clean a MAC and check that it is valid
func CleanMAC(value string) (string, bool) {
	str := ""
	for _, chr := range strings.ToLower(value) {
		if (chr >= '0' && chr <= '9') || (chr >= 'a' && chr <= 'f') {
			str = str + string([]rune{chr})
		}
	}
	return str, len(str) == 12
}

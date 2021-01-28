package core

import (
	"fmt"
	"os"
	"strings"
)

const (
	userVLANLogin = "@vlan."
	userLogin     = ":"
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

// NewUserLogin creates a login name for a user+token
func NewUserLogin(user, token string) string {
	return fmt.Sprintf("%s%s%s", user, userLogin, token)
}

// NewUserVLANLogin creates a new user+vlan login name
func NewUserVLANLogin(user, vlan string) string {
	return fmt.Sprintf("%s%s%s", user, userVLANLogin, vlan)
}

// GetTokenFromLogin gets the user's token part from a FQDN user+token+vlan login
func GetTokenFromLogin(input string) (string, string) {
	token := input
	if strings.Contains(input, userVLANLogin) {
		token = strings.Split(input, userVLANLogin)[0]
	}
	if !strings.Contains(token, userLogin) {
		return "", ""
	}
	parts := strings.Split(token, userLogin)
	return parts[0], strings.Join(parts[1:], userLogin)
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

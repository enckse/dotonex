package internal

import (
	"os"
	"strings"
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

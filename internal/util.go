package internal

import (
	"fmt"
	"os"
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

// NewManifestEntry creates a new manifest entry object
func NewManifestEntry(user, mac string) string {
	return fmt.Sprintf("%s.%s", user, mac)
}

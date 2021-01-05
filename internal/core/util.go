package core

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// PathExists reports if a path exists or does not exist
func PathExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	return true
}

// Compare will indicate (and optionally show) differences between byte sets
func Compare(prev, now []byte, show bool) bool {
	p := strings.Split(string(prev), "\n")
	n := strings.Split(string(now), "\n")
	if diff := cmp.Diff(p, n); diff != "" {
		if show {
			WriteInfo("======")
			fmt.Println(diff)
			WriteInfo("======")
		}
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

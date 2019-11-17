package authem

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// PathExists will report if a path already exists
func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// DefaultYaml gets the location of the default yaml config files
func DefaultYaml(name string) string {
	return filepath.Join("/etc", "authem", name+".yaml")
}

// Compare will indicate (and optionally show) differences between byte sets
func Compare(prev, now []byte, show bool) bool {
	p := strings.Split(string(prev), "\n")
	n := strings.Split(string(now), "\n")
	if diff := cmp.Diff(p, n); diff != "" {
		if show {
			Info("======")
			Info(diff)
			Info("======")
		}
		return false
	}
	return true
}

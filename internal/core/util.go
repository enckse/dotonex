package core

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/go-cmp/cmp"
)

type (
	// KeyValue represents a simple key/value object
	KeyValue struct {
		Key   string
		Value string
	}

	// KeyValueStore represents a store of KeyValue objects
	KeyValueStore struct {
		KeyValues []KeyValue
		DropEmpty bool
	}
)

// Add adds a key value object to the store
func (kv *KeyValueStore) Add(key, val string) {
	kv.KeyValues = append(kv.KeyValues, KeyValue{Key: key, Value: val})
}

// String converts the KeyValue to a string representation
func (kv KeyValue) String() string {
	return fmt.Sprintf("%s = %s", kv.Key, kv.Value)
}

// Strings gets all strings from a store
func (kv KeyValueStore) Strings() []string {
	var objs []string
	offset := ""
	for _, k := range kv.KeyValues {
		if kv.DropEmpty && len(k.Value) == 0 {
			continue
		}
		objs = append(objs, fmt.Sprintf("%s%s", offset, k.String()))
		offset = "  "
	}
	return objs
}

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

// In will check if an integer is in a list
func In(i int, list []int) bool {
	for _, obj := range list {
		if obj == i {
			return true
		}
	}
	return false
}

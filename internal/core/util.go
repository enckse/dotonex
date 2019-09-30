package core

import (
	"fmt"
	"os"
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

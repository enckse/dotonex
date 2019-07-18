package core

import (
	"fmt"
	"os"
)

type KeyValue struct {
	Key   string
	Value string
}

type KeyValueStore struct {
	KeyValues []KeyValue
	DropEmpty bool
}

func (kv *KeyValueStore) Add(key, val string) {
	kv.KeyValues = append(kv.KeyValues, KeyValue{Key: key, Value: val})
}

func (kv KeyValueStore) Strings() []string {
	var objs []string
	offset := ""
	for _, k := range kv.KeyValues {
		if kv.DropEmpty && len(k.Value) == 0 {
			continue
		}
		objs = append(objs, fmt.Sprintf("%s%s = %s", offset, k.Key, k.Value))
		offset = "  "
	}
	return objs
}

func PathExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

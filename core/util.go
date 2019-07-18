package core

import (
	"bytes"
	"fmt"
	"os"
)

type KeyValue struct {
	Key   string
	Value string
}

type KeyValueStore struct {
	KeyValues []KeyValue
}

func (kv KeyValueStore) String() string {
	var b bytes.Buffer
	offset := ""
	for idx, k := range kv.KeyValues {
		if idx > 0 {
			b.Write([]byte("\n"))
		}
		b.Write([]byte(fmt.Sprintf("%s%s = %s", offset, k.Key, k.Value)))
		offset = "  "
	}
	return b.String()
}

func PathExists(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

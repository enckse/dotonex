package authem

import (
	"testing"
)

func TestDefaultYaml(t *testing.T) {
	if DefaultYaml("test") != "/etc/authem/test.yaml" {
		t.Error("invalid default yaml")
	}
}

func TestCompare(t *testing.T) {
	diff := Compare([]byte(""), []byte("a"), false)
	if diff {
		t.Error("different")
	}
	diff = Compare([]byte("a"), []byte("a"), false)
	if !diff {
		t.Error("different")
	}
}

package core

import (
	"testing"
)

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

func TestManifestEntry(t *testing.T) {
	if NewManifestEntry("test", "object") != "test.object" {
		t.Error("invalid manifest entry")
	}
}

package internal

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

func TestIn(t *testing.T) {
	if IntegerIn(1, []int{0, 2}) {
		t.Error("in was wrong")
	}
	if !IntegerIn(3, []int{1, 2, 3}) {
		t.Error("in should be right...")
	}
}

func TestManifestEntry(t *testing.T) {
	if NewManifestEntry("test", "object") != "test.object" {
		t.Error("invalid manifest entry")
	}
}

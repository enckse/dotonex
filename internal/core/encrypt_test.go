package core

import (
	"testing"
)

func TestMD4(t *testing.T) {
	if o := MD4("test"); o != "0cb6948805f797bf2a82807973b89537" {
		t.Error("invalid md4")
	}
}

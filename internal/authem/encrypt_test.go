package authem

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := "aaaaaaaabbbbbbbbccccccccdddddddd"
	enc, err := Encrypt(key, "secret")
	if err != nil {
		t.Errorf("invalid request %v", err)
	}
	if enc == "" {
		t.Error("invalid encryption string", enc)
	}
	dec, err := Decrypt(key, enc)
	if err != nil {
		t.Errorf("invalid request %v", err)
	}
	if dec != "secret" {
		t.Error("decryption error", dec)
	}
	_, err = Decrypt("garbage", enc)
	if err == nil {
		t.Error("invalid request")
	}
	_, err = Encrypt("garbage", enc)
	if err == nil {
		t.Error("invalid request")
	}
}

func TestMD4(t *testing.T) {
	if o := MD4("test"); o != "0cb6948805f797bf2a82807973b89537" {
		t.Error("invalid md4")
	}
}

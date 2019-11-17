package authem

import (
	"testing"
)

var (
	testSecrets = []*Secret{
		&Secret{"abc", "garbage", false},
		&Secret{"xyz", "more", false},
	}
)

func TestInflate(t *testing.T) {
	u := &User{}
	if err := u.Inflate("", []*Secret{}); err.Error() != "user secrets not configured" {
		t.Error("should fail")
	}
	u.UserName = "abc"
	key := "aaaaaaaabbbbbbbbccccccccdddddddd"
	if err := u.Inflate(key, testSecrets); err != nil {
		t.Error("should pass")
	}
	if u.MD4 != "730b18608a90bf41c7f771ac71f28036" {
		t.Error("invalid md4")
	}
}

func TestLoginName(t *testing.T) {
	u := &User{}
	u.UserName = "test"
	if u.LoginName() != "test" {
		t.Error("invalid user name")
	}
	u.LoginAs = "test1"
	if u.LoginName() != "test1" {
		t.Error("invalid login name")
	}
}

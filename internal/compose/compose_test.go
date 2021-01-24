package compose

import (
	"testing"

	"github.com/tidwall/buntdb"
)

func TestNewKey(t *testing.T) {
	s := Store{}
	if "serverhash=>key" != s.NewKey(ServerHashKey, "key") {
		t.Error("invalid key")
	}
}

func TestSaveGet(t *testing.T) {
	s := Store{}
	db, err := buntdb.Open(":memory:")
	if err != nil {
		t.Error("memory db failed")
	}
	defer db.Close()
	s.db = db
	val, ok, err := s.Get("TEST")
	if ok || err != nil || val != "" {
		t.Error("not found")
	}
	err = s.Save("TEST", "TEST2")
	if err != nil {
		t.Error("unable to save")
	}
	val, ok, err = s.Get("TEST")
	if err != nil || !ok {
		t.Error("invalid response")
	}
	if val != "TEST2" {
		t.Error("wrong value")
	}
}

func TestValidateMembership(t *testing.T) {
	d := Definition{}
	if err := d.ValidateMembership(); err == nil {
		t.Error("no memberships")
	}
	d = Definition{}
	d.Membership = append(d.Membership, Member{VLAN: ""})
	if err := d.ValidateMembership(); err == nil {
		t.Error("invalid membership")
	}
	d = Definition{}
	d.Membership = append(d.Membership, Member{VLAN: ""})
	d.Membership = append(d.Membership, Member{VLAN: "a"})
	if err := d.ValidateMembership(); err == nil {
		t.Error("invalid membership")
	}
	d = Definition{}
	d.Membership = append(d.Membership, Member{VLAN: "a"})
	if err := d.ValidateMembership(); err != nil {
		t.Error("valid membership")
	}
}

func TestIsVLAN(t *testing.T) {
	d := Definition{}
	if _, ok := d.IsVLAN("test"); ok {
		t.Error("not a vlan")
	}
	d = Definition{}
	d.VLANs = append(d.VLANs, VLAN{Name: "test", ID: "abc"})
	if _, ok := d.IsVLAN("aaa"); ok {
		t.Error("not a vlan")
	}
	d.VLANs = append(d.VLANs, VLAN{Name: "aaa", ID: "abc"})
	id, ok := d.IsVLAN("aaa")
	if !ok || id != "abc" {
		t.Error("valid vlan")
	}
}

func TestValidateVLANs(t *testing.T) {
	d := Definition{}
	if err := d.ValidateVLANs(); err == nil {
		t.Error("no VLANs")
	}
	d = Definition{}
	d.VLANs = append(d.VLANs, VLAN{Name: "a", ID: ""})
	if err := d.ValidateVLANs(); err == nil {
		t.Error("invalid VLANs")
	}
	d = Definition{}
	d.VLANs = append(d.VLANs, VLAN{Name: "", ID: "b"})
	if err := d.ValidateVLANs(); err == nil {
		t.Error("invalid VLANs")
	}
	d = Definition{}
	d.VLANs = append(d.VLANs, VLAN{Name: "", ID: ""})
	if err := d.ValidateVLANs(); err == nil {
		t.Error("invalid VLANs")
	}
	d = Definition{}
	d.VLANs = append(d.VLANs, VLAN{Name: "a", ID: "b"})
	if err := d.ValidateVLANs(); err != nil {
		t.Error("valid VLANs")
	}
	d = Definition{}
	d.VLANs = append(d.VLANs, VLAN{Name: "", ID: ""})
	d.VLANs = append(d.VLANs, VLAN{Name: "b", ID: "a"})
	if err := d.ValidateVLANs(); err == nil {
		t.Error("invalid VLANs")
	}
}

func TestTryGetUserArray(t *testing.T) {
	valid := func(u string) bool {
		return u == "test"
	}
	user, err := TryGetUser([]byte("[{}]"), valid)
	if err == nil {
		t.Error("json is valid but no user")
	}
	user, err = TryGetUser([]byte("[{}, {}]"), valid)
	if err == nil {
		t.Error("json is valid but no user")
	}
	user, err = TryGetUser([]byte("[]"), valid)
	if err == nil {
		t.Error("json is valid but no user")
	}
	user, err = TryGetUser([]byte("[{\"usernazme\": \"test\"}]"), valid)
	if err == nil {
		t.Error("json is valid and has invalid key")
	}
	user, err = TryGetUser([]byte("[{\"username\": \"test2\"}]"), valid)
	if err == nil {
		t.Error("json is valid and has invalid user")
	}
	user, err = TryGetUser([]byte("[{\"userID\": \"test\"}]"), valid)
	if err != nil {
		t.Error("json is valid and has user")
	}
	if user != "test" {
		t.Error("wrong user return")
	}
	user, err = TryGetUser([]byte("[{\"username\": \"test\"}]"), valid)
	if err != nil {
		t.Error("json is valid and has user")
	}
	if user != "test" {
		t.Error("wrong user return")
	}
}

func TestTryGetUserSingleton(t *testing.T) {
	valid := func(u string) bool {
		return u == "test"
	}
	user, err := TryGetUser([]byte(""), valid)
	if err == nil {
		t.Error("json is invalid")
	}
	user, err = TryGetUser([]byte("{}"), valid)
	if err == nil {
		t.Error("json is valid but no user")
	}
	user, err = TryGetUser([]byte("{\"usernazme\": \"test\"}"), valid)
	if err == nil {
		t.Error("json is valid and has invalid key")
	}
	user, err = TryGetUser([]byte("{\"username\": \"test2\"}"), valid)
	if err == nil {
		t.Error("json is valid and has invalid user")
	}
	user, err = TryGetUser([]byte("{\"userID\": \"test\"}"), valid)
	if err != nil {
		t.Error("json is valid and has user")
	}
	if user != "test" {
		t.Error("wrong user return")
	}
	user, err = TryGetUser([]byte("{\"username\": \"test\"}"), valid)
	if err != nil {
		t.Error("json is valid and has user")
	}
	if user != "test" {
		t.Error("wrong user return")
	}
}

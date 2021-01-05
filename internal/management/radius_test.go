package management

import (
	"fmt"
	"sort"
	"testing"
)

var (
	vlans = []*VLAN{
		&VLAN{
			ID:   1,
			Name: "test1",
		},
		&VLAN{
			ID:   2,
			Name: "test2",
		},
	}
	systems = []*System{
		&System{
			Type: "sys1",
		},
		&System{
			Type: "sys2",
		},
	}
)

func TestMACAddRADIUS(t *testing.T) {
	d := MACAddRADIUS("test", 10)
	if d != `"TEST" MD5 "TEST"
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:10` {
	}
}

func TestUserAddRADIUS(t *testing.T) {
	d := UserAddRADIUS("test", "abcd", 10)
	if d != `"test" PEAP

"test" MSCHAPV2 hash:abcd [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:10` {
	}
}

func TestCheckMAC(t *testing.T) {
	for _, mac := range []string{"", "     ", "   aa aeiajeiajea"} {
		if err := CheckMAC(mac); err.Error() != fmt.Sprintf("invalid MAC (length): %s", mac) {
			t.Error("length failure")
		}
	}
	for _, mac := range []string{"            ", "aAbDEFda1232", "zzzAJI:12309"} {
		if err := CheckMAC(mac); err.Error() != fmt.Sprintf("invalid MAC (char): %s", mac) {
			t.Error("char failure")
		}
	}
	for _, mac := range []string{"000000000000", "1234567890ab", "cdefcdefcdef"} {
		if err := CheckMAC(mac); err != nil {
			t.Error("valid")
		}
	}
}

func testUser() User {
	return User{Perms: UserPermissions{IsRADIUS: true}}
}

func TestForRADIUS(t *testing.T) {
	_, err := User{Perms: UserPermissions{IsRADIUS: false}}.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "user disabled in RADIUS" {
		t.Error("disabled")
	}
	u := testUser()
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "no user name" {
		t.Error("user")
	}
	u.UserName = "test"
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "no md4" {
		t.Error("md4")
	}
	u.MD4 = "test"
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "no VLANs" {
		t.Error("vlans")
	}
	u.VLANs = []string{"test1"}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "no MACs" {
		t.Error("macs")
	}
	u.VLANs = []string{"test1", "test1TEST"}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "unknown VLAN: test1TEST" {
		t.Error("vlan?")
	}
	u.VLANs = []string{"test1"}
	u.Systems = []UserSystem{UserSystem{}}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "system without id (idx: 0)" {
		t.Error("system id")
	}
	u.Systems = []UserSystem{UserSystem{ID: "test"}}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "system type unknown:  (id test)" {
		t.Error("system type")
	}
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1"}}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "no MACs" {
		t.Error("system macs")
	}
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{VLAN: ""}}}}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "invalid VLAN  for []" {
		t.Error("system vlans")
	}
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{VLAN: "test1", MACs: []string{"aabbccddeeff"}}}}}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "missing hostapd and/or manifest entries, user can NOT login" {
		t.Error("user has no permissions")
	}
	u.Perms.IsPEAP = true
	_, err = u.ForRADIUS(vlans, systems)
	if err != nil {
		t.Error("user has no permissions")
	}
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{MAB: true, VLAN: "test1", MACs: []string{"aabbccddeeff", "aabbccddeeff"}}}}}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "MAC already added for user: aabbccddeeff" {
		t.Error(err.Error())
	}
	u.Perms.IsPEAP = false
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "MAC already added for user: aabbccddeeff" {
		t.Error(err.Error())
	}
	u.VLANs = []string{"test1", "test2"}
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{MAB: true, VLAN: "test1", MACs: []string{"aabbccddeeff"}}, MACMap{MAB: true, VLAN: "test2", MACs: []string{"aabbccddeeff"}}}}}
	_, err = u.ForRADIUS(vlans, systems)
	if err == nil || err.Error() != "MAC already added for user: aabbccddeeff" {
		t.Error(err.Error())
	}
	u.VLANs = []string{"test1"}
	u.Perms.IsPEAP = true
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{VLAN: "test1", MACs: []string{"aabbccddeeff"}}}}}
	_, err = u.ForRADIUS(vlans, systems)
	if err != nil {
		t.Error("valid")
	}
}

func TestLoginAs(t *testing.T) {
	u := testUser()
	u.MD4 = "test"
	u.UserName = "test"
	u.VLANs = []string{"test1"}
	u.Perms.IsPEAP = true
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{VLAN: "test1", MACs: []string{"aabbccddeeff"}}}}}
	o, err := u.ForRADIUS(vlans, systems)
	if err != nil {
		t.Error("valid")
	}
	if len(o.Manifest) != 2 || len(o.Hostapd) != 2 {
		t.Error("invalid manifest/hostapd")
	}
	sort.Strings(o.Manifest)
	if o.Manifest[0] != "test.aabbccddeeff" || o.Manifest[1] != "test1.test.aabbccddeeff" {
		t.Error("wrong manifest entries")
	}
	if fmt.Sprintf("%v", o.Hostapd) != `["test" PEAP

"test" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1 "test1.test" PEAP

"test1.test" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1]` {
		t.Error("invalid hostapd")
	}
	u.LoginAs = "a"
	o, err = u.ForRADIUS(vlans, systems)
	if err != nil {
		t.Error("valid")
	}
	if len(o.Manifest) != 2 || len(o.Hostapd) != 2 {
		t.Error("invalid manifest/hostapd")
	}
	sort.Strings(o.Manifest)
	if o.Manifest[0] != "a.aabbccddeeff" || o.Manifest[1] != "test1.a.aabbccddeeff" {
		t.Error("wrong manifest entries")
	}
	if fmt.Sprintf("%v", o.Hostapd) != `["a" PEAP

"a" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1 "test1.a" PEAP

"test1.a" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1]` {
		t.Error("invalid hostapd")
	}
}

func TestRADIUSOutputs(t *testing.T) {
	u := testUser()
	u.MD4 = "test"
	u.UserName = "test"
	u.Perms.IsPEAP = true
	u.VLANs = []string{"test1", "test2"}
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{MAB: true, VLAN: "test1", MACs: []string{"aabbc1ddeeff"}}, MACMap{MAB: true, VLAN: "test2", MACs: []string{"aabbccddeeff"}}}}}
	o, err := u.ForRADIUS(vlans, systems)
	if err != nil {
		t.Error("valid")
	}
	if len(o.MACs) != 2 {
		t.Error("wrong mac count")
	}
	if len(o.Hostapd) != 5 {
		t.Error("wrong hostapd count")
	}
	if len(o.Manifest) != 8 {
		t.Error("manifest count is wrong")
	}
	sort.Strings(o.Manifest)
	if fmt.Sprintf("%v", o.Manifest) != "[aabbc1ddeeff.aabbc1ddeeff aabbccddeeff.aabbccddeeff test.aabbc1ddeeff test.aabbccddeeff test1.test.aabbc1ddeeff test1.test.aabbccddeeff test2.test.aabbc1ddeeff test2.test.aabbccddeeff]" {
		t.Error("invalid manifest")
	}
	sort.Strings(o.Hostapd)
	if fmt.Sprintf("%v", o.Hostapd) != `["AABBC1DDEEFF" MD5 "AABBC1DDEEFF"
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1 "AABBCCDDEEFF" MD5 "AABBCCDDEEFF"
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:2 "test" PEAP

"test" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1 "test1.test" PEAP

"test1.test" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1 "test2.test" PEAP

"test2.test" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:2]` {
		t.Error("invalid hostapd")
	}
	u.Systems = []UserSystem{UserSystem{ID: "test", Type: "sys1", MACs: []MACMap{MACMap{VLAN: "test1", MACs: []string{"aabbc1ddeeff"}}, MACMap{VLAN: "test2", MACs: []string{"aabbccddeeff"}}}}}
	o, err = u.ForRADIUS(vlans, systems)
	if err != nil {
		t.Error("valid")
	}
	if len(o.MACs) != 2 {
		t.Error("wrong mac count")
	}
	if len(o.Hostapd) != 3 {
		t.Error("wrong hostapd count")
	}
	if len(o.Manifest) != 6 {
		t.Error("manifest count is wrong")
	}
	sort.Strings(o.Manifest)
	if fmt.Sprintf("%v", o.Manifest) != "[test.aabbc1ddeeff test.aabbccddeeff test1.test.aabbc1ddeeff test1.test.aabbccddeeff test2.test.aabbc1ddeeff test2.test.aabbccddeeff]" {
		t.Error("invalid manifest")
	}
	sort.Strings(o.Hostapd)
	if fmt.Sprintf("%v", o.Hostapd) != `["test" PEAP

"test" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1 "test1.test" PEAP

"test1.test" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:1 "test2.test" PEAP

"test2.test" MSCHAPV2 hash:test [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:2]` {
		t.Error("invalid hostapd")
	}
}

func TestMergeRADIUS(t *testing.T) {
	if _, err := MergeRADIUS([]*UserRADIUS{}); err.Error() != "no radius users" {
		t.Error("merge should fail")
	}
	u1 := &UserRADIUS{}
	if _, err := MergeRADIUS([]*UserRADIUS{u1}); err.Error() != "user was not properly radius configured" {
		t.Error("merge should fail")
	}
	u1.MACs = []string{"11aabbccddee"}
	if _, err := MergeRADIUS([]*UserRADIUS{u1}); err.Error() != "user was not properly radius configured" {
		t.Error("merge should fail")
	}
	u1.Hostapd = []string{"garbage"}
	if _, err := MergeRADIUS([]*UserRADIUS{u1}); err.Error() != "user was not properly radius configured" {
		t.Error("merge should fail")
	}
	u1.Manifest = []string{"manifest"}
	if _, err := MergeRADIUS([]*UserRADIUS{u1}); err != nil {
		t.Error("merge should work")
	}
	u2 := &UserRADIUS{}
	u2.MACs = []string{"aabbccddeeff", "11aabbccddee"}
	u2.Manifest = []string{"abc"}
	u2.Hostapd = []string{"garbage2"}
	if _, err := MergeRADIUS([]*UserRADIUS{u1, u2}); err.Error() != "11aabbccddee MAC is already configured for another user" {
		t.Error("merge should fail")
	}
	u2.MACs = []string{"aabbccddeeff"}
	v, err := MergeRADIUS([]*UserRADIUS{u1, u2})
	if err != nil {
		t.Error("merge should work")
	}
	if len(v.Manifest) != 2 || v.Manifest[1] != "manifest" || v.Manifest[0] != "abc" {
		t.Error("invalid manifest")
	}
	if string(v.Hostapd) != `garbage

garbage2` {
		t.Error("invalid hostapd")
	}
}

func TestVLAN(t *testing.T) {
	v := VLAN{}
	v.ID = -1
	if err := v.Check(); err.Error() != "no vlan name" {
		t.Error("invalid vlan not caught")
	}
	v.Name = "a"
	if err := v.Check(); err.Error() != "no vlan description" {
		t.Error("invalid vlan not caught")
	}
	v.Description = "b"
	if err := v.Check(); err.Error() != "invalid vlan id" {
		t.Error("invalid vlan not caught")
	}
	v.ID = 5000
	if err := v.Check(); err.Error() != "invalid vlan id" {
		t.Error("invalid vlan not caught")
	}
	v.ID = 1
	if err := v.Check(); err.Error() != "no route on vlan" {
		t.Error("invalid vlan not caught")
	}
	v.Route = "x"
	if err := v.Check(); err.Error() != "no net associated with vlan" {
		t.Error("invalid vlan not caught")
	}
	v.Net = "n"
	if err := v.Check(); err != nil {
		t.Error("valid")
	}
}

func TestSystem(t *testing.T) {
	s := System{}
	if err := s.Check(); err.Error() != "system description is incomplete" {
		t.Error("invalid system not caught")
	}
	s.Type = "a"
	if err := s.Check(); err.Error() != "system description is incomplete" {
		t.Error("invalid system not caught")
	}
	s.Make = "a"
	if err := s.Check(); err.Error() != "system description is incomplete" {
		t.Error("invalid system not caught")
	}
	s.Model = "b"
	if err := s.Check(); err.Error() != "system description is incomplete" {
		t.Error("invalid system not caught")
	}
	s.Revision = "c"
	if err := s.Check(); err != nil {
		t.Error("system valid")
	}
}

package internal

import (
	"testing"
)

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

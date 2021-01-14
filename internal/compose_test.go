package internal

import (
	"testing"
)

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

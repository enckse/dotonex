package op

import (
	"fmt"
)

type (
	// VLAN for composing vlan definitions
	VLAN struct {
		Name string
		ID   string
	}
	// Member indicates something is a member of a VLAN
	Member struct {
		VLAN string
	}
	// Definition is a shared configuration for composition
	Definition struct {
		VLANs      []VLAN
		Membership []Member
	}
)

// ValidateMembership will check if membership settings are valid
func (d Definition) ValidateMembership() error {
	if len(d.Membership) == 0 {
		return fmt.Errorf("no membership")
	}
	for _, m := range d.Membership {
		if m.VLAN == "" {
			return fmt.Errorf("invalid vlan")
		}
	}
	return nil
}

// ValidateVLANs will check VLAN definitions for correctness
func (d Definition) ValidateVLANs() error {
	if len(d.VLANs) == 0 {
		return fmt.Errorf("no vlans")
	}
	for _, v := range d.VLANs {
		if v.Name == "" || v.ID == "" {
			return fmt.Errorf("invalid vlan")
		}
	}
	return nil
}

// IsVLAN gets and checks if a vlan is valid in the definition
func (d Definition) IsVLAN(name string) (string, bool) {
	for _, v := range d.VLANs {
		if v.Name == name {
			return v.ID, true
		}
	}
	return "", false
}

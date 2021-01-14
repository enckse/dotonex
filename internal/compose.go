package internal

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

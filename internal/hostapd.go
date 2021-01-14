package internal

import (
	"fmt"
)

type (
	// Hostapd is the backend hostapd configuration handler for file generation
	Hostapd struct {
		name     string
		password string
		vlan     string
		mab      bool
	}
)

const (
	attributes = `
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:%s`

	mab   = `"%s" MD5 "%s"` + attributes
	login = `"%s" PEAP

"%s" MSCHAPV2 hash:%s [2]` + attributes
)

// String
func (h Hostapd) String() string {
	if h.mab {
		return fmt.Sprintf(mab, h.name, h.name, h.vlan)
	}
	return fmt.Sprintf(login, h.name, h.name, h.password, h.vlan)
}

// NewHostapd generates a new hostapd configuration setup
func NewHostapd(name, password, vlanID string) Hostapd {
	mab := name == password
	return Hostapd{name: name, password: password, vlan: vlanID, mab: mab}
}

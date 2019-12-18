package authem

import (
	"fmt"
	"sort"
	"strings"

	"voidedtech.com/radiucal/internal/core"
)

// UserAddRADIUS creates an user entry for RADIUS
func UserAddRADIUS(user, md4 string, vlan int) string {
	return fmt.Sprintf(`"%s" PEAP

"%s" MSCHAPV2 hash:%s [2]
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:%d`, user, user, md4, vlan)
}

// MACAddRADIUS creates a MAC entry for RADIUS
func MACAddRADIUS(mac string, vlan int) string {
	upper := strings.ToUpper(mac)
	return fmt.Sprintf(`"%s" MD5 "%s"
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:%d`, upper, upper, vlan)
}

// Check will verify a system has basic definition requirements
func (s System) Check() error {
	for _, t := range [...]string{s.Type, s.Make, s.Model, s.Revision} {
		if isEmpty(t) {
			return fmt.Errorf("system description is incomplete")
		}
	}
	return nil
}

// Check will check vlan configurations for errors
func (v VLAN) Check() error {
	if isEmpty(v.Name) {
		return fmt.Errorf("no vlan name")
	}
	if isEmpty(v.Description) {
		return fmt.Errorf("no vlan description")
	}
	if v.ID < 0 || v.ID > 4096 {
		return fmt.Errorf("invalid vlan id")
	}
	if isEmpty(v.Route) {
		return fmt.Errorf("no route on vlan")
	}
	if isEmpty(v.Net) {
		return fmt.Errorf("no net associated with vlan")
	}
	return nil
}

// MergeRADIUS merges user radius configurations into a RADIUS server configuration
func MergeRADIUS(u []*UserRADIUS) (*RADIUSConfig, error) {
	if len(u) == 0 {
		return nil, fmt.Errorf("no radius users")
	}
	macs := make(map[string]bool)
	var hostapd []string
	var manifest []string
	for _, user := range u {
		if len(user.MACs) == 0 || len(user.Hostapd) == 0 || len(user.Manifest) == 0 {
			return nil, fmt.Errorf("user was not properly radius configured")
		}
		for _, m := range user.MACs {
			if _, ok := macs[m]; ok {
				return nil, fmt.Errorf("%s MAC is already configured for another user", m)
			}
			macs[m] = true
		}
		hostapd = append(hostapd, user.Hostapd...)
		manifest = append(manifest, user.Manifest...)
	}
	sort.Strings(hostapd)
	sort.Strings(manifest)
	return &RADIUSConfig{
		Manifest: manifest,
		Hostapd:  []byte(strings.Join(hostapd, "\n\n")),
	}, nil
}

// ForRADIUS converts a user for radius usage
func (u User) ForRADIUS(vlans []*VLAN, systems []*System, options RADIUSOptions) (*UserRADIUS, error) {
	if !u.Perms.IsRADIUS {
		return nil, fmt.Errorf("user disabled in RADIUS")
	}
	login := u.LoginName()
	if isEmpty(login) {
		return nil, fmt.Errorf("no user name")
	}
	if isEmpty(u.MD4) {
		return nil, fmt.Errorf("no md4")
	}
	r := &UserRADIUS{}
	internalVLANs := make(map[string]int)
	first := true
	for _, v := range u.VLANs {
		found := false
		for _, match := range vlans {
			internalVLANs[v] = match.ID
			if match.Name == v {
				found = true
				if u.Perms.IsPEAP {
					if first {
						r.Hostapd = append(r.Hostapd, UserAddRADIUS(login, u.MD4, match.ID))
					}
					r.Hostapd = append(r.Hostapd, UserAddRADIUS(fmt.Sprintf("%s.%s", v, login), u.MD4, match.ID))
				}
				first = false
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown VLAN: %s", v)
		}
	}
	if first {
		return nil, fmt.Errorf("no VLANs")
	}
	hasMAC := false
	trackMACs := make(map[string]bool)
	for idx, s := range u.Systems {
		if len(s.ID) == 0 {
			return nil, fmt.Errorf("system without id (idx: %d)", idx)
		}
		found := false
		for _, sType := range systems {
			if sType.Type == s.Type {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("system type unknown: %s (id %s)", s.Type, s.ID)
		}
		for _, macs := range s.MACs {
			vlan := macs.VLAN
			id, ok := internalVLANs[vlan]
			if !ok {
				return nil, fmt.Errorf("invalid VLAN %s for %v", vlan, macs.MACs)
			}
			for _, mac := range macs.MACs {
				if err := CheckMAC(mac); err != nil {
					return nil, err
				}
				if trackMACs[mac] {
					return nil, fmt.Errorf("MAC already added for user: %s", mac)
				}
				mab := macs.MAB
				trackMACs[mac] = mab
				if mab {
					r.Hostapd = append(r.Hostapd, MACAddRADIUS(mac, id))
				}
				hasMAC = true
			}
		}
	}
	if !hasMAC {
		return nil, fmt.Errorf("no MACs")
	}
	for mac, mab := range trackMACs {
		r.MACs = append(r.MACs, mac)
		if mab {
			r.Manifest = append(r.Manifest, core.NewManifestEntry(mac, mac))
		}
		for _, uVLAN := range append([]string{""}, u.VLANs...) {
			credentials := login
			if !isEmpty(uVLAN) {
				credentials = fmt.Sprintf("%s.%s", uVLAN, credentials)
			}
			if u.Perms.IsPEAP {
				r.Manifest = append(r.Manifest, core.NewManifestEntry(credentials, mac))
			}
		}
	}
	if len(r.Manifest) == 0 || len(r.Hostapd) == 0 {
		return nil, fmt.Errorf("missing hostapd and/or manifest entries, user can NOT login")
	}
	return r, nil
}

// CheckMAC verifies if a MAC looks right
func CheckMAC(mac string) error {
	if len(mac) != 12 {
		return fmt.Errorf("invalid MAC (length): %s", mac)
	}
	for _, r := range mac {
		if (r >= 'a' && r <= 'f') || (r >= '0' && r <= '9') {
			continue
		}
		return fmt.Errorf("invalid MAC (char): %s", mac)
	}
	return nil
}

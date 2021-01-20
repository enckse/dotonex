package compose

import (
	"encoding/json"
	"fmt"
)

var (
	userNameFields = []string{"username", "name", "user", "userid", "userName", "UserName", "userId", "userID"}
)

type (
	userMap map[string]interface{}

	// GetUser is a callback to verify if a user is valid within the backend system
	GetUser func(string) bool

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

func tryUserMap(m userMap, verify GetUser) (string, error) {
	for _, k := range userNameFields {
		if _, ok := m[k]; !ok {
			continue
		}
		user, ok := m[k].(string)
		if !ok {
			continue
		}
		if verify(user) {
			return user, nil
		}
	}

	return "", fmt.Errorf("no user found in map")
}

func tryUserArray(data []byte, verify GetUser) (string, error) {
	var object []userMap
	if err := json.Unmarshal(data, &object); err != nil {
		return "", err
	}
	if len(object) != 1 {
		return "", fmt.Errorf("invalid object returned: not 1")
	}

	return tryUserMap(object[0], verify)
}

func trySingletonObject(data []byte, verify GetUser) (string, error) {
	object := make(userMap)
	if err := json.Unmarshal(data, &object); err != nil {
		return "", err
	}
	return tryUserMap(object, verify)
}

// TryGetUser will try and find a user in json output and validate it
func TryGetUser(data []byte, verify GetUser) (string, error) {
	sUser, sErr := trySingletonObject(data, verify)
	if sErr != nil {
		mUser, mErr := tryUserArray(data, verify)
		if mErr != nil {
			return "", fmt.Errorf("unable to detect user (%v,%v)", sErr, mErr)
		}
		return mUser, nil
	}
	return sUser, nil
}

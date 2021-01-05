package management

import (
	"fmt"
	"os"
	"strings"

	"voidedtech.com/radiucal/internal/core"
)

type (
	// UserConfig contains user configuration data
	UserConfig struct {
		Users []*User
	}

	// System represents a system definition
	System struct {
		Type     string
		Make     string
		Model    string
		Revision string
	}

	// Secret represents a user's secret information
	Secret struct {
		UserName string
		Password string
		Email    string
		Fake     bool `yaml:"-"`
	}

	// VLAN represents a textual VLAN description
	VLAN struct {
		Name        string
		ID          int
		Description string
		Initiate    []string
		Route       string
		Net         string
	}

	// UserPermissions reflect controlled permissions for a user within all of management
	UserPermissions struct {
		IsRADIUS bool
		IsPEAP   bool
		IsRoot   bool
		Trusts   []string
	}

	// MACMap represents mac control for control within a VLAN for auth
	MACMap struct {
		VLAN string
		MACs []string
		MAB  bool
	}

	// UserSystem represents how we attach systems to users
	UserSystem struct {
		Type string
		ID   string
		MACs []MACMap
	}

	// User definitions
	User struct {
		UserName string
		FullName string
		MD4      string
		VLANs    []string
		Systems  []UserSystem
		Perms    UserPermissions
		LoginAs  string
	}

	// UserRADIUS represents a login available for radius
	UserRADIUS struct {
		Manifest []string
		Hostapd  []string
		MACs     []string
	}

	// RADIUSConfig contains a radius configuration to write to disk
	RADIUSConfig struct {
		Manifest []string
		Hostapd  []byte
	}
)

func isEmpty(value string) bool {
	return len(strings.TrimSpace(value)) == 0
}

// LoginName gets the user's login name for system logins
func (u *User) LoginName() string {
	if !isEmpty(u.LoginAs) {
		return u.LoginAs
	}
	return u.UserName
}

// Inflate populates a user with secret information for configuration in other tools
func (u *User) Inflate(key string, secrets []*Secret) error {
	for _, s := range secrets {
		if s.UserName == u.UserName {
			if s.Fake {
				u.MD4 = core.MD4("")
				return nil
			}
			u.MD4 = core.MD4(s.Password)
			return nil
		}
	}
	return fmt.Errorf("user secrets not configured")
}

// GetKey retrieves the management key used for secrets
func GetKey(optional bool) (string, error) {
	k := os.Getenv("AUTHEM_KEY")
	if strings.TrimSpace(k) == "" {
		if optional {
			return "", nil
		}
		return "", fmt.Errorf("AUTHEM_KEY not set")
	}
	return k, nil
}

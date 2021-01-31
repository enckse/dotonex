package compose

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/tidwall/buntdb"
	"voidedtech.com/dotonex/internal/core"
)

const (
	inArrayPre  = "inarray["
	inArrayPost = "]"
)

type (
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

	// Store is backend handling of data
	Store struct {
		core.ComposeFlags
		db *buntdb.DB
	}
)

// Get will get a store value
func (s Store) Get(key string) (string, bool, error) {
	var val string
	rErr := s.db.View(func(tx *buntdb.Tx) error {
		var err error
		val, err = tx.Get(key)
		if err != nil {
			return err
		}
		return nil
	})
	if rErr == nil {
		return val, true, nil
	}
	if rErr == buntdb.ErrNotFound {
		return "", false, nil
	}
	return "", false, rErr
}

// Save commits a value to the store
func (s Store) Save(key, value string) error {
	err := s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(key, value, nil)
		return err
	})
	return err
}

// NewKey creates a database key
func (s Store) NewKey(name string) string {
	return fmt.Sprintf("root=>%s", name)
}

// NewStore initializes a storage backend
func NewStore(flags core.ComposeFlags, db *buntdb.DB) Store {
	return Store{ComposeFlags: flags, db: db}
}

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

// TryGetUser will try and find a user in json output and validate it
func TryGetUser(layout []string, data []byte, verify GetUser) (string, error) {
	var errors []error
	for idx, l := range layout {
		next := layout[idx+1:]
		isArray := strings.HasPrefix(l, inArrayPre) && strings.HasSuffix(l, inArrayPost)
		if isArray {
			indexer := strings.Replace(l, inArrayPre, "", 1)
			indexer = strings.TrimSpace(indexer[0 : len(indexer)-1])
			useIndex := -1
			if indexer != "" {
				i, err := strconv.Atoi(indexer)
				if err != nil {
					return "", err
				}
				useIndex = i
			}
			var obj []interface{}
			if err := json.Unmarshal(data, &obj); err != nil {
				return "", err
			}
			if len(obj) == 0 {
				return "", fmt.Errorf("zero array found")
			}
			for subIdx, sub := range obj {
				if useIndex >= 0 {
					if subIdx != useIndex {
						continue
					}
				}
				b, err := json.Marshal(sub)
				if err != nil {
					return "", err
				}
				found, err := TryGetUser(next, b, verify)
				if found != "" {
					return found, nil
				}
				errors = append(errors, err)
			}
		} else {
			m := make(map[string]interface{})
			if err := json.Unmarshal(data, &m); err != nil {
				return "", err
			}
			sub, ok := m[l]
			if !ok {
				return "", fmt.Errorf("subkey not found")
			}
			user, ok := sub.(string)
			if ok {
				if user == "" {
					return "", fmt.Errorf("empty string found")
				}
				return user, nil
			}
			b, err := json.Marshal(sub)
			if err != nil {
				return "", err
			}
			found, err := TryGetUser(next, b, verify)
			if found != "" {
				return found, nil
			}
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		var multi []string
		for _, e := range errors {
			multi = append(multi, e.Error())
		}
		return "", fmt.Errorf(strings.Join(multi, "\n"))
	}
	return "", fmt.Errorf("unable to find a user")
}

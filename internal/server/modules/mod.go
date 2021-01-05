package modules

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/internal/server/processing"
)

var (
	lockUserAuth     = &sync.Mutex{}
	userAuthManifest = make(map[string]bool)
)

// SetUserAuths does a lock-safe update the set of user+mac combinations
func SetUserAuths(set []string) {
	lockUserAuth.Lock()
	defer lockUserAuth.Unlock()
	userAuthManifest = make(map[string]bool)
	for _, u := range set {
		userAuthManifest[u] = true
	}
}

// Access logging of requests for auth endpoints
func Access(mode processing.ModuleMode, packet *processing.ClientPacket) {
	go func() {
		username, err := rfc2865.UserName_LookupString(packet.Packet)
		if err != nil {
			username = ""
		}
		calling, err := rfc2865.CallingStationID_LookupString(packet.Packet)
		if err != nil {
			calling = ""
		}
		kv := processing.KeyValueStore{}
		kv.DropEmpty = true
		kv.Add("Mode", fmt.Sprintf("%d", mode))
		kv.Add("Code", packet.Packet.Code.String())
		kv.Add("Id", strconv.Itoa(int(packet.Packet.Identifier)))
		kv.Add("User-Name", username)
		kv.Add("Calling-Station-Id", calling)
		processing.LogModuleMessages("ACCESS", kv.Strings())
	}()
}

// LogPacket will log packet and mode (useful for acct and auth)
func LogPacket(mode processing.ModuleMode, packet *processing.ClientPacket) {
	go func() {
		dump := processing.NewRequestDump(packet, fmt.Sprintf("%d", mode))
		messages := dump.DumpPacket(processing.KeyValue{})
		processing.LogModuleMessages("LOG", messages)
	}()
}

// AuthorizeUserMAC validates if a user+MAC combo should be allowed
func AuthorizeUserMAC(packet *processing.ClientPacket) bool {
	return checkUserMAC(packet) == nil
}

func clean(in string) string {
	result := ""
	for _, c := range strings.ToLower(in) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' {
			result = result + string(c)
		}
	}
	return result
}

func checkUserMAC(p *processing.ClientPacket) error {
	username, err := rfc2865.UserName_LookupString(p.Packet)
	if err != nil {
		return err
	}
	calling, err := rfc2865.CallingStationID_LookupString(p.Packet)
	if err != nil {
		return err
	}
	username = clean(username)
	calling = clean(calling)
	fqdn := newManifestEntry(username, calling)
	success := true
	var failure error
	lockUserAuth.Lock()
	_, ok := userAuthManifest[fqdn]
	lockUserAuth.Unlock()
	if !ok {
		failure = fmt.Errorf("failed preauth: %s %s", username, calling)
		success = false
	}
	go mark(success, username, calling, p, false)
	return failure
}

func mark(success bool, user, calling string, p *processing.ClientPacket, cached bool) {
	nas := clean(rfc2865.NASIdentifier_GetString(p.Packet))
	if len(nas) == 0 {
		nas = "unknown"
	}
	nasipraw := rfc2865.NASIPAddress_Get(p.Packet)
	nasip := "noip"
	if nasipraw == nil {
		if p.NASIP != "" {
			nasip = p.NASIP
		}
	} else {
		nasip = nasipraw.String()
	}
	nasport := rfc2865.NASPort_Get(p.Packet)
	result := "PASSED"
	if !success {
		result = "FAILED"
	}
	kv := processing.KeyValueStore{}
	kv.Add("Result", result)
	kv.Add("User-Name", user)
	kv.Add("Calling-Station-Id", calling)
	kv.Add("NAS-Id", nas)
	kv.Add("NAS-IPAddress", nasip)
	kv.Add("NAS-Port", fmt.Sprintf("%d", nasport))
	kv.Add("Id", strconv.Itoa(int(p.Packet.Identifier)))
	processing.LogModuleMessages("USERMAC", kv.Strings())
}

func newManifestEntry(user, mac string) string {
	return fmt.Sprintf("%s.%s", user, mac)
}

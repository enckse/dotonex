package modules

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/dotonex/internal/core"
	"voidedtech.com/dotonex/internal/op"
)

type (
	ProxyModule struct {
	}
)

func (l *ProxyModule) Name() string {
	return "proxy"
}

func (l *ProxyModule) Pre(packet *op.ClientPacket) bool {
	return checkUserMac(packet) == nil
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

func checkUserMac(p *op.ClientPacket) error {
	userName, err := rfc2865.UserName_LookupString(p.Packet)
	if err != nil {
		return err
	}
	calling, err := rfc2865.CallingStationID_LookupString(p.Packet)
	if err != nil {
		return err
	}
	token := userName
	calling = clean(calling)
	success := true
	var failure error
	valid := true
	cleaned, isMAC := core.CleanMAC(calling)
	if isMAC {
		if calling != clean(token) {
			// This is NOT a MAB situation
			valid = op.CheckTokenMAC(token, cleaned)
		}
	} else {
		valid = false
	}
	if !valid {
		failure = fmt.Errorf("failed preauth: %s %s", userName, calling)
		success = false
	}
	go mark(success, userName, calling, p, false)
	return failure
}

func mark(success bool, user, calling string, p *op.ClientPacket, cached bool) {
	nas := clean(rfc2865.NASIdentifier_GetString(p.Packet))
	if len(nas) == 0 {
		nas = "unknown"
	}
	nasipraw := rfc2865.NASIPAddress_Get(p.Packet)
	nasip := "noip"
	if nasipraw == nil {
		if p.ClientAddr != nil {
			h, _, err := net.SplitHostPort(p.ClientAddr.String())
			if err == nil {
				nasip = h
			}
		}
	} else {
		nasip = nasipraw.String()
	}
	nasport := rfc2865.NASPort_Get(p.Packet)
	result := "PASSED"
	if !success {
		result = "FAILED"
	}
	kv := keyValueStore{}
	kv.add("Result", result)
	kv.add("User-Name", user)
	kv.add("Calling-Station-Id", calling)
	kv.add("NAS-Id", nas)
	kv.add("NAS-IPAddress", nasip)
	kv.add("NAS-Port", fmt.Sprintf("%d", nasport))
	kv.add("Id", strconv.Itoa(int(p.Packet.Identifier)))
	logPluginMessages("proxy", kv.strings())
}

func (kv *keyValueStore) add(key, val string) {
	kv.keyValues = append(kv.keyValues, keyValue{key: key, value: val})
}

func (kv keyValueStore) strings() []string {
	var objs []string
	offset := ""
	for _, k := range kv.keyValues {
		objs = append(objs, fmt.Sprintf("%s%s", offset, k.str()))
		offset = "  "
	}
	return objs
}

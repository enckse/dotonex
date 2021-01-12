package modules

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/dotonex/internal"
)

type (
	proxyModule struct {
	}
)

func (l *proxyModule) Name() string {
	return "proxy"
}

var (
	// ProxyModule represents the instance for the system
	ProxyModule proxyModule
)

func (l *proxyModule) Setup(ctx *internal.PluginContext) error {
	return nil
}

func (l *proxyModule) Pre(packet *internal.ClientPacket) bool {
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

func checkUserMac(p *internal.ClientPacket) error {
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
	cleaned, isMAC := internal.CleanMAC(calling)
	if isMAC {
		if calling != clean(token) {
			// This is NOT a MAB situation
			valid = internal.CheckTokenMAC(token, cleaned)
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

func mark(success bool, user, calling string, p *internal.ClientPacket, cached bool) {
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
	kv := internal.KeyValueStore{}
	kv.Add("Result", result)
	kv.Add("User-Name", user)
	kv.Add("Calling-Station-Id", calling)
	kv.Add("NAS-Id", nas)
	kv.Add("NAS-IPAddress", nasip)
	kv.Add("NAS-Port", fmt.Sprintf("%d", nasport))
	kv.Add("Id", strconv.Itoa(int(p.Packet.Identifier)))
	internal.LogPluginMessages(&ProxyModule, kv.Strings())
}

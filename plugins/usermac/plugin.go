package main

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/core"
)

type umac struct {
}

func (l *umac) Name() string {
	return "usermac"
}

var (
	cache                = make(map[string]bool)
	lock     *sync.Mutex = new(sync.Mutex)
	canCache bool
	db       string
	// Plugin represents the instance for the system
	Plugin   umac
	instance string
)

func (l *umac) Reload() {
	lock.Lock()
	defer lock.Unlock()
	cache = make(map[string]bool)
}

func (l *umac) Setup(ctx *core.PluginContext) error {
	canCache = ctx.Cache
	instance = ctx.Instance
	db = filepath.Join(ctx.Lib, "users")
	return nil
}

func (l *umac) Pre(packet *core.ClientPacket) bool {
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

func checkUserMac(p *core.ClientPacket) error {
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
	fqdn := fmt.Sprintf("%s.%s", username, calling)
	lock.Lock()
	good, ok := cache[fqdn]
	lock.Unlock()
	if canCache && ok {
		core.WriteDebug("object is preauthed", fqdn)
		go mark(good, username, calling, p, true)
		if good {
			return nil
		}
		return fmt.Errorf("%s is blacklisted", fqdn)
	}
	core.WriteDebug("not preauthed", fqdn)
	path := filepath.Join(db, fqdn)
	success := true
	var failure error
	res := core.PathExists(path)
	lock.Lock()
	cache[fqdn] = res
	lock.Unlock()
	if !res {
		failure = fmt.Errorf("failed preauth: %s %s", username, calling)
		success = false
	}
	go mark(success, username, calling, p, false)
	return failure
}

func mark(success bool, user, calling string, p *core.ClientPacket, cached bool) {
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
	kv := core.KeyValueStore{}
	kv.Add("Result", result)
	kv.Add("User-Name", user)
	kv.Add("Calling-Station-Id", calling)
	kv.Add("NAS-Id", nas)
	kv.Add("NAS-IPAddress", nasip)
	kv.Add("NAS-Port", fmt.Sprintf("%d", nasport))
	kv.Add("Id", strconv.Itoa(int(p.Packet.Identifier)))
	core.LogPluginMessages(&Plugin, kv.Strings())
}

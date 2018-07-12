package main

import (
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"

	"github.com/epiphyte/goutils"
	"github.com/epiphyte/radiucal/core"
	. "layeh.com/radius/rfc2865"
)

type umac struct {
}

func (l *umac) Name() string {
	return "usermac"
}

var (
	cache    map[string]bool = make(map[string]bool)
	lock     *sync.Mutex     = new(sync.Mutex)
	fileLock *sync.Mutex     = new(sync.Mutex)
	canCache bool
	db       string
	logs     string
	Plugin   umac
	instance string
	// Function callback on failed/passed
	doCallback bool
	callback   []string
	onFail     bool
	onPass     bool
)

func (l *umac) Reload() {
	lock.Lock()
	defer lock.Unlock()
	cache = make(map[string]bool)
}

func (l *umac) Setup(ctx *core.PluginContext) {
	canCache = ctx.Cache
	logs = ctx.Logs
	instance = ctx.Instance
	db = filepath.Join(ctx.Lib, "users")
	callback = ctx.Section.GetArrayOrEmpty("callback")
	doCallback = len(callback) > 0
	onFail = !ctx.Section.GetFalse("nofail")
	onPass = !ctx.Section.GetFalse("nopass")
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
	username, err := UserName_LookupString(p.Packet)
	if err != nil {
		return err
	}
	calling, err := CallingStationID_LookupString(p.Packet)
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
		goutils.WriteDebug("object is preauthed", fqdn)
		if good {
			return nil
		} else {
			return errors.New(fmt.Sprintf("%s is blacklisted", fqdn))
		}
	} else {
		goutils.WriteDebug("not preauthed", fqdn)
	}
	path := filepath.Join(db, fqdn)
	success := true
	var failure error
	res := goutils.PathExists(path)
	lock.Lock()
	cache[fqdn] = res
	lock.Unlock()
	if !res {
		failure = errors.New(fmt.Sprintf("failed preauth: %s %s", username, calling))
		success = false
	}
	go mark(success, username, calling, p)
	return failure
}

func mark(success bool, user, calling string, p *core.ClientPacket) {
	nas := clean(NASIdentifier_GetString(p.Packet))
	if len(nas) == 0 {
		nas = "unknown"
	}
	nasipraw := NASIPAddress_Get(p.Packet)
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
	nasport := NASPort_Get(p.Packet)
	fileLock.Lock()
	defer fileLock.Unlock()
	f, t := core.DatedAppendFile(logs, "audit", instance)
	if f == nil {
		return
	}
	defer f.Close()
	result := "passed"
	if !success {
		result = "failed"
	}
	msg := fmt.Sprintf("%s (mac:%s) (nas:%s,ip:%s,port:%d)", user, calling, nas, nasip, nasport)
	if doCallback {
		reportFail := !success && onFail
		reportPass := success && onPass
		if reportFail || reportPass {
			goutils.WriteDebug("perform callback", callback...)
			args := callback[1:]
			args = append(args, fmt.Sprintf("%s -> %s", result, msg))
			goutils.RunCommand(callback[0], args...)
		}
	}
	core.FormatLog(f, t, result, msg)
}

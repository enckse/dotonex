package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "layeh.com/radius/rfc2865"
	"voidedtech.com/goutils/logger"
	"voidedtech.com/goutils/opsys"
	"voidedtech.com/radiucal/core"
)

type umac struct {
}

func (l *umac) Name() string {
	return "usermac"
}

var (
	cache    map[string]bool     = make(map[string]bool)
	lock     *sync.Mutex         = new(sync.Mutex)
	fileLock *sync.Mutex         = new(sync.Mutex)
	logged   map[string]struct{} = make(map[string]struct{})
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
	fileLock.Lock()
	defer fileLock.Unlock()
	cache = make(map[string]bool)
	logged = make(map[string]struct{})
}

func (l *umac) Setup(ctx *core.PluginContext) {
	canCache = ctx.Cache
	logs = ctx.Logs
	instance = ctx.Instance
	db = filepath.Join(ctx.Lib, "users")
	callback = ctx.Section.GetArrayOrEmpty("callback")
	if len(callback) > 0 {
		onFail = ctx.Section.GetTrue("onfail")
		onPass = ctx.Section.GetTrue("onpass")
		doCallback = onFail || onPass
	}
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
		logger.WriteDebug("object is preauthed", fqdn)
		go mark(good, username, calling, p, true)
		if good {
			return nil
		} else {
			return errors.New(fmt.Sprintf("%s is blacklisted", fqdn))
		}
	} else {
		logger.WriteDebug("not preauthed", fqdn)
	}
	path := filepath.Join(db, fqdn)
	success := true
	var failure error
	res := opsys.PathExists(path)
	lock.Lock()
	cache[fqdn] = res
	lock.Unlock()
	if !res {
		failure = errors.New(fmt.Sprintf("failed preauth: %s %s", username, calling))
		success = false
	}
	go mark(success, username, calling, p, false)
	return failure
}

func formatLog(f *os.File, t time.Time, indicator, message string) {
	l := fmt.Sprintf("%s [%s] %s\n", t.Format("2006-01-02T15:04:05"), strings.ToUpper(indicator), message)
	if _, ok := logged[l]; !ok {
		f.Write([]byte(l))
	}
}

func mark(success bool, user, calling string, p *core.ClientPacket, cached bool) {
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
	if doCallback && !cached {
		reportFail := !success && onFail
		reportPass := success && onPass
		if reportFail || reportPass {
			logger.WriteDebug("perform callback", callback...)
			args := callback[1:]
			opts := &opsys.RunOptions{}
			opts.Stdin = append(opts.Stdin, fmt.Sprintf("%s -> %s", result, msg))
			opsys.RunCommandWithOptions(opts, callback[0], args...)
		}
	}
	formatLog(f, t, result, msg)
}

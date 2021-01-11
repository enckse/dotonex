package modules

import (
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/internal"
)

type (
	proxyModule struct {
	}
)

func (l *proxyModule) Name() string {
	return "proxy"
}

var (
	lock     = &sync.Mutex{}
	file     string
	manifest = make(map[string]bool)
	// Plugin represents the instance for the system
	ProxyModule proxyModule
)

func (l *proxyModule) load() error {
	if !internal.PathExists(file) {
		return fmt.Errorf("%s is missing", file)
	}
	lock.Lock()
	defer lock.Unlock()
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	manifest = make(map[string]bool)
	data := strings.Split(string(b), "\n")
	kv := internal.KeyValueStore{}
	kv.Add("Manfiest", "load")
	idx := 0
	for _, d := range data {
		if strings.TrimSpace(d) == "" {
			continue
		}
		kv.Add(fmt.Sprintf("Manifest-%d", idx), d)
		manifest[d] = true
		idx++
	}
	internal.LogPluginMessages(&ProxyModule, kv.Strings())
	return nil
}

func (l *proxyModule) Setup(ctx *internal.PluginContext) error {
	file = filepath.Join(ctx.Lib, "manifest")
	if err := l.load(); err != nil {
		return err
	}
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
	fqdn := internal.NewManifestEntry(username, calling)
	success := true
	var failure error
	lock.Lock()
	_, ok := manifest[fqdn]
	lock.Unlock()
	if !ok {
		failure = fmt.Errorf("failed preauth: %s %s", username, calling)
		success = false
	}
	go mark(success, username, calling, p, false)
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

package main

import (
	"net"
	"strings"
	"sync"

	yaml "gopkg.in/yaml.v2"
	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/core"
)

const (
	noIP = "noip"
	star = "*"
)

type nwl struct {
}

func (l *nwl) Name() string {
	return "naswhitelist"
}

var (
	// Plugin represents the system instance of the module
	Plugin    nwl
	lock      *sync.Mutex = new(sync.Mutex)
	whitelist             = make(map[string]bool)
	enabled   bool
	order     []string
)

func (l *nwl) Reload() {
}

// NasWhitelistConfig represents the plugin's specific configuration
type NasWhitelistConfig struct {
	NasWhitelist struct {
		Whitelist []string
	}
}

func (l *nwl) Setup(ctx *core.PluginContext) error {
	conf := &NasWhitelistConfig{}
	err := yaml.Unmarshal(ctx.Backing, conf)
	if err != nil {
		return err
	}
	array := conf.NasWhitelist.Whitelist
	l.startup(array)
	return nil
}

func (l *nwl) startup(array []string) {
	enabled = false
	lock.Lock()
	defer lock.Unlock()
	whitelist = make(map[string]bool)
	order = []string{}
	if len(array) > 0 {
		tracked := make(map[string]int)
		for _, ip := range array {
			ipSplit := len(strings.Split(ip, "."))
			if ipSplit > 4 {
				core.WriteWarn("invalid ip", ip)
				continue
			}
			enabled = true
			isBlacklist := false
			if strings.HasPrefix(ip, "!") {
				isBlacklist = true
				ip = ip[1:len(ip)]
			}
			if i, ok := tracked[ip]; ok {
				order = append(order[:i], order[i+1:]...)
			}
			tracked[ip] = len(order)
			order = append(order, ip)
			whitelist[ip] = isBlacklist
		}

		core.WriteDebug("ips (ordered)", order...)
	}
}

func (l *nwl) Pre(packet *core.ClientPacket) bool {
	if !enabled {
		return true
	}
	nasipraw := rfc2865.NASIPAddress_Get(packet.Packet)
	nasip := noIP
	if nasipraw == nil {
		if packet.ClientAddr != nil {
			h, _, err := net.SplitHostPort(packet.ClientAddr.String())
			if err == nil {
				nasip = h
			}
		}
	} else {
		nasip = nasipraw.String()
	}
	if nasip == noIP {
		return false
	}
	lock.Lock()
	defer lock.Unlock()
	last := false
	valid := false
	for _, k := range order {
		v, ok := whitelist[k]
		if !ok {
			core.WriteWarn("internal error")
			return false
		}
		match := false
		if strings.HasSuffix(k, ".") {
			if strings.HasPrefix(nasip, k) {
				match = true
			}
		} else {
			if nasip == k {
				match = true
			}
		}
		if match {
			valid = true
			last = v
			if !last {
			}
		}
	}
	if !valid {
		return false
	}
	return !last
}

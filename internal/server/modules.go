package server

import (
	"fmt"
	"strconv"

	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"sync"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/internal/core"
)

type (
	logger struct {
	}
	umac struct {
	}
	access struct {
	}
)

var (
	ModuleLog     logger
	lockMAC       = &sync.Mutex{}
	fileMAC       string
	manifest      = make(map[string]bool)
	ModuleUserMAC umac
	ModuleAccess  access
)

func (l *access) Name() string {
	return "access"
}

func (l *access) Setup(ctx *ModuleContext) error {
	return nil
}

func (l *access) Process(packet *ClientPacket, mode ModuleMode) bool {
	l.write(mode, packet)
	return true
}

func (l *access) write(mode ModuleMode, packet *ClientPacket) {
	go func() {
		username, err := rfc2865.UserName_LookupString(packet.Packet)
		if err != nil {
			username = ""
		}
		calling, err := rfc2865.CallingStationID_LookupString(packet.Packet)
		if err != nil {
			calling = ""
		}
		kv := KeyValueStore{}
		kv.DropEmpty = true
		kv.Add("Mode", fmt.Sprintf("%d", mode))
		kv.Add("Code", packet.Packet.Code.String())
		kv.Add("Id", strconv.Itoa(int(packet.Packet.Identifier)))
		kv.Add("User-Name", username)
		kv.Add("Calling-Station-Id", calling)
		LogModuleMessages(l, kv.Strings())
	}()
}

func (l *logger) Name() string {
	return "logger"
}

func (l *logger) Setup(ctx *ModuleContext) error {
	return nil
}

func (l *logger) Process(packet *ClientPacket, mode ModuleMode) bool {
	l.write(mode, packet)
	return true
}

func (l *logger) write(mode ModuleMode, packet *ClientPacket) {
	go func() {
		dump := NewRequestDump(packet, fmt.Sprintf("%d", mode))
		messages := dump.DumpPacket(KeyValue{})
		LogModuleMessages(l, messages)
	}()
}

func (l *umac) Name() string {
	return "usermac"
}

func (l *umac) load() error {
	if !core.PathExists(fileMAC) {
		return fmt.Errorf("%s is missing", fileMAC)
	}
	lockMAC.Lock()
	defer lockMAC.Unlock()
	b, err := ioutil.ReadFile(fileMAC)
	if err != nil {
		return err
	}
	manifest = make(map[string]bool)
	data := strings.Split(string(b), "\n")
	kv := KeyValueStore{}
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
	LogModuleMessages(l, kv.Strings())
	return nil
}

func (l *umac) Setup(ctx *ModuleContext) error {
	fileMAC = filepath.Join(ctx.Lib, "manifest")
	if err := l.load(); err != nil {
		return err
	}
	return nil
}

func (l *umac) Process(packet *ClientPacket, mode ModuleMode) bool {
	if mode == PreProcess {
		return l.checkUserMac(packet) == nil
	}
	return true
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

func (l *umac) checkUserMac(p *ClientPacket) error {
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
	fqdn := core.NewManifestEntry(username, calling)
	success := true
	var failure error
	lockMAC.Lock()
	_, ok := manifest[fqdn]
	lockMAC.Unlock()
	if !ok {
		failure = fmt.Errorf("failed preauth: %s %s", username, calling)
		success = false
	}
	go l.mark(success, username, calling, p, false)
	return failure
}

func (l *umac) mark(success bool, user, calling string, p *ClientPacket, cached bool) {
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
	kv := KeyValueStore{}
	kv.Add("Result", result)
	kv.Add("User-Name", user)
	kv.Add("Calling-Station-Id", calling)
	kv.Add("NAS-Id", nas)
	kv.Add("NAS-IPAddress", nasip)
	kv.Add("NAS-Port", fmt.Sprintf("%d", nasport))
	kv.Add("Id", strconv.Itoa(int(p.Packet.Identifier)))
	LogModuleMessages(l, kv.Strings())
}

// LoadModule loads a plugin from the name and into a module object
func LoadModule(name string, ctx *ModuleContext) (Module, error) {
	mod, err := getModule(name)
	if err != nil {
		return nil, err
	}
	if err := mod.Setup(ctx.CloneContext()); err != nil {
		return nil, err
	}
	return mod, nil
}

func getModule(name string) (Module, error) {
	switch name {
	case "usermac":
		return &ModuleUserMAC, nil
	case "log":
		return &ModuleLog, nil
	case "access":
		return &ModuleAccess, nil
	}
	return nil, fmt.Errorf("unknown plugin type %s", name)
}

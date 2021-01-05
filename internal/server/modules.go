package server

import (
	"fmt"
	"bytes"
	"io"
	"log"
	"time"
	"strconv"

	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"sync"

	"voidedtech.com/radiucal/internal/core"
	"layeh.com/radius/rfc2865"
)

type (
	tracer struct {
	modes  []string
	}

	logTrace struct {
		io.Writer
		data bytes.Buffer
	modes  []string
	}
	logger struct {
	modes  []string
	}
	umac struct {
	modes  []string
	}
	access struct{
	modes  []string
}
)

var (
	ModuleDebug tracer
	ModuleLog   logger
	lockMAC        = &sync.Mutex{}
	fileMAC    string
	manifest = make(map[string]bool)
	ModuleUserMAC umac
	ModuleAccess access
)

func (l *access) Name() string {
	return "access"
}

func (l *access) Setup(ctx *ModuleContext) error {
	l.modes = DisabledModes(l, ctx)
	return nil
}

func (l *access) Pre(packet *ClientPacket) bool {
	return NoopPre(packet, l.write)
}

func (l *access) Post(packet *ClientPacket) bool {
	return NoopPost(packet, l.write)
}

func (l *access) Trace(t TraceType, packet *ClientPacket) {
	l.write(TracingMode, t, packet)
}

func (l *access) Account(packet *ClientPacket) {
	l.write(AccountingMode, NoTrace, packet)
}

func (l *access) write(mode string, objType TraceType, packet *ClientPacket) {
	go func() {
		if Disabled(mode, l.modes) {
			return
		}
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
		kv.Add("Mode", fmt.Sprintf("%s", mode))
		kv.Add("Code", packet.Packet.Code.String())
		kv.Add("Id", strconv.Itoa(int(packet.Packet.Identifier)))
		kv.Add("User-Name", username)
		kv.Add("Calling-Station-Id", calling)
		LogModuleMessages(l, kv.Strings())
	}()
}


func (t *tracer) Name() string {
	return "debugger"
}

func (t *tracer) Setup(ctx *ModuleContext) error {
	t.modes = DisabledModes(t, ctx)
	return nil
}

func (t *tracer) Pre(packet *ClientPacket) bool {
	return NoopPre(packet, t.write)
}

func (t *tracer) Post(packet *ClientPacket) bool {
	return NoopPost(packet, t.write)
}

func (t *tracer) Trace(objType TraceType, packet *ClientPacket) {
	t.write(TracingMode, objType, packet)
}

func (t *tracer) Account(packet *ClientPacket) {
	t.write(AccountingMode, NoTrace, packet)
}

func (t *logTrace) Write(b []byte) (int, error) {
	return t.data.Write(b)
}

func (t *logTrace) dump() {
	log.Println(t.data.String())
}

func (l *tracer) write(mode string, objType TraceType, packet *ClientPacket) {
	go func() {
		if Disabled(mode, l.modes) {
			return
		}
		t := &logTrace{}
		writeTrace(t, mode, objType, packet, time.Now())
		t.dump()
	}()
}

func writeTrace(tracing io.Writer, mode string, objType TraceType, packet *ClientPacket, t time.Time) {
	dump := NewRequestDump(packet, mode)
	for _, m := range dump.DumpPacket(KeyValue{Key: "TraceType", Value: fmt.Sprintf("%d", objType)}) {
		tracing.Write([]byte(fmt.Sprintf("%s\n", m)))
	}
}

func (l *logger) Name() string {
	return "logger"
}

func (l *logger) Setup(ctx *ModuleContext) error {
	l.modes = DisabledModes(l, ctx)
	return nil
}

func (l *logger) Pre(packet *ClientPacket) bool {
	return NoopPre(packet, l.write)
}

func (l *logger) Post(packet *ClientPacket) bool {
	return NoopPost(packet, l.write)
}

func (l *logger) Trace(t TraceType, packet *ClientPacket) {
	l.write(TracingMode, t, packet)
}

func (l *logger) Account(packet *ClientPacket) {
	l.write(AccountingMode, NoTrace, packet)
}

func (l *logger) write(mode string, objType TraceType, packet *ClientPacket) {
	go func() {
		if Disabled(mode, l.modes) {
			return
		}
		dump := NewRequestDump(packet, mode)
		messages := dump.DumpPacket(KeyValue{Key: "Info", Value: fmt.Sprintf("%d", int(objType))})
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

func (l *umac) Pre(packet *ClientPacket) bool {
	return l.checkUserMac(packet) == nil
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
	case "debug":
		return &ModuleDebug, nil
	case "access":
		return &ModuleAccess, nil
	}
	return nil, fmt.Errorf("unknown plugin type %s", name)
}

package runner

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"layeh.com/radius/debug"
	"layeh.com/radius/rfc2865"
	"voidedtech.com/dotonex/internal/core"
)

var (
	pluginLock *sync.Mutex = new(sync.Mutex)
	pluginLogs             = []string{}
	pluginLID  int
)

type (
	// RequestDump represents the interfaces available to log/dump a request
	requestDump struct {
		data *ClientPacket
		mode string
	}

	// KeyValue represents a simple key/value object
	keyValue struct {
		key   string
		value string
	}

	keyValueStore struct {
		keyValues []keyValue
	}
)

func newRequestDump(packet *ClientPacket, mode string) *requestDump {
	return &requestDump{data: packet, mode: mode}
}

func (packet *requestDump) dumpPacket(kv keyValue) []string {
	var w bytes.Buffer
	io.WriteString(&w, fmt.Sprintf(fmt.Sprintf("Mode = %s\n", packet.mode)))
	if packet.data.ClientAddr != nil {
		io.WriteString(&w, fmt.Sprintf("UDPAddr = %s\n", packet.data.ClientAddr.String()))
	}
	conf := &debug.Config{}
	conf.Dictionary = debug.IncludedDictionary
	debug.Dump(&w, conf, packet.data.Packet)
	results := []string{kv.str()}
	for _, m := range strings.Split(w.String(), "\n") {
		if len(m) == 0 {
			continue
		}
		results = append(results, m)
	}
	return results
}

func newFile(path, instance string, appending bool) *os.File {
	flags := os.O_RDWR | os.O_CREATE
	if appending {
		flags = flags | os.O_APPEND
	}
	t := time.Now()
	inst := instance
	if len(inst) == 0 {
		inst = fmt.Sprintf("default.%d", t.UnixNano())
	}
	logPath := filepath.Join(path, fmt.Sprintf("%s.%s", inst, t.Format("2006-01-02")))
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		core.WriteError(fmt.Sprintf("unable to create file: %s", logPath), err)
		return nil
	}
	return f
}

// WritePluginMessages supports writing plugin messages to disk
func WritePluginMessages(path, instance string) {
	pluginLock.Lock()
	defer pluginLock.Unlock()
	var f *os.File
	if len(pluginLogs) == 0 {
		return
	}
	f = newFile(path, instance, true)
	if f == nil {
		return
	}
	defer f.Close()
	for _, m := range pluginLogs {
		f.Write([]byte(m))
	}
	pluginLogs = pluginLogs[:0]
	pluginLID = 0
}

func logPluginMessages(module string, messages []string) {
	pluginLock.Lock()
	defer pluginLock.Unlock()
	name := strings.ToUpper(module)
	t := time.Now().Format("2006-01-02T15:04:05.000")
	idx := pluginLID
	for _, m := range messages {
		pluginLogs = append(pluginLogs, fmt.Sprintf("%s [%s] (%d) %s\n", t, name, idx, m))
	}
	pluginLID++
}

func (kv keyValue) str() string {
	return fmt.Sprintf("%s = %s", kv.key, kv.value)
}

func moduleWrite(mode string, objType TraceType, packet *ClientPacket) {
	go func() {
		dump := newRequestDump(packet, mode)
		messages := dump.dumpPacket(keyValue{key: "Info", value: fmt.Sprintf("%d", int(objType))})
		logPluginMessages(mode, messages)
	}()
}

// AccountPacket will do accounting operations
func AccountPacket(packet *ClientPacket) {
	moduleWrite("accounting", NoTrace, packet)
}

// PrePacket performs pre-authorization checks
func PrePacket(packet *ClientPacket) bool {
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

func checkUserMac(p *ClientPacket) error {
	userName, err := rfc2865.UserName_LookupString(p.Packet)
	if err != nil {
		return err
	}
	calling, err := rfc2865.CallingStationID_LookupString(p.Packet)
	if err != nil {
		return err
	}
	calling = clean(calling)
	var failure error
	reason := ""
	cleaned, isMAC := core.CleanMAC(calling)
	if isMAC {
		// MAB case is if the calling != clean the token
		if calling != clean(userName) {
			tokenUser, token := core.GetTokenFromLogin(userName)
			if token == "" || tokenUser == "" {
				reason = "INVALIDTOKEN"
			} else {
				reason = "TOKENMACFAIL"
				if CheckTokenMAC(tokenUser, token, cleaned) {
					reason = ""
				}
			}
		}
	} else {
		reason = "INVALIDMAC"
	}
	if reason != "" {
		failure = fmt.Errorf("failed preauth: %s %s (%s)", userName, calling, reason)
	}
	go mark(reason, userName, calling, p, false)
	return failure
}

func mark(reason, user, calling string, p *ClientPacket, cached bool) {
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
	showReason := reason != ""
	if showReason {
		result = "FAILED"
	}
	kv := keyValueStore{}
	kv.add("Result", result)
	if showReason {
		kv.add("Reason", reason)
	}
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

// TracePacket for running trace of packets
func TracePacket(t TraceType, packet *ClientPacket) {
	moduleWrite("trace", t, packet)
}

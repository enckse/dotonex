package modules

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"layeh.com/radius/debug"
	"voidedtech.com/dotonex/internal/core"
	"voidedtech.com/dotonex/internal/op"
)

var (
	pluginLock *sync.Mutex = new(sync.Mutex)
	pluginLogs             = []string{}
	pluginLID  int
)

type (
	// RequestDump represents the interfaces available to log/dump a request
	requestDump struct {
		data *op.ClientPacket
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

func newRequestDump(packet *op.ClientPacket, mode string) *requestDump {
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

func moduleWrite(mode string, objType op.TraceType, packet *op.ClientPacket) {
	go func() {
		dump := newRequestDump(packet, mode)
		messages := dump.dumpPacket(keyValue{key: "Info", value: fmt.Sprintf("%d", int(objType))})
		logPluginMessages(mode, messages)
	}()
}

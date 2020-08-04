package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/debug"
	"voidedtech.com/radiucal/internal/core"
)

const (
	// PreProcess is the "pre" packet processing before full EAP/RADIUS
	PreProcess ModuleMode = 1
	// PostProcess is the "post" packet processing after full EAP/RADIUS
	PostProcess ModuleMode = 2
	// AccountingProcess is for accounting processing
	AccountingProcess ModuleMode = 3
)

var (
	moduleLock *sync.Mutex = new(sync.Mutex)
	moduleLogs             = []string{}
	moduleLID  int
)

type (
	// RequestDump represents the interfaces available to log/dump a request
	RequestDump struct {
		data *ClientPacket
		mode string
	}

	// ModuleMode is the processing mode for the packet (e.g. pre, post, accounting)
	ModuleMode int

	// ModuleContext is the context given to a module
	ModuleContext struct {
		// Backing config
		config *Configuration
	}

	// Module represents a module module for packet checking
	Module interface {
		Setup(*ModuleContext) error
		Name() string
		Process(*ClientPacket, ModuleMode) bool
	}

	// ClientPacket represents the radius packet from the client
	ClientPacket struct {
		ClientAddr *net.UDPAddr
		Buffer     []byte
		Packet     *radius.Packet
		Error      error
	}

	// KeyValue represents a simple key/value object
	KeyValue struct {
		Key   string
		Value string
	}

	// KeyValueStore represents a store of KeyValue objects
	KeyValueStore struct {
		KeyValues []KeyValue
		DropEmpty bool
	}
)

// NewClientPacket creates a client packet from an input data packet
func NewClientPacket(buffer []byte, addr *net.UDPAddr) *ClientPacket {
	return &ClientPacket{Buffer: buffer, ClientAddr: addr}
}

// NewModuleContext prepares a context from a configuration
func NewModuleContext(config *Configuration) *ModuleContext {
	p := &ModuleContext{}
	p.config = config
	return p
}

// NewRequestDump prepares a packet request for dumping
func NewRequestDump(packet *ClientPacket, mode string) *RequestDump {
	return &RequestDump{data: packet, mode: mode}
}

// DumpPacket dumps packet information to a string array of outputs
func (packet *RequestDump) DumpPacket(kv KeyValue) []string {
	var w bytes.Buffer
	io.WriteString(&w, fmt.Sprintf(fmt.Sprintf("Mode = %s\n", packet.mode)))
	if packet.data.ClientAddr != nil {
		io.WriteString(&w, fmt.Sprintf("UDPAddr = %s\n", packet.data.ClientAddr.String()))
	}
	conf := &debug.Config{}
	conf.Dictionary = debug.IncludedDictionary
	debug.Dump(&w, conf, packet.data.Packet)
	results := []string{kv.String()}
	for _, m := range strings.Split(w.String(), "\n") {
		if len(m) == 0 {
			continue
		}
		results = append(results, m)
	}
	return results
}

func newFile(path string, appending bool) *os.File {
	flags := os.O_RDWR | os.O_CREATE
	if appending {
		flags = flags | os.O_APPEND
	}
	t := time.Now()
	logPath := filepath.Join(path, t.Format("2006-01-02"))
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		core.WriteError(fmt.Sprintf("unable to create file: %s", logPath), err)
		return nil
	}
	return f
}

// WriteModuleMessages supports writing module messages to disk
func WriteModuleMessages(path string) {
	moduleLock.Lock()
	defer moduleLock.Unlock()
	var f *os.File
	if len(moduleLogs) == 0 {
		return
	}
	f = newFile(path, true)
	if f == nil {
		return
	}
	defer f.Close()
	for _, m := range moduleLogs {
		f.Write([]byte(m))
	}
	moduleLogs = moduleLogs[:0]
	moduleLID = 0
}

// LogModuleMessages adds messages to the module log queue
func LogModuleMessages(mod Module, messages []string) {
	moduleLock.Lock()
	defer moduleLock.Unlock()
	name := strings.ToUpper(mod.Name())
	t := time.Now().Format("2006-01-02T15:04:05.000")
	idx := moduleLID
	for _, m := range messages {
		moduleLogs = append(moduleLogs, fmt.Sprintf("%s [%s] (%d) %s\n", t, name, idx, m))
	}
	moduleLID++
}

// Add adds a key value object to the store
func (kv *KeyValueStore) Add(key, val string) {
	kv.KeyValues = append(kv.KeyValues, KeyValue{Key: key, Value: val})
}

// String converts the KeyValue to a string representation
func (kv KeyValue) String() string {
	return fmt.Sprintf("%s = %s", kv.Key, kv.Value)
}

// Strings gets all strings from a store
func (kv KeyValueStore) Strings() []string {
	var objs []string
	offset := ""
	for _, k := range kv.KeyValues {
		if kv.DropEmpty && len(k.Value) == 0 {
			continue
		}
		objs = append(objs, fmt.Sprintf("%s%s", offset, k.String()))
		offset = "  "
	}
	return objs
}

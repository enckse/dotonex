package core

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
)

const (
	// NoTrace indicates no tracing to occur
	NoTrace TraceType = 0
	// TraceRequest indicate to trace the request
	TraceRequest TraceType = 1
	// AccountingMode for accounting
	AccountingMode = "accounting"
	// TracingMode for tracing
	TracingMode = "trace"
	// PreAuthMode for pre-auth
	PreAuthMode = "preauth"
	// PostAuthMode for post-auth
	PostAuthMode = "postauth"
)

var (
	pluginLock *sync.Mutex = new(sync.Mutex)
	pluginLogs             = []string{}
	pluginLID  int
)

type (
	// RequestDump represents the interfaces available to log/dump a request
	RequestDump struct {
		data *ClientPacket
		mode string
	}
	// TraceType indicates how to trace a request
	TraceType int

	// NoopCall represents module calls that perform no operation (mocks)
	NoopCall func(string, TraceType, *ClientPacket)

	// PluginContext is the context given to a plugin module
	PluginContext struct {
		// Backing config
		config *Configuration
		// Lib represents the library path for radiucal
		Lib string
	}

	// Module represents a plugin module for packet checking
	Module interface {
		Setup(*PluginContext) error
		Name() string
	}

	// PreAuth represents the interface required to pre-authorize a packet
	PreAuth interface {
		Module
		Pre(*ClientPacket) bool
	}

	// PostAuth represents the interface required to post-authorize a packet
	PostAuth interface {
		Module
		Post(*ClientPacket) bool
	}

	// Tracing represents the interface required to trace requests
	Tracing interface {
		Module
		Trace(TraceType, *ClientPacket)
	}

	// Accounting represents the interface required to handle accounting
	Accounting interface {
		Module
		Account(*ClientPacket)
	}

	// ClientPacket represents the radius packet from the client
	ClientPacket struct {
		ClientAddr *net.UDPAddr
		Buffer     []byte
		Packet     *radius.Packet
		Error      error
	}
)

// NewClientPacket creates a client packet from an input data packet
func NewClientPacket(buffer []byte, addr *net.UDPAddr) *ClientPacket {
	return &ClientPacket{Buffer: buffer, ClientAddr: addr}
}

// NewPluginContext prepares a context from a configuration
func NewPluginContext(config *Configuration) *PluginContext {
	p := &PluginContext{}
	p.config = config
	p.Lib = config.Dir
	return p
}

// CloneContext a plugin context to a copy for use in other plugins
func (p *PluginContext) CloneContext() *PluginContext {
	return NewPluginContext(p.config)
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

// Disabled indicates if a given mode is disabled
func Disabled(mode string, modes []string) bool {
	if len(modes) == 0 {
		return false
	}
	for _, m := range modes {
		if m == mode {
			return true
		}
	}
	return false
}

// NoopPost is a no-operation post authorization call
func NoopPost(packet *ClientPacket, call NoopCall) bool {
	return noopAuth(PostAuthMode, packet, call)
}

// NoopPre is a no-operation pre authorization call
func NoopPre(packet *ClientPacket, call NoopCall) bool {
	return noopAuth(PreAuthMode, packet, call)
}

func noopAuth(mode string, packet *ClientPacket, call NoopCall) bool {
	call(mode, NoTrace, packet)
	return true
}

func isFlagged(list []string, name string) bool {
	for _, v := range list {
		if name == v {
			return true
		}
	}
	return false
}

// DisabledModes returns the modes disable for module
func DisabledModes(m Module, ctx *PluginContext) []string {
	name := m.Name()
	noAccounting := isFlagged(ctx.config.Disable.Accounting, name)
	noTracing := isFlagged(ctx.config.Disable.Trace, name)
	noPreauth := isFlagged(ctx.config.Disable.Preauth, name)
	noPostauth := isFlagged(ctx.config.Disable.Postauth, name)
	var modes []string
	if noAccounting {
		modes = append(modes, AccountingMode)
	}
	if noTracing {
		modes = append(modes, TracingMode)
	}
	if noPreauth {
		modes = append(modes, PreAuthMode)
	}
	if noPostauth {
		modes = append(modes, PostAuthMode)
	}
	return modes
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
		WriteError(fmt.Sprintf("unable to create file: %s", logPath), err)
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

// LogPluginMessages adds messages to the plugin log queue
func LogPluginMessages(mod Module, messages []string) {
	pluginLock.Lock()
	defer pluginLock.Unlock()
	name := strings.ToUpper(mod.Name())
	t := time.Now().Format("2006-01-02T15:04:05.000")
	idx := pluginLID
	for _, m := range messages {
		pluginLogs = append(pluginLogs, fmt.Sprintf("%s [%s] (%d) %s\n", t, name, idx, m))
	}
	pluginLID++
}

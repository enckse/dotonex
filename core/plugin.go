package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"

	"layeh.com/radius/debug"
)

const (
	AccountingMode = "accounting"
	TracingMode    = "trace"
	PreAuthMode    = "preauth"
	PostAuthMode   = "postauth"
)

var (
	pluginLock *sync.Mutex = new(sync.Mutex)
	pluginLogs             = []string{}
)

type TraceType int

type NoopCall func(string, TraceType, *ClientPacket)

const (
	NoTrace      TraceType = iota
	TraceRequest TraceType = iota
)

type PluginContext struct {
	// Location of logs directory
	Logs string
	// Location of the general lib directory
	Lib string
	// Backing config
	config *Configuration
	// Instance name
	Instance string
	// Enable caching
	Cache bool
	// Backing configuration data
	Backing []byte
}

type Module interface {
	Reload()
	Setup(*PluginContext) error
	Name() string
}

type PreAuth interface {
	Module
	Pre(*ClientPacket) bool
}

type PostAuth interface {
	Module
	Post(*ClientPacket) bool
}

type Tracing interface {
	Module
	Trace(TraceType, *ClientPacket)
}

type Accounting interface {
	Module
	Account(*ClientPacket)
}

func NewPluginContext(config *Configuration) *PluginContext {
	p := &PluginContext{}
	p.Cache = config.Cache
	p.config = config
	p.Backing = config.backing
	return p
}

func (p *PluginContext) clone(moduleName string) *PluginContext {
	n := &PluginContext{}
	n.Logs = p.Logs
	n.Lib = p.Lib
	n.Instance = p.Instance
	n.Cache = p.Cache
	n.config = p.config
	n.Backing = p.Backing
	return n
}

func (p *PluginContext) GetBackingConfig() []byte {
	return p.config.backing
}

type requestDump struct {
	data *ClientPacket
	mode string
}

func NewRequestDump(packet *ClientPacket, mode string) *requestDump {
	return &requestDump{data: packet, mode: mode}
}

func (packet *requestDump) DumpPacket(header string) []string {
	var w bytes.Buffer
	io.WriteString(&w, fmt.Sprintf(fmt.Sprintf("Mode = %s\n", packet.mode)))
	if packet.data.ClientAddr != nil {
		io.WriteString(&w, fmt.Sprintf("UDPAddr = %s\n", packet.data.ClientAddr.String()))
	}
	conf := &debug.Config{}
	conf.Dictionary = debug.IncludedDictionary
	debug.Dump(&w, conf, packet.data.Packet)
	results := []string{header}
	for _, m := range strings.Split(w.String(), "\n") {
		if len(m) == 0 {
			continue
		}
		results = append(results, m)
	}
	return results
}

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

func NoopPost(packet *ClientPacket, call NoopCall) bool {
	return noopAuth(PostAuthMode, packet, call)
}

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
		inst = "default"
	}
	logPath := filepath.Join(path, fmt.Sprintf("%s.%s", inst, t.Format("2006-01-02")))
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		WriteError(fmt.Sprintf("unable to create file: %s", logPath), err)
		return nil
	}
	return f
}

func LoadPlugin(path string, ctx *PluginContext) (Module, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	v, err := p.Lookup("Plugin")
	if err != nil {
		return nil, err
	}
	var mod Module
	mod, ok := v.(Module)
	if !ok {
		return nil, errors.New(fmt.Sprintf("unable to load plugin %s", path))
	}
	err = mod.Setup(ctx.clone(mod.Name()))
	if err != nil {
		return nil, err
	}
	return mod, nil
	switch t := mod.(type) {
	default:
		return nil, errors.New(fmt.Sprintf("unknown type: %T", t))
	case Accounting:
		return t.(Accounting), nil
	case PreAuth:
		return t.(PreAuth), nil
	case Tracing:
		return t.(Tracing), nil
	case PostAuth:
		return t.(PostAuth), nil
	}
}

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
	for _, m := range pluginLogs {
		f.Write([]byte(m))
	}
	pluginLogs = pluginLogs[:0]
}

func LogPluginMessages(mod Module, messages []string) {
	pluginLock.Lock()
	defer pluginLock.Unlock()
	name := mod.Name()
	t := time.Now().Format("2006-01-02T15:04:05")
	for _, m := range messages {
		pluginLogs = append(pluginLogs, fmt.Sprintf("%s [%s] %s\n", t, name, m))
	}
}

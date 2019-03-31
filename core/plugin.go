package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"time"

	"layeh.com/radius/debug"
	"voidedtech.com/goutils/logger"
	"voidedtech.com/goutils/preyaml"
)

const (
	AccountingMode = "accounting"
	TracingMode    = "trace"
	PreAuthMode    = "preauth"
	PostAuthMode   = "postauth"
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
	return p
}

func (p *PluginContext) clone(moduleName string) *PluginContext {
	n := &PluginContext{}
	n.Logs = p.Logs
	n.Lib = p.Lib
	n.Instance = p.Instance
	n.Cache = p.Cache
	n.config = p.config
	return n
}

func (p *PluginContext) SetupBackingConfig() ([]byte, *preyaml.Directives) {
	d := &preyaml.Directives{}
	return p.config.backing, d
}

type requestDump struct {
	data  *ClientPacket
	mode  string
	stamp time.Time
}

func NewRequestDump(packet *ClientPacket, mode string, timestamp time.Time) *requestDump {
	return &requestDump{data: packet, mode: mode, stamp: timestamp}
}

func (packet *requestDump) DumpPacket(w io.Writer) {
	io.WriteString(w, fmt.Sprintf(fmt.Sprintf("Mode = %s (%s)\n", packet.mode, packet.stamp)))
	if packet.data.ClientAddr != nil {
		io.WriteString(w, fmt.Sprintf("UDPAddr = %s\n", packet.data.ClientAddr.String()))
	}
	conf := &debug.Config{}
	conf.Dictionary = debug.IncludedDictionary
	debug.Dump(w, conf, packet.data.Packet)
}

func DatedAppendFile(path, name, instance string) (*os.File, time.Time) {
	return newFile(path, name, instance, true)
}

func NewFilePath(path, name, instance string) (string, time.Time) {
	t := time.Now()
	inst := instance
	if len(inst) > 0 {
		inst = fmt.Sprintf("%s.", inst)
	}
	logPath := filepath.Join(path, fmt.Sprintf("%s%s.%s", inst, name, t.Format("2006-01-02")))
	return logPath, t
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

func newFile(path, name, instance string, appending bool) (*os.File, time.Time) {
	flags := os.O_RDWR | os.O_CREATE
	if appending {
		flags = flags | os.O_APPEND
	}
	logPath, t := NewFilePath(path, name, instance)
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		logger.WriteError(fmt.Sprintf("unable to create file: %s", logPath), err)
		return nil, t
	}
	return f, t
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

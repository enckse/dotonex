package core

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"time"

	"github.com/epiphyte/goutils"
	"layeh.com/radius/debug"
)

const (
	AccountingMode = "accounting"
	TracingMode    = "trace"
	PreAuthMode    = "preauth"
	PostAuthMode   = "postauth"
)

type TraceType int

const (
	NoTrace      TraceType = iota
	TraceRequest TraceType = iota
)

type PluginContext struct {
	// Location of logs directory
	Logs string
	// Location of the general lib directory
	Lib string
	// Plugin section (subsection of config)
	Section *goutils.Config
	// Backing config
	config *goutils.Config
	// Instance name
	Instance string
	// Enable caching
	Cache bool
}

type Module interface {
	Reload()
	Setup(*PluginContext)
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

func NewPluginContext(config *goutils.Config) *PluginContext {
	p := &PluginContext{}
	p.Cache = config.GetTrue("cache")
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
	n.Section = p.config.GetSection(fmt.Sprintf("[%s]", moduleName))
	return n
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
		inst = fmt.Sprintf(".%s", inst)
	}
	logPath := filepath.Join(path, fmt.Sprintf("radiucal%s.%s.%s", inst, name, t.Format("2006-01-02")))
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

func DisabledModes(m Module, ctx *PluginContext) []string {
	name := m.Name()
	accounting := ctx.config.GetTrue(fmt.Sprintf("%s_disable_accounting", name))
	tracing := ctx.config.GetTrue(fmt.Sprintf("%s_disable_trace", name))
	preauth := ctx.config.GetTrue(fmt.Sprintf("%s_disable_preauth", name))
	postauth := ctx.config.GetTrue(fmt.Sprintf("%s_disable_postauth", name))
	var modes []string
	if accounting {
		modes = append(modes, AccountingMode)
	}
	if tracing {
		modes = append(modes, TracingMode)
	}
	if preauth {
		modes = append(modes, PreAuthMode)
	}
	if postauth {
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
		goutils.WriteError(fmt.Sprintf("unable to create file: %s", logPath), err)
		return nil, t
	}
	return f, t
}

func FormatLog(f *os.File, t time.Time, indicator, message string) {
	f.Write([]byte(fmt.Sprintf("%s [%s] %s\n", t.Format("2006-01-02T15:04:05"), strings.ToUpper(indicator), message)))
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
	mod.Setup(ctx.clone(mod.Name()))
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

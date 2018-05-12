package plugins

import (
	"errors"
	"fmt"
	"github.com/epiphyte/goutils"
	"layeh.com/radius"
	"layeh.com/radius/debug"
	. "layeh.com/radius/rfc2865"
	"net"
	"os"
	"path/filepath"
	"plugin"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	AccountingMode = "accounting"
	AuthingMode    = "auth"
	PreAuthMode    = "preauth"
)

type ClientPacket struct {
	ClientAddr *net.UDPAddr
	Buffer     []byte
	Packet     *radius.Packet
}

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

type Authing interface {
	Module
	Auth(*ClientPacket)
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

func NewClientPacket(buffer []byte, addr *net.UDPAddr) *ClientPacket {
	return &ClientPacket{Buffer: buffer, ClientAddr: addr}
}

// Get attributes as Type/Value string arrays
func KeyValueStrings(packet *ClientPacket) []string {
	var datum []string
	if packet.ClientAddr != nil {
		datum = append(datum, fmt.Sprintf("UDPAddr: %s", packet.ClientAddr.String()))
	}
	for t, a := range packet.Packet.Attributes {
		name := resolveType(t)
		datum = append(datum, fmt.Sprintf("Type: %d (%s)", t, name))
		for _, s := range a {
			unknown := true
			val := ""
			if t == NASIPAddress_Type {
				ip, err := radius.IPAddr(s)
				if err == nil {
					unknown = false
					val = fmt.Sprintf("(ip) %s", ip.String())
				}
			}

			if unknown {
				i, err := radius.Integer(s)
				if err == nil {
					unknown = false
					val = fmt.Sprintf("(int) %d", i)
				}
			}

			if unknown {
				d, err := radius.Date(s)
				if err == nil {
					unknown = false
					val = fmt.Sprintf("(time) %s", d.Format(time.RFC3339))
				}
			}

			if unknown {
				val = string(s)
				unknown = false
				for _, c := range val {
					if !unicode.IsPrint(c) {
						unknown = true
						break
					}
				}
			}

			if unknown {
				val = fmt.Sprintf("(hex) %x", s)
			}
			datum = append(datum, fmt.Sprintf("Value: %s", val))
		}
	}
	return datum
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
	authing := ctx.config.GetTrue(fmt.Sprintf("%s_disable_auth", name))
	preauth := ctx.config.GetTrue(fmt.Sprintf("%s_disable_preauth", name))
	var modes []string
	if accounting {
		modes = append(modes, AccountingMode)
	}
	if authing {
		modes = append(modes, AuthingMode)
	}
	if preauth {
		modes = append(modes, PreAuthMode)
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
	case Authing:
		return t.(Authing), nil
	}
}

func resolveType(t radius.Type) string {
	attr := debug.IncludedDictionary.AttributeByOID(strconv.Itoa(int(t)))
	if attr == nil {
		return "Unknown"
	}
	return attr.Name
}

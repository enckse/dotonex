package stats

import (
	"fmt"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	"voidedtech.com/radiucal/internal/core"
)

const (
	timeFormat = "2006-01-02T15:04:05"
)

// Config represents the configuration for the stats plugin
type Config struct {
	Stats struct {
		Flush int
	}
}

type modedata struct {
	first time.Time
	last  time.Time
	name  string
	count int
}

var (
	lock *sync.Mutex = new(sync.Mutex)
	// Plugin represents the instance for the system
	Plugin   stats
	info     = make(map[string]*modedata)
	modes    []string
	instance string
	flush    int
	flushIdx int
)

type stats struct {
}

func (s *stats) Name() string {
	return "stats"
}

func (s *stats) Reload() {
	lock.Lock()
	defer lock.Unlock()
	info = make(map[string]*modedata)
}

func (s *stats) Setup(ctx *core.PluginContext) error {
	conf := &Config{}
	err := yaml.Unmarshal(ctx.Backing, conf)
	if err != nil {
		return err
	}
	flush = conf.Stats.Flush
	if flush < 0 {
		flush = 0
	}
	instance = ctx.Instance
	modes = core.DisabledModes(s, ctx)
	return nil
}

func (s *stats) Post(packet *core.ClientPacket) bool {
	return core.NoopPost(packet, write)
}

func (s *stats) Pre(packet *core.ClientPacket) bool {
	return core.NoopPre(packet, write)
}

func (s *stats) Trace(t core.TraceType, packet *core.ClientPacket) {
	write(core.TracingMode, t, nil)
}

func (s *stats) Account(packet *core.ClientPacket) {
	write(core.AccountingMode, core.NoTrace, nil)
}

func write(mode string, objType core.TraceType, packet *core.ClientPacket) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		if core.Disabled(mode, modes) {
			return
		}
		key := fmt.Sprintf("%s.%d", mode, int(objType))
		t := time.Now()
		if _, ok := info[key]; !ok {
			info[key] = &modedata{first: t, count: 0, name: key}
		}
		m, _ := info[key]
		m.last = t
		m.count++
		if flush == 0 || flushIdx > flush {
			flushIdx = 0
			kv := core.KeyValueStore{}
			kv.Add("Time", t.Format(timeFormat))
			kv.Add("First", m.first.Format(timeFormat))
			kv.Add("Last", m.first.Format(timeFormat))
			kv.Add("Count", fmt.Sprintf("%d", m.count))
			kv.Add("Name", m.name)
			core.LogPluginMessages(&Plugin, kv.Strings())
		} else {
			flushIdx++
		}
	}()
}

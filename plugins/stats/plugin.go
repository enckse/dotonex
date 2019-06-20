package main

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"voidedtech.com/radiucal/core"
)

type modedata struct {
	first time.Time
	last  time.Time
	name  string
	count int
}

func (m *modedata) String() string {
	return fmt.Sprintf("first: %s\nlast: %s\ncount: %d\nname: %s\n",
		m.first.Format("2006-01-02T15:04:05"),
		m.last.Format("2006-01-02T15:04:05"),
		m.count,
		m.name)
}

var (
	lock     *sync.Mutex = new(sync.Mutex)
	dir      string
	Plugin   stats
	info     map[string]*modedata = make(map[string]*modedata)
	modes    []string
	instance string
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
	dir = ctx.Logs
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
		f, t := core.NewFilePath(dir, fmt.Sprintf("stats.%s", key), instance)
		if _, ok := info[key]; !ok {
			info[key] = &modedata{first: t, count: 0, name: key}
		}
		m, _ := info[key]
		m.last = t
		m.count++
		ioutil.WriteFile(f, []byte(m.String()), 0644)
	}()
}

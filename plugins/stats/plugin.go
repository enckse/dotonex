package main

import (
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/epiphyte/radiucal/plugins"
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

func (s *stats) Setup(ctx *plugins.PluginContext) {
	dir = ctx.Logs
	instance = ctx.Instance
	modes = plugins.DisabledModes(s, ctx)
}

func (s *stats) Pre(packet *plugins.ClientPacket) bool {
	write(plugins.PreAuthMode, plugins.NotAuth)
	return true
}

func (s *stats) Auth(t plugins.AuthType, packet *plugins.ClientPacket) {
	write(plugins.AuthingMode, t)
}

func (s *stats) Account(packet *plugins.ClientPacket) {
	write(plugins.AccountingMode, plugins.NotAuth)
}

func write(mode string, objType plugins.AuthType) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		if plugins.Disabled(mode, modes) {
			return
		}
		key := fmt.Sprintf("%s.%d", mode, int(objType))
		f, t := plugins.NewFilePath(dir, fmt.Sprintf("stats.%s", key), instance)
		if _, ok := info[key]; !ok {
			info[key] = &modedata{first: t, count: 0, name: key}
		}
		m, _ := info[key]
		m.last = t
		m.count++
		ioutil.WriteFile(f, []byte(m.String()), 0644)
	}()
}

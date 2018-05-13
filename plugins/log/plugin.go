package main

import (
	"fmt"
	"sync"

	"github.com/epiphyte/radiucal/plugins"
)

var (
	lock     *sync.Mutex = new(sync.Mutex)
	logs     string
	Plugin   logger
	modes    []string
	instance string
)

type logger struct {
}

func (l *logger) Name() string {
	return "logger"
}

func (l *logger) Reload() {
}

func (l *logger) Setup(ctx *plugins.PluginContext) {
	logs = ctx.Logs
	modes = plugins.DisabledModes(l, ctx)
	instance = ctx.Instance
}

func (l *logger) Pre(packet *plugins.ClientPacket) bool {
	write(plugins.PreAuthMode, packet)
	return true
}

func (l *logger) Auth(packet *plugins.ClientPacket) {
	write(plugins.AuthingMode, packet)
}

func (l *logger) Account(packet *plugins.ClientPacket) {
	write(plugins.AccountingMode, packet)
}

func write(mode string, packet *plugins.ClientPacket) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		if plugins.Disabled(mode, modes) {
			return
		}
		f, t := plugins.DatedAppendFile(logs, mode, instance)
		if f == nil {
			return
		}
		f.Write([]byte(fmt.Sprintf("id -> %s (%s)\n", mode, t)))
		plugins.DumpPacket(packet, f)
	}()
}

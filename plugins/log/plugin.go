package main

import (
	"fmt"
	"sync"

	"voidedtech.com/radiucal/core"
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

func (l *logger) Setup(ctx *core.PluginContext) {
	logs = ctx.Logs
	modes = core.DisabledModes(l, ctx)
	instance = ctx.Instance
}

func (l *logger) Pre(packet *core.ClientPacket) bool {
	return core.NoopPre(packet, write)
}

func (l *logger) Post(packet *core.ClientPacket) bool {
	return core.NoopPost(packet, write)
}

func (l *logger) Trace(t core.TraceType, packet *core.ClientPacket) {
	write(core.TracingMode, t, packet)
}

func (l *logger) Account(packet *core.ClientPacket) {
	write(core.AccountingMode, core.NoTrace, packet)
}

func write(mode string, objType core.TraceType, packet *core.ClientPacket) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		if core.Disabled(mode, modes) {
			return
		}
		f, t := core.DatedAppendFile(logs, mode, instance)
		if f == nil {
			return
		}
		defer f.Close()
		f.Write([]byte(fmt.Sprintf("id -> %s %d (%s)\n", mode, int(objType), t)))
		dump := core.NewRequestDump(packet, mode, t)
		dump.DumpPacket(f)
	}()
}

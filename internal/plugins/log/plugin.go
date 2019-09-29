package log

import (
	"fmt"

	"voidedtech.com/radiucal/internal/core"
)

var (
	// Plugin represents the system instance of the module
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

func (l *logger) Setup(ctx *core.PluginContext) error {
	modes = core.DisabledModes(l, ctx)
	instance = ctx.Instance
	return nil
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
		if core.Disabled(mode, modes) {
			return
		}
		dump := core.NewRequestDump(packet, mode)
		messages := dump.DumpPacket(core.KeyValue{Key: "Info", Value: fmt.Sprintf("%d", int(objType))})
		core.LogPluginMessages(&Plugin, messages)
	}()
}

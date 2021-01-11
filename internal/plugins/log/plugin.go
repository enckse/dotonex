package log

import (
	"fmt"

	"voidedtech.com/radiucal/internal"
)

var (
	// Plugin represents the system instance of the module
	Plugin logger
	modes  []string
)

type (
	logger struct {
	}
)

func (l *logger) Name() string {
	return "logger"
}

func (l *logger) Setup(ctx *internal.PluginContext) error {
	modes = internal.DisabledModes(l, ctx)
	return nil
}

func (l *logger) Pre(packet *internal.ClientPacket) bool {
	return internal.NoopPre(packet, write)
}

func (l *logger) Trace(t internal.TraceType, packet *internal.ClientPacket) {
	write(internal.TracingMode, t, packet)
}

func (l *logger) Account(packet *internal.ClientPacket) {
	write(internal.AccountingMode, internal.NoTrace, packet)
}

func write(mode string, objType internal.TraceType, packet *internal.ClientPacket) {
	go func() {
		if internal.Disabled(mode, modes) {
			return
		}
		dump := internal.NewRequestDump(packet, mode)
		messages := dump.DumpPacket(internal.KeyValue{Key: "Info", Value: fmt.Sprintf("%d", int(objType))})
		internal.LogPluginMessages(&Plugin, messages)
	}()
}

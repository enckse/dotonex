package log

import (
	"fmt"

	"voidedtech.com/radiucal/internal/server"
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

func (l *logger) Setup(ctx *server.PluginContext) error {
	modes = server.DisabledModes(l, ctx)
	return nil
}

func (l *logger) Pre(packet *server.ClientPacket) bool {
	return server.NoopPre(packet, write)
}

func (l *logger) Post(packet *server.ClientPacket) bool {
	return server.NoopPost(packet, write)
}

func (l *logger) Trace(t server.TraceType, packet *server.ClientPacket) {
	write(server.TracingMode, t, packet)
}

func (l *logger) Account(packet *server.ClientPacket) {
	write(server.AccountingMode, server.NoTrace, packet)
}

func write(mode string, objType server.TraceType, packet *server.ClientPacket) {
	go func() {
		if server.Disabled(mode, modes) {
			return
		}
		dump := server.NewRequestDump(packet, mode)
		messages := dump.DumpPacket(server.KeyValue{Key: "Info", Value: fmt.Sprintf("%d", int(objType))})
		server.LogPluginMessages(&Plugin, messages)
	}()
}

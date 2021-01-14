package modules

import (
	"fmt"

	"voidedtech.com/dotonex/internal"
)

var (
	// AccountingModule is an instance of the module
	AccountingModule accountingModule
)

type (
	accountingModule struct {
	}
)

func (l *accountingModule) Name() string {
	return "accounting"
}

func (l *accountingModule) Setup(ctx *internal.PluginContext) error {
	return nil
}

func (l *accountingModule) Trace(t internal.TraceType, packet *internal.ClientPacket) {
	write("tracing", t, packet)
}

func (l *accountingModule) Account(packet *internal.ClientPacket) {
	write("accounting", internal.NoTrace, packet)
}

func write(mode string, objType internal.TraceType, packet *internal.ClientPacket) {
	go func() {
		dump := internal.NewRequestDump(packet, mode)
		messages := dump.DumpPacket(internal.KeyValue{Key: "Info", Value: fmt.Sprintf("%d", int(objType))})
		internal.LogPluginMessages(&AccountingModule, messages)
	}()
}

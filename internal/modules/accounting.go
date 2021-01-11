package modules

import (
	"fmt"

	"voidedtech.com/radiucal/internal"
)

var (
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
	write(internal.TracingMode, t, packet)
}

func (l *accountingModule) Account(packet *internal.ClientPacket) {
	write(internal.AccountingMode, internal.NoTrace, packet)
}

func write(mode string, objType internal.TraceType, packet *internal.ClientPacket) {
	go func() {
		dump := internal.NewRequestDump(packet, mode)
		messages := dump.DumpPacket(internal.KeyValue{Key: "Info", Value: fmt.Sprintf("%d", int(objType))})
		internal.LogPluginMessages(&AccountingModule, messages)
	}()
}

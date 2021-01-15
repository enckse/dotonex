package modules

import (
	"voidedtech.com/dotonex/internal/op"
)

type (
	TraceModule struct {
	}
)

func (l *TraceModule) Trace(t op.TraceType, packet *op.ClientPacket) {
	moduleWrite("trace", t, packet)
}

func (l *TraceModule) Name() string {
	return "trace"
}

package modules

import (
	"voidedtech.com/dotonex/internal/op"
)

type (
	// TraceModule for tracing requests
	TraceModule struct {
	}
)

// Trace for running trace of packets
func (l *TraceModule) Trace(t op.TraceType, packet *op.ClientPacket) {
	moduleWrite("trace", t, packet)
}

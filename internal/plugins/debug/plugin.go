package debug

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"voidedtech.com/radiucal/internal/core"
)

type (
	tracer struct {
	}

	logTrace struct {
		io.Writer
		data bytes.Buffer
	}
)

var (
	// Plugin represents the system instance of the module
	Plugin tracer
	modes  []string
)

func (t *tracer) Name() string {
	return "debugger"
}

func (t *tracer) Unload() {
}

func (t *tracer) Setup(ctx *core.PluginContext) error {
	modes = core.DisabledModes(t, ctx)
	return nil
}

func (t *tracer) Pre(packet *core.ClientPacket) bool {
	return core.NoopPre(packet, dump)
}

func (t *tracer) Post(packet *core.ClientPacket) bool {
	return core.NoopPost(packet, dump)
}

func (t *tracer) Trace(objType core.TraceType, packet *core.ClientPacket) {
	dump(core.TracingMode, objType, packet)
}

func (t *tracer) Account(packet *core.ClientPacket) {
	dump(core.AccountingMode, core.NoTrace, packet)
}

func (t *logTrace) Write(b []byte) (int, error) {
	return t.data.Write(b)
}

func (t *logTrace) dump() {
	log.Println(t.data.String())
}

func dump(mode string, objType core.TraceType, packet *core.ClientPacket) {
	go func() {
		if core.Disabled(mode, modes) {
			return
		}
		t := &logTrace{}
		write(t, mode, objType, packet, time.Now())
		t.dump()
	}()
}

func write(tracing io.Writer, mode string, objType core.TraceType, packet *core.ClientPacket, t time.Time) {
	dump := core.NewRequestDump(packet, mode)
	for _, m := range dump.DumpPacket(core.KeyValue{Key: "TraceType", Value: fmt.Sprintf("%d", objType)}) {
		tracing.Write([]byte(fmt.Sprintf("%s\n", m)))
	}
}

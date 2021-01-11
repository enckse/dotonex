package debug

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"voidedtech.com/radiucal/internal"
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

func (t *tracer) Setup(ctx *internal.PluginContext) error {
	modes = internal.DisabledModes(t, ctx)
	return nil
}

func (t *tracer) Pre(packet *internal.ClientPacket) bool {
	return internal.NoopPre(packet, dump)
}

func (t *tracer) Post(packet *internal.ClientPacket) bool {
	return internal.NoopPost(packet, dump)
}

func (t *tracer) Trace(objType internal.TraceType, packet *internal.ClientPacket) {
	dump(internal.TracingMode, objType, packet)
}

func (t *tracer) Account(packet *internal.ClientPacket) {
	dump(internal.AccountingMode, internal.NoTrace, packet)
}

func (t *logTrace) Write(b []byte) (int, error) {
	return t.data.Write(b)
}

func (t *logTrace) dump() {
	log.Println(t.data.String())
}

func dump(mode string, objType internal.TraceType, packet *internal.ClientPacket) {
	go func() {
		if internal.Disabled(mode, modes) {
			return
		}
		t := &logTrace{}
		write(t, mode, objType, packet, time.Now())
		t.dump()
	}()
}

func write(tracing io.Writer, mode string, objType internal.TraceType, packet *internal.ClientPacket, t time.Time) {
	dump := internal.NewRequestDump(packet, mode)
	for _, m := range dump.DumpPacket(internal.KeyValue{Key: "TraceType", Value: fmt.Sprintf("%d", objType)}) {
		tracing.Write([]byte(fmt.Sprintf("%s\n", m)))
	}
}

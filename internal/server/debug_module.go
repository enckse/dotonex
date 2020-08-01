package debug

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"voidedtech.com/radiucal/internal/server"
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

func (t *tracer) Setup(ctx *server.PluginContext) error {
	modes = server.DisabledModes(t, ctx)
	return nil
}

func (t *tracer) Pre(packet *server.ClientPacket) bool {
	return server.NoopPre(packet, dump)
}

func (t *tracer) Post(packet *server.ClientPacket) bool {
	return server.NoopPost(packet, dump)
}

func (t *tracer) Trace(objType server.TraceType, packet *server.ClientPacket) {
	dump(server.TracingMode, objType, packet)
}

func (t *tracer) Account(packet *server.ClientPacket) {
	dump(server.AccountingMode, server.NoTrace, packet)
}

func (t *logTrace) Write(b []byte) (int, error) {
	return t.data.Write(b)
}

func (t *logTrace) dump() {
	log.Println(t.data.String())
}

func dump(mode string, objType server.TraceType, packet *server.ClientPacket) {
	go func() {
		if server.Disabled(mode, modes) {
			return
		}
		t := &logTrace{}
		write(t, mode, objType, packet, time.Now())
		t.dump()
	}()
}

func write(tracing io.Writer, mode string, objType server.TraceType, packet *server.ClientPacket, t time.Time) {
	dump := server.NewRequestDump(packet, mode)
	for _, m := range dump.DumpPacket(server.KeyValue{Key: "TraceType", Value: fmt.Sprintf("%d", objType)}) {
		tracing.Write([]byte(fmt.Sprintf("%s\n", m)))
	}
}

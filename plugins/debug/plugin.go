package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/epiphyte/radiucal/core"
)

type tracer struct {
}

var (
	Plugin tracer
	modes  []string
)

func (t *tracer) Reload() {
}

func (t *tracer) Name() string {
	return "debugger"
}

func (t *tracer) Setup(ctx *core.PluginContext) {
	modes = core.DisabledModes(t, ctx)
}

func (t *tracer) Pre(packet *core.ClientPacket) bool {
	return authDump(core.PreAuthMode, packet)
}

func (t *tracer) Post(packet *core.ClientPacket) bool {
	return authDump(core.PostAuthMode, packet)
}

func authDump(mode string, packet *core.ClientPacket) bool {
	dump(mode, core.NoTrace, packet)
	return true
}

func (t *tracer) Trace(objType core.TraceType, packet *core.ClientPacket) {
	dump(core.TracingMode, objType, packet)
}

func (t *tracer) Account(packet *core.ClientPacket) {
	dump(core.AccountingMode, core.NoTrace, packet)
}

type logTrace struct {
	io.Writer
	data bytes.Buffer
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
	dump := core.NewRequestDump(packet, mode, t)
	tracing.Write([]byte(fmt.Sprintf("tracetype: %d\n", objType)))
	dump.DumpPacket(tracing)
}

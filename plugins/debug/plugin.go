package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/epiphyte/radiucal/plugins"
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

func (t *tracer) Setup(ctx *plugins.PluginContext) {
	modes = plugins.DisabledModes(t, ctx)
}

func (t *tracer) Pre(packet *plugins.ClientPacket) bool {
	dump(plugins.PreAuthMode, plugins.NoTrace, packet)
	return true
}

func (t *tracer) Trace(objType plugins.TraceType, packet *plugins.ClientPacket) {
	dump(plugins.TracingMode, objType, packet)
}

func (t *tracer) Account(packet *plugins.ClientPacket) {
	dump(plugins.AccountingMode, plugins.NoTrace, packet)
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

func dump(mode string, objType plugins.TraceType, packet *plugins.ClientPacket) {
	go func() {
		if plugins.Disabled(mode, modes) {
			return
		}
		t := &logTrace{}
		write(t, mode, objType, packet, time.Now())
		t.dump()
	}()
}

func write(tracing io.Writer, mode string, objType plugins.TraceType, packet *plugins.ClientPacket, t time.Time) {
	dump := plugins.NewRequestDump(packet, mode, t)
	tracing.Write([]byte(fmt.Sprintf("tracetype: %d\n", objType)))
	dump.DumpPacket(tracing)
}

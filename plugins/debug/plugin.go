package main

import (
	"bytes"
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
	dump(plugins.PreAuthMode, plugins.None, packet)
	return true
}

func (t *tracer) Trace(objType plugins.TraceType, packet *plugins.ClientPacket) {
	dump(plugins.TracingMode, objType, packet)
}

func (t *tracer) Account(packet *plugins.ClientPacket) {
	dump(plugins.AccountingMode, plugins.None, packet)
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
		tracer := &logTrace{}
		dump := plugins.NewRequestDump(packet, mode, time.Now())
		log.Println("tracetype: ", objType)
		dump.DumpPacket(tracer)
		tracer.dump()
	}()
}

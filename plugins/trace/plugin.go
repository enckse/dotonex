package main

import (
	"github.com/epiphyte/radiucal/plugins"
	"io"
	"log"
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
	return "tracer"
}

func (t *tracer) Setup(ctx *plugins.PluginContext) {
	modes = plugins.DisabledModes(t, ctx)
}

func (t *tracer) Pre(packet *plugins.ClientPacket) bool {
	dump(plugins.PreAuthMode, packet)
	return true
}

func (t *tracer) Auth(packet *plugins.ClientPacket) {
	dump(plugins.AuthingMode, packet)
}

func (t *tracer) Account(packet *plugins.ClientPacket) {
	dump(plugins.AccountingMode, packet)
}

type logTrace struct {
	io.Writer
}

func (t *logTrace) Write(b []byte) (int, error) {
	log.Println(string(b))
	return 0, nil
}

func dump(mode string, packet *plugins.ClientPacket) {
	go func() {
		if plugins.Disabled(mode, modes) {
			return
		}
		tracer := &logTrace{}
		log.Println(mode)
		plugins.DumpPacket(packet, tracer)
	}()
}

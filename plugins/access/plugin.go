package main

import (
	"fmt"
	"io"
	"strconv"
	"sync"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/core"
)

var (
	lock     *sync.Mutex = new(sync.Mutex)
	logs     string
	Plugin   access
	modes    []string
	instance string
)

type access struct {
}

func (l *access) Name() string {
	return "access"
}

func (l *access) Reload() {
}

func (l *access) Setup(ctx *core.PluginContext) error {
	logs = ctx.Logs
	modes = core.DisabledModes(l, ctx)
	instance = ctx.Instance
	return nil
}

func (l *access) Pre(packet *core.ClientPacket) bool {
	return core.NoopPre(packet, write)
}

func (l *access) Post(packet *core.ClientPacket) bool {
	return core.NoopPost(packet, write)
}

func (l *access) Trace(t core.TraceType, packet *core.ClientPacket) {
	write(core.TracingMode, t, packet)
}

func (l *access) Account(packet *core.ClientPacket) {
	write(core.AccountingMode, core.NoTrace, packet)
}

func keyValWrite(f io.Writer, key, val string) {
	if len(val) == 0 {
		return
	}
	f.Write([]byte(fmt.Sprintf("  %s => %s\n", key, val)))
}

func write(mode string, objType core.TraceType, packet *core.ClientPacket) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		if core.Disabled(mode, modes) {
			return
		}
		f, t := core.DatedAppendFile(logs, "access", instance)
		if f == nil {
			return
		}
		username, err := rfc2865.UserName_LookupString(packet.Packet)
		if err != nil {
			username = ""
		}
		calling, err := rfc2865.CallingStationID_LookupString(packet.Packet)
		if err != nil {
			calling = ""
		}
		defer f.Close()
		f.Write([]byte(fmt.Sprintf("Info -> %s %d (%s)\n", mode, int(objType), t)))
		keyValWrite(f, "Code", packet.Packet.Code.String())
		keyValWrite(f, "Id", strconv.Itoa(int(packet.Packet.Identifier)))
		keyValWrite(f, "User-Name", username)
		keyValWrite(f, "Calling-Station-Id", calling)
	}()
}

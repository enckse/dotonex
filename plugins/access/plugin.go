package main

import (
	"fmt"
	"strconv"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/core"
)

var (
	Plugin access
	modes  []string
)

type access struct {
}

func (l *access) Name() string {
	return "access"
}

func (l *access) Reload() {
}

func (l *access) Setup(ctx *core.PluginContext) error {
	modes = core.DisabledModes(l, ctx)
	return nil
}

func (l *access) Pre(packet *core.ClientPacket) bool {
	return core.NoopPre(packet, l.write)
}

func (l *access) Post(packet *core.ClientPacket) bool {
	return core.NoopPost(packet, l.write)
}

func (l *access) Trace(t core.TraceType, packet *core.ClientPacket) {
	l.write(core.TracingMode, t, packet)
}

func (l *access) Account(packet *core.ClientPacket) {
	l.write(core.AccountingMode, core.NoTrace, packet)
}

func keyValWrite(messages []string, key, val string) []string {
	if len(val) == 0 {
		return messages
	}
	return append(messages, fmt.Sprintf("  %s = %s", key, val))
}

func (l *access) write(mode string, objType core.TraceType, packet *core.ClientPacket) {
	go func() {
		if core.Disabled(mode, modes) {
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
		var messages []string
		messages = append(messages, fmt.Sprintf("Info = %s %d", mode, int(objType)))
		messages = keyValWrite(messages, "Code", packet.Packet.Code.String())
		messages = keyValWrite(messages, "Id", strconv.Itoa(int(packet.Packet.Identifier)))
		messages = keyValWrite(messages, "User-Name", username)
		messages = keyValWrite(messages, "Calling-Station-Id", calling)
		core.LogPluginMessages(l, messages)
	}()
}

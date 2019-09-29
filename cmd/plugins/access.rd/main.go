package main

import (
	"fmt"
	"strconv"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/core"
)

var (
	// Plugin represents the plugin instance for the system
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

func write(mode string, objType core.TraceType, packet *core.ClientPacket) {
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
		kv := core.KeyValueStore{}
		kv.DropEmpty = true
		kv.Add("Mode", fmt.Sprintf("%s", mode))
		kv.Add("Code", packet.Packet.Code.String())
		kv.Add("Id", strconv.Itoa(int(packet.Packet.Identifier)))
		kv.Add("User-Name", username)
		kv.Add("Calling-Station-Id", calling)
		core.LogPluginMessages(&Plugin, kv.Strings())
	}()
}

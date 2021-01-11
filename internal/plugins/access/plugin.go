package access

import (
	"fmt"
	"strconv"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/internal"
)

var (
	// Plugin represents the plugin instance for the system
	Plugin access
)

type (
	access struct{}
)

func (l *access) Name() string {
	return "access"
}

func (l *access) Setup(ctx *internal.PluginContext) error {
	return nil
}

func (l *access) Pre(packet *internal.ClientPacket) bool {
	return internal.NoopPre(packet, write)
}

func (l *access) Trace(t internal.TraceType, packet *internal.ClientPacket) {
	write(internal.TracingMode, t, packet)
}

func (l *access) Account(packet *internal.ClientPacket) {
	write(internal.AccountingMode, internal.NoTrace, packet)
}

func write(mode string, objType internal.TraceType, packet *internal.ClientPacket) {
	if mode == internal.TracingMode {
		return
	}
	go func() {
		username, err := rfc2865.UserName_LookupString(packet.Packet)
		if err != nil {
			username = ""
		}
		calling, err := rfc2865.CallingStationID_LookupString(packet.Packet)
		if err != nil {
			calling = ""
		}
		kv := internal.KeyValueStore{}
		kv.DropEmpty = true
		kv.Add("Mode", fmt.Sprintf("%s", mode))
		kv.Add("Code", packet.Packet.Code.String())
		kv.Add("Id", strconv.Itoa(int(packet.Packet.Identifier)))
		kv.Add("User-Name", username)
		kv.Add("Calling-Station-Id", calling)
		internal.LogPluginMessages(&Plugin, kv.Strings())
	}()
}

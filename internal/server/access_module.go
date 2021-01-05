package access

import (
	"fmt"
	"strconv"

	"layeh.com/radius/rfc2865"
	"voidedtech.com/radiucal/internal/server"
)

var (
	// Plugin represents the plugin instance for the system
	Plugin access
	modes  []string
)

type (
	access struct{}
)

func (l *access) Name() string {
	return "access"
}

func (l *access) Setup(ctx *server.PluginContext) error {
	modes = server.DisabledModes(l, ctx)
	return nil
}

func (l *access) Pre(packet *server.ClientPacket) bool {
	return server.NoopPre(packet, write)
}

func (l *access) Post(packet *server.ClientPacket) bool {
	return server.NoopPost(packet, write)
}

func (l *access) Trace(t server.TraceType, packet *server.ClientPacket) {
	write(server.TracingMode, t, packet)
}

func (l *access) Account(packet *server.ClientPacket) {
	write(server.AccountingMode, server.NoTrace, packet)
}

func write(mode string, objType server.TraceType, packet *server.ClientPacket) {
	go func() {
		if server.Disabled(mode, modes) {
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
		kv := server.KeyValueStore{}
		kv.DropEmpty = true
		kv.Add("Mode", fmt.Sprintf("%s", mode))
		kv.Add("Code", packet.Packet.Code.String())
		kv.Add("Id", strconv.Itoa(int(packet.Packet.Identifier)))
		kv.Add("User-Name", username)
		kv.Add("Calling-Station-Id", calling)
		server.LogPluginMessages(&Plugin, kv.Strings())
	}()
}

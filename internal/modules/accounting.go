package modules

import (
	"voidedtech.com/dotonex/internal/op"
)

type (
	// AccountingModule for accounting servers
	AccountingModule struct {
	}
)

// Account will do accounting operations
func (l *AccountingModule) Account(packet *op.ClientPacket) {
	moduleWrite("accounting", op.NoTrace, packet)
}

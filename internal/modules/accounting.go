package modules

import (
	"voidedtech.com/dotonex/internal/op"
)

type (
	AccountingModule struct {
	}
)

func (l *AccountingModule) Name() string {
	return "accounting"
}

func (l *AccountingModule) Account(packet *op.ClientPacket) {
	moduleWrite("accounting", op.NoTrace, packet)
}

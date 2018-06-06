package core

import (
	"github.com/epiphyte/goutils"
)

func LogError(message string, err error) bool {
	if err == nil {
		return false
	}
	goutils.WriteError(message, err)
	return true
}

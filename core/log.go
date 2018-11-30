package core

import (
	"github.com/epiphyte/goutils/logger"
)

func LogError(message string, err error) bool {
	if err == nil {
		return false
	}
	logger.WriteError(message, err)
	return true
}

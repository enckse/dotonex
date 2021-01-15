package core

import (
	"fmt"
	"log"
)

var (
	debugging = false
	instance  = ""
)

// ConfigureLogging will configure the underlying logging options
// (this should be called at startup)
func ConfigureLogging(dbg bool, inst string) {
	debugging = dbg
	if len(inst) > 0 {
		instance = fmt.Sprintf("- %s - ", inst)
	}
}

func init() {
	log.SetFlags(0)
}

// Fatal will log a fatal message and error
func Fatal(message string, err error) {
	if err != nil {
		WriteError(message, err)
	}
	log.Fatal(message)
}

// WriteInfo logs informational messages
func WriteInfo(message string, messages ...string) {
	write("INFO", message, messages...)
}

// WriteWarn logs warning messages
func WriteWarn(message string, messages ...string) {
	write("WARN", message, messages...)
}

// WriteError logs error messages
func WriteError(message string, err error) {
	write("ERROR", message, fmt.Sprintf("%s", err))
}

// WriteDebug logs debugging messages
func WriteDebug(message string, messages ...string) {
	if !debugging {
		return
	}
	write("DEBUG", message, messages...)
}

func write(cat, message string, messages ...string) {
	category := ""
	vars := ""
	category = fmt.Sprintf("[%s] ", cat)
	if len(messages) > 0 {
		vars = fmt.Sprintf(" (%s)", messages)
	}
	msg := fmt.Sprintf("%s%s%s%s", category, instance, message, vars)
	log.Print(msg)
}

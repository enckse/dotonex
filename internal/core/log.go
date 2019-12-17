package core

import (
	"fmt"
	"log"
	"os"
)

const (
	// ExitFailure indicates something went wrong
	ExitFailure = 2
	// ExitSignal indicates a signal exit (ok but do something)
	ExitSignal = 1
)

var (
	debugging = false
	instance  = ""
)

// ConfigureLogging will configure the underlying logging options
// (this should be called at startup)
func ConfigureLogging(dbg bool, instance string) {
	debugging = dbg
	if len(instance) > 0 {
		instance = fmt.Sprintf("- %s - ", instance)
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

// WriteInfoDetail logs detail for informational messages
func WriteInfoDetail(message string) {
	write("INFO", fmt.Sprintf(" => %s", message))
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

func write(cat string, message string, messages ...string) {
	category := ""
	vars := ""
	category = fmt.Sprintf("[%s] ", cat)
	if len(messages) > 0 {
		vars = fmt.Sprintf(" (%s)", messages)
	}
	msg := fmt.Sprintf("%s%s%s%s", category, instance, message, vars)
	log.Print(msg)
}

// Version prints version information
func Version(vers, details string) {
	d := ""
	if len(details) > 0 {
		d = fmt.Sprintf(" (%s)", details)
	}
	WriteInfo(fmt.Sprintf("Version: %s%s", vers, d))
}

// ExitNow prints an error message and calls Exit (do NOT call when defers are running)
func ExitNow(message string, err error) {
	WriteError(message, err)
	os.Exit(ExitFailure)
}

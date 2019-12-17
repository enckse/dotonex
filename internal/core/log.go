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
	debugging   = false
	erroring    = true
	information = true
	warning     = true
	instance    = ""
)

type (
	// LogOptions specify how to output logging information
	LogOptions struct {
		Debug bool
		Info  bool
		Warn  bool
		Error bool
		// Instance name to use (default is empty
		Instance string
	}
)

// NewLogOptions creates a new logging configuration option set
func NewLogOptions() *LogOptions {
	return &LogOptions{Warn: true, Error: true}
}

// ConfigureLogging will configure the underlying logging options
// (this should be called at startup)
func ConfigureLogging(options *LogOptions) {
	if options.Debug {
		debugging = true
	}
	if debugging || options.Info {
		information = true
	}
	if debugging || information || options.Warn {
		warning = true
	}
	if warning || debugging || information || options.Error {
		erroring = true
	}
	if len(options.Instance) > 0 {
		instance = fmt.Sprintf("- %s - ", options.Instance)
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
	write(information || debugging, "INFO", message, messages...)
}

// WriteInfoDetail logs detail for informational messages
func WriteInfoDetail(message string) {
	write(information || debugging, "INFO", fmt.Sprintf(" => %s", message))
}

// WriteWarn logs warning messages
func WriteWarn(message string, messages ...string) {
	write(warning || debugging || information, "WARN", message, messages...)
}

// WriteError logs error messages
func WriteError(message string, err error) {
	write(erroring || warning || debugging || information, "ERROR", message, fmt.Sprintf("%s", err))
}

// WriteDebug logs debugging messages
func WriteDebug(message string, messages ...string) {
	write(debugging, "DEBUG", message, messages...)
}

func write(condition bool, cat string, message string, messages ...string) {
	if condition {
		category := ""
		vars := ""
		category = fmt.Sprintf("[%s] ", cat)

		if len(messages) > 0 {
			vars = fmt.Sprintf(" (%s)", messages)
		}
		msg := fmt.Sprintf("%s%s%s%s", category, instance, message, vars)
		log.Print(msg)

	}
}

// Version prints version information
func Version(vers string) {
	WriteInfo(fmt.Sprintf("Version: %s", vers))
}

// ExitNow prints an error message and calls Exit (do NOT call when defers are running)
func ExitNow(message string, err error) {
	WriteError(message, err)
	os.Exit(ExitFailure)
}

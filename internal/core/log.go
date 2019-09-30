package core

import (
	"fmt"
	"log"
)

// LogError to log errors within the system
func LogError(message string, err error) bool {
	if err == nil {
		return false
	}
	WriteError(message, err)
	return true
}

const (
	defaultFlags = 0
)

var (
	debugging   = false
	erroring    = true
	information = true
	warning     = true
	quiet       = false
	instance    = ""
	level       = true
	variadic    = true
)

type (
	// LogOptions specify how to output logging information
	LogOptions struct {
		Debug bool
		Info  bool
		Warn  bool
		Error bool
		// Enable timestamps
		Timestamps bool
		// Disable all logging
		Quiet bool
		// Instance name to use (default is empty
		Instance string
		// Indicates if level output should be disabled
		NoLevel bool
		// Indicates if the variadic args should be displayed
		NoVariadic bool
	}
)

// NewLogOptions creates a new logging configuration option set
func NewLogOptions() *LogOptions {
	return &LogOptions{Warn: true, Error: true, Timestamps: false}
}

// ConfigureLogging will configure the underlying logging options
// (this should be called at startup)
func ConfigureLogging(options *LogOptions) {
	if !options.Timestamps {
		log.SetFlags(defaultFlags)
	}
	if options.Quiet {
		quiet = true
	}
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
	if options.NoLevel {
		level = false
	}
	if options.NoVariadic {
		variadic = false
	}
	if len(options.Instance) > 0 {
		instance = fmt.Sprintf("- %s - ", options.Instance)
	}
}

func init() {
	log.SetFlags(defaultFlags)
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
		if !quiet {
			category := ""
			vars := ""
			if level {
				category = fmt.Sprintf("[%s] ", cat)
			}
			if variadic && len(messages) > 0 {
				vars = fmt.Sprintf(" (%s)", messages)
			}
			msg := fmt.Sprintf("%s%s%s%s", category, instance, message, vars)
			log.Print(msg)
		}
	}
}

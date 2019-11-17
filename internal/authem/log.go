package authem

import (
	"fmt"
	"os"
)

const (
	// ExitFailure indicates something went wrong
	ExitFailure = 2
	// ExitSignal indicates a signal exit (ok but do something)
	ExitSignal = 1
)

// Error writes error information out
func Error(message string, err error) {
	msg := message
	if err != nil {
		msg = fmt.Sprintf("%s -> %v", msg, err)
	}
	fmt.Println(fmt.Sprintf("ERROR: %s", msg))
}

// Info prints informational messages
func Info(message string) {
	fmt.Println(message)
}

// InfoDetail prints detailed informational messages
func InfoDetail(message string) {
	Info(fmt.Sprintf("  => %s", message))
}

// Fatal performs a error output and panics
func Fatal(message string, err error) {
	Error(message, err)
	panic("fatal error detected")
}

// Version prints version information
func Version(vers string) {
	Info(fmt.Sprintf("Version: %s", vers))
}

// ExitNow prints an error message and calls Exit (do NOT call when defers are running)
func ExitNow(message string, err error) {
	Error(message, err)
	os.Exit(ExitFailure)
}

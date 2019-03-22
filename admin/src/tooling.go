package main

import (
	"flag"
	"fmt"
	"os"

	"voidedtech.com/goutils/logger"
)

var (
	vers = "master"
)

func die(err error) {
	dieNow("unrecoverable error", err, err != nil)
}

func dieNow(message string, err error, now bool) {
	messaged := false
	if err != nil {
		messaged = true
		logger.WriteError(message, err)
	}
	if now {
		if !messaged {
			logger.WriteWarn(message)
		}
		os.Exit(1)
	}
}

func main() {
	cmd := flag.String("command", "", "command to execute")
	flag.Parse()
	action := *cmd
	switch action {
	case "netconf":
		netconfRun()
	case "version":
		fmt.Println(vers)
	default:
		fmt.Println("unknown command")
	}
}

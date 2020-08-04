package main

import (
	"flag"
	"fmt"
	"os"

	"voidedtech.com/radiucal/internal/core"
	"voidedtech.com/radiucal/internal/server"
)

const (
	vers = "master"
)

func main() {
	serve := flag.NewFlagSet("serve", flag.ContinueOnError)
	cfg := serve.String("config", "/etc/authem/configurator.yaml", "config file (server mode)")

	args := []string{}
	if len(os.Args) > 1 {
		args = os.Args[2:]
	}

	switch os.Args[1] {
	case "serve":
		if err := serve.Parse(args); err != nil {
			core.Fatal("invalid flags for configurate", err)
		}
		server.Run(*cfg)
	case "version":
		fmt.Println(vers)
	}
}

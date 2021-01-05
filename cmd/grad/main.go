package main

import (
	"flag"
	"fmt"
	"os"

	"voidedtech.com/radiucal/internal/core"
	"voidedtech.com/radiucal/internal/management"
)

const (
	vers = "master"
)

func main() {
	configurate := flag.NewFlagSet("configurate", flag.ContinueOnError)
	cfg := configurate.String("config", "/etc/authem/configurator.yaml", "config file (server mode)")
	forceScript := configurate.Bool("run-scripts", false, "run the scripts regardless of configuration changes")
	password := flag.NewFlagSet("passwd", flag.ContinueOnError)
	user := password.String("user", "", "user to change")
	email := password.String("email", "", "user's email address")
	force := password.Bool("force", false, "force change a user's secret")
	show := password.Bool("show", false, "show the user's secrets, perform no changes")
	pwd := password.String("password", "", "use this password")
	length := password.Int("length", 64, "default password length")

	args := []string{}
	if len(os.Args) > 1 {
		args = os.Args[2:]
	}

	switch os.Args[1] {
	case "configurate":
		if err := configurate.Parse(args); err != nil {
			core.Fatal("invalid flags for configurate", err)
		}
		scripts := configurate.Args()
		management.Configurate(*cfg, scripts, *forceScript)
	case "passwd":
		if err := password.Parse(args); err == nil {
			core.Fatal("invalid flags for passwd", err)
		}
		management.Password(user, email, pwd, show, force, length)
	case "version":
		fmt.Println(vers)
	}
}

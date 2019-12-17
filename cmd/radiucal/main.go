package main

import (
	"fmt"
	"os"
	"os/exec"

	"voidedtech.com/radiucal/internal/core"
)

func main() {
	flags := core.Flags()
	i := fmt.Sprintf("instance: %s", flags.Instance)
	args := flags.Args()
	if flags.Debug {
		core.WriteDebug(fmt.Sprintf("flags: %v", args))
	}
	for {
		core.WriteInfo("starting " + i)
		cmd := exec.Command("radiucal-runner", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			core.WriteWarn(fmt.Sprintf("radiucal runner ended %s (%v)", i, err))
		}
	}
}

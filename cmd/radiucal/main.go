package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"voidedtech.com/radiucal/internal/core"
	"voidedtech.com/radiucal/internal/server"
)

func main() {
	flags := server.Flags()
	i := fmt.Sprintf("instance: %s", flags.Instance)
	args := flags.Args()
	if flags.Debug {
		core.WriteDebug(fmt.Sprintf("flags: %v", args))
	}
	last := time.Now()
	errors := 0
	for {
		core.WriteInfo("starting " + i)
		cmd := exec.Command("radiucal-runner", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			core.WriteWarn(fmt.Sprintf("radiucal runner ended %s (%v)", i, err))
		}
		now := time.Now()
		sleep := 10 * time.Millisecond
		if now.Sub(last).Seconds() < 30 {
			if errors > 3 {
				core.WriteWarn("cool down for restart")
				sleep = 5 * time.Second
			} else {
				errors++
			}
		} else {
			errors = 0
		}
		time.Sleep(sleep)
		last = now
	}
}

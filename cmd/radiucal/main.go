package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"voidedtech.com/radiucal/internal"
)

func main() {
	flags := internal.Flags()
	i := fmt.Sprintf("instance: %s", flags.Instance)
	args := flags.Args()
	if flags.Debug {
		internal.WriteDebug(fmt.Sprintf("flags: %v", args))
	}
	last := time.Now()
	errors := 0
	for {
		internal.WriteInfo("starting " + i)
		cmd := exec.Command("radiucal-runner", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			internal.WriteWarn(fmt.Sprintf("radiucal runner ended %s (%v)", i, err))
		}
		now := time.Now()
		sleep := 10 * time.Millisecond
		if now.Sub(last).Seconds() < 30 {
			if errors > 3 {
				internal.WriteWarn("cool down for restart")
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

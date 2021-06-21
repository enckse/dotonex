package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"voidedtech.com/dotonex/internal/core"
)

func main() {
	flags := core.Flags()
	instances := []string{}
	options, err := os.ReadDir(flags.Directory)
	if err != nil {
		core.Fatal("unable to read possible instances", err)
	}
	for _, option := range options {
		name := option.Name()
		if strings.HasSuffix(name, core.InstanceConfig) {
			instances = append(instances, strings.Replace(name, core.InstanceConfig, "", -1))
		}
	}

	if len(instances) == 0 {
		core.Fatal("no instances found", fmt.Errorf("please configure some instances"))
	}
	for _, i := range instances {
		go runInstance(i, flags)
	}

	duration := 10 * time.Second
	for {
		time.Sleep(duration)
	}
}

func runInstance(instance string, arguments core.ProcessFlags) {
	args := arguments.Args(instance)
	if arguments.Debug {
		core.WriteDebug(fmt.Sprintf("flags: %v", args))
	}
	last := time.Now()
	errors := 0
	for {
		core.WriteInfo("starting " + instance)
		cmd := exec.Command("dotonex-runner", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			core.WriteWarn(fmt.Sprintf("dotonex runner ended %s (%v)", instance, err))
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

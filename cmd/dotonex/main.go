package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"voidedtech.com/dotonex/internal"
)

func main() {
	flags := internal.Flags()
	instances := []string{}
	options, err := ioutil.ReadDir(flags.Directory)
	if err != nil {
		internal.Fatal("unable to read possible instances", err)
	}
	for _, option := range options {
		name := option.Name()
		if strings.HasSuffix(name, internal.InstanceConfig) {
			instances = append(instances, strings.Replace(name, internal.InstanceConfig, "", -1))
		}
	}

	if len(instances) == 0 {
		internal.Fatal("no instances found", fmt.Errorf("please configure some instances"))
	}
	for _, i := range instances {
		go runInstance(i, flags)
	}

	duration := 10 * time.Second
	for {
		time.Sleep(duration)
	}
}

func runInstance(instance string, arguments internal.ProcessFlags) {
	args := arguments.Args(instance)
	if arguments.Debug {
		internal.WriteDebug(fmt.Sprintf("flags: %v", args))
	}
	last := time.Now()
	errors := 0
	for {
		internal.WriteInfo("starting " + instance)
		cmd := exec.Command("dotonex-runner", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			internal.WriteWarn(fmt.Sprintf("dotonex runner ended %s (%v)", instance, err))
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

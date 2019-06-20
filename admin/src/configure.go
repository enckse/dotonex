package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"voidedtech.com/goutils/logger"
	"voidedtech.com/goutils/opsys"
)

const (
	hash     = outputDir + "last"
	prevHash = hash + ".prev"
	eapBin   = outputDir + eapUsers
	daily    = "/tmp/"
	changed  = outputDir + "changed"
	varLib   = "/var/lib/radiucal/"
	varHome  = varLib + "users/"
	manBin   = outputDir + manifest
)

func bashCommand(command string, canDie bool) {
	_, err := opsys.RunCommand("bash", "-c", command)
	if canDie {
		die(err)
	}
}

func runCommand(command string) {
	bashCommand(command, true)
}

func signal() {
	logger.WriteInfo("signal apps")
	bashCommand("kill -HUP $(pidof hostapd)", false)
	bashCommand("kill -2 $(pidof radiucal)", false)
}

func setup() {
	if opsys.PathNotExists(outputDir) {
		runCommand(fmt.Sprintf("mkdir -p %s", outputDir))
	}
	if opsys.PathExists(hash) {
		runCommand(fmt.Sprintf("cp %s %s", hash, prevHash))
	}
	runCommand(fmt.Sprintf("rm -f %s", changed))
}

func callReports(client bool) {
	signal()
}

func update(client bool) {
	if opsys.PathNotExists(manBin) {
		logger.Fatal("missing manifest file", nil)
	}
	runCommand(fmt.Sprintf("mkdir -p %s", varHome))
	f, err := ioutil.ReadDir(varHome)
	die(err)
	b, err := ioutil.ReadFile(manBin)
	current := make(map[string]struct{})
	for _, l := range strings.Split(string(b), "\n") {
		current[l] = struct{}{}
	}
	die(err)
	for _, file := range f {
		n := file.Name()
		if _, ok := current[n]; !ok {
			p := filepath.Join(varHome, n)
			logger.WriteInfo("dropping file", n, p)
			runCommand(fmt.Sprintf("rm -f %s", p))
		}
	}
	for k, _ := range current {
		runCommand(fmt.Sprintf("touch %s%s", varHome, k))
	}
	runCommand(fmt.Sprintf("cp %s %s/%s", eapBin, varLib, eapUsers))
	callReports(client)
}

func configure(client bool) {
	t := time.Now().Format("2006-01-02")
	logger.WriteInfo("updating network configuration", t)
	setup()
	netconfRun()
	server := !client
	if server {
		logger.WriteInfo("checking for daily operations")
		dailyRun := filepath.Join(daily, fmt.Sprintf(".radius-%s", t))
		if opsys.PathNotExists(dailyRun) {
			logger.WriteInfo("daily updates")
			callReports(client)
			runCommand(fmt.Sprintf("touch %s", dailyRun))
		}
	}
	runCommand(fmt.Sprintf("cat %s* | sha256sum | cut -d ' ' -f 1 > %s", userDir, hash))
	diffed := true
	if opsys.PathExists(hash) && opsys.PathExists(prevHash) {
		runCommand(fmt.Sprintf("diff -u %s %s; if [ $? -ne 0 ]; then touch %s; fi", prevHash, hash, changed))
		diffed = opsys.PathExists(changed)
	}
	if diffed {
		logger.WriteInfo("configuration updated")
		if server {
			update(client)
		}
	}
}

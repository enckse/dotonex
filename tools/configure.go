package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"voidedtech.com/dotonex/internal/core"
)

type (
	// Make are the template inputs for the Makefile
	Make struct {
		Gitlab           bool
		GoFlags          string
		HostapdVersion   string
		SharedKey        string
		RADIUSKey        string
		GitlabFQDN       string
		ServerRepository string
		BuildOnly        bool
		errored          bool
		Accounting       string
		To               bool
		Bind             string
		file             string
		Configuration    *Config
	}

	// Config generation
	Config struct {
		Accounting string
		To         bool
		Bind       string
		file       string
	}
)

var (
	generated = []string{"Makefile", "clients", "env", "secrets"}
)

const (
	hostapdFlag    = "hostap-version"
	gitlabFlag     = "enable-gitlab"
	gitlabFQDNFlag = "gitlab-fqdn"
	repoFlag       = "server-repository"
	sharedFlag     = "shared-key"
	radiusFlag     = "radius-key"
	toolDir        = "tools"
)

func show(cat, message string) {
	fmt.Println(fmt.Sprintf("[%s] %s", cat, message))
}

func (m *Make) nonEmptyFatal(cat, key, value string) {
	if strings.TrimSpace(value) == "" {
		category := cat
		if len(category) == 0 {
			category = "global"
		} else {
			category = fmt.Sprintf("-%s", category)
		}
		m.fail(fmt.Errorf("[%s] '-%s' must be set", category, key), false)
	}
}

func (m *Make) fail(err error, exit bool) {
	show("ERROR", fmt.Sprintf("%v", err))
	m.errored = true
	if exit {
		os.Exit(1)
	}
}

func main() {
	hostapd := flag.String(hostapdFlag, "hostap_2_9", "hostapd version to build")
	buildOnly := flag.Bool("build-only", false, "build only, no setup/install")
	doGitlab := flag.Bool(gitlabFlag, true, "enable gitlab configuration")
	gitlabFQDN := flag.String(gitlabFQDNFlag, "", "gitlab fully-qualified-domain-name")
	repo := flag.String(repoFlag, "", "server repository for backend management")
	radiusKey := flag.String(radiusFlag, "", "radius key between server and networking components")
	sharedKey := flag.String(sharedFlag, "", "shared radius key for all users given unique tokens")
	goFlags := flag.String("go-flags", "-ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w' -trimpath -buildmode=pie -mod=readonly -modcacherw", "flags for go building")
	flag.Parse()
	m := Make{BuildOnly: *buildOnly, Gitlab: *doGitlab, GoFlags: *goFlags, HostapdVersion: *hostapd, GitlabFQDN: *gitlabFQDN, RADIUSKey: *radiusKey, SharedKey: *sharedKey, ServerRepository: *repo}
	cleanup := generated
	files, err := ioutil.ReadDir(".")
	if err != nil {
		m.fail(err, true)
	}
	for _, f := range files {
		name := f.Name()
		if strings.HasSuffix(name, core.InstanceConfig) {
			cleanup = append(cleanup, name)
		}
	}
	for _, g := range cleanup {
		show("cleanup", g)
		if core.PathExists(g) {
			if err := os.Remove(g); err != nil {
				m.fail(err, true)
			}
		}
	}
	m.errored = false
	m.nonEmptyFatal("", hostapdFlag, m.HostapdVersion)
	defaults := true
	defaultGitlab := true
	if !m.BuildOnly {
		defaults = false
		m.nonEmptyFatal("", radiusFlag, m.RADIUSKey)
		m.nonEmptyFatal("", sharedFlag, m.SharedKey)
		if m.Gitlab {
			defaultGitlab = false
			m.nonEmptyFatal(gitlabFlag, gitlabFQDNFlag, m.GitlabFQDN)
			m.nonEmptyFatal(gitlabFlag, repoFlag, m.ServerRepository)
		}
	}
	if defaults {
		m.RADIUSKey = "radiuskey"
		m.SharedKey = "sharedkey"
	}
	if defaultGitlab {
		m.GitlabFQDN = "gitlab.example.com"
		m.ServerRepository = "."
	}
	if m.errored {
		os.Exit(1)
	}
	for _, file := range []string{"Makefile", "clients", "env", "secrets"} {
		show("generating", file)
		b, err := ioutil.ReadFile(filepath.Join(toolDir, file+".in"))
		if err != nil {
			m.fail(err, true)
		}
		tmpl, err := template.New(file).Parse(string(b))
		if err != nil {
			m.fail(err, true)
		}
		var buffer bytes.Buffer
		if err := tmpl.Execute(&buffer, &m); err != nil {
			m.fail(err, true)
		}
		if err := ioutil.WriteFile(file, buffer.Bytes(), 0644); err != nil {
			m.fail(err, true)
		}
	}

	b, err := ioutil.ReadFile(filepath.Join(toolDir, "conf.in"))
	if err != nil {
		m.fail(err, true)
	}
	tmpl, err := template.New("script").Parse(string(b))
	if err != nil {
		m.fail(err, true)
	}
	proxy := &Config{Accounting: "false", To: true, Bind: "1812", file: "proxy"}
	accounting := &Config{Accounting: "true", To: false, Bind: "1813", file: "accounting"}
	for _, c := range []*Config{proxy, accounting} {
		m.Configuration = c
		output := c.file + core.InstanceConfig
		show("configs", output)
		var b bytes.Buffer
		if err := tmpl.Execute(&b, &m); err != nil {
			m.fail(err, true)
		}
		if err := ioutil.WriteFile(output, b.Bytes(), 0644); err != nil {
			m.fail(err, true)
		}
	}
}

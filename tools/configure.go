package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

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
		errored          bool
		To               bool
		file             string
		Configuration    *Config
		Static           string
		CFlags           string
		LDFlags          string
		CertKey          string
	}

	// Config generation
	Config struct {
		IsAccounting bool
		file         string
	}
)

var (
	generated     = []string{"Makefile", "clients", "env"}
	randomLetters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
)

const (
	hostapdFlag    = "hostap-version"
	gitlabFlag     = "enable-gitlab"
	gitlabFQDNFlag = "gitlab-fqdn"
	repoFlag       = "server-repository"
	sharedFlag     = "shared-key"
	radiusFlag     = "radius-key"
	toolDir        = "tools"
	certKeyFlag    = "hostapd-certkey"
)

func show(cat, message string) {
	fmt.Println(fmt.Sprintf("[%s] %s", cat, message))
}

func (m *Make) nonEmptyFatalKey(key, value string) {
	m.nonEmptyFatal("", key, value)
	for _, c := range strings.ToLower(value) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			continue
		}
		m.deferFatal("", key, "invalid character ([a-zA-Z0-9])")
	}
}

func (m *Make) nonEmptyFatal(cat, key, value string) {
	if strings.TrimSpace(value) == "" {
		m.deferFatal(cat, key, "must be set")
	}
}

func (m *Make) deferFatal(cat, key, reason string) {
	category := cat
	if len(category) == 0 {
		category = "global"
	} else {
		category = fmt.Sprintf("-%s", category)
	}
	m.fail(fmt.Errorf("[%s] '-%s' %s", category, key, reason), false)
}

func (m *Make) fail(err error, exit bool) {
	show("ERROR", fmt.Sprintf("%v", err))
	m.errored = true
	if exit {
		os.Exit(1)
	}
}

func randSequence(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = randomLetters[rand.Intn(len(randomLetters))]
	}
	return string(b)
}

func useOrRandom(name, input string) string {
	if len(input) > 0 {
		return input
	}
	val := randSequence(20)
	show("randomize", fmt.Sprintf("'%s' -> %s", name, val))
	return val
}

func main() {
	rand.Seed(time.Now().UnixNano())
	hostapd := flag.String(hostapdFlag, "hostap_2_9", "hostapd version to build")
	cFlags := flag.String("cflags", "-march=x86-64 -mtune=generic -O2 -pipe -fno-plt", "CFLAGS for hostapd build")
	ldFlags := flag.String("ldflags", "-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now", "LDFLAGS for hostapd build")
	certKey := flag.String(certKeyFlag, "", "hostapd certificate password key")
	doGitlab := flag.Bool(gitlabFlag, false, "enable gitlab configuration")
	gitlabFQDN := flag.String(gitlabFQDNFlag, "", "gitlab fully-qualified-domain-name")
	repo := flag.String(repoFlag, "", "server repository for backend management")
	radiusKey := flag.String(radiusFlag, "", "radius key between server and networking components")
	sharedKey := flag.String(sharedFlag, "", "shared radius key for all users given unique tokens")
	goFlags := flag.String("go-flags", "-ldflags '-linkmode external -extldflags $(LDFLAGS) -s -w' -trimpath -buildmode=pie -mod=readonly -modcacherw", "flags for go building")
	flag.Parse()
	m := Make{CertKey: *certKey, CFlags: *cFlags, LDFlags: *ldFlags, Gitlab: *doGitlab, GoFlags: *goFlags, HostapdVersion: *hostapd, GitlabFQDN: *gitlabFQDN, RADIUSKey: *radiusKey, SharedKey: *sharedKey, ServerRepository: *repo}
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
	m.Static = "true"
	defaults = false
	m.RADIUSKey = useOrRandom(radiusFlag, m.RADIUSKey)
	m.SharedKey = useOrRandom(sharedFlag, m.SharedKey)
	m.CertKey = useOrRandom(certKeyFlag, m.CertKey)
	if m.Gitlab {
		m.Static = "false"
		defaultGitlab = false
		m.nonEmptyFatal(gitlabFlag, gitlabFQDNFlag, m.GitlabFQDN)
		for _, c := range m.GitlabFQDN {
			if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' {
				continue
			}
			m.fail(fmt.Errorf("invalid character in FQDN"), false)
		}
		m.nonEmptyFatal(gitlabFlag, repoFlag, m.ServerRepository)
	}
	if defaults {
		m.CertKey = "certkey"
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
	for _, file := range generated {
		show("generating", file)
		tmpl, err := getTemplate(file)
		if err != nil {
			m.fail(err, true)
		}
		if err := writeTemplate(tmpl, file, m, 0644); err != nil {
			m.fail(err, true)
		}
	}
	tmpl, err := getTemplate("hostapd.configure")
	if err != nil {
		m.fail(err, true)
	}
	if err := writeTemplate(tmpl, filepath.Join("hostap", "configure"), m, 0755); err != nil {
		m.fail(err, true)
	}
	tmpl, err = getTemplate("conf")
	if err != nil {
		m.fail(err, true)
	}
	proxy := &Config{IsAccounting: false, file: "proxy"}
	accounting := &Config{IsAccounting: true, file: "accounting"}
	for _, c := range []*Config{proxy, accounting} {
		m.Configuration = c
		output := c.file + core.InstanceConfig
		show("configs", output)
		if err := writeTemplate(tmpl, output, m, 0644); err != nil {
			m.fail(err, true)
		}
	}
}

func writeTemplate(tmpl *template.Template, file string, m Make, mode os.FileMode) error {
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, &m); err != nil {
		m.fail(err, true)
	}
	if err := ioutil.WriteFile(file, buffer.Bytes(), mode); err != nil {
		m.fail(err, true)
	}
	return nil
}

func getTemplate(file string) (*template.Template, error) {
	b, err := ioutil.ReadFile(filepath.Join(toolDir, file+".in"))
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(file).Parse(string(b))
	if err != nil {
		return nil, err
	}
	return tmpl, err
}

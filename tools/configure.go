package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
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
		AllowInstall     bool
	}
)

const (
	hostapdFlag    = "hostap-version"
	gitlabFlag     = "enable-gitlab"
	gitlabFQDNFlag = "gitlab-fqdn"
	repoFlag       = "server-repository"
	sharedFlag     = "shared-key"
	radiusFlag     = "radius-key"
)

func nonEmptyFatal(cat, key, value string) {
	if strings.TrimSpace(value) == "" {
		category := cat
		if len(category) > 0 {
			category = fmt.Sprintf("[-%s] ", category)
		}
		fail(fmt.Errorf("%s'-%s' must be set", category, key))
	}
}

func fail(err error) {
	fmt.Println("[ERROR] %v", err)
	os.Exit(1)
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
	m := Make{AllowInstall: !*buildOnly, Gitlab: *doGitlab, GoFlags: *goFlags, HostapdVersion: *hostapd, GitlabFQDN: *gitlabFQDN, RADIUSKey: *radiusKey, SharedKey: *sharedKey, ServerRepository: *repo}
	if m.AllowInstall {
		nonEmptyFatal("", hostapdFlag, m.HostapdVersion)
		nonEmptyFatal("", radiusFlag, m.RADIUSKey)
		nonEmptyFatal("", sharedFlag, m.SharedKey)
		if m.Gitlab {
			nonEmptyFatal(gitlabFlag, gitlabFQDNFlag, m.GitlabFQDN)
			nonEmptyFatal(gitlabFlag, repoFlag, m.ServerRepository)
		}
	}
	b, err := ioutil.ReadFile("tools/Makefile.in")
	if err != nil {
		fail(err)
	}
	tmpl, err := template.New("make").Parse(string(b))
	if err != nil {
		fail(err)
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, &m); err != nil {
		fail(err)
	}
	if err := ioutil.WriteFile("Makefile", buffer.Bytes(), 0644); err != nil {
		fail(err)
	}
}

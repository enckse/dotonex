package main

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"

	"voidedtech.com/dotonex/internal/core"
)

type (
	// Config is the generation of the config for templating
	Config struct {
		Accounting string
		To         bool
		Bind       string
		file       string
	}
)

const (
	target = "configs/"
	config = `# dotonex config

# to support caching operations (false)
cache: true

# host (to bind to, default is localhost)
host: localhost

# accounting mode (false)
accounting: {{ .Accounting }}

# do NOT respond with a radius reject
noreject: true
{{ if .To }}
# proxy binding (not applicable in accounting mode, default: 1814)
to: 1814
{{ end }}
# bind port (1812 by default, 1813 for accounting)
bind: {{ .Bind }}

# working directory (/var/lib/dotonex/)
dir: /var/lib/dotonex/

# log dir
log: /var/log/dotonex/

# backend configuration management
configurator:
	# utilizies internal payload instead of backend scripts (false)
	static: false
	# repository path (/var/cache/dotonex/config)
	repository: /var/cache/dotonex/config
	# payload command to run to validate a user OR static list of token+mac pairs ([])
	payload: ["curl", "-s", "https://gitlab.url/api/v4/user?access_token={}"]
	# shared login key for all users (empty)
	serverkey: secretkey
	# refresh time for how often to rebuild dynamic config in minutes (5)
	refresh: 5
	# timeout for how long the backend script can run in seconds (30)
	timeout: 30
	# debug is enabled for backend
	debug: false

# internal operations (do NOT change except for debugging)
internals:
    # disable exit on interrupt
    nointerrupt: false
    # disable log buffering
    nologs: false
    # how long (seconds, default 10) to buffer logs
    logs: 10
    # how long should a runner last (hours: default 12)
    lifespan: 12
    # how often should a runner check for lifespan (hours: default 1)
    spancheck: 1
    # hour range in which a recycle is allowed based on lifespan (day hour 0-23, default: 22, 23, 0, 1, 2, 3, 4, 5)
    lifehours: [22, 23, 0, 1, 2, 3, 4, 5]
`
)

func main() {
	tmpl, err := template.New("script").Parse(strings.Replace(config, "\t", "    ", -1))
	if err != nil {
		core.Fatal("unable to parse template", err)
	}
	proxy := &Config{Accounting: "false", To: true, Bind: "1812", file: "proxy"}
	accounting := &Config{Accounting: "true", To: false, Bind: "1813", file: "accounting"}
	for _, c := range []*Config{proxy, accounting} {
		output := filepath.Join(target, c.file+core.InstanceConfig)
		core.WriteInfo(output)
		var b bytes.Buffer
		if err := tmpl.Execute(&b, c); err != nil {
			core.Fatal("template execution failure", err)
		}
		if err := ioutil.WriteFile(output, b.Bytes(), 0644); err != nil {
			core.Fatal("unable to write file", err)
		}
	}
}

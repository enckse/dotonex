package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/epiphyte/goutils"
	"golang.org/x/crypto/md4"
)

const (
	userDir      = "users/"
	bash         = "bash"
	pwdKey       = "pwd"
	useraddKey   = "useradd"
	netconfKey   = "netconf"
	configureKey = "configure"
	reportsKey   = "reports"
	packKey      = "pack"
	versionKey   = "version"
)

var (
	vers        = "master"
	scripting   = make(map[string][]string)
	skipUserDir = make(map[string]struct{})
	embed       = []string{}
)

func init() {
	skipUserDir[packKey] = struct{}{}
	skipUserDir[versionKey] = struct{}{}
	skipUserDir[pwdKey] = struct{}{}
	embed = append(embed, fmt.Sprintf("%s.py", netconfKey))
	embed = append(embed, fmt.Sprintf("%s.sh", reportsKey))
	embed = append(embed, fmt.Sprintf("%s.sh", configureKey))
}

func utf16le(s string) []byte {
	codes := utf16.Encode([]rune(s))
	b := make([]byte, len(codes)*2)
	for i, r := range codes {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return b
}

func getPass() (string, string) {
	out, err := goutils.RunBashCommand("pwgen 64 1")
	die(err)
	pass := strings.TrimSpace(strings.Join(out, ""))
	h := md4.New()
	h.Write(utf16le(pass))
	return pass, fmt.Sprintf("%x", string(h.Sum(nil)))
}

func password() {
	p, h := getPass()
	fmt.Println(fmt.Sprintf("\npassword:\n%s\n\nmd4:\n%s\n", p, h))
}

func die(err error) {
	dieNow("unrecoverable error", err, err != nil)
}

func dieNow(message string, err error, now bool) {
	messaged := false
	if err != nil {
		messaged = true
		goutils.WriteError(message, err)
	}
	if now {
		if !messaged {
			goutils.WriteWarn(message)
		}
		os.Exit(1)
	}
}

func useradd() {
	scanner := bufio.NewScanner(os.Stdin)
	text := ""
	fmt.Println("please input user name:")
	for scanner.Scan() {
		text = strings.TrimSpace(scanner.Text())
		break
	}
	user := ""
	for _, c := range text {
		if c >= 'a' && c <= 'z' {
			user += string(c)
		}
	}
	dieNow("invalid username", nil, len(user) == 0)
	p, h := getPass()
	script := fmt.Sprintf(`
import netconf as __config__
import users.common as common

u_obj = __config__.Assignment()
u_obj.password = '%s'
u_obj.vlan = None
u_obj.macs = None
`, h)
	ioutil.WriteFile(filepath.Join(userDir, fmt.Sprintf("user_%s.py", user)), []byte(script), 0644)
	fmt.Println(fmt.Sprintf("%s was create with a password of %s", user, p))
}

func runScript(name, interpreter string, client bool) {
	m := goutils.MemoryStringCompression{}
	v, ok := scripting[name]
	dieNow("no script", nil, !ok)
	m.Content = v
	res, err := m.Decompress()
	die(err)
	script := []string{res}
	logging := goutils.NewLogOptions()
	logging.NoVariadic = true
	logging.NoLevel = true
	logging.Info = true
	goutils.ConfigureLogging(logging)
	opts := &goutils.RunOptions{}
	opts.OnError = goutils.DumpStdall
	opts.StdoutDelimiter = "\n"
	updated := ""
	var useScript []string
	if interpreter == bash {
		isClient := 1
		if !client {
			isClient = 0
		}
		updated = "#!/bin/bash"
		useScript = append(useScript, updated)
		useScript = append(useScript, fmt.Sprintf("IS_LOCAL=%d", isClient))
	}
	for _, l := range script {
		if len(updated) > 0 && strings.HasPrefix(l, updated) {
			continue
		}
		useScript = append(useScript, l)
	}
	opts.WorkingDir = "."
	opts.Stdin = []string{strings.Join(useScript, "\n")}
	o, err := goutils.RunCommandWithOptions(opts, interpreter)
	die(err)
	for _, l := range o {
		fmt.Println(l)
	}
}

func main() {
	cmd := flag.String("command", "", "command to execute")
	client := flag.Bool("client", true, "indicate client or server (true is client)")
	flag.Parse()
	action := *cmd
	errored := false
	if _, ok := skipUserDir[action]; !ok {
		for _, check := range []string{userDir} {
			if goutils.PathNotExists(check) {
				errored = true
				fmt.Println(fmt.Sprintf("missing required file/directory: %s", check))
			}
		}
	}
	dieNow("see previous error", nil, errored)
	clientInd := *client
	switch action {
	case pwdKey:
		password()
	case useraddKey:
		useradd()
	case netconfKey:
		runScript(action, "python", clientInd)
	case configureKey:
		runScript(action, bash, clientInd)
	case reportsKey:
		runScript(action, bash, clientInd)
	case packKey:
		pack()
	case versionKey:
		fmt.Println(vers)
	default:
		dieNow("unknown command", nil, true)
	}
}

func pack() {
	file := []string{}
	file = append(file, "// this file is auto-generated, do NOT edit it")
	file = append(file, "package main")
	opts := goutils.NewCompressionOptions()
	opts.Length = 100
	file = append(file, "")
	file = append(file, "func init() {")
	for _, f := range embed {
		dieNow("missing file", nil, goutils.PathNotExists(f))
		fname := strings.Split(f, ".")[0]
		name := fmt.Sprintf("%sScript", fname)
		r, err := ioutil.ReadFile(f)
		die(err)
		c := goutils.MemoryStringCompression{}
		err = c.Compress(opts, string(r))
		die(err)
		file = append(file, fmt.Sprintf("\t// %s compression", name))
		file = append(file, fmt.Sprintf("\tvar %s []string", name))
		for _, l := range c.Content {
			file = append(file, fmt.Sprintf("\t%s = append(%s, `%s`)", name, name, l))
		}
		file = append(file, fmt.Sprintf("\tscripting[\"%s\"] = %s", fname, name))
	}
	file = append(file, "}")
	file = append(file, "")
	err := ioutil.WriteFile("generated.go", []byte(strings.Join(file, "\n")), 0644)
	die(err)
}

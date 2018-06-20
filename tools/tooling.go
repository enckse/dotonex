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
	if err != nil {
		goutils.WriteError("pwgen", err)
		panic("unable to generate password")
	}
	pass := strings.TrimSpace(strings.Join(out, ""))
	h := md4.New()
	h.Write(utf16le(pass))
	return pass, fmt.Sprintf("%x", string(h.Sum(nil)))
}

func password() {
	p, h := getPass()
	fmt.Println(fmt.Sprintf("\npassword:\n%s\n\nmd4:\n%s\n", p, h))
}

const (
	userDir     = "users/"
	resourceDir = "/usr/share/radiucal/"
)

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
	if len(user) == 0 {
		fmt.Println("invalid username")
		return
	}
	p, h := getPass()
	script := fmt.Sprintf(`
import users.__config__ as __config__
import users.common as common

u_obj = __config__.Assignment()
u_obj.password = '%s'
u_obj.vlan = None
u_obj.macs = None
`, h)
	ioutil.WriteFile(filepath.Join(userDir, fmt.Sprintf("user_%s.py", user)), []byte(script), 0644)
	fmt.Println(fmt.Sprintf("%s was create with a password of %s", user, p))
}

func copyFiles(files []string, source, destination string) {
	src := resourceDir
	if len(source) > 0 {
		src = filepath.Join(resourceDir, source)
	}
	for _, f := range files {
		file := filepath.Join(src, f)
		if goutils.PathNotExists(file) {
			continue
		}
		dest := filepath.Join(destination, f)
		if goutils.PathExists(dest) {
			os.Remove(dest)
		}
		cmd := fmt.Sprintf("cp %s %s", file, dest)
		_, err := goutils.RunBashCommand(cmd)
		if err != nil {
			fmt.Println("error copying resources")
			fmt.Println(cmd)
			fmt.Println(err)
		}
		if strings.HasPrefix(f, ".sh") {
			_, err := goutils.RunBashCommand(fmt.Sprintf("chmod u+x %s", file))
			if err != nil {
				fmt.Println("unable to set x bit on file")
				fmt.Println(file)
				fmt.Println(err)
			}
		}
	}
}

func bootstrap() {
	copyFiles([]string{"configure.sh", "reports.sh", "netconf.py"}, "", ".")
	copyFiles([]string{"__config__.py", "__init__.py"}, userDir, fmt.Sprintf("./%s", userDir))
}

func main() {
	cmd := flag.String("command", "", "command to execute")
	flag.Parse()
	errored := false
	for _, check := range []string{userDir, resourceDir} {
		if goutils.PathNotExists(check) {
			fmt.Println(fmt.Sprintf("missing required file/directory: %s", check))
		}
	}
	if errored {
		fmt.Println("see previous errors")
		return
	}
	switch *cmd {
	case "pwd":
		password()
	case "useradd":
		useradd()
	case "bootstrap":
		bootstrap()
	default:
		fmt.Println("unknown command")
	}
}

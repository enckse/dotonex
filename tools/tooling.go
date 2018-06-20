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

	"golang.org/x/crypto/md4"
	"github.com/epiphyte/goutils"
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

const userDir = "users/"

func useradd() {
	scanner := bufio.NewScanner(os.Stdin)
	text := ""
	fmt.Println("please input user name:")
    for scanner.Scan() {
		text = strings.TrimSpace(scanner.Text())
		break
    }
	user := ""
	for _,c := range text {
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

func bootstrap() {

}

func main() {
	cmd := flag.String("command", "", "command to execute")
	flag.Parse()
	if goutils.PathNotExists(userDir) {
		fmt.Println("can only be run from a configuration location")
		fmt.Println(fmt.Sprintf("missing: %s", userDir))
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

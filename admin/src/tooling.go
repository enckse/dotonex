package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/epiphyte/goutils/logger"
	"github.com/epiphyte/goutils/opsys"
	"github.com/epiphyte/goutils/random"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/md4"
)

const (
	userDir      = "users/"
	pwdKey       = "pwd"
	useraddKey   = "useradd"
	netconfKey   = "netconf"
	configureKey = "configure"
	versionKey   = "version"
	outputDir    = "bin/"
	eapUsers     = "eap_users"
	manifest     = "manifest"
	encKey       = "enc"
	decKey       = "dec"
	passwordFile = "passwords"
	userPass     = userDir + passwordFile
	userPassEnc  = userDir + passwordFile + ".keys"
	// input args
	userArg = "user"
	passArg = "pass"
)

type inputArgs struct {
	user string
	pass string
}

func (i *inputArgs) Set(param string) error {
	parts := strings.Split(param, "=")
	if len(parts) != 2 {
		return errors.New(fmt.Sprintf("invalid parameter: %s (not k=v)", param))
	}
	cleaned := strings.TrimSpace(parts[1])
	switch parts[0] {
	case userArg:
		i.user = cleaned
	case passArg:
		i.pass = cleaned
	default:
		return errors.New(fmt.Sprintf("unknown parameters: %s", param))
	}
	return nil
}

func (i *inputArgs) String() string {
	return fmt.Sprintf("%s = %s; %s = %s", userArg, i.user, passArg, i.pass)
}

var (
	vers        = "master"
	skipUserDir = make(map[string]struct{})
)

func init() {
	skipUserDir[versionKey] = struct{}{}
	skipUserDir[pwdKey] = struct{}{}
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

func getPass(param *inputArgs) (string, string, string) {
	pass := param.pass
	if len(pass) == 0 {
		random.SeedTime()
		pass = random.RandomString(64)
	}
	h := md4.New()
	h.Write(utf16le(pass))
	b, e := bcrypt.GenerateFromPassword([]byte(pass), 10)
	if e != nil {
		logger.Fatal("unable to bcrypt password", e)
	}
	return pass, fmt.Sprintf("%x", string(h.Sum(nil))), string(b)
}

func password(params *inputArgs) {
	p, h, b := getPass(params)
	fmt.Println(fmt.Sprintf("\npassword: %s\n\nmd4: %s\n\nbcrypt: %s\n", p, h, b))
}

func die(err error) {
	dieNow("unrecoverable error", err, err != nil)
}

func dieNow(message string, err error, now bool) {
	messaged := false
	if err != nil {
		messaged = true
		logger.WriteError(message, err)
	}
	if now {
		if !messaged {
			logger.WriteWarn(message)
		}
		os.Exit(1)
	}
}

func useradd(param *inputArgs) {
	text := param.user
	if len(text) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("please input user name:")
		for scanner.Scan() {
			text = strings.TrimSpace(scanner.Text())
			break
		}
	}
	user := ""
	for _, c := range text {
		if c >= 'a' && c <= 'z' {
			user += string(c)
		}
	}
	dieNow("invalid username", nil, len(user) == 0)
	p, h, b := getPass(param)
	script := fmt.Sprintf(`--user %s
object = network:Define(<device_type_str>, <device_id_str>)
object.Macs = {}
object:Assigned(<VLAN_NUMBER>)
`, user)
	ioutil.WriteFile(filepath.Join(userDir, fmt.Sprintf("user_%s.lua", user)), []byte(script), 0644)
	fmt.Println(fmt.Sprintf("%s was created with password: %s (bcrypt: %s)", user, p, b))
	opsys.RunBashCommand(fmt.Sprintf("echo '%s,%s' >> %s/passwords", user, h, userDir))
}

func main() {
	cmd := flag.String("command", "", "command to execute")
	client := flag.Bool("client", true, "running as client/admin")
	var parsed inputArgs
	flag.Var(&parsed, "parameters", "function parameters")
	flag.Parse()
	action := *cmd
	errored := false
	if _, ok := skipUserDir[action]; !ok {
		for _, check := range []string{userDir} {
			if opsys.PathNotExists(check) {
				errored = true
				fmt.Println(fmt.Sprintf("missing required file/directory: %s", check))
			}
		}
	}
	dieNow("see previous error", nil, errored)
	clientInd := *client
	switch action {
	case pwdKey:
		password(&parsed)
	case useraddKey:
		useradd(&parsed)
	case netconfKey:
		netconfRun()
	case encKey:
		encrypt(&parsed)
	case decKey:
		decrypt(&parsed)
	case configureKey:
		logger.WriteInfo("running configuration", vers)
		configure(clientInd)
	case versionKey:
		fmt.Println(vers)
	default:
		dieNow("unknown command", nil, true)
	}
}

func encDecInit(inFile string, args *inputArgs) ([]byte, cipher.Block) {
	if len(args.pass) == 0 {
		panic("invalid password")
	}
	key := []byte(args.pass)
	bytes, err := ioutil.ReadFile(inFile)
	die(err)
	b, err := aes.NewCipher(key)
	die(err)
	return bytes, b
}

func decrypt(args *inputArgs) {
	ciphertext, block := encDecInit(userPassEnc, args)
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	err := ioutil.WriteFile(userPass, ciphertext, 0644)
	die(err)
}

func encrypt(args *inputArgs) {
	bytes, block := encDecInit(userPass, args)
	ciphertext := make([]byte, aes.BlockSize+len(bytes))
	iv := ciphertext[:aes.BlockSize]
	_, err := io.ReadFull(rand.Reader, iv)
	die(err)

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], bytes)
	err = ioutil.WriteFile(userPassEnc, ciphertext, 0644)
	die(err)
}

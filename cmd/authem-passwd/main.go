package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
	"voidedtech.com/radiucal/internal/authem"
)

const (
	targetDir = "bin"
	alphaNum  = "abcdefghijklmnopqrstuvwxyz0123456789"
	userBase  = `username: %s
email:
vlans: []
perms:
    isradius: true
    ispeap: true
systems: []
`
)

var (
	vers = "master"
)

func newUserSecret(l int, pass bool) (string, error) {
	password := ""
	if pass {
		alphaNumeric := []rune(alphaNum)
		b := make([]rune, l)
		runes := len(alphaNumeric)
		for i := range b {
			b[i] = alphaNumeric[rand.Intn(runes)]
		}
		password = string(b)
	}
	return password, nil
}

func passwd(user, userFile, key, pwd string, force bool, length int) error {
	if authem.PathExists(userFile) {
		if !force {
			return fmt.Errorf("%s already exists, use force to overwrite", userFile)
		}
		if err := os.Remove(userFile); err != nil {
			return err
		}
	}
	authem.Info("")
	authem.Info(user)
	authem.Info("")
	authem.InfoDetail(userFile)
	password := pwd
	needPass := len(password) == 0
	if needPass {
		p, err := newUserSecret(length, needPass)
		if err != nil {
			return err
		}
		if needPass {
			password = p
		}
	}
	s := authem.Secret{
		UserName: user,
		Password: password,
		Fake:     false,
	}
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	enc, err := authem.Encrypt(key, string(b))
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(userFile, []byte(enc), 0644); err != nil {
		return err
	}
	userDef := filepath.Join(authem.UserDir, user+".yaml")
	authem.InfoDetail(userDef)
	if !authem.PathExists(userDef) {
		if err := ioutil.WriteFile(userDef, []byte(fmt.Sprintf(userBase, user)), 0644); err != nil {
			return err
		}
	}
	return nil
}

func showObject(userFile, key string) error {
	if !authem.PathExists(userFile) {
		return fmt.Errorf("user does not exist")
	}
	b, err := ioutil.ReadFile(userFile)
	if err != nil {
		return err
	}
	dec, err := authem.Decrypt(key, string(b))
	if err != nil {
		return err
	}

	authem.Info("")
	authem.Info("=======file========")
	authem.Info("")
	authem.Info(fmt.Sprintf("%s", dec))
	authem.Info("===================")
	authem.Info("")
	return nil
}

func updatePwd(user, pwd string, show, force bool, length int) error {
	k, err := authem.GetKey(false)
	if err != nil {
		return err
	}
	if len(user) == 0 {
		return fmt.Errorf("no user given")
	}
	userFile := filepath.Join(authem.SecretsDir, user+".yaml")
	if !show {
		if err := passwd(user, userFile, k, pwd, force, length); err != nil {
			return err
		}
	}
	if err := showObject(userFile, k); err != nil {
		return err
	}
	return nil
}

func main() {
	user := flag.String("user", "", "user to change")
	force := flag.Bool("force", false, "force change a user's secret")
	show := flag.Bool("show", false, "show the user's secrets, perform no changes")
	pwd := flag.String("password", "", "use this password")
	length := flag.Int("length", 64, "default password length")
	flag.Parse()
	authem.Version(vers)
	if err := updatePwd(*user, *pwd, *show, *force, *length); err != nil {
		authem.ExitNow("failed to perform operation", err)
	}
}

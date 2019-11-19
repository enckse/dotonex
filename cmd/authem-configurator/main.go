package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	yaml "gopkg.in/yaml.v2"
	"voidedtech.com/radiucal/internal/authem"
	"voidedtech.com/radiucal/internal/core"
)

const (
	manifest = "manifest"
	eap      = "eap_users"
	usersCfg = "config.yaml"
)

var (
	trackedFiles = [...]string{manifest, eap, usersCfg}
	vers         = "master"
)

type (
	// Config is the configurator specific handler
	Config struct {
		Verbose bool
		Key     string
		Cache   string
		Scripts []string
		Diffs   bool
		Deploy  bool
	}

	configuratorError struct {
	}
)

func (c *configuratorError) Error() string {
	return "configuration change"
}

func unchanged(cfg *Config, radius *authem.RADIUSConfig, users, rawConfig []byte) (bool, error) {
	if !core.PathExists(authem.TempDir) {
		if err := os.Mkdir(authem.TempDir, 0755); err != nil {
			return false, err
		}
	}
	valid := 0
	manifestBytes := []byte(strings.Join(radius.Manifest, "\n"))
	hostapdBytes := append(radius.Hostapd, []byte("\n")...)
	core.WriteInfo("[overall]")
	for _, f := range trackedFiles {
		if cfg.Verbose {
			core.WriteInfoDetail(f)
		}
		path := filepath.Join(authem.TempDir, f)
		if core.PathExists(path) {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return false, err
			}
			if err := ioutil.WriteFile(path+".prev", b, 0644); err != nil {
				return false, err
			}
			if cfg.Verbose {
				core.WriteInfoDetail("performing diff")
			}
			switch f {
			case manifest:
				if core.Compare(b, manifestBytes, cfg.Diffs) {
					valid++
				}
			case eap:
				if core.Compare(b, hostapdBytes, cfg.Diffs) {
					valid++
				}
			case usersCfg:
				if core.Compare(b, users, cfg.Diffs) {
					valid++
				}
			default:
				return false, fmt.Errorf("unknown track file: %s", f)
			}
		}
	}
	for k, v := range map[string][]byte{
		manifest: manifestBytes,
		eap:      hostapdBytes,
		usersCfg: users,
	} {
		paths := []string{authem.TempDir}
		if len(cfg.Cache) > 0 {
			paths = append(paths, cfg.Cache)
		}
		for _, f := range paths {
			p := filepath.Join(f, k)
			if cfg.Verbose {
				core.WriteInfoDetail(fmt.Sprintf("writing %s (%s)", k, p))
			}
			data := v
			if f != authem.TempDir && k == usersCfg && cfg.Deploy {
				data = rawConfig
			}
			if err := ioutil.WriteFile(p, data, 0644); err != nil {
				return false, err
			}
		}
	}
	return valid == len(trackedFiles), nil
}

func getConfig(f string, scripts []string, verbose bool) (*Config, error) {
	if !core.PathExists(f) {
		k, err := authem.GetKey(true)
		if err != nil {
			return nil, err
		}
		return &Config{
			Key:     k,
			Verbose: verbose,
			Scripts: scripts,
			Diffs:   true,
			Deploy:  false,
		}, nil
	}
	c := &Config{}
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	c.Verbose = c.Verbose || verbose
	c.Diffs = c.Diffs || verbose
	if len(scripts) > 0 {
		c.Scripts = append(c.Scripts, scripts...)
	}
	return c, nil
}

func hash(value string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
}

func configurate(cfg string, scripts []string, verbose, scripting bool) error {
	config, err := getConfig(cfg, scripts, verbose)
	if err != nil {
		return err
	}
	loader := authem.LoadingOptions{
		Verbose: config.Verbose,
		Sync:    config.Verbose,
		Key:     config.Key,
		NoKey:   config.Key == "",
	}
	vlans, err := loader.LoadVLANs()
	if err != nil {
		return err
	}
	systems, err := loader.LoadSystems()
	if err != nil {
		return err
	}
	secrets, err := loader.LoadSecrets()
	if err != nil {
		return err
	}
	users, radius, err := loader.LoadUsers(vlans, systems, secrets)
	if err != nil {
		return err
	}
	merged, err := authem.MergeRADIUS(radius)
	if err != nil {
		return err
	}
	u := authem.UserConfig{
		Users: users,
	}
	b, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	var postProcess []authem.BashRunner
	if len(config.Scripts) > 0 {
		for _, f := range config.Scripts {
			if len(f) == 0 {
				continue
			}
			var scriptables = authem.ToScriptable(u, vlans, systems, secrets)
			scriptBytes, err := ioutil.ReadFile(f)
			if err != nil {
				return err
			}
			tmpl, err := template.New("t").Parse(string(scriptBytes))
			if err != nil {
				return err
			}
			var buffer bytes.Buffer
			if err := tmpl.Execute(&buffer, scriptables); err != nil {
				return err
			}
			postProcess = append(postProcess, authem.BashRunner{buffer.Bytes(), filepath.Base(f)})
		}
	}
	raw, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	var newUsers []*authem.User
	for _, user := range u.Users {
		user.MD4 = hash(user.MD4)
		newUsers = append(newUsers, user)
	}
	u.Users = newUsers
	b, err = yaml.Marshal(u)
	if err != nil {
		return err
	}
	same, err := unchanged(config, merged, b, raw)
	if err != nil {
		return err
	}
	postScript := scripting
	if same {
		core.WriteInfoDetail("no changes")
	} else {
		core.WriteInfo("changes detected")
		postScript = true
	}
	if postScript && len(postProcess) > 0 {
		first := true
		for _, post := range postProcess {
			if len(post.Data) > 0 {
				if first {
					core.WriteInfo("[scripts]")
					first = false
				}
				core.WriteInfoDetail(post.Name)
				if err := post.Execute(); err != nil {
					return err
				}
			}
		}
	}
	if !same {
		return &configuratorError{}
	}
	return nil
}

func main() {
	cfg := flag.String("config", "/etc/authem/configurator.yaml", "config file (server mode)")
	verbose := flag.Bool("verbose", false, "enable verbose outputs")
	forceScript := flag.Bool("run-scripts", false, "run the scripts regardless of configuration changes")
	flag.Parse()
	remainders := flag.Args()
	core.Version(vers)
	err := configurate(*cfg, remainders, *verbose, *forceScript)
	if err != nil {
		if _, ok := err.(*configuratorError); !ok {
			core.ExitNow("unable to configure", err)
		}
		os.Exit(core.ExitSignal)
	}
}

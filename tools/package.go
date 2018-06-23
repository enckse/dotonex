package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/epiphyte/goutils"
)

func die(err error) {
	if err != nil {
		goutils.WriteError("unrecoverable error", err)
		os.Exit(1)
	}
}

func gen() {
	fmt.Println("// this file is auto-generated, do NOT edit it")
	fmt.Println("package main")
	fmt.Println("")
	fmt.Println("var (")
	m := make(map[string]*goutils.MemoryStringCompression)
	opts := goutils.NewCompressionOptions()
	opts.Length = 100
	for _, f := range []string{"configure.sh", "reports.sh", "netconf.py"} {
		name := strings.Split(f, ".")[0] + "Script"
		r, err := ioutil.ReadFile(f)
		die(err)
		c, err := goutils.MemoryStringCompress(opts, string(r))
		die(err)
		fmt.Println(fmt.Sprintf("    // %s script", name))
		fmt.Println(fmt.Sprintf("    %s = []string{}", name))
		m[name] = c
	}
	fmt.Println(")")
	fmt.Println("")
	fmt.Println("func init() {")
	for k, v := range m {
		fmt.Println(fmt.Sprintf("    // %s compression", k))
		for _, l := range v.Content {
			fmt.Println(fmt.Sprintf("    %s = append(%s, `%s`)", k, k, l))
		}
	}
	fmt.Println("}")
}

func main() {
	gen()
}

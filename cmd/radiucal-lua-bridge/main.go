package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
	"voidedtech.com/radiucal/internal/core"
)

const (
	includesFile = "includes.lua"
	userFile     = "user_"
	vlanFile     = "vlan_"
	luaExtension = ".lua"
	configDir    = "config/"
	// definition indicators
	idColumn    = "id"
	invalidVLAN = -1
	// assignment type
	ownType assignType = iota
	mabType
	logType
)

var (
	state        definition
	stateDisable = false
)

type (
	assignType int

	assignment struct {
		objectType assignType
		mac        string
		vlan       int
	}

	// Systems represents system definitions
	Systems struct {
		definition
		file    string
		user    string
		objects []*assignment
		desc    map[string]map[string][]string
	}

	// VLAN represents actual VLAN definitions
	VLAN struct {
		file     string
		number   int
		name     string
		initiate []string
		route    string
		net      string
		owner    string
		desc     string
		group    string
	}

	network struct {
		definition
		systems []*Systems
		vlans   []*VLAN
		refVLAN map[int]struct{}
		mab     map[string]struct{}
		own     map[string]struct{}
		login   map[string]struct{}
	}

	outputs struct {
		audits     [][]string
		manifest   []string
		eap        map[string]string
		eapKeys    []string
		trackLines map[string]struct{}
		sysTrack   map[string]map[string][]string
		sysCols    map[string]struct{}
	}
	definition interface {
		Segment(int, string, []string, string, string, string, string, string)
		Object(assignType, string, int)
		Describe(id, key, value string)
	}

	entityAdd func(mac string)

	entity struct {
		Macs     []string
		ID       string
		Typed    string
		Make     string
		Model    string
		XAttr    []string
		Revision string
	}

	segment struct {
		Name     string
		Num      int
		Initiate []string
		Route    string
		Net      string
		Owner    string
		Desc     string
		Group    string
	}
)

func fatal(message string, err error) {
	msg := message
	if err != nil {
		msg = fmt.Sprintf("%s: %v", message, err)
	}
	core.Fatal(msg, nil)
}

// Segment defines a new segment (VLAN)
func (n *network) Segment(num int, name string, initiate []string, route, net, owner, desc, group string) {
	if num < 0 || num > 4096 || strings.TrimSpace(name) == "" {
		fatal(fmt.Sprintf("invalid vlan definition: name or number is invalid (%s or %d)", name, num), nil)
	}
	v := &VLAN{}
	v.name = name
	v.initiate = initiate
	v.route = route
	v.net = net
	v.owner = owner
	v.desc = desc
	v.group = group
	v.number = num
	n.vlans = append(n.vlans, v)
}

func (o *outputs) trackLine(lineType, line string) {
	actual := fmt.Sprintf("%s -> %s", lineType, line)
	if _, ok := o.trackLines[actual]; ok {
		fatal(fmt.Sprintf("invalid config, detected duplicate object (%s -> %s)", lineType, line), nil)
	}
	o.trackLines[actual] = struct{}{}
}

func createOutputs(o *outputs, name, pass string, v *assignment, vlans map[int]string, isMAB, defaultUser bool) {
	vlan := vlans[v.vlan]
	m := v.mac
	objs := []string{name, vlan, m}
	audit := strings.Join(objs, ",")
	fqdn := name
	if !defaultUser {
		fqdn = fmt.Sprintf("%s.%s", vlan, name)
	}
	if isMAB {
		fqdn = m
	}
	phasing := ""
	key := "1"
	if isMAB {
		key = "2"
		upMAC := strings.ToUpper(m)
		phasing = fmt.Sprintf("\"%s\" MD5 \"%s\"", upMAC, upMAC)
	} else {
		phasing = fmt.Sprintf(`"%s" PEAP

"%s" MSCHAPV2 hash:%s [2]`, fqdn, fqdn, pass)
	}
	radius := fmt.Sprintf(`%s
radius_accept_attr=64:d:13
radius_accept_attr=65:d:6
radius_accept_attr=81:s:%d
`, phasing, v.vlan)
	useKey := fmt.Sprintf("%s %s", key, phasing)
	o.eapKeys = append(o.eapKeys, useKey)
	o.eap[useKey] = radius
	mani := fmt.Sprintf("%s.%s", fqdn, m)
	o.trackLine("manifest", mani)
	o.manifest = append(o.manifest, mani)
	if defaultUser {
		return
	}
	// audit doesn't support default because default is just a special normal user
	o.trackLine("audit", audit)
	o.audits = append(o.audits, objs)
}

func (o *outputs) eapWrite() {
	values := o.eapKeys
	sort.Strings(values)
	content := []string{}
	track := make(map[string]struct{})
	for _, k := range values {
		if _, ok := track[k]; ok {
			continue
		}
		track[k] = struct{}{}
		content = append(content, o.eap[k])
	}
	writeContent("eap_users", content)
}

func writeFile(file string, values []string) {
	lines := values
	sort.Strings(lines)
	writeContent(file, lines)
}

func rawWrite(file string, content []byte) {
	if err := ioutil.WriteFile(filepath.Join("bin/", file), content, 0644); err != nil {
		fatal("unable to write", err)
	}
}

func writeContent(file string, lines []string) {
	content := strings.Join(lines, "\n")
	rawWrite(file, []byte(content))
}

func (o *outputs) add(user string, desc map[string]map[string][]string) {
	for k := range desc {
		values := desc[k]
		values["user"] = []string{user}
		for a, val := range values {
			cur, ok := o.sysTrack[k]
			if !ok {
				cur = make(map[string][]string)
			}
			if _, ok := o.sysCols[a]; !ok {
				o.sysCols[a] = struct{}{}
			}
			exist, ok := cur[a]
			if ok {
				for _, v := range val {
					exist = append(exist, v)
				}
				cur[a] = exist
			} else {
				cur[a] = val
			}
			o.sysTrack[k] = cur
		}
	}
}

func vlanReports(vlans []*VLAN) {
	segments := [][]string{}
	segments = append(segments, []string{"cell", "segment", "lan", "vlan", "owner", "description"})
	diagram := []string{"digraph g {", "    size=\"6,6\";", "    node [color=lightblue2, style=filled];"}
	for _, vlan := range vlans {
		diagram = append(diagram, fmt.Sprintf("    \"%s\" [shape=\"record\"]", vlan.name))
		if vlan.route != "none" {
			diagram = append(diagram, fmt.Sprintf("    \"%s\" -> \"%s\" [color=red]", vlan.name, vlan.route))
		}
		if len(vlan.initiate) > 0 {
			for _, o := range vlan.initiate {
				diagram = append(diagram, fmt.Sprintf("    \"%s\" -> \"%s\"", vlan.name, o))
			}
		}
		entry := []string{vlan.group, vlan.name, vlan.net, fmt.Sprintf("%d", vlan.number), vlan.owner, vlan.desc}
		segments = append(segments, entry)
	}
	diagram = append(diagram, "}")
	writeContent("segment-diagram.dot", diagram)
	writeCSV("segments", segments, true)
}

func writeCSV(name string, content [][]string, hasHeader bool) {
	cnt := ""
	b := bytes.NewBufferString(cnt)
	w := csv.NewWriter(b)
	datum := content
	if hasHeader {
		datum = datum[1:]
		if err := w.Write(content[0]); err != nil {
			fatal(fmt.Sprintf("unable to write csv header: %s", name), err)
		}
	}

	for _, r := range datum {
		if err := w.Write(r); err != nil {
			fatal(fmt.Sprintf("unable to write csv row: %s", name), err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		fatal(fmt.Sprintf("unable to write csv: %s", name), err)
	}
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	out := []string{}
	if hasHeader {
		out = append(out, lines[0])
		lines = lines[1:]
	}
	sort.Strings(lines)
	for _, l := range lines {
		out = append(out, l)
	}
	rawWrite(fmt.Sprintf("%s.csv", name), []byte(strings.Join(out, "\n")))
}

func (o *outputs) systemInfo() [][]string {
	cols := []string{}
	for k := range o.sysCols {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	sysinfo := [][]string{}
	for k, v := range o.sysTrack {
		vals := []string{k}
		for _, c := range cols {
			if c == idColumn {
				continue
			}
			value, ok := v[c]
			if !ok {
				value = []string{}
			}
			uniques := make(map[string]struct{})
			colVals := []string{}
			for _, u := range value {
				if _, ok := uniques[u]; ok {
					continue
				}
				uniques[u] = struct{}{}
				colVals = append(colVals, u)
			}
			vals = append(vals, strings.Join(colVals, ";"))
		}
		sysinfo = append(sysinfo, vals)
	}
	sysinfo = append([][]string{cols}, sysinfo...)
	return sysinfo
}

func (n *network) process() {
	if len(n.vlans) > 0 && len(n.systems) > 0 {
		vlans := make(map[int]string)
		output := &outputs{}
		for _, v := range n.vlans {
			if _, ok := vlans[v.number]; ok {
				fatal(fmt.Sprintf("vlan redefined (%d %s)", v.number, v.name), nil)
			}
			vlans[v.number] = v.name
		}
		for k := range n.refVLAN {
			if _, ok := vlans[k]; !ok {
				fatal(fmt.Sprintf("%d -> unknown VLAN reference", k), nil)
			}
		}
		passes := readPasses()
		output.eap = make(map[string]string)
		output.trackLines = make(map[string]struct{})
		output.sysTrack = make(map[string]map[string][]string)
		output.sysCols = make(map[string]struct{})
		defaultVLANs := make(map[string]int)
		for _, s := range n.systems {
			s.user = strings.Replace(strings.Replace(s.file, userFile, "", -1), luaExtension, "", -1)
			if _, ok := defaultVLANs[s.user]; !ok {
				defaultVLANs[s.user] = invalidVLAN
			}
			output.add(s.user, s.desc)
			pass, ok := passes[s.user]
			if !ok {
				fatal(fmt.Sprintf("%s does not have a password", s.user), nil)
			}
			for _, o := range s.objects {
				if o.objectType == ownType {
					output.audits = append(output.audits, []string{s.user, "n/a", o.mac})
				} else {
					isMAB := o.objectType == mabType
					userGen := []bool{false}
					defVLAN := invalidVLAN
					if !isMAB {
						defVLAN, _ = defaultVLANs[s.user]
						if defVLAN < 0 {
							defVLAN = o.vlan
							defaultVLANs[s.user] = o.vlan
						}
						if defVLAN == o.vlan {
							userGen = append(userGen, true)
						}
					}
					for _, u := range userGen {
						createOutputs(output, s.user, pass, o, vlans, isMAB, u)
					}
				}
			}
		}
		fmt.Println("checks completed")
		writeCSV("audit", output.audits, false)
		writeFile("manifest", output.manifest)
		writeCSV("sysinfo", output.systemInfo(), true)
		vlanReports(n.vlans)
		output.eapWrite()
		return
	}
	fatal("invalid network definition (no systems or no vlans)", nil)
}

// Object defines a new object (system)
func (s *Systems) Object(t assignType, mac string, vlan int) {
	checkMAC(mac)
	o := &assignment{}
	o.objectType = t
	o.mac = mac
	o.vlan = vlan
	s.objects = append(s.objects, o)
}

// Describe is used to add more to a system
func (s *Systems) Describe(id, key, value string) {
	vals := make(map[string][]string)
	if v, ok := s.desc[id]; ok {
		vals = v
	}
	vals[key] = append(vals[key], value)
	s.desc[id] = vals
}

func fileToScript(fileName string) string {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		fatal(fmt.Sprintf("unable to write script: %s", fileName), err)
	}
	return string(b)
}

func getScript(fileName string) string {
	include := "-- included"
	options, err := ioutil.ReadDir(configDir)
	if err != nil {
		fatal("unable to search for inclusions", err)
	}
	for _, opt := range options {
		n := opt.Name()
		if n == includesFile || strings.HasSuffix(n, fmt.Sprintf(".%s", includesFile)) {
			cnt := fileToScript(filepath.Join(configDir, n))
			include = fmt.Sprintf("%s\n%s", include, cnt)
		}
	}
	script := fileToScript(fileName)
	return fmt.Sprintf("%s\n%s", include, script)
}

func isIn(mac string, in map[string]struct{}) {
	if _, ok := in[mac]; ok {
		fatal(fmt.Sprintf("%s must be assigned OR mab'd OR owned", mac), nil)
	}
}

func tracked(mac string, in, or, self map[string]struct{}, checkSelf bool) {
	isIn(mac, in)
	isIn(mac, or)
	if checkSelf {
		isIn(mac, self)
	}
	self[mac] = struct{}{}
}

func (n *network) addSystem(s *Systems) {
	for _, o := range s.objects {
		getVLAN := true
		switch o.objectType {
		case ownType:
			tracked(o.mac, n.mab, n.login, n.own, true)
			getVLAN = false
		case mabType:
			tracked(o.mac, n.own, n.login, n.mab, true)
		case logType:
			tracked(o.mac, n.mab, n.own, n.login, false)
		}
		if getVLAN {
			n.refVLAN[o.vlan] = struct{}{}
		}
	}
	n.systems = append(n.systems, s)
}

func main() {
	f, err := ioutil.ReadDir(configDir)
	if err != nil {
		fatal("unable to run netconf", err)
	}
	net := &network{}
	net.mab = make(map[string]struct{})
	net.own = make(map[string]struct{})
	net.login = make(map[string]struct{})
	net.refVLAN = make(map[int]struct{})
	for _, file := range f {
		name := file.Name()
		if (strings.HasPrefix(name, userFile) || strings.HasPrefix(name, vlanFile)) && strings.HasSuffix(name, luaExtension) {
			path := filepath.Join(configDir, name)
			if core.PathExists(path) {
				fmt.Println(fmt.Sprintf("reading %s", name))
				if strings.HasPrefix(name, userFile) {
					s := &Systems{}
					s.file = name
					s.desc = make(map[string]map[string][]string)
					buildSystems(path, s)
					net.addSystem(s)
				} else {
					n := &network{}
					buildSystems(path, n)
					for _, v := range n.vlans {
						v.file = name
						net.vlans = append(net.vlans, v)
					}
				}
			}
		}
	}
	net.process()
}

func readPasses() map[string]string {
	userPasses := make(map[string]string)
	path := filepath.Join(configDir, "passwords")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fatal(fmt.Sprintf("unable to read file: %s", path), err)
	}
	tracked := make(map[string]string)
	r := csv.NewReader(strings.NewReader(string(data)))
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fatal("unable to read passwords", err)
		}
		u := record[0]
		p := record[1]
		if _, ok := userPasses[u]; ok {
			fatal(fmt.Sprintf("user %s already has defined password", u), nil)
		}
		if _, ok := tracked[p]; ok {
			fatal(fmt.Sprintf("%s password is not unique", u), nil)
		}
		userPasses[u] = p
	}
	return userPasses
}

func checkMAC(mac string) {
	if len(mac) == 12 {
		valid := true
		for _, c := range mac {
			if (c >= 'a' && c <= 'f') || (c >= '0' && c <= '9') {
				continue
			}
			valid = false
			break
		}
		if valid {
			return
		}
	}
	fatal(fmt.Sprintf("invalid mac detected: %s", mac), nil)
}

func (e *entity) Disabled() {
	stateDisable = true
}

func (s *segment) Add() {
	state.Segment(s.Num, s.Name, s.Initiate, s.Route, s.Net, s.Owner, s.Desc, s.Group)
}

func (s *segment) Define(num int, name string) *segment {
	return &segment{Num: num, Name: name}
}

func (e *entity) Define(typed, id string) *entity {
	return &entity{ID: id, Typed: typed}
}

func (e *entity) Assign(vlan int, entities []*entity) {
	for _, entity := range entities {
		entity.Assigned(vlan)
	}
}

func (e *entity) describeItem(key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	state.Describe(e.ID, key, value)
}

func (e *entity) add(vlan int, adding entityAdd) {
	if stateDisable {
		return
	}
	for _, m := range e.Macs {
		adding(m)
	}
	e.describeItem("make", e.Make)
	e.describeItem("model", e.Model)
	e.describeItem("revision", e.Revision)
	e.describeItem("xattr", strings.Join(e.XAttr, ";"))
	e.describeItem("objType", e.Typed)
	e.describeItem(idColumn, e.ID)
}

func (e *entity) Assigned(vlan int) {
	e.add(vlan, func(mac string) {
		state.Object(logType, mac, vlan)
	})
}

func (e *entity) Owned() {
	e.add(invalidVLAN, func(mac string) {
		state.Object(ownType, mac, invalidVLAN)
	})
}

func (e *entity) Mabed(vlan int) {
	e.add(vlan, func(mac string) {
		state.Object(mabType, mac, vlan)
	})
}

func buildSystems(path string, s definition) {
	state = s
	stateDisable = false
	L := lua.NewState()
	defer L.Close()
	e := &entity{}
	seg := &segment{Num: invalidVLAN}
	L.SetGlobal("network", luar.New(L, e))
	L.SetGlobal("segments", luar.New(L, seg))
	script := getScript(path)
	if err := L.DoString(script); err != nil {
		fmt.Println(script)
		fatal("^ script error", err)
	}
}

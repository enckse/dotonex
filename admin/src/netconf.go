package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"voidedtech.com/goutils/logger"
	"voidedtech.com/goutils/opsys"
)

const (
	includesFile = "includes.lua"
	userFile     = "user_"
	vlanFile     = "vlan_"
	luaExtension = ".lua"
)

type assignment struct {
	objectType assignType
	mac        string
	vlan       int
}

type Systems struct {
	definition
	file    string
	user    string
	objects []*assignment
	desc    map[string]map[string][]string
}

type VLAN struct {
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

type network struct {
	definition
	systems []*Systems
	vlans   []*VLAN
	refVLAN map[int]struct{}
	mab     map[string]struct{}
	own     map[string]struct{}
	login   map[string]struct{}
}

type outputs struct {
	audits     []string
	manifest   []string
	eap        map[string]string
	eapKeys    []string
	trackLines map[string]struct{}
	sysTrack   map[string]map[string][]string
	sysCols    map[string]struct{}
}

func (n *network) Segment(num int, name string, initiate []string, route, net, owner, desc, group string) {
	if num < 0 || num > 4096 || strings.TrimSpace(name) == "" {
		logger.Fatal(fmt.Sprintf("invalid vlan definition: name or number is invalid (%s or %d)", name, num), nil)
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
		logger.Fatal(fmt.Sprintf("invalid config, detected duplicate object (%s -> %s)", lineType, line), nil)
	}
	o.trackLines[actual] = struct{}{}
}

func createOutputs(o *outputs, name, pass string, v *assignment, vlans map[int]string, isMAB, defaultUser bool) {
	vlan := vlans[v.vlan]
	m := v.mac
	audit := fmt.Sprintf("%s,%s,%s", name, vlan, m)
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
	o.audits = append(o.audits, audit)
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
	writeContent(eapUsers, content)
}

func writeFile(file string, values []string) {
	lines := values
	sort.Strings(lines)
	writeContent(file, lines)
}

func writeContent(file string, lines []string) {
	content := strings.Join(lines, "\n")
	err := ioutil.WriteFile(filepath.Join(outputDir, file), []byte(content), 0644)
	die(err)
}

func (s *outputs) add(user string, desc map[string]map[string][]string) {
	for k, _ := range desc {
		values := desc[k]
		values["user"] = []string{user}
		for a, val := range values {
			cur, ok := s.sysTrack[k]
			if !ok {
				cur = make(map[string][]string)
			}
			if _, ok := s.sysCols[a]; !ok {
				s.sysCols[a] = struct{}{}
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
			s.sysTrack[k] = cur
		}
	}
}

func vlanReports(vlans []*VLAN) {
	segments := [][]string{}
	segments = append(segments, []string{"cell", "segment", "lan", "vlan", "owner", "description"})
	segments = append(segments, []string{"---", "---", "---", "---", "---", "---"})
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
	markdown := []string{}
	for _, line := range segments {
		l := fmt.Sprintf("| %s |", strings.Join(line, " | "))
		markdown = append(markdown, l)
	}
	writeContent("segment-diagram.dot", diagram)
	writeContent("segments.md", markdown)
}

func (output *outputs) systemInfo() []string {
	cols := []string{}
	for k, _ := range output.sysCols {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	sysinfo := []string{}
	for k, v := range output.sysTrack {
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
		sysinfo = append(sysinfo, strings.Join(vals, ","))
	}
	sort.Strings(sysinfo)
	sysinfo = append([]string{strings.Join(cols, ",")}, sysinfo...)
	return sysinfo
}

func (n *network) process() {
	if len(n.vlans) > 0 && len(n.systems) > 0 {
		vlans := make(map[int]string)
		output := &outputs{}
		for _, v := range n.vlans {
			if _, ok := vlans[v.number]; ok {
				logger.Fatal(fmt.Sprintf("vlan redefined (%d %s)", v.number, v.name), nil)
			}
			vlans[v.number] = v.name
		}
		for k, _ := range n.refVLAN {
			if _, ok := vlans[k]; !ok {
				logger.Fatal(fmt.Sprintf("%d -> unknown VLAN reference", k), nil)
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
				logger.Fatal(fmt.Sprintf("%s does not have a password", s.user), nil)
			}
			for _, o := range s.objects {
				if o.objectType == ownType {
					output.audits = append(output.audits, fmt.Sprintf("%s,n/a,%s", s.user, o.mac))
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
		logger.WriteInfo("checks completed")
		writeFile("audit.csv", output.audits)
		writeFile(manifest, output.manifest)
		writeContent("sysinfo.csv", output.systemInfo())
		vlanReports(n.vlans)
		output.eapWrite()
		return
	}
	logger.Fatal("invalid network definition (no systems or no vlans)", nil)
}

func (s *Systems) Object(t assignType, mac string, vlan int) {
	checkMAC(mac)
	o := &assignment{}
	o.objectType = t
	o.mac = mac
	o.vlan = vlan
	s.objects = append(s.objects, o)
}

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
	die(err)
	return string(b)
}

func getScript(fileName string) string {
	include := ""
	p := filepath.Join(configDir, includesFile)
	if opsys.PathExists(p) {
		include = fileToScript(p)
	}
	script := fileToScript(fileName)
	return fmt.Sprintf("%s\n%s", include, script)
}

func isIn(mac string, in map[string]struct{}) {
	if _, ok := in[mac]; ok {
		logger.Fatal(fmt.Sprintf("%s must be assigned OR mab'd OR owned", mac), nil)
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

func netconfRun() {
	f, err := ioutil.ReadDir(configDir)
	die(err)
	net := &network{}
	net.mab = make(map[string]struct{})
	net.own = make(map[string]struct{})
	net.login = make(map[string]struct{})
	net.refVLAN = make(map[int]struct{})
	for _, file := range f {
		name := file.Name()
		if (strings.HasPrefix(name, userFile) || strings.HasPrefix(name, vlanFile)) && strings.HasSuffix(name, luaExtension) {
			path := filepath.Join(configDir, name)
			if opsys.PathExists(path) {
				logger.WriteInfo("reading", name)
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
	path := filepath.Join(configDir, passwordFile)
	data, err := ioutil.ReadFile(path)
	die(err)
	tracked := make(map[string]string)
	r := csv.NewReader(strings.NewReader(string(data)))
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		die(err)
		u := record[0]
		p := record[1]
		if _, ok := userPasses[u]; ok {
			logger.Fatal(fmt.Sprintf("user %s already has defined password", u), nil)
		}
		if _, ok := tracked[p]; ok {
			logger.Fatal(fmt.Sprintf("%s password is not unique", u), nil)
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
	logger.Fatal(fmt.Sprintf("invalid mac detected: %s", mac), nil)
}

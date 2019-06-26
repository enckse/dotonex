package main

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
	"voidedtech.com/goutils/logger"
)

type assignType int

const (
	idColumn    = "id"
	invalidVLAN = -1
	// assignment type
	ownType assignType = iota
	mabType
	logType
)

type definition interface {
	Segment(int, string, []string, string, string, string, string, string)
	Object(assignType, string, int)
	Describe(id, key, value string)
}

var (
	state        definition
	stateDisable = false
	flagged      = ""
)

type entityAdd func(mac string)

type entity struct {
	Macs     []string
	Id       string
	Typed    string
	Make     string
	Model    string
	Verbose  string
	Revision string
	describe bool
}

type segment struct {
	Name     string
	Num      int
	Initiate []string
	Route    string
	Net      string
	Owner    string
	Desc     string
	Group    string
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
	return &entity{Id: id, Typed: typed, describe: true}
}

func (e *entity) Tag(value string) {
	flagged = value
}

func (e *entity) Untag() {
	flagged = ""
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
	state.Describe(e.Id, key, value)
}

func (e *entity) add(vlan int, adding entityAdd) {
	if stateDisable {
		return
	}
	for _, m := range e.Macs {
		adding(m)
	}
	sysType := "0"
	if flagged != "" {
		e.describeItem("tagged", flagged)
	}
	if e.describe {
		e.describeItem("make", e.Make)
		e.describeItem("model", e.Model)
		e.describeItem("revision", e.Revision)
		e.describeItem("verbose", strings.Replace(e.Verbose, ",", " ", -1))
		e.describeItem("objType", e.Typed)
		e.describeItem(idColumn, e.Id)
		sysType = "1"
	}
	state.Describe(e.Id, "system_type", sysType)
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

func (e *entity) Own(id string, macs []string) {
	o := e.Define("n/a", id)
	o.Macs = macs
	o.describe = false
	o.Owned()
}

func buildSystems(path string, s definition) {
	state = s
	stateDisable = false
	flagged = ""
	L := lua.NewState()
	defer L.Close()
	e := &entity{}
	seg := &segment{Num: invalidVLAN}
	L.SetGlobal("network", luar.New(L, e))
	L.SetGlobal("segments", luar.New(L, seg))
	script := getScript(path)
	if err := L.DoString(script); err != nil {
		logger.WriteWarn(script)
		die(err)
	}
}

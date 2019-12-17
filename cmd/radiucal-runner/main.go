package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	"layeh.com/radius"
	"voidedtech.com/radiucal/internal/core"
	"voidedtech.com/radiucal/internal/plugins"
	"voidedtech.com/radiucal/internal/server"
)

var (
	vers          = "master"
	proxy         *net.UDPConn
	serverAddress *net.UDPAddr
	clients                   = make(map[string]*connection)
	clientLock    *sync.Mutex = new(sync.Mutex)
)

type (
	connection struct {
		client *net.UDPAddr
		server *net.UDPConn
	}
)

func newConnection(srv, cli *net.UDPAddr) *connection {
	conn := new(connection)
	conn.client = cli
	serverUDP, err := net.DialUDP("udp", nil, srv)
	if err != nil {
		core.WriteError("dial udp", err)
		return nil
	}
	conn.server = serverUDP
	return conn
}

func setup(hostport string, port int) error {
	proxyAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	proxyUDP, err := net.ListenUDP("udp", proxyAddr)
	if err != nil {
		return err
	}
	proxy = proxyUDP
	serverAddr, err := net.ResolveUDPAddr("udp", hostport)
	if err != nil {
		return err
	}
	serverAddress = serverAddr
	return nil
}

func runConnection(ctx *server.Context, conn *connection) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, err := conn.server.Read(buffer[0:])
		if err != nil {
			core.WriteError("unable to read buffer", err)
			continue
		}
		buffered := []byte(buffer[0:n])
		if !checkAuth("post", server.PostAuthorize, ctx, buffered, conn.client, conn.client) {
			continue
		}
		if _, err := proxy.WriteToUDP(buffer[0:n], conn.client); err != nil {
			core.WriteError("error relaying", err)
		}
	}
}

func checkAuth(name string, fxn server.AuthorizePacket, ctx *server.Context, b []byte, addr, client *net.UDPAddr) bool {
	auth := server.HandleAuth(fxn, ctx, b, addr, func(buffer []byte) {
		proxy.WriteToUDP(buffer, client)
	})
	if !auth {
		core.WriteDebug("client failed auth check", name)
	}
	return auth
}

func runProxy(ctx *server.Context) {
	if ctx.Debug {
		core.WriteInfo("=============WARNING==================")
		core.WriteInfo("debugging is enabled!")
		core.WriteInfo("dumps from debugging may contain secrets")
		core.WriteInfo("do NOT share debugging dumps")
		core.WriteInfo("=============WARNING==================")
		ctx.DebugDump()
	}
	var buffer [radius.MaxPacketLength]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if err != nil {
			core.WriteError("read from udp", err)
			continue
		}
		addr := cliaddr.String()
		clientLock.Lock()
		conn, found := clients[addr]
		if !found {
			conn = newConnection(serverAddress, cliaddr)
			if conn == nil {
				clientLock.Unlock()
				continue
			}
			clients[addr] = conn
			clientLock.Unlock()
			go runConnection(ctx, conn)
		} else {
			clientLock.Unlock()
		}
		buffered := []byte(buffer[0:n])
		if !checkAuth("pre", server.PreAuthorize, ctx, buffered, cliaddr, conn.client) {
			continue
		}
		if _, err := conn.server.Write(buffer[0:n]); err != nil {
			core.WriteError("unable to write to the server", err)
		}
	}
}

func account(ctx *server.Context) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if err != nil {
			core.WriteError("accounting udp error", err)
			continue
		}
		ctx.Account(core.NewClientPacket(buffer[0:n], cliaddr))
	}
}

func main() {
	core.Version(vers)
	p := core.Flags()
	b, err := ioutil.ReadFile(p.Config)
	if err != nil {
		core.Fatal("unable to load config", err)
	}
	conf := &core.Configuration{}
	if err := yaml.Unmarshal(b, conf); err != nil {
		core.Fatal("unable to parse config", err)
	}
	conf.Defaults(b)
	debug := conf.Debug || p.Debug
	core.ConfigureLogging(debug, p.Instance)
	if debug {
		conf.Dump()
	}
	to := 1814
	if !conf.Accounting {
		if conf.To > 0 {
			to = conf.To
		}
	}
	addr := fmt.Sprintf("%s:%d", conf.Host, to)
	if err := setup(addr, conf.Bind); err != nil {
		core.Fatal("proxy setup", err)
	}

	ctx := &server.Context{Debug: debug}
	ctx.FromConfig(conf.Dir, conf)
	pCtx := core.NewPluginContext(conf)
	for _, p := range conf.Plugins {
		core.WriteInfo("loading plugin", p)
		obj, err := plugins.LoadPlugin(p, pCtx)
		if err != nil {
			core.Fatal(fmt.Sprintf("unable to load plugin: %s", p), err)
		}
		if i, ok := obj.(core.Accounting); ok {
			ctx.AddAccounting(i)
		}
		if i, ok := obj.(core.Tracing); ok {
			ctx.AddTrace(i)
		}
		if i, ok := obj.(core.PreAuth); ok {
			ctx.AddPreAuth(i)
		}
		if i, ok := obj.(core.PostAuth); ok {
			ctx.AddPostAuth(i)
		}
		ctx.AddModule(obj)
	}

	if !conf.Internals.NoLogs {
		logBuffer := time.Duration(conf.Internals.Logs) * time.Second
		go func() {
			for {
				time.Sleep(logBuffer)
				if ctx.Debug {
					core.WriteDebug("flushing logs")
				}
				core.WritePluginMessages(conf.Log, p.Instance)
			}
		}()
	}
	wait := make(chan bool)
	if !conf.Internals.NoInterrupt {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				if ctx.Debug {
					core.WriteDebug("interrupt signal received")
				}
				wait <- true
			}
		}()
	}
	timeout := make(chan bool)
	check := time.Duration(conf.Internals.SpanCheck) * time.Hour
	end := time.Now().Add(time.Duration(conf.Internals.Lifespan) * time.Hour)
	go func() {
		for {
			time.Sleep(check)
			if ctx.Debug {
				core.WriteDebug("lifespan wakeup")
			}
			now := time.Now()
			if !core.IntegerIn(now.Hour(), conf.Internals.LifeHours) {
				continue
			}
			if now.After(end) {
				core.WriteInfo("lifespan reached")
				timeout <- true
			}
		}
	}()
	if conf.Accounting {
		core.WriteInfo("accounting mode")
		go account(ctx)
	} else {
		core.WriteInfo("proxy mode")
		go runProxy(ctx)
	}
	select {
	case <-wait:
		core.WriteInfo("signal...")
	case <-timeout:
		core.WriteInfo("lifecyle...")
	}
	core.WritePluginMessages(conf.Log, p.Instance)
	os.Exit(0)
}

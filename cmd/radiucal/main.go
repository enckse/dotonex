// Implementation of a UDP proxy

package main

import (
	"flag"
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
		age    time.Time
	}
)

func newConnection(srv, cli *net.UDPAddr) *connection {
	conn := new(connection)
	conn.client = cli
	conn.age = time.Now()
	serverUDP, err := net.DialUDP("udp", nil, srv)
	if core.LogError("dial udp", err) {
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
		if core.LogError("unable to read", err) {
			continue
		}
		buffered := []byte(buffer[0:n])
		if !checkAuth("post", server.PostAuthorize, ctx, buffered, conn.client, conn.client) {
			continue
		}
		_, err = proxy.WriteToUDP(buffer[0:n], conn.client)
		core.LogError("relaying", err)
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
		if core.LogError("read from udp", err) {
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
		_, err = conn.server.Write(buffer[0:n])
		core.LogError("server write", err)
	}
}

func account(ctx *server.Context) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if core.LogError("accounting udp error", err) {
			continue
		}
		ctx.Account(core.NewClientPacket(buffer[0:n], cliaddr))
	}
}

func dropConnections(debug bool, lifespan time.Duration) {
	clientLock.Lock()
	newClients := make(map[string]*connection)
	n := time.Now()
	for k, v := range clients {
		if v.age.Add(lifespan).Before(n) {
			if debug {
				core.WriteDebug(fmt.Sprintf("closing connection: %s", k))
			}
			v.server.Close()
			continue
		}
		newClients[k] = v
	}
	clients = newClients
	clientLock.Unlock()
}

func main() {
	core.WriteInfo(fmt.Sprintf("radiucal (%s)", vers))
	var cfg = flag.String("config", "/etc/radiucal/radiucal.conf", "Configuration file")
	var instance = flag.String("instance", "", "Instance name")
	var debugging = flag.Bool("debug", false, "debugging")
	flag.Parse()
	b, err := ioutil.ReadFile(*cfg)
	if err != nil {
		core.Fatal("unable to load config", err)
	}
	conf := &core.Configuration{}
	if err := yaml.Unmarshal(b, conf); err != nil {
		core.Fatal("unable to parse config", err)
	}
	conf.Defaults(b)
	debug := conf.Debug || *debugging
	logOpts := core.NewLogOptions()
	logOpts.Debug = debug
	logOpts.Info = true
	logOpts.Instance = *instance
	core.ConfigureLogging(logOpts)
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
	err = setup(addr, conf.Bind)
	if core.LogError("proxy setup", err) {
		panic("unable to proceed")
	}

	ctx := &server.Context{Debug: debug}
	ctx.FromConfig(conf.Dir, conf)
	pCtx := core.NewPluginContext(conf)
	pCtx.Logs = conf.Log
	pCtx.Lib = conf.Dir
	pCtx.Instance = *instance
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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if ctx.Debug {
				core.WriteDebug("reload signal received")
			}
			clientLock.Lock()
			for _, v := range clients {
				v.server.Close()
			}
			clients = make(map[string]*connection)
			clientLock.Unlock()
			ctx.Reload()
			core.WritePluginMessages(pCtx.Logs, pCtx.Instance)
		}
	}()

	logBuffer := time.Duration(conf.LogBuffer) * time.Second
	go func() {
		for {
			time.Sleep(logBuffer)
			if ctx.Debug {
				core.WriteDebug("flushing logs")
			}
			core.WritePluginMessages(pCtx.Logs, pCtx.Instance)
		}
	}()

	connAge := time.Duration(conf.ConnAge) * time.Hour
	go func() {
		for {
			time.Sleep(connAge)
			if ctx.Debug {
				core.WriteDebug("cleaning up old connections")
			}
			dropConnections(ctx.Debug, connAge)
		}
	}()

	if conf.Accounting {
		core.WriteInfo("accounting mode")
		account(ctx)
	} else {
		runProxy(ctx)
	}
}
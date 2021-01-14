package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	"layeh.com/radius"
	"voidedtech.com/dotonex/internal"
	"voidedtech.com/dotonex/internal/modules"
)

var (
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
		internal.WriteError("dial udp", err)
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

func runConnection(ctx *internal.Context, conn *connection) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, err := conn.server.Read(buffer[0:])
		if err != nil {
			internal.WriteError("unable to read buffer", err)
			continue
		}
		if _, err := proxy.WriteToUDP(buffer[0:n], conn.client); err != nil {
			internal.WriteError("error relaying", err)
		}
	}
}

func runProxy(ctx *internal.Context) {
	if ctx.Debug {
		internal.WriteInfo("=============WARNING==================")
		internal.WriteInfo("debugging is enabled!")
		internal.WriteInfo("dumps from debugging may contain secrets")
		internal.WriteInfo("do NOT share debugging dumps")
		internal.WriteInfo("=============WARNING==================")
		ctx.DebugDump()
	}
	var buffer [radius.MaxPacketLength]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if err != nil {
			internal.WriteError("read from udp", err)
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
		auth := internal.HandleAuth(internal.PreAuthorize, ctx, buffered, cliaddr, func(buffer []byte) {
			proxy.WriteToUDP(buffer, conn.client)
		})
		if !auth {
			internal.WriteDebug("client failed preauth check")
			continue
		}
		if _, err := conn.server.Write(buffer[0:n]); err != nil {
			internal.WriteError("unable to write to the server", err)
		}
	}
}

func account(ctx *internal.Context) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if err != nil {
			internal.WriteError("accounting udp error", err)
			continue
		}
		ctx.Account(internal.NewClientPacket(buffer[0:n], cliaddr))
	}
}

func main() {
	p := internal.Flags()
	internal.ConfigureLogging(p.Debug, p.Instance)
	b, err := ioutil.ReadFile(filepath.Join(p.Directory, p.Instance+internal.InstanceConfig))
	if err != nil {
		internal.Fatal("unable to load config", err)
	}
	conf := &internal.Configuration{}
	if err := yaml.Unmarshal(b, conf); err != nil {
		internal.Fatal("unable to parse config", err)
	}
	conf.Defaults(b)
	if p.Debug {
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
		internal.Fatal("proxy setup", err)
	}

	ctx := &internal.Context{Debug: p.Debug}
	ctx.FromConfig(conf.Dir, conf)
	internal.WriteInfo("loading plugins")
	var plugin internal.Module
	if conf.Accounting {
		plugin = &modules.AccountingModule
	} else {
		plugin = &modules.ProxyModule
	}

	if err := plugin.Setup(internal.NewPluginContext(conf)); err != nil {
		internal.Fatal("unable to create internal plugin", err)
	}

	if i, ok := plugin.(internal.Accounting); ok {
		ctx.AddAccounting(i)
	}
	if i, ok := plugin.(internal.Tracing); ok {
		ctx.AddTrace(i)
	}
	if i, ok := plugin.(internal.PreAuth); ok {
		ctx.AddPreAuth(i)
	}
	ctx.AddModule(plugin)

	if !conf.Internals.NoLogs {
		logBuffer := time.Duration(conf.Internals.Logs) * time.Second
		go func() {
			for {
				time.Sleep(logBuffer)
				if ctx.Debug {
					internal.WriteDebug("flushing logs")
				}
				internal.WritePluginMessages(conf.Log, p.Instance)
			}
		}()
	}
	interrupt := make(chan bool)
	if !conf.Internals.NoInterrupt {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				if ctx.Debug {
					internal.WriteDebug("interrupt signal received")
				}
				interrupt <- true
			}
		}()
	}
	lifecycle := make(chan bool)
	check := time.Duration(conf.Internals.SpanCheck) * time.Hour
	end := time.Now().Add(time.Duration(conf.Internals.Lifespan) * time.Hour)
	go func() {
		for {
			time.Sleep(check)
			if ctx.Debug {
				internal.WriteDebug("lifespan wakeup")
			}
			now := time.Now()
			if !internal.IntegerIn(now.Hour(), conf.Internals.LifeHours) {
				if ctx.Debug {
					internal.WriteDebug("lifespan in quiet hours")
				}
				continue
			}
			if now.After(end) {
				lifecycle <- true
			}
		}
	}()
	if conf.Accounting {
		internal.WriteInfo("accounting mode")
		go account(ctx)
	} else {
		internal.WriteInfo("proxy mode")
		if conf.Configurator.Static {
			internal.SetAllowed(conf.Configurator.Payload)
		} else {
			if err := internal.Manage(conf); err != nil {
				internal.Fatal("unable to setup management of configs", err)
			}
		}
		go runProxy(ctx)
	}
	select {
	case <-interrupt:
		internal.WriteInfo("interrupt...")
	case <-lifecycle:
		internal.WriteInfo("lifecyle...")
	}
	internal.WritePluginMessages(conf.Log, p.Instance)
	os.Exit(0)
}

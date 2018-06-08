// Implementation of a UDP proxy

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	"github.com/epiphyte/goutils"
	"github.com/epiphyte/radiucal/core"
	"github.com/epiphyte/radiucal/server"
	"layeh.com/radius"
)

var vers = "master"

var (
	proxy         *net.UDPConn
	serverAddress *net.UDPAddr
	clients       map[string]*connection = make(map[string]*connection)
	clientLock    *sync.Mutex            = new(sync.Mutex)
)

type connection struct {
	client *net.UDPAddr
	server *net.UDPConn
}

func newConnection(srv, cli *net.UDPAddr) *connection {
	conn := new(connection)
	conn.client = cli
	serverUdp, err := net.DialUDP("udp", nil, srv)
	if core.LogError("dial udp", err) {
		return nil
	}
	conn.server = serverUdp
	return conn
}

func setup(hostport string, port int) error {
	proxyAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	proxyUdp, err := net.ListenUDP("udp", proxyAddr)
	if err != nil {
		return err
	}
	proxy = proxyUdp
	serverAddr, err := net.ResolveUDPAddr("udp", hostport)
	if err != nil {
		return err
	}
	serverAddress = serverAddr
	return nil
}

func runConnection(conn *connection) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, err := conn.server.Read(buffer[0:])
		if core.LogError("unable to read", err) {
			continue
		}
		_, err = proxy.WriteToUDP(buffer[0:n], conn.client)
		core.LogError("relaying", err)
	}
}

func runProxy(ctx *server.Context) {
	if ctx.Debug {
		goutils.WriteInfo("=============WARNING==================")
		goutils.WriteInfo("debugging is enabled!")
		goutils.WriteInfo("dumps from debugging may contain secrets")
		goutils.WriteInfo("do NOT share debugging dumps")
		goutils.WriteInfo("=============WARNING==================")
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
			go runConnection(conn)
		} else {
			clientLock.Unlock()
		}
		buffered := []byte(buffer[0:n])
		preauthed := server.HandleAuth(server.PreAuthorize, ctx, buffered, cliaddr, func(b []byte) {
			proxy.WriteToUDP(b, conn.client)
		})
		if !preauthed {
			goutils.WriteDebug("client failed preauth")
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

func main() {
	goutils.WriteInfo(fmt.Sprintf("radiucal (%s)", vers))
	var config = flag.String("config", "/etc/radiucal/radiucal.conf", "Configuration file")
	var instance = flag.String("instance", "", "Instance name")
	var debugging = flag.Bool("debug", false, "debugging")
	flag.Parse()
	conf, err := goutils.LoadConfig(*config, goutils.NewConfigSettings())
	if err != nil {
		goutils.WriteError("unable to load config", err)
		panic("invalid/unable to load config")
	}
	debug := conf.GetTrue("debug") || *debugging
	logOpts := goutils.NewLogOptions()
	logOpts.Debug = debug
	logOpts.Info = true
	logOpts.Instance = *instance
	goutils.ConfigureLogging(logOpts)
	host := conf.GetStringOrDefault("host", "localhost")
	var to int = 1814
	accounting := conf.GetTrue("accounting")
	defaultBind := 1812
	if accounting {
		defaultBind = 1813
	} else {
		to, err = conf.GetIntOrDefault("to", 1814)
		if err != nil {
			goutils.WriteError("unable to get bind-to", err)
			panic("cannot bind to another socket")
		}
	}
	bind, err := conf.GetIntOrDefault("bind", defaultBind)
	if err != nil {
		goutils.WriteError("unable to bind address", err)
		panic("unable to bind")
	}
	addr := fmt.Sprintf("%s:%d", host, to)
	err = setup(addr, bind)
	if core.LogError("proxy setup", err) {
		panic("unable to proceed")
	}

	lib := conf.GetStringOrDefault("dir", "/var/lib/radiucal/")
	ctx := &server.Context{Debug: debug}
	ctx.FromConfig(lib, conf)
	mods := conf.GetArrayOrEmpty("plugins")
	pCtx := core.NewPluginContext(conf)
	pCtx.Logs = filepath.Join(lib, "log")
	pCtx.Lib = lib
	pCtx.Instance = *instance
	pPath := filepath.Join(lib, "plugins")
	for _, p := range mods {
		oPath := filepath.Join(pPath, fmt.Sprintf("%s.rd", p))
		goutils.WriteInfo("loading plugin", p, oPath)
		obj, err := core.LoadPlugin(oPath, pCtx)
		if err != nil {
			goutils.WriteError(fmt.Sprintf("unable to load plugin: %s", p), err)
			panic("unable to load plugin")
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
		ctx.AddModule(obj)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			clientLock.Lock()
			clients = make(map[string]*connection)
			clientLock.Unlock()
			ctx.Reload()
		}
	}()

	if accounting {
		goutils.WriteInfo("accounting mode")
		account(ctx)
	} else {
		runProxy(ctx)
	}
}

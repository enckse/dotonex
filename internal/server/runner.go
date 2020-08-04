package server

import (
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	"layeh.com/radius"
	"voidedtech.com/radiucal/internal/core"
)

var (
	authClients             = make(map[string]*connection)
	authLock    *sync.Mutex = new(sync.Mutex)
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

func setup(hostport string, port int) (*net.UDPConn, *net.UDPAddr, error) {
	proxyAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, nil, err
	}
	proxyUDP, err := net.ListenUDP("udp", proxyAddr)
	if err != nil {
		return nil, nil, err
	}
	serverAddr, err := net.ResolveUDPAddr("udp", hostport)
	if err != nil {
		return nil, nil, err
	}
	return proxyUDP, serverAddr, nil
}

func runConnection(proxy *net.UDPConn, ctx *Context, conn *connection) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, err := conn.server.Read(buffer[0:])
		if err != nil {
			core.WriteError("unable to read buffer", err)
			continue
		}
		buffered := []byte(buffer[0:n])
		if !checkAuth("post", PostAuthorize, ctx, buffered, conn.client, conn.client) {
			continue
		}
		if _, err := proxy.WriteToUDP(buffer[0:n], conn.client); err != nil {
			core.WriteError("error relaying", err)
		}
	}
}

func checkAuth(name string, fxn AuthorizePacket, ctx *Context, b []byte, addr, client *net.UDPAddr) bool {
	auth := HandleAuth(fxn, ctx, b, addr)
	if !auth {
		core.WriteDebug("client failed auth check", name)
	}
	return auth
}

func runProxy(proxy *net.UDPConn, server *net.UDPAddr, ctx *Context) {
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
		authLock.Lock()
		conn, found := authClients[addr]
		if !found {
			conn = newConnection(server, cliaddr)
			if conn == nil {
				authLock.Unlock()
				continue
			}
			authClients[addr] = conn
			authLock.Unlock()
			go runConnection(proxy, ctx, conn)
		} else {
			authLock.Unlock()
		}
		buffered := []byte(buffer[0:n])
		if !checkAuth("pre", PreAuthorize, ctx, buffered, cliaddr, conn.client) {
			continue
		}
		if _, err := conn.server.Write(buffer[0:n]); err != nil {
			core.WriteError("unable to write to the server", err)
		}
	}
}

func account(proxy *net.UDPConn, ctx *Context) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if err != nil {
			core.WriteError("accounting udp error", err)
			continue
		}
		ctx.Account(NewClientPacket(buffer[0:n], cliaddr))
	}
}

// Run the serving of proxy endpoints
func Run(config string) {
	b, err := ioutil.ReadFile(config)
	if err != nil {
		core.Fatal("unable to load config", err)
	}
	conf := &Configuration{}
	if err := yaml.Unmarshal(b, conf); err != nil {
		core.Fatal("unable to parse config", err)
	}
	core.ConfigureLogging(conf.Debug)
	if conf.Debug {
		conf.Dump()
	}
	go serveEndpoint(conf.Auth, conf, false)
	go serveEndpoint(conf.Acct, conf, true)
	logBuffer := time.Duration(conf.Logging.Flush) * time.Second
	go func() {
		for {
			time.Sleep(logBuffer)
			if conf.Debug {
				core.WriteDebug("flushing logs")
			}
			WriteModuleMessages(conf.Logging.Dir)
		}
	}()

	for {
		time.Sleep(30 * time.Second)
	}
}

func serveEndpoint(endpoint Endpoint, config *Configuration, accounting bool) {
	if !endpoint.Enable {
		return
	}
	addr := fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
	proxy, address, err := setup(addr, endpoint.To)
	if err != nil {
		core.Fatal("proxy setup", err)
	}

	ctx := &Context{Debug: config.Debug}
	pCtx := NewModuleContext(config)
	for _, p := range endpoint.Mods {
		core.WriteInfo("loading module", p)
		obj, err := LoadModule(p, pCtx)
		if err != nil {
			core.Fatal(fmt.Sprintf("unable to load module: %s", p), err)
		}
		ctx.AddModule(obj)
	}

	if accounting {
		account(proxy, ctx)
	} else {
		runProxy(proxy, address, ctx)
	}
}

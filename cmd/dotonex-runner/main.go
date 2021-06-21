package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	"layeh.com/radius"
	"voidedtech.com/dotonex/internal/core"
	"voidedtech.com/dotonex/internal/runner"
)

var (
	proxy         *net.UDPConn
	serverAddress *net.UDPAddr
	clients       = make(map[string]*connection)
	clientLock    = new(sync.Mutex)
	erroredCount  = 0
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

func runConnection(ctx *runner.Context, conn *connection) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, err := conn.server.Read(buffer[0:])
		if err != nil {
			core.WriteError("unable to read buffer", err)
			continue
		}
		if _, err := proxy.WriteToUDP(buffer[0:n], conn.client); err != nil {
			core.WriteError("error relaying", err)
		}
	}
}

func runProxy(ctx *runner.Context) {
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
				erroredCount++
				clientLock.Unlock()
				continue
			}
			erroredCount = 0
			clients[addr] = conn
			clientLock.Unlock()
			go runConnection(ctx, conn)
		} else {
			clientLock.Unlock()
		}
		buffered := []byte(buffer[0:n])
		auth := runner.HandlePreAuth(ctx, buffered, cliaddr, func(buffer []byte) {
			if _, err := proxy.WriteToUDP(buffer, conn.client); err != nil {
				core.WriteError("unable to proxy", err)
			}
		})
		if !auth {
			core.WriteDebug("client failed preauth check")
			continue
		}
		if _, err := conn.server.Write(buffer[0:n]); err != nil {
			core.WriteError("unable to write to the server", err)
		}
	}
}

func account(ctx *runner.Context) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if err != nil {
			core.WriteError("accounting udp error", err)
			continue
		}
		ctx.Account(runner.NewClientPacket(buffer[0:n], cliaddr))
	}
}

func monitorCount(debug bool, title string, c chan bool, state core.MonitorState, callback func() int) {
	if state.Check > 0 {
		core.WriteInfo(fmt.Sprintf("performing %s management", title))
		check := time.Duration(state.Check) * time.Minute
		go func() {
			for {
				time.Sleep(check)
				if debug {
					core.WriteDebug(fmt.Sprintf("%s check", title))
				}
				clientLock.Lock()
				total := callback()
				clientLock.Unlock()
				if debug {
					core.WriteDebug(fmt.Sprintf("%s: %d", title, total))
				}
				if total > state.Count {
					c <- true
				}
			}
		}()
	}
}

func main() {
	p := core.Flags()
	core.ConfigureLogging(p.Debug, p.Instance)
	b, err := os.ReadFile(filepath.Join(p.Directory, p.Instance+core.InstanceConfig))
	if err != nil {
		core.Fatal("unable to load config", err)
	}
	conf := &core.Configuration{}
	if err := yaml.Unmarshal(b, conf); err != nil {
		core.Fatal("unable to parse config", err)
	}
	if conf.Preload != nil && len(conf.Preload) > 0 {
		combined := &core.Configuration{}
		for _, preload := range conf.Preload {
			core.WriteInfo(fmt.Sprintf("preloading: %s", preload))
			loaded, err := os.ReadFile(preload)
			if err != nil {
				core.Fatal(fmt.Sprintf("unable to preload: %s", preload), err)
			}
			if err := yaml.Unmarshal(loaded, combined); err != nil {
				core.Fatal(fmt.Sprintf("unable to parse yaml: %s", preload), err)
			}
		}
		conf = combined
		if err := yaml.Unmarshal(b, conf); err != nil {
			core.Fatal("unable to overlay root config", err)
		}
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
		core.Fatal("proxy setup", err)
	}

	ctx := &runner.Context{Debug: p.Debug}
	ctx.FromConfig(conf)

	if !conf.Internals.NoLogs {
		logBuffer := time.Duration(conf.Internals.Logs) * time.Second
		go func() {
			for {
				time.Sleep(logBuffer)
				if ctx.Debug {
					core.WriteDebug("flushing logs")
				}
				runner.WritePluginMessages(conf.Log, p.Instance)
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
					core.WriteDebug("interrupt signal received")
				}
				interrupt <- true
			}
		}()
	}
	lifecycle := make(chan bool)
	if conf.Internals.LifeCheck > 0 {
		core.WriteInfo("performing lifespan management")
		check := time.Duration(conf.Internals.LifeCheck) * time.Hour
		end := time.Now().Add(time.Duration(conf.Internals.Lifespan) * time.Hour)
		go func() {
			for {
				time.Sleep(check)
				if ctx.Debug {
					core.WriteDebug("lifespan wakeup")
				}
				now := time.Now()
				if !core.IntegerIn(now.Hour(), conf.Internals.LifeHours) {
					if ctx.Debug {
						core.WriteDebug("lifespan in quiet hours")
					}
					continue
				}
				if now.After(end) {
					lifecycle <- true
				}
			}
		}()
	}
	maxConns := make(chan bool)
	clientFailures := make(chan bool)
	if conf.Accounting {
		core.WriteInfo("accounting mode")
		go account(ctx)
	} else {
		core.WriteInfo("proxy mode")
		if conf.Compose.Static {
			runner.SetAllowed(conf.Compose.Payload)
		} else {
			if err := runner.Manage(conf); err != nil {
				core.Fatal("unable to setup management of configs", err)
			}
		}
		monitorCount(ctx.Debug, "max connection", maxConns, conf.Internals.MaxConnections, func() int {
			return len(clients)
		})
		monitorCount(ctx.Debug, "client errors", clientFailures, conf.Internals.ClientFailures, func() int {
			return erroredCount
		})
		go runProxy(ctx)
	}
	select {
	case <-clientFailures:
		core.WriteInfo("client failures...")
	case <-maxConns:
		core.WriteInfo("connections...")
	case <-interrupt:
		core.WriteInfo("interrupt...")
	case <-lifecycle:
		core.WriteInfo("lifecyle...")
	}
	runner.WritePluginMessages(conf.Log, p.Instance)
	if conf.Quit.Wait {
		core.WriteInfo("shutting down")
		cleanup := make(chan bool)
		timedOut := make(chan bool)
		go func() {
			clientLock.Lock()
			runner.ShutdownModules()
			runner.ShutdownValidator()
			cleanup <- true
		}()
		if conf.Quit.Timeout > 0 {
			go func() {
				time.Sleep(time.Duration(conf.Quit.Timeout) * time.Second)
				timedOut <- true
			}()
		}
		select {
		case <-cleanup:
			core.WriteInfo("cleanup completed")
		case <-timedOut:
			core.WriteInfo("cleanup timed out")
		}
	}
	os.Exit(0)
}

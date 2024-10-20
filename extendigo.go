package main

import (
	"io"
	"net/rpc"
	"os"
	"os/exec"

	nio "github.com/mrnavastar/assist/io"
)

type Loader struct {
	Plugins []Plugin
}

type Plugin struct {
	client  *rpc.Client
	server  *rpc.Server
	Id      string
	Version string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	cmd     *exec.Cmd
}

func NewPlugin(id string) (p Plugin) {
	p.Id = id
	return p
}

func (p *Plugin) Register(a any) Plugin {
	p.server.Register(a)
	return *p
}

func (p *Plugin) Shutdown() error {
	return p.cmd.Cancel()
}

func (p *Plugin) Start() Plugin {
	clientIn := os.NewFile(uintptr(4), "cin")
	clientOut := os.NewFile(uintptr(5), "cout")
	serverIn := os.NewFile(uintptr(6), "sin")
	serverOut := os.NewFile(uintptr(7), "sout")

	p.client = rpc.NewClient(nio.RWCloser{clientIn, clientOut})
	p.server = rpc.NewServer()
	p.server.ServeConn(nio.RWCloser{serverIn, serverOut})
	return *p
}

func (p *Plugin) Call(method string, args any, reply any) {
	p.client.Call(method, args, reply)
}

func (l *Loader) Load(pluginPath string) (plugin Plugin, err error) {
	clientIn, _, err := os.Pipe()
	if err != nil {
		return plugin, err
	}
	_, clientOut, err := os.Pipe()
	if err != nil {
		return plugin, err
	}
	serverIn, _, err := os.Pipe()
	if err != nil {
		return plugin, err
	}
	_, serverOut, err := os.Pipe()
	if err != nil {
		return plugin, err
	}

	plugin.client = rpc.NewClient(nio.RWCloser{clientIn, clientOut})
	plugin.server = rpc.NewServer()
	plugin.server.ServeConn(nio.RWCloser{serverIn, serverOut})

	plugin.cmd = exec.Command(pluginPath)
	plugin.Stdin = plugin.cmd.Stdin
	plugin.Stdout = plugin.cmd.Stdout
	plugin.Stderr = plugin.cmd.Stderr
	plugin.cmd.ExtraFiles = []*os.File{clientIn, clientOut, serverIn, serverOut}
	return plugin, plugin.cmd.Start()
}

package main

import (
	"fmt"
	"github.com/niftynei/glightning/glightning"
	"github.com/niftynei/glightning/jrpc2"
	"log"
	"os"
)

type Hello struct{}

func (h *Hello) New() interface{} {
	return &Hello{}
}

func (h *Hello) Name() string {
	return "say-hi"
}

func (h *Hello) Call() (jrpc2.Result, error) {
	name := plugin.GetOptionValue("name")
	return fmt.Sprintf("Howdy %s!", name), nil
}

var lightning *glightning.Lightning
var plugin *glightning.Plugin

// ok, let's try plugging this into c-lightning
func main() {
	plugin = glightning.NewPlugin(onInit)
	lightning = glightning.NewLightning()

	registerOptions(plugin)
	registerMethods(plugin)
	registerSubscriptions(plugin)
	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func secondMain(config *glightning.Config) {
	// set up lightning .. thing
	lightning.StartUp(config.RpcFile, config.LightningDir)
	channels, _ := lightning.ListChannels()
	log.Printf("You know about %d channels", len(channels))
}

func registerOptions(p *glightning.Plugin) {
	p.RegisterOption(glightning.NewOption("name", "How you'd like to be called", "Mary"))
}

func registerMethods(p *glightning.Plugin) {
	rpcHello := glightning.NewRpcMethod(&Hello{}, "Say hello!")
	rpcHello.LongDesc = `Send a message! Name is set
by the 'name' option, passed in at startup `
	p.RegisterMethod(rpcHello)
}

func OnConnect(c *glightning.ConnectEvent) {
	log.Printf("connect called: id %s at %s:%d", c.PeerId, c.Address.Addr, c.Address.Port)
}

func OnDisconnect(d *glightning.DisconnectEvent) {
	log.Printf("disconnect called for %s\n", d.PeerId)
}

func registerSubscriptions(p *glightning.Plugin) {
	p.SubscribeConnect(OnConnect)
	p.SubscribeDisconnect(OnDisconnect)
}

func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("successfully init'd! %s\n", config.RpcFile)
	secondMain(config)
}

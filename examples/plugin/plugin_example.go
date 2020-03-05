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
	name, err := plugin.GetOptionValue("name")
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Howdy %s\n\n", name), nil
}

type PrettyHello struct{}

func (h *PrettyHello) New() interface{} {
	return &PrettyHello{}
}

func (h *PrettyHello) Name() string {
	return "pretty-hi"
}

func (h *PrettyHello) Call() (jrpc2.Result, error) {
	name, err := plugin.GetOptionValue("name")
	if err != nil {
		return nil, err
	}
	result := &struct {
		Result string `json:"hi"`
		// If you want the result returned to be 'simply' formatted
		// return a field called "format-hint" set to `FormatSimple`
		FormatHint string `json:"format-hint,omitempty"`
	}{
		Result:     fmt.Sprintf("\n\tHowdy %s!\n\n", name),
		FormatHint: glightning.FormatSimple,
	}

	return &result, nil
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
	plugin.RegisterHooks(&glightning.Hooks{
		PeerConnected:  OnPeerConnect,
		RpcCommand:     OnRpcCommand,
		InvoicePayment: OnInvoicePayment,
		OpenChannel:    OnOpenChannel,
		HtlcAccepted:   OnHtlcAccepted,
	})
	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

// This is called after the plugin starts up successfully
func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("successfully init'd! %s\n", config.RpcFile)

	// Here's how you'd use the config's lightning-dir to
	//   start up an RPC client for the node.
	lightning.StartUp(config.RpcFile, config.LightningDir)
	channels, _ := lightning.ListChannels()
	log.Printf("You know about %d channels", len(channels))

	// If 'initialization' happened at the same time as the plugin starts,
	//   then the 'startup' will be true. Otherwise, you've been
	//   initialized by the 'dynamic' plugin command.
	//   Note that you have to opt-into dynamic startup.
	log.Printf("Is this initial node startup? %v\n", config.Startup)
}

func registerOptions(p *glightning.Plugin) {
	p.RegisterOption(glightning.NewOption("name", "How you'd like to be called", "Mary"))
}

func registerMethods(p *glightning.Plugin) {
	rpcHello := glightning.NewRpcMethod(&Hello{}, "Say hello!")
	rpcHello.LongDesc = `Send a message! Name is set
by the 'name' option, passed in at startup `
	rpcHello.Category = "utility"
	p.RegisterMethod(rpcHello)

	rpcPretty := glightning.NewRpcMethod(&PrettyHello{}, "Pretty hello!")
	rpcPretty.LongDesc = `Pretty much the same as 'say-hi' but formatted`
	rpcPretty.Category = "utility"
	p.RegisterMethod(rpcPretty)
}

/* Subscription Examples */
func OnConnect(c *glightning.ConnectEvent) {
	log.Printf("connect called: id %s at %s:%d", c.PeerId, c.Address.Addr, c.Address.Port)
}

func OnDisconnect(d *glightning.DisconnectEvent) {
	log.Printf("disconnect called for %s\n", d.PeerId)
}

func OnInvoicePaid(payment *glightning.Payment) {
	log.Printf("invoice paid for amount %s with preimage %s", payment.MilliSatoshis, payment.PreImage)
}

func OnChannelOpened(co *glightning.ChannelOpened) {
	log.Printf("channel opened with %s for %s. is locked? %v", co.PeerId, co.FundingSatoshis, co.FundingLocked)
}

func OnWarning(warn *glightning.Warning) {
	log.Printf("Got a warning!! %s", warn.Log)
}

func OnSendPaySuccess(sps *glightning.SendPaySuccess) {
	log.Printf("send pay success!")
}

func OnSendPayFailure(spf *glightning.SendPayFailure) {
	log.Printf("send pay failure!")
}

func registerSubscriptions(p *glightning.Plugin) {
	p.SubscribeConnect(OnConnect)
	p.SubscribeDisconnect(OnDisconnect)
	p.SubscribeInvoicePaid(OnInvoicePaid)
	p.SubscribeChannelOpened(OnChannelOpened)
	p.SubscribeWarnings(OnWarning)
	p.SubscribeSendPaySuccess(OnSendPaySuccess)
	p.SubscribeSendPayFailure(OnSendPayFailure)
}

/* Hook Examples */
func OnPeerConnect(event *glightning.PeerConnectedEvent) (*glightning.PeerConnectedResponse, error) {
	log.Printf("peer connected called\n")

	// See also: Disconnect(errMsg)
	return event.Continue(), nil
}

func OnDbWrite(event *glightning.DbWriteEvent) (bool, error) {
	log.Printf("dbwrite called\n")
	// You can also return false
	return true, nil
}

func OnInvoicePayment(event *glightning.InvoicePaymentEvent) (*glightning.InvoicePaymentResponse, error) {
	log.Printf("invoice payment called\n")

	// See also: Fail(failureCode)
	return event.Continue(), nil
}

func OnOpenChannel(event *glightning.OpenChannelEvent) (*glightning.OpenChannelResponse, error) {
	log.Printf("openchannel called\n")

	// See also: Reject(errorMsg)
	return event.Continue(), nil
}

func OnHtlcAccepted(event *glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
	log.Printf("htlc_accepted called\n")

	// See also: Fail(failureCode), Resolve(paymentKey)
	return event.Continue(), nil
}

func OnRpcCommand(event *glightning.RpcCommandEvent) (*glightning.RpcCommandResponse, error) {
	cmd := event.X.Cmd
	id, _ := cmd.Id()
	log.Printf("command %s called id %s", cmd.MethodName, id)

	return event.Continue(), nil
}

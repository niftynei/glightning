package main

import (
	"github.com/niftynei/glightning/glightning"
	"log"
	"os"
)

func main() {
	plugin := glightning.NewPlugin(onInit)
	plugin.RegisterHooks(&glightning.Hooks{
		OpenChannel: OnOpenChannel,
	})

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("successfully init'd! %s\n", config.RpcFile)
}

func OnOpenChannel(event *glightning.OpenChannelEvent) (*glightning.OpenChannelResponse, error) {
	log.Printf("openchannel called\n")

	// I want to close to this address, please
	addr := "bcrt1q8q4xevfuwgsm7mxant8aadz50xt67768s4332d"

	return event.ContinueWithCloseTo(addr), nil
}

package main

import (
	"github.com/niftynei/glightning/glightning"
	"log"
	"os"
)

func main() {
	plugin := glightning.NewPlugin(onInit)
	plugin.RegisterHooks(&glightning.Hooks{
		DbWrite: OnDbWrite,
	})

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("successfully init'd! %s\n", config.RpcFile)
}

func OnDbWrite(event *glightning.DbWriteEvent) (*glightning.DbWriteResponse, error) {
	log.Printf("dbwrite called %d", event.DataVersion)
	// can also call 'Fail'
	return event.Continue(), nil
}

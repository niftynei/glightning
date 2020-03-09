package main

import (
	"github.com/niftynei/glightning/glightning"
	"log"
	"math/big"
	"os"
)

func main() {
	plugin := glightning.NewPlugin(onInit)

	// we use big-int because they're .. very big ints
	var a, b, c big.Int
	a.Exp(big.NewInt(2), big.NewInt(101), nil)
	plugin.AddInitFeatures(a.Bytes())

	b.Exp(big.NewInt(2), big.NewInt(103), nil)
	plugin.AddNodeFeatures(b.Bytes())

	c.Exp(big.NewInt(2), big.NewInt(105), nil)
	plugin.AddInvoiceFeatures(c.Bytes())

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("successfully init'd! %s\n", config.RpcFile)

}

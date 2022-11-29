package main

import (
	"log"
	"math/big"
	"os"

	"github.com/niftynei/glightning/glightning"
)

func main() {
	plugin := glightning.NewPlugin(onInit)

	// we use big-int because they're .. very big ints
	var a, b, c, d big.Int
	a.Exp(big.NewInt(2), big.NewInt(101), nil)
	plugin.AddInitFeatures(a.Bytes())

	b.Exp(big.NewInt(2), big.NewInt(103), nil)
	plugin.AddNodeFeatures(b.Bytes())

	c.Exp(big.NewInt(2), big.NewInt(105), nil)
	plugin.AddInvoiceFeatures(c.Bytes())

	d.Exp(big.NewInt(2), big.NewInt(107), nil)
	plugin.AddChannelFeatures(d.Bytes())

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]glightning.Option, config *glightning.Config) {
	log.Printf("successfully init'd! %s\n", config.RpcFile)

}

package main

import (
	"github.com/niftynei/glightning/gbitcoin"
	"github.com/niftynei/glightning/glightning"
	"log"
	"os"
	"strconv"
)

var btc *gbitcoin.Bitcoin

func main() {

	plugin := glightning.NewPlugin(onInit)
	bb := glightning.NewBitcoinBackend(plugin)

	bb.RegisterGetUtxOut(GetUtxOut)
	bb.RegisterGetChainInfo(GetChainInfo)
	bb.RegisterGetFeeRate(GetFeeRate)
	bb.RegisterSendRawTransaction(SendRawTx)
	bb.RegisterGetRawBlockByHeight(BlockByHeight)

	// register options for bitcoind auth, port + directory
	// matches existing options so we can swap this out with
	// bcli seamlessly
	plugin.RegisterNewOption("bitcoin-datadir", "Bitcoind data directory", "~/.bitcoin")
	plugin.RegisterNewOption("bitcoin-rpcport", "RPC port number for bitcoind", "8332")
	plugin.RegisterNewOption("bitcoin-rpcuser", "Username for RPC auth", "btcuser")
	plugin.RegisterNewOption("bitcoin-rpcpassword", "Authentication for RPC", "btcpass")

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("successfully init'd! %s\n", config.RpcFile)

	// btc info is set via plugin 'options'
	btcDir := options["bitcoin-datadir"]
	btcUser := options["bitcoin-rpcuser"]
	btcPass := options["bitcoin-rpcpassword"]
	btcPort, err := strconv.ParseUint(options["bitcoin-rpcport"], 10, 32)
	if err != nil {
		log.Fatal(err)
	}

	// default startup
	btc = gbitcoin.NewBitcoin(btcUser, btcPass)
	btc.StartUp("", btcDir, uint(btcPort))
}

func GetUtxOut(txid string, vout uint32) (string, string, error) {
	log.Printf("called getutxo")

	txout, err := btc.GetTxOut(txid, vout)
	if err != nil {
		log.Printf("there's an error! %s", err)
		return "", "", err
	}

	// gettxout sends back an empty if there's nothing found,
	// which is ok, we just need to pass this info along
	if txout == nil {
		return "", "", nil
	}

	log.Printf("txout is %v", txout)
	amt := glightning.ConvertBtc(txout.Value)
	return amt.AsMsat(), txout.ScriptPubKey.Hex, nil
}

func GetChainInfo() (*glightning.Btc_ChainInfo, error) {
	log.Printf("called getchaininfo")

	c, err := btc.GetChainInfo()
	if err != nil {
		log.Printf("error returned: %s", err)
		return nil, err
	}

	return &glightning.Btc_ChainInfo{
		Chain:                c.Chain,
		HeaderCount:          c.Headers,
		BlockCount:           c.Blocks,
		InitialBlockDownload: c.InitialBlockDownload,
	}, nil
}

func GetFeeRate(blocks uint32, mode string) (uint64, error) {
	log.Printf("called getfeerate %d %s", blocks, mode)

	fees, err := btc.EstimateFee(blocks, mode)
	if err != nil {
		return 0, err
	}

	// feerate's response must be denominated in satoshi per kilo-vbyte
	return fees.SatPerKb(), nil
}

func SendRawTx(tx string) error {
	txid, err := btc.SendRawTx(tx)

	log.Printf("called sendrawtransaction %s(%s)", txid, err)

	return err
}

// return a blockhash, block, error
func BlockByHeight(height uint32) (string, string, error) {
	log.Printf("called blockbyheight %d", height)

	hash, err := btc.GetBlockHash(height)
	if err != nil {
		return "", "", err
	}

	raw, err := btc.GetRawBlock(hash)
	if err != nil {
		return "", "", err
	}

	return hash, raw, nil
}

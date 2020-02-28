package glightning_test

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/niftynei/glightning/gbitcoin"
	"github.com/niftynei/glightning/glightning"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"syscall"
	"testing"
	"time"
)

const defaultTimeout int = 10

func check(t *testing.T, err error) {
	if err != nil {
		debug.PrintStack()
		t.Fatal(err)
	}
}

func advanceChain(n *Node, btc *gbitcoin.Bitcoin, numBlocks uint) error {
	timeout := time.Now().Add(time.Duration(defaultTimeout) * time.Second)

	info, _ := n.rpc.GetInfo()
	blockheight := info.Blockheight
	mineBlocks(numBlocks, btc)
	for {
		info, _ = n.rpc.GetInfo()
		if info.Blockheight >= uint(blockheight)+numBlocks {
			return nil
		}
		if time.Now().After(timeout) {
			return errors.New("timed out waiting for chain to advance")
		}
	}
}

func waitForChannelActive(n *Node, scid string) error {
	timeout := time.Now().Add(time.Duration(defaultTimeout) * time.Second)
	for {
		chans, _ := n.rpc.GetChannel(scid)
		// both need to be active
		active := 0
		for i := 0; i < len(chans); i++ {
			if chans[i].IsActive {
				active += 1
			}
		}
		if active == 2 {
			return nil
		}
		if time.Now().After(timeout) {
			return errors.New(fmt.Sprintf("timed out waiting for scid %s", scid))
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func waitForChannelReady(t *testing.T, from, to *Node) {
	timeout := time.Now().Add(time.Duration(defaultTimeout) * time.Second)
	for {
		info, err := to.rpc.GetInfo()
		check(t, err)
		peer, err := from.rpc.GetPeer(info.Id)
		check(t, err)
		if peer.Channels == nil {
			t.Fatal("no channels for peer")
		}
		if peer.Channels[0].State == "CHANNELD_NORMAL" {
			return
		}
		if time.Now().After(timeout) {
			t.Fatal("timed out waiting for channel normal")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func Init(t *testing.T) (string, string, int, *gbitcoin.Bitcoin) {
	// let's put it in a temporary directory
	testDir, err := ioutil.TempDir("", "gltests-")
	check(t, err)
	dataDir, _, btcPort, btc := SpinUpBitcoind(t, testDir)
	return testDir, dataDir, btcPort, btc
}

func CleanUp(testDir string) {
	os.Remove(testDir)
}

type BNode struct {
	rpc  *gbitcoin.Bitcoin
	dir  string
	port uint
	pid  uint
}

// Returns a bitcoin node w/ RPC client
func SpinUpBitcoind(t *testing.T, dir string) (string, int, int, *gbitcoin.Bitcoin) {
	// make some dirs!
	bitcoindDir := filepath.Join(dir, "bitcoind")
	err := os.Mkdir(bitcoindDir, os.ModeDir|0755)
	check(t, err)

	bitcoinPath, err := exec.LookPath("bitcoind")
	check(t, err)
	btcPort, err := getPort()
	check(t, err)
	btcUser := "btcuser"
	btcPass := "btcpass"
	bitcoind := exec.Command(bitcoinPath, "-regtest",
		fmt.Sprintf("-datadir=%s", bitcoindDir),
		"-server", "-logtimestamps", "-nolisten",
		fmt.Sprintf("-rpcport=%d", btcPort),
		"-addresstype=bech32", "-logtimestamps", "-txindex",
		fmt.Sprintf("-rpcpassword=%s", btcPass),
		fmt.Sprintf("-rpcuser=%s", btcUser))

	bitcoind.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}
	log.Printf("starting %s on %d...", bitcoinPath, btcPort)
	err = bitcoind.Start()
	check(t, err)
	log.Printf(" bitcoind started (%d)!\n", bitcoind.Process.Pid)

	btc := gbitcoin.NewBitcoin(btcUser, btcPass)
	btc.SetTimeout(uint(2))
	// Waits til bitcoind is up
	btc.StartUp("", bitcoindDir, uint(btcPort))
	// Go ahead and run 50 blocks
	addr, err := btc.GetNewAddress(gbitcoin.Bech32)
	check(t, err)
	_, err = btc.GenerateToAddress(addr, 101)
	check(t, err)
	return bitcoindDir, bitcoind.Process.Pid, btcPort, btc
}

func (node *Node) waitForLog(phrase string, timeoutSec int) error {
	logfile, _ := os.Open(filepath.Join(node.dir, "log"))
	defer logfile.Close()

	timeout := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	reader := bufio.NewReader(logfile)
	for timeoutSec == 0 || time.Now().Before(timeout) {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(100 * time.Millisecond)
			} else {
				return err
			}
		}
		m, err := regexp.MatchString(phrase, line)
		if err != nil {
			return err
		}
		if m {
			return nil
		}
	}

	return errors.New(fmt.Sprintf("Unable to find \"%s\" in %s/log", phrase, node.dir))
}

func getPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

type Node struct {
	rpc *glightning.Lightning
	dir string
}

func LnNode(testDir, dataDir string, btcPort int, name string) (*Node, error) {
	var err error
	lightningPath := os.Getenv("LIGHTNINGD_PATH")
	if lightningPath == "" {
		// assume it's just a thing i can call
		lightningPath, err = exec.LookPath("lightningd")
		if err != nil {
			return nil, err
		}
	}

	lightningdDir := filepath.Join(testDir, fmt.Sprintf("lightningd-%s", name))
	err = os.Mkdir(lightningdDir, os.ModeDir|0755)
	if err != nil {
		return nil, err
	}

	port, err := getPort()
	if err != nil {
		return nil, err
	}
	lightningd := exec.Command(lightningPath,
		fmt.Sprintf("--lightning-dir=%s", lightningdDir),
		fmt.Sprintf("--bitcoin-datadir=%s", dataDir),
		"--network=regtest", "--funding-confirms=3",
		fmt.Sprintf("--addr=localhost:%d", port),
		fmt.Sprintf("--bitcoin-rpcport=%d", btcPort),
		"--log-file=log",
		"--log-level=debug",
		"--bitcoin-rpcuser=btcuser",
		"--bitcoin-rpcpassword=btcpass",
		"--dev-fast-gossip",
		"--dev-bitcoind-poll=1",
		"--allow-deprecated-apis=false")

	lightningd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}
	log.Printf("starting %s on %d...", lightningPath, port)
	if err := lightningd.Start(); err != nil {
		return nil, err
	}

	time.Sleep(200 * time.Millisecond)

	lightningdDir = filepath.Join(lightningdDir, "regtest")
	node := &Node{nil, lightningdDir}
	log.Printf("starting node in %s\n", lightningdDir)
	err = node.waitForLog("Server started with public key", 30)
	if err != nil {
		return nil, err
	}
	log.Printf(" lightningd started (%d)!\n", lightningd.Process.Pid)

	node.rpc = glightning.NewLightning()
	node.rpc.StartUp("lightning-rpc", lightningdDir)

	return node, nil
}

func short(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

func TestBitcoinProxy(t *testing.T) {
	short(t)

	testDir, _, _, btc := Init(t)
	defer CleanUp(testDir)
	addr, err := btc.GetNewAddress(gbitcoin.Bech32)
	check(t, err)
	assert.NotNil(t, addr)

}

func TestConnectRpc(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	l1Info, _ := l1.rpc.GetInfo()
	assert.Equal(t, 1, len(l1Info.Binding))

	l1Addr := l1Info.Binding[0]
	l2, err := LnNode(testDir, dataDir, btcPid, "two")
	peerId, err := l2.rpc.Connect(l1Info.Id, l1Addr.Addr, uint(l1Addr.Port))
	check(t, err)
	assert.Equal(t, peerId, l1Info.Id)
}

func TestConfigsRpc(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	configs, err := l1.rpc.ListConfigs()
	check(t, err)
	assert.Equal(t, "lightning-rpc", configs["rpc-file"])
	assert.Equal(t, false, configs["always-use-proxy"])

	network, err := l1.rpc.GetConfig("network")
	check(t, err)
	assert.Equal(t, "regtest", network)
}

func TestHelpRpc(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	commands, err := l1.rpc.Help()
	check(t, err)
	if len(commands) == 0 {
		t.Error("No help commands returned")
	}

	cmd, err := l1.rpc.HelpFor("help")
	check(t, err)
	assert.Equal(t, "help [command]", cmd.NameAndUsage)
}

func TestSignCheckMessage(t *testing.T) {
	short(t)

	msg := "hello there"
	testDir, dataDir, btcPid, _ := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)
	l2, err := LnNode(testDir, dataDir, btcPid, "two")
	check(t, err)

	l1Info, _ := l1.rpc.GetInfo()

	signed, err := l1.rpc.SignMessage(msg)
	check(t, err)

	v, err := l2.rpc.CheckMessageVerify(msg, signed.ZBase, l1Info.Id)
	check(t, err)
	assert.True(t, v)
}

func TestListTransactions(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, btc := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	err = fundNode("1.0", l1, btc)
	check(t, err)
	err = fundNode("1.0", l1, btc)
	check(t, err)
	waitToSync(l1)
	trans, err := l1.rpc.ListTransactions()
	check(t, err)
	assert.Equal(t, len(trans), 2)
}

func connect(l1, l2 *Node) error {
	l2Info, _ := l2.rpc.GetInfo()
	_, err := l1.rpc.Connect(l2Info.Id, l2Info.Binding[0].Addr, uint(l2Info.Binding[0].Port))
	return err
}

func fundNode(amount string, n *Node, b *gbitcoin.Bitcoin) error {
	addr, err := n.rpc.NewAddr()
	if err != nil {
		return err
	}
	_, err = b.SendToAddress(addr, amount)
	if err != nil {
		return err
	}

	return mineBlocks(1, b)
}

// n is number of blocks to mine
func mineBlocks(n uint, b *gbitcoin.Bitcoin) error {
	addr, err := b.GetNewAddress(gbitcoin.Bech32)
	if err != nil {
		return err
	}
	_, err = b.GenerateToAddress(addr, n)
	if err != nil {
		return err
	}
	return nil
}

func waitToSync(n *Node) {
	for {
		info, _ := n.rpc.GetInfo()
		if info.IsLightningdSync() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestCreateOnion(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	hops := []glightning.Hop{
		glightning.Hop{
			Pubkey:  "02eec7245d6b7d2ccb30380bfbe2a3648cd7a942653f5aa340edcea1f283686619",
			Payload: "000000000000000000000000000000000000000000000000000000000000000000",
		},
		glightning.Hop{
			Pubkey:  "0324653eac434488002cc06bbfb7f10fe18991e35f9fe4302dbea6d2353dc0ab1c",
			Payload: "140101010101010101000000000000000100000001",
		},
		glightning.Hop{
			Pubkey:  "027f31ebc5462c1fdce1b737ecff52d37d75dea43ce11c74d25aa297165faa2007",
			Payload: "fd0100000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f606162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6a7a8a9aaabacadaeafb0b1b2b3b4b5b6b7b8b9babbbcbdbebfc0c1c2c3c4c5c6c7c8c9cacbcccdcecfd0d1d2d3d4d5d6d7d8d9dadbdcdddedfe0e1e2e3e4e5e6e7e8e9eaebecedeeeff0f1f2f3f4f5f6f7f8f9fafbfcfdfeff",
		},
		glightning.Hop{
			Pubkey:  "032c0b7cf95324a07d05398b240174dc0c2be444d96b159aa6c7f7b1e668680991",
			Payload: "140303030303030303000000000000000300000003",
		},
		glightning.Hop{
			Pubkey:  "02edabbd16b41c8371b92ef2f04c1185b4f03b6dcd52ba9b78d9d7c89c8f221145",
			Payload: "000404040404040404000000000000000400000004000000000000000000000000",
		},
	}

	privateHash := "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	resp, err := l1.rpc.CreateOnion(hops, privateHash, "")
	check(t, err)

	assert.Equal(t, len(resp.SharedSecrets), len(hops))
	assert.Equal(t, len(resp.Onion), 2*1366)

	privateHash = "4242424242424242424242424242424242424242424242424242424242424242"
	sessionKey := "4141414141414141414141414141414141414141414141414141414141414141"
	resp, err = l1.rpc.CreateOnion(hops, privateHash, sessionKey)
	check(t, err)

	firstHop := glightning.FirstHop{
		ShortChannelId: "100x1x1",
		Direction:      1,
		AmountMsat:     "1000sat",
		Delay:          8,
	}

	// Ideally we'd do a 'real' send onion but we don't
	// need to know if c-lightning works, only that the API
	// functions correctly...
	_, err = l1.rpc.SendOnionWithDetails(resp.Onion, firstHop, privateHash, "label", resp.SharedSecrets, nil)

	// ... which means we expect an error back!
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "204:No connection to first peer found")
}

func getShortChannelId(t *testing.T, node1, node2 *Node) string {
	info, err := node2.rpc.GetInfo()
	check(t, err)
	peer, err := node1.rpc.GetPeer(info.Id)
	check(t, err)
	if peer == nil || len(peer.Channels) == 0 {
		t.Fatal(fmt.Sprintf("peer %s not found", info.Id))
	}
	return peer.Channels[0].ShortChannelId
}

// ok, now let's check the plugin subs+hooks etc
func TestPlugins(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, btc := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	plugins, err := l1.rpc.ListPlugins()
	check(t, err)
	pluginCount := len(plugins)

	// Get the path to our current test binary
	var val string
	var ok bool
	if val, ok = os.LookupEnv("PLUGIN_EXAMPLE"); !ok {
		t.Skip("No plugin example path (PLUGIN_EXAMPLE) passed in")
	}

	exPlugin := filepath.Join(val, "plugin_example")
	plugins, err = l1.rpc.StartPlugin(exPlugin)
	check(t, err)
	assert.Equal(t, pluginCount+1, len(plugins))
	err = l1.waitForLog(`Is this initial node startup\? false`, 1)
	check(t, err)

	l1Info, _ := l1.rpc.GetInfo()
	assert.Equal(t, 1, len(l1Info.Binding))

	l1Addr := l1Info.Binding[0]
	l2, err := LnNode(testDir, dataDir, btcPid, "two")
	plugins, err = l2.rpc.StartPlugin(exPlugin)
	check(t, err)
	err = l2.waitForLog(`Is this initial node startup\? false`, 1)
	check(t, err)

	// We should have a third node!
	l3, err := LnNode(testDir, dataDir, btcPid, "three")
	check(t, err)

	peerId, err := l2.rpc.Connect(l1Info.Id, "localhost", uint(l1Addr.Port))
	check(t, err)

	l3Info, _ := l3.rpc.GetInfo()
	peer3, err := l2.rpc.Connect(l3Info.Id, "localhost", uint(l3Info.Binding[0].Port))
	check(t, err)

	err = fundNode("1.0", l2, btc)
	check(t, err)
	waitToSync(l1)
	waitToSync(l2)

	// open a channel
	amount := glightning.NewAmount(10000000)
	feerate := glightning.NewFeeRate(glightning.SatPerKiloSipa, uint(253))
	_, err = l2.rpc.FundChannelExt(peerId, amount, feerate, true, nil)
	check(t, err)

	// wait til the change is onchain
	advanceChain(l2, btc, 1)

	// fund a second channel!
	_, err = l2.rpc.FundChannelExt(peer3, amount, feerate, true, nil)
	check(t, err)

	mineBlocks(6, btc)

	waitForChannelReady(t, l2, l3)
	waitForChannelReady(t, l2, l1)

	// there's two now??
	scid23 := getShortChannelId(t, l2, l3)
	err = l2.waitForLog(fmt.Sprintf(`Received channel_update for channel %s/. now ACTIVE`, scid23), 20)
	check(t, err)
	scid21 := getShortChannelId(t, l2, l1)
	err = l2.waitForLog(fmt.Sprintf(`Received channel_update for channel %s/. now ACTIVE`, scid21), 20)
	check(t, err)

	// wait for everybody to know about other channels
	waitForChannelActive(l1, scid23)
	waitForChannelActive(l3, scid21)

	// warnings go off because of feerate misfires
	err = l1.waitForLog("Got a warning!!", 1)
	check(t, err)
	// open channel hook called ?? why no working
	/*
		err = l1.waitForLog("openchannel called", 1)
		check(t, err)
	*/
	// channel opened notification
	err = l1.waitForLog("channel opened", 1)
	check(t, err)

	invAmt := uint64(100000)
	invAmt2 := uint64(10000)
	inv, err := l1.rpc.CreateInvoice(invAmt, "push pay", "money", 100, nil, "", false)
	inv2, err := l3.rpc.CreateInvoice(invAmt, "push pay two", "money two", 100, nil, "", false)
	check(t, err)

	route, err := l2.rpc.GetRouteSimple(peerId, invAmt, 1.0)
	check(t, err)

	// l2 -> l1
	_, err = l2.rpc.SendPayLite(route, inv.PaymentHash)
	check(t, err)
	_, err = l2.rpc.WaitSendPay(inv.PaymentHash, 0)
	check(t, err)

	// SEND PAY SUCCESS
	err = l2.waitForLog("send pay success!", 1)
	check(t, err)
	err = l1.waitForLog("invoice paid", 1)
	check(t, err)

	/* ?? why no work
	err = l2.waitForLog("invoice payment called", 1)
	check(t, err)
	*/

	// now try to route from l1 -> l3 (but with broken middle)
	route2, err := l1.rpc.GetRouteSimple(peer3, invAmt2, 1.0)
	check(t, err)

	_, err = l2.rpc.CloseNormal(peer3)
	check(t, err)
	mineBlocks(1, btc)

	_, err = l1.rpc.SendPayLite(route2, inv2.PaymentHash)
	check(t, err)
	_, err = l1.rpc.WaitSendPay(inv2.PaymentHash, 0)
	assert.NotNil(t, err)

	// SEND PAY FAILURE
	err = l1.waitForLog("send pay failure!", 1)
	check(t, err)

}

func TestCloseTo(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, btc := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	l1Info, _ := l1.rpc.GetInfo()
	assert.Equal(t, 1, len(l1Info.Binding))

	l1Addr := l1Info.Binding[0]
	l2, err := LnNode(testDir, dataDir, btcPid, "two")

	peerId, err := l2.rpc.Connect(l1Info.Id, "localhost", uint(l1Addr.Port))
	check(t, err)

	err = fundNode("1.0", l2, btc)
	check(t, err)
	waitToSync(l1)
	waitToSync(l2)

	closeTo, err := btc.GetNewAddress(gbitcoin.Bech32)
	check(t, err)
	feerate := glightning.NewFeeRate(glightning.SatPerKiloSipa, uint(253))
	amount := uint64(100000)
	starter, err := l2.rpc.StartFundChannel(peerId, amount, true, feerate, closeTo)
	check(t, err)

	// build a transaction
	outs := []*gbitcoin.TxOut{
		&gbitcoin.TxOut{
			Address: starter.Address,
			Satoshi: amount,
		},
	}
	rawtx, err := btc.CreateRawTx(nil, outs, nil, nil)
	check(t, err)
	fundedtx, err := btc.FundRawTx(rawtx)
	check(t, err)
	tx, err := btc.DecodeRawTx(fundedtx.TxString)
	check(t, err)
	txout, err := tx.FindOutputIndex(starter.Address)
	check(t, err)
	_, err = l2.rpc.CompleteFundChannel(peerId, tx.TxId, txout)
	check(t, err)

	peer, err := l2.rpc.GetPeer(peerId)
	check(t, err)

	assert.Equal(t, closeTo, peer.Channels[0].CloseToAddress)
}

func TestInvoiceFieldsOnPaid(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, btc := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	l1Info, _ := l1.rpc.GetInfo()
	assert.Equal(t, 1, len(l1Info.Binding))

	l1Addr := l1Info.Binding[0]
	l2, err := LnNode(testDir, dataDir, btcPid, "two")

	peerId, err := l2.rpc.Connect(l1Info.Id, "localhost", uint(l1Addr.Port))
	check(t, err)

	err = fundNode("1.0", l2, btc)
	check(t, err)
	waitToSync(l1)
	waitToSync(l2)

	// open a channel
	amount := glightning.NewAmount(10000000)
	feerate := glightning.NewFeeRate(glightning.SatPerKiloSipa, uint(253))
	_, err = l2.rpc.FundChannelExt(peerId, amount, feerate, true, nil)
	check(t, err)

	// wait til the change is onchain
	advanceChain(l2, btc, 6)
	waitForChannelReady(t, l2, l1)

	invAmt := uint64(100000)
	invO, err := l1.rpc.CreateInvoice(invAmt, "pay me", "money", 100, nil, "", false)
	check(t, err)

	_, err = l2.rpc.PayBolt(invO.Bolt11)
	check(t, err)

	invA, err := l1.rpc.GetInvoice("pay me")
	check(t, err)

	assert.Equal(t, invAmt, invA.MilliSatoshiReceivedRaw)
	assert.True(t, len(invA.PaymentHash) > 0)
}

// let's try out some hooks!
func TestHooks(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init(t)
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	check(t, err)

	// Get the path to our current test binary
	var val string
	var ok bool
	if val, ok = os.LookupEnv("PLUGIN_EXAMPLE"); !ok {
		t.Skip("No plugin example path (PLUGIN_EXAMPLE) passed in")
	}

	exPlugin := filepath.Join(val, "plugin_example")
	_, err = l1.rpc.StartPlugin(exPlugin)
	check(t, err)
	l1.waitForLog("successfully init'd!", 1)

	l1Info, _ := l1.rpc.GetInfo()
	l1Addr := l1Info.Binding[0]

	l2, err := LnNode(testDir, dataDir, btcPid, "two")
	peerId, err := l2.rpc.Connect(l1Info.Id, l1Addr.Addr, uint(l1Addr.Port))
	check(t, err)
	assert.Equal(t, peerId, l1Info.Id)
	err = l1.waitForLog("peer connected called", 1)
	check(t, err)

	l2.rpc.Disconnect(l1Info.Id, true)
	err = l1.waitForLog("disconnect called for", 1)
	check(t, err)
}

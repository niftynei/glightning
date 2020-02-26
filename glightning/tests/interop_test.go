package glightning_test

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/niftynei/glightning/glightning"
	"github.com/niftynei/glightning/gbitcoin"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func Init() (string, string, int, *gbitcoin.Bitcoin) {
	// let's put it in a temporary directory
	testDir, err := ioutil.TempDir("", "gltests-")
	if err != nil {
		log.Fatal(err)
	}
	dataDir, _, btcPort, btc := SpinUpBitcoind(testDir)
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
func SpinUpBitcoind(dir string) (string, int, int, *gbitcoin.Bitcoin) {
	// make some dirs!
	bitcoindDir := filepath.Join(dir, "bitcoind")
	err := os.Mkdir(bitcoindDir, os.ModeDir|0755)
	if err != nil {
		log.Fatal(err)
	}

	bitcoinPath, err := exec.LookPath("bitcoind")
	if err != nil {
		log.Fatal(err)
	}
	btcPort := getPort()
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
	if err := bitcoind.Start(); err != nil {
		log.Fatal(err)
	}
	log.Printf(" bitcoind started (%d)!\n", bitcoind.Process.Pid)

	btc := gbitcoin.NewBitcoin(btcUser, btcPass)
	btc.SetTimeout(uint(2))
	// Waits til bitcoind is up
	btc.StartUp("", bitcoindDir, uint(btcPort))
	// Go ahead and run 50 blocks
	addr, err := btc.GetNewAddress(gbitcoin.Bech32)
	if err != nil {
		log.Fatal(err)
	}
	_, err = btc.GenerateToAddress(addr, 101)
	if err != nil {
		log.Fatal(err)
	}
	return bitcoindDir, bitcoind.Process.Pid, btcPort, btc
}

func (node *Node) waitForLog(phrase string, timeoutSec int) error {
	logfile, err := os.Open(filepath.Join(node.dir, "log"))
	if err != nil {
		return err
	}
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
		if strings.Contains(line, phrase) {
			return nil
		}
	}

	return errors.New(fmt.Sprintf("Unable to find \"%s\" in log", phrase))
}

func getPort() int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
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

	port := getPort()
	lightningd := exec.Command(lightningPath,
		fmt.Sprintf("--lightning-dir=%s", lightningdDir),
		fmt.Sprintf("--bitcoin-datadir=%s", dataDir),
		"--network=regtest", "--funding-confirms=3",
		fmt.Sprintf("--addr=localhost:%d", port),
		fmt.Sprintf("--bitcoin-rpcport=%d", btcPort),
		"--log-file=log",
		"--bitcoin-rpcuser=btcuser",
		"--bitcoin-rpcpassword=btcpass",
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
	err = node.waitForLog("Server started with public key", 5)
	if err != nil {
		return nil, err
	}
	log.Printf(" lightningd started (%d)!\n", lightningd.Process.Pid)
	log.Printf("Live at %s\n", lightningdDir)

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

	testDir, _, _, btc := Init()
	defer CleanUp(testDir)
	addr, err := btc.GetNewAddress(gbitcoin.Bech32)
	if err != nil {
		log.Fatal(err)
	}
	assert.NotNil(t, addr)

}

func TestConnectRpc(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init()
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	if err != nil {
		log.Fatal(err)
	}

	l1Info, _ := l1.rpc.GetInfo()
	assert.Equal(t, 1, len(l1Info.Binding))

	l1Addr := l1Info.Binding[0]
	l2, err := LnNode(testDir, dataDir, btcPid, "two")
	peerId, err := l2.rpc.Connect(l1Info.Id, l1Addr.Addr, uint(l1Addr.Port))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, peerId, l1Info.Id)
}

func TestConfigsRpc(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init()
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	if err != nil {
		log.Fatal(err)
	}

	configs, err := l1.rpc.ListConfigs()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "lightning-rpc", configs["rpc-file"])
	assert.Equal(t, false, configs["always-use-proxy"])

	network, err := l1.rpc.GetConfig("network")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "regtest", network)
}

func TestHelpRpc(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init()
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	if err != nil {
		log.Fatal(err)
	}

	commands, err := l1.rpc.Help()
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) == 0 {
		t.Error("No help commands returned")
	}

	cmd, err := l1.rpc.HelpFor("help")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "help [command]", cmd.NameAndUsage)
}

func connect(l1, l2 *Node) {
	l2Info, _ := l2.rpc.GetInfo()
	_, err := l1.rpc.Connect(l2Info.Id, l2Info.Binding[0].Addr, uint(l2Info.Binding[0].Port))
	if err != nil {
		log.Fatal(err)
	}
}

func fundNode(amount string, n *Node, b *gbitcoin.Bitcoin) {
	addr, err := n.rpc.NewAddr()
	if err != nil {
		log.Fatal(err)
	}
	_, err = b.SendToAddress(addr, amount)
	if err != nil {
		log.Fatal(err)
	}

	mineBlocks(1, b)
}

// n is number of blocks to mine
func mineBlocks(n uint, b *gbitcoin.Bitcoin) {
	addr, err := b.GetNewAddress(gbitcoin.Bech32)
	if err != nil {
		log.Fatal(err)
	}
	_, err = b.GenerateToAddress(addr, n)
	if err != nil {
		log.Fatal(err)
	}
}

func waitToSync(n *Node) {
	for {
		info, _ := n.rpc.GetInfo()
		if info.IsLightningdSync() {
			break
		}
	}
}

func TestCreateOnion(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init()
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	if err != nil {
		log.Fatal(err)
	}

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
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, len(resp.SharedSecrets), len(hops))
	assert.Equal(t, len(resp.Onion), 2*1366)

	privateHash = "4242424242424242424242424242424242424242424242424242424242424242"
	sessionKey := "4141414141414141414141414141414141414141414141414141414141414141"
	resp, err = l1.rpc.CreateOnion(hops, privateHash, sessionKey)
	if err != nil {
		log.Fatal(err)
	}

	onlen := len(resp.Onion)
	assert.Equal(t, resp.Onion[onlen-22:onlen], "9400f45a48e6dc8ddbaeb3")

	firstHop := glightning.FirstHop{
		ShortChannelId: "100x1x1",
		Direction: 1,
		AmountMsat: "1000sat",
		Delay: 8,
	}

	// Ideally we'd do a 'real' send onion but we don't 
	// need to know if c-lightning works, only that the API
	// functions correctly...
	_, err = l1.rpc.SendOnionWithDetails(resp.Onion, firstHop, privateHash, "label", resp.SharedSecrets, nil)

	// ... which means we expect an error back!
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "204:No connection to first peer found")
}

// ok, now let's check the dynamic plugin loader
func TestPlugins(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init()
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	if err != nil {
		log.Fatal(err)
	}

	plugins, err := l1.rpc.ListPlugins()
	if err != nil {
		log.Fatal(err)
	}
	pluginCount := len(plugins)

	// Get the path to our current test binary
	var val string
	var ok bool
	if val, ok = os.LookupEnv("PLUGIN_EXAMPLE"); !ok {
		t.Fatal("No plugin example path passed in")
	}

	exPlugin := filepath.Join(val, "plugin_example")
	plugins, err = l1.rpc.StartPlugin(exPlugin)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, pluginCount+1, len(plugins))
	err = l1.waitForLog("Is this initial node startup? false", 1)
	if err != nil {
		t.Fatal(err)
	}

	l1Info, _ := l1.rpc.GetInfo()
	assert.Equal(t, 1, len(l1Info.Binding))

	l1Addr := l1Info.Binding[0]
	l2, err := LnNode(testDir, dataDir, btcPid, "two")
	peerId, err := l2.rpc.Connect(l1Info.Id, l1Addr.Addr, uint(l1Addr.Port))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, peerId, l1Info.Id)
	err = l1.waitForLog("connect called: ", 1)
	if err != nil {
		t.Fatal(err)
	}

	l2.rpc.Disconnect(peerId, true)
	err = l1.waitForLog("disconnect called for", 1)
	if err != nil {
		t.Fatal(err)
	}
}

// let's try out some hooks!
func TestHooks(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid, _ := Init()
	defer CleanUp(testDir)
	l1, err := LnNode(testDir, dataDir, btcPid, "one")
	if err != nil {
		log.Fatal(err)
	}

	// Get the path to our current test binary
	var val string
	var ok bool
	if val, ok = os.LookupEnv("PLUGIN_EXAMPLE"); !ok {
		t.Fatal("No plugin example path passed in")
	}

	exPlugin := filepath.Join(val, "plugin_example")
	_, err = l1.rpc.StartPlugin(exPlugin)
	if err != nil {
		t.Fatal(err)
	}
	l1.waitForLog("successfully init'd!", 1)

	l1Info, _ := l1.rpc.GetInfo()
	l1Addr := l1Info.Binding[0]

	l2, err := LnNode(testDir, dataDir, btcPid, "two")
	peerId, err := l2.rpc.Connect(l1Info.Id, l1Addr.Addr, uint(l1Addr.Port))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, peerId, l1Info.Id)
	err = l1.waitForLog("peer connected called", 1)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: we need a bitcoind rpc to trigger the rest
}

package glightning_test

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/niftynei/glightning/glightning"
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

func Init() (string, string, int) {
	// let's put it in a temporary directory
	testDir, err := ioutil.TempDir("", "gltests-")
	if err != nil {
		log.Fatal(err)
	}
	dataDir, _, btcPort := SpinUpBitcoind(testDir)
	return testDir, dataDir, btcPort
}

func CleanUp(testDir string) {
	os.Remove(testDir)
}

// Returns bitcoind PID
func SpinUpBitcoind(dir string) (string, int, int) {
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
	bitcoind := exec.Command(bitcoinPath, "-regtest",
		fmt.Sprintf("-datadir=%s", bitcoindDir),
		"-server", "-logtimestamps", "-nolisten",
		fmt.Sprintf("-rpcport=%d", btcPort),
		"-addresstype=bech32", "-logtimestamps", "-txindex",
		"-rpcpassword=btcpass", "-rpcuser=btcuser")

	bitcoind.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}
	log.Printf("starting %s on %d...", bitcoinPath, btcPort)
	if err := bitcoind.Start(); err != nil {
		log.Fatal(err)
	}
	log.Printf(" bitcoind started (%d)!\n", bitcoind.Process.Pid)

	return bitcoindDir, bitcoind.Process.Pid, btcPort
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
		"--bitcoin-rpcpassword=btcpass")

	lightningd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}
	log.Printf("starting %s on %d...", lightningPath, port)
	if err := lightningd.Start(); err != nil {
		return nil, err
	}

	time.Sleep(200 * time.Millisecond)

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

func TestConnectRpc(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid := Init()
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

	testDir, dataDir, btcPid := Init()
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

// ok, now let's check the dynamic plugin loader
func TestPlugins(t *testing.T) {
	short(t)

	testDir, dataDir, btcPid := Init()
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
	assert.Equal(t, pluginCount + 1, len(plugins))
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

	testDir, dataDir, btcPid := Init()
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

package glightning_test

import (
	"bufio"
	"fmt"
	"github.com/niftynei/glightning/glightning"
	"github.com/niftynei/glightning/jrpc2"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"os"
	"testing"
)

type HiMethod struct {
	plugin *glightning.Plugin
}

func NewHiMethod(p *glightning.Plugin) *HiMethod {
	return &HiMethod{
		plugin: p,
	}
}

func (hi *HiMethod) Name() string {
	return "hi"
}

func (hi *HiMethod) New() interface{} {
	return NewHiMethod(hi.plugin)
}

func (hi *HiMethod) Call() (jrpc2.Result, error) {
	gOpt := hi.plugin.GetOption("greeting")
	return fmt.Sprintf("Hello, %s", gOpt.Value()), nil
}

func getInitFunc(t *testing.T, testFn func(t *testing.T, opt map[string]string, config *glightning.Config)) func(*glightning.Plugin, map[string]string, *glightning.Config) {
	return func(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
		testFn(t, options, config)
	}
}

func nullInitFunc(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	// does nothing
}

func TestLogsGeneralInfra(t *testing.T) {
	plugin := glightning.NewPlugin(nullInitFunc)

	progIn, _, _ := os.Pipe()
	testIn, progOut, _ := os.Pipe()

	go func(in, out *os.File, t *testing.T) {
		err := plugin.Start(in, out)
		if err != nil {
			t.Fatal(err)
		}
	}(progIn, progOut, t)

	in, out := io.Pipe()
	go func(in io.Reader) {
		// everytime we get a new message, log it thru c-lightning
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			plugin.Log(scanner.Text(), glightning.Info)
		}
		if err := scanner.Err(); err != nil {
			// print errors with logging to stderr
			fmt.Fprintln(os.Stderr, "error with logging pipe:", err)
		}
	}(in)
	log.SetFlags(0)
	log.SetOutput(out)

	log.Print("this is a log line")

	scanner := bufio.NewScanner(testIn)
	scanner.Split(func(data []byte, eof bool) (advance int, token []byte, err error) {
		for i := 0; i < len(data); i++ {
			if data[i] == '\n' && (i+1) < len(data) && data[i+1] == '\n' {
				return i + 2, data[:i], nil
			}
		}
		return 0, nil, nil
	})
	if !scanner.Scan() {
		t.Log(scanner.Err())
		t.FailNow()
	}
	bytesRead := scanner.Bytes()
	assert.Equal(t, "{\"jsonrpc\":\"2.0\",\"method\":\"log\",\"params\":{\"level\":\"info\",\"message\":\"this is a log line\"}}", string(bytesRead))
}

// test the plugin's handling of init
func TestInit(t *testing.T) {

	initTestFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		assert.Equal(t, "Jenny", options["greeting"])
		assert.Equal(t, "rpc.file", config.RpcFile)
		assert.Equal(t, "dirforlightning", config.LightningDir)
		assert.Equal(t, true, config.Startup)
	})
	plugin := glightning.NewPlugin(initTestFn)
	plugin.RegisterOption(glightning.NewOption("greeting", "How you'd like to be called", "Mary"))
	plugin.RegisterMethod(glightning.NewRpcMethod(NewHiMethod(plugin), "Send a greeting."))

	initJson := "{\"jsonrpc\":\"2.0\",\"method\":\"init\",\"params\":{\"options\":{\"greeting\":\"Jenny\"},\"configuration\":{\"rpc-file\":\"rpc.file\",\"startup\":true,\"lightning-dir\":\"dirforlightning\"}},\"id\":1}\n\n"

	expectedJson := "{\"jsonrpc\":\"2.0\",\"result\":\"ok\",\"id\":1}"
	runTest(t, plugin, initJson, expectedJson)
}

func HandleConnect(event *glightning.ConnectEvent) {
	// do nothing
}

func TestGetManifest(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterMethod(glightning.NewRpcMethod(NewHiMethod(plugin), "Send a greeting."))
	plugin.RegisterOption(glightning.NewOption("greeting", "How you'd like to be called", "Mary"))
	plugin.SubscribeConnect(HandleConnect)

	msg := "{\"jsonrpc\":\"2.0\",\"method\":\"getmanifest\",\"id\":\"aloha\"}\n\n"
	resp := "{\"jsonrpc\":\"2.0\",\"result\":{\"options\":[{\"name\":\"greeting\",\"type\":\"string\",\"default\":\"Mary\",\"description\":\"How you'd like to be called\"}],\"rpcmethods\":[{\"name\":\"hi\",\"description\":\"Send a greeting.\"}],\"subscriptions\":[\"connect\"]},\"id\":\"aloha\"}"
	runTest(t, plugin, msg, resp)
}

func runTest(t *testing.T, plugin *glightning.Plugin, inputMsg, expectedMsg string) {
	progIn, testOut, err := os.Pipe()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	testIn, progOut, err := os.Pipe()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	go func(in, out *os.File) {
		err := plugin.Start(in, out)
		if err != nil {
			panic(err)
		}
	}(progIn, progOut)

	// call the method
	// would using a client implementation be nice here?
	// the pylightning plugin handler probably uses regular
	testOut.Write([]byte(inputMsg))

	scanner := bufio.NewScanner(testIn)
	scanner.Split(func(data []byte, eof bool) (advance int, token []byte, err error) {
		for i := 0; i < len(data); i++ {
			if data[i] == '\n' && (i+1) < len(data) && data[i+1] == '\n' {
				return i + 2, data[:i], nil
			}
		}
		return 0, nil, nil
	})
	if !scanner.Scan() {
		t.Log(scanner.Err())
		t.FailNow()
	}
	bytesRead := scanner.Bytes()
	assert.Equal(t, expectedMsg, string(bytesRead))
}

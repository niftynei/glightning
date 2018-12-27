package golight_test

import (
	"bufio"
	"io"
	"fmt"
	"os"
	"github.com/niftynei/golight"
	"github.com/niftynei/golight/jrpc2"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

type HiMethod struct {
	plugin *golight.Plugin
}

func NewHiMethod(p *golight.Plugin) *HiMethod {
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

func getInitFunc(t *testing.T, testFn func(t *testing.T, opt map[string]string, config *golight.Config)) func(*golight.Plugin, map[string]string, *golight.Config) {
	return func (plugin *golight.Plugin, options map[string]string, config *golight.Config) {
		testFn(t, options, config)
	}
}

func nullInitFunc(plugin *golight.Plugin, options map[string]string, config *golight.Config) {
	// does nothing
}

// check that we're sending all our logs out 
// over the wire
func TestLoggingRedirect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	os.Setenv("LIGHTNINGD_PLUGIN", "")
	plugin := golight.NewPlugin(nullInitFunc)

	progIn, _, _ := os.Pipe()
	testIn, progOut, _ := os.Pipe()

	go func(in, out *os.File, t *testing.T) {
		err := plugin.Start(in, out)
		if err != nil {
			t.Fatal(err)
		}
	}(progIn, progOut, t)

	time.Sleep(1)
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
	assert.Equal(t, "", string(bytesRead))
}

func TestLogsGeneralInfra(t *testing.T) {
	plugin := golight.NewPlugin(nullInitFunc)

	progIn, _ , _ := os.Pipe()
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
			plugin.Log(scanner.Text(), golight.Info)
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

	initTestFn := getInitFunc(t ,func(t *testing.T, options map[string]string, config *golight.Config) {
		assert.Equal(t, "Jenny", options["greeting"])
		assert.Equal(t, "rpc.file", config.RpcFile)
		assert.Equal(t, "dirforlightning", config.LightningDir)
	})
	plugin := golight.NewPlugin(initTestFn)
	plugin.RegisterOption(golight.NewOption("greeting", "How you'd like to be called", "Mary"))
	plugin.RegisterMethod(golight.NewRpcMethod(NewHiMethod(plugin), "Send a greeting."))

	initJson := "{\"jsonrpc\":\"2.0\",\"method\":\"init\",\"params\":{\"options\":{\"greeting\":\"Jenny\"},\"configuration\":{\"rpc-file\":\"rpc.file\",\"lightning-dir\":\"dirforlightning\"}}}\n\n"

	expectedJson := "{\"jsonrpc\":\"2.0\",\"result\":\"ok\",\"id\":null}"
	runTest(t, plugin, initJson, expectedJson)
}

func TestGetManifest(t *testing.T) {
	initFn := getInitFunc(t ,func(t *testing.T, options map[string]string, config *golight.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := golight.NewPlugin(initFn)
	plugin.RegisterMethod(golight.NewRpcMethod(NewHiMethod(plugin), "Send a greeting."))
	plugin.RegisterOption(golight.NewOption("greeting", "How you'd like to be called", "Mary"))

	msg := "{\"jsonrpc\":\"2.0\",\"method\":\"getmanifest\",\"id\":\"aloha\"}\n\n"
	resp := "{\"jsonrpc\":\"2.0\",\"result\":{\"options\":[{\"name\":\"greeting\",\"type\":\"string\",\"default\":\"Mary\",\"description\":\"How you'd like to be called\"}],\"rpcmethods\":[{\"name\":\"hi\",\"description\":\"Send a greeting.\"}]},\"id\":\"aloha\"}"
	runTest(t, plugin, msg, resp)
}

func runTest(t *testing.T, plugin *golight.Plugin, inputMsg, expectedMsg string) {
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

// todo: try using stdin and stdout as pipes?

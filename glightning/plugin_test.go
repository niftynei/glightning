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
	"sync"
	"testing"
	"time"
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
	gOpt, err := hi.plugin.GetOption("greeting")
	if err != nil {
		return nil, err
	}
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

func OnPeerConnect(event *glightning.PeerConnectedEvent) (*glightning.PeerConnectedResponse, error) {
	return nil, nil
}

func OnDbWrite(event *glightning.DbWriteEvent) (bool, error) {
	return false, nil
}

func OnInvoicePayment(event *glightning.InvoicePaymentEvent) (*glightning.InvoicePaymentResponse, error) {
	return nil, nil
}

func OnOpenChannel(*glightning.OpenChannelEvent) (*glightning.OpenChannelResponse, error) {
	return nil, nil
}

func OnHtlcAccepted(*glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
	return nil, nil
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

func TestMissingOptionRpcCall(t *testing.T) {
	initTestFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initTestFn)
	// No 'greeting' options is registered, should return an error
	plugin.RegisterMethod(glightning.NewRpcMethod(NewHiMethod(plugin), "Send a greeting."))

	initJson := "{\"jsonrpc\":\"2.0\",\"method\":\"hi\",\"params\":{},\"id\":1}\n\n"
	expectedJson := "{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-1,\"message\":\"Option 'greeting' not found\"},\"id\":1}"
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
	plugin.SetDynamic(true)

	msg := "{\"jsonrpc\":\"2.0\",\"method\":\"getmanifest\",\"id\":\"aloha\"}\n\n"
	resp := "{\"jsonrpc\":\"2.0\",\"result\":{\"options\":[{\"name\":\"greeting\",\"type\":\"string\",\"default\":\"Mary\",\"description\":\"How you'd like to be called\"}],\"rpcmethods\":[{\"name\":\"hi\",\"description\":\"Send a greeting.\"}],\"subscriptions\":[\"connect\"],\"dynamic\":true},\"id\":\"aloha\"}"
	runTest(t, plugin, msg, resp)
}

func TestManifestWithHooks(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		PeerConnected:  OnPeerConnect,
		DbWrite:        OnDbWrite,
		InvoicePayment: OnInvoicePayment,
		OpenChannel:    OnOpenChannel,
		HtlcAccepted:   OnHtlcAccepted,
	})

	msg := "{\"jsonrpc\":\"2.0\",\"method\":\"getmanifest\",\"id\":\"aloha\"}\n\n"
	resp := `{"jsonrpc":"2.0","result":{"options":[],"rpcmethods":[],"hooks":["db_write","peer_connected","invoice_payment","openchannel","htlc_accepted"],"dynamic":true},"id":"aloha"}`
	runTest(t, plugin, msg, resp)
}

func TestHook_DbWriteOk(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		DbWrite: func(event *glightning.DbWriteEvent) (bool, error) {
			assert.Equal(t, 3, len(event.Writes))
			writesExp := []string{
				"BEGIN TRANSACTION;",
				"UPDATE vars SET val='2' WHERE name='bip32_max_index';",
				"COMMIT;",
			}
			assert.Equal(t, writesExp, event.Writes)
			return true, nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"db_write","params":{"writes":["BEGIN TRANSACTION;","UPDATE vars SET val='2' WHERE name='bip32_max_index';","COMMIT;"]}}`
	resp := `{"jsonrpc":"2.0","result":true,"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_DbWriteFail(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		DbWrite: func(event *glightning.DbWriteEvent) (bool, error) {
			assert.Equal(t, 3, len(event.Writes))
			writesExp := []string{
				"BEGIN TRANSACTION;",
				"UPDATE vars SET val='2' WHERE name='bip32_max_index';",
				"COMMIT;",
			}
			assert.Equal(t, writesExp, event.Writes)
			return false, nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"db_write","params":{"writes":["BEGIN TRANSACTION;","UPDATE vars SET val='2' WHERE name='bip32_max_index';","COMMIT;"]}}`
	resp := `{"jsonrpc":"2.0","result":false,"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_PeerConnectedOk(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		PeerConnected: func(event *glightning.PeerConnectedEvent) (*glightning.PeerConnectedResponse, error) {
			expected := glightning.PeerEvent{
				PeerId:   "02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6",
				Addr:     "127.0.0.1:58366",
				Features: "aa",
			}
			assert.Equal(t, expected, event.Peer)
			return event.Continue(), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"peer_connected","params":{"peer":{"id":"02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6","addr":"127.0.0.1:58366","features":"aa"}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"continue"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_PeerConnectedDisconnect(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		PeerConnected: func(event *glightning.PeerConnectedEvent) (*glightning.PeerConnectedResponse, error) {
			expected := glightning.PeerEvent{
				PeerId:   "02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6",
				Addr:     "127.0.0.1:58366",
				Features: "aa",
			}
			assert.Equal(t, expected, event.Peer)
			return event.Disconnect("there is a problem"), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"peer_connected","params":{"peer":{"id":"02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6","addr":"127.0.0.1:58366","features":"aa"}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"disconnect","error_message":"there is a problem"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_OpenChannelOk(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		OpenChannel: func(event *glightning.OpenChannelEvent) (*glightning.OpenChannelResponse, error) {
			expected := glightning.OpenChannel{
				PeerId:                            "02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6",
				FundingSatoshis:                   "16000000000msat",
				PushMilliSatoshis:                 "0msat",
				DustLimitSatoshis:                 "546000msat",
				MaxHtlcValueInFlightMilliSatoshis: "18446744073709551615msat",
				ChannelReserveSatoshis:            "160000000msat",
				HtlcMinimumMillisatoshis:          "0msat",
				FeeratePerKw:                      253,
				ToSelfDelay:                       6,
				MaxAcceptedHtlcs:                  483,
				ChannelFlags:                      1,
			}
			assert.Equal(t, expected, event.OpenChannel)
			return event.Continue(), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"openchannel","params":{"openchannel":{"id":"02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6","funding_satoshis":"16000000000msat","push_msat":"0msat","dust_limit_satoshis":"546000msat","max_htlc_value_in_flight_msat":"18446744073709551615msat","channel_reserve_satoshis":"160000000msat","htlc_minimum_msat":"0msat","feerate_per_kw":253,"to_self_delay":6,"max_accepted_htlcs":483,"channel_flags":1}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"continue"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_OpenChannelReject(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		OpenChannel: func(event *glightning.OpenChannelEvent) (*glightning.OpenChannelResponse, error) {
			return event.Reject("unwanted"), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"openchannel","params":{"openchannel":{"id":"02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6","funding_satoshis":"16000000000msat","push_msat":"0msat","dust_limit_satoshis":"546000msat","max_htlc_value_in_flight_msat":"18446744073709551615msat","channel_reserve_satoshis":"160000000msat","htlc_minimum_msat":"0msat","feerate_per_kw":253,"to_self_delay":6,"max_accepted_htlcs":483,"channel_flags":1}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"reject","error_message":"unwanted"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_AddHtlc(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		HtlcAccepted: func(event *glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
			expected := &glightning.HtlcAcceptedEvent{
				Onion: glightning.Onion{
					Payload: "0000000000000000000000000000c3500000014b000000000000000000000000",
					PerHop: glightning.PerHop{
						Realm:                      "00",
						ShortChannelId:             "0x0x0",
						ForwardAmountMilliSatoshis: "50000msat",
						OutgoingCltvValue:          331,
					},
					NextOnion:    "0003ff512b805f80ba69052342482401cda5f4986ed8600316ba8caf72b4fcc5826f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000171bd5ff2190087d1572def9e35dd7c14e9ab3e06ce797b0480c9ccc8fb00a7d2390daf2fcead26d6775d7feac22f21fce0353b7fe77491401d18f7d69379c336d0000000000000000000000000000000000000000000000000000000000000000",
					SharedSecret: "eb5ab3e3db045e589597687e0eba89a98af0d19fd1967e1f24feb6f2814cb9c5",
				},
				Htlc: glightning.HtlcOffer{
					Amount:             "50000msat",
					CltvExpiry:         331,
					CltvExpiryRelative: 23,
					PaymentHash:        "6440c8f51f2ee53213ef9f2e58ffdf46982fe91dd7c9228a92a557450ae2f2f5",
				},
			}
			assert.Equal(t, expected.Onion, event.Onion)
			assert.Equal(t, expected.Htlc, event.Htlc)
			return event.Continue(), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"htlc_accepted","params":{"onion":{"payload":"0000000000000000000000000000c3500000014b000000000000000000000000","per_hop_v0":{"realm":"00","short_channel_id":"0x0x0","forward_amount":"50000msat","outgoing_cltv_value":331},"next_onion":"0003ff512b805f80ba69052342482401cda5f4986ed8600316ba8caf72b4fcc5826f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000171bd5ff2190087d1572def9e35dd7c14e9ab3e06ce797b0480c9ccc8fb00a7d2390daf2fcead26d6775d7feac22f21fce0353b7fe77491401d18f7d69379c336d0000000000000000000000000000000000000000000000000000000000000000","shared_secret":"eb5ab3e3db045e589597687e0eba89a98af0d19fd1967e1f24feb6f2814cb9c5"},"htlc":{"amount":"50000msat","cltv_expiry":331,"cltv_expiry_relative":23,"payment_hash":"6440c8f51f2ee53213ef9f2e58ffdf46982fe91dd7c9228a92a557450ae2f2f5"}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"continue"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_HtlcAcceptResolve(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		HtlcAccepted: func(event *glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
			return event.Resolve("payment_key"), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"htlc_accepted","params":{"onion":{"payload":"0000000000000000000000000000c3500000014b000000000000000000000000","per_hop_v0":{"realm":"00","short_channel_id":"0x0x0","forward_amount":"50000msat","outgoing_cltv_value":331},"next_onion":"0003ff512b805f80ba69052342482401cda5f4986ed8600316ba8caf72b4fcc5826f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000171bd5ff2190087d1572def9e35dd7c14e9ab3e06ce797b0480c9ccc8fb00a7d2390daf2fcead26d6775d7feac22f21fce0353b7fe77491401d18f7d69379c336d0000000000000000000000000000000000000000000000000000000000000000","shared_secret":"eb5ab3e3db045e589597687e0eba89a98af0d19fd1967e1f24feb6f2814cb9c5"},"htlc":{"amount":"50000msat","cltv_expiry":331,"cltv_expiry_relative":23,"payment_hash":"6440c8f51f2ee53213ef9f2e58ffdf46982fe91dd7c9228a92a557450ae2f2f5"}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"resolve","payment_key":"payment_key"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_HtlcAcceptedFail(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		HtlcAccepted: func(event *glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
			return event.Fail(uint16(55)), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"htlc_accepted","params":{"onion":{"payload":"0000000000000000000000000000c3500000014b000000000000000000000000","per_hop_v0":{"realm":"00","short_channel_id":"0x0x0","forward_amount":"50000msat","outgoing_cltv_value":331},"next_onion":"0003ff512b805f80ba69052342482401cda5f4986ed8600316ba8caf72b4fcc5826f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000171bd5ff2190087d1572def9e35dd7c14e9ab3e06ce797b0480c9ccc8fb00a7d2390daf2fcead26d6775d7feac22f21fce0353b7fe77491401d18f7d69379c336d0000000000000000000000000000000000000000000000000000000000000000","shared_secret":"eb5ab3e3db045e589597687e0eba89a98af0d19fd1967e1f24feb6f2814cb9c5"},"htlc":{"amount":"50000msat","cltv_expiry":331,"cltv_expiry_relative":23,"payment_hash":"6440c8f51f2ee53213ef9f2e58ffdf46982fe91dd7c9228a92a557450ae2f2f5"}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"fail","failure_code":55},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_InvoicePayment(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		InvoicePayment: func(event *glightning.InvoicePaymentEvent) (*glightning.InvoicePaymentResponse, error) {
			expected := glightning.Payment{
				Label:         "test_4",
				PreImage:      "09d686f01fbbc6d36996f6c68b09d62600b9da32bd249892904350e31bc51c6e",
				MilliSatoshis: "50000msat",
			}
			assert.Equal(t, expected, event.Payment)
			return event.Continue(), nil
		},
	})
	msg := `{"jsonrpc":"2.0","id":"aloha","method":"invoice_payment","params":{"payment":{"label":"test_4","preimage":"09d686f01fbbc6d36996f6c68b09d62600b9da32bd249892904350e31bc51c6e","msat":"50000msat"}}}`
	resp := `{"jsonrpc":"2.0","result":{},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_InvoicePaymentFail(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		InvoicePayment: func(event *glightning.InvoicePaymentEvent) (*glightning.InvoicePaymentResponse, error) {
			return event.Fail(uint16(44)), nil
		},
	})
	msg := `{"jsonrpc":"2.0","id":"aloha","method":"invoice_payment","params":{"payment":{"label":"test_4","preimage":"09d686f01fbbc6d36996f6c68b09d62600b9da32bd249892904350e31bc51c6e","msat":"50000msat"}}}`
	resp := `{"jsonrpc":"2.0","result":{"failure_code":44},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestSubscription_SendPaySuccess(t *testing.T) {
	var wg sync.WaitGroup
	defer await(t, &wg)

	wg.Add(1)
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.SubscribeSendPaySuccess(func(event *glightning.SendPaySuccess) {
		defer wg.Done()
		expected := &glightning.SendPaySuccess{
			Id:                     1,
			PaymentHash:            "5c85bf402b87d4860f4a728e2e58a2418bda92cd7aea0ce494f11670cfbfb206",
			Destination:            "035d2b1192dfba134e10e540875d366ebc8bc353d5aa766b80c090b39c3a5d885d",
			MilliSatoshi:           100000000,
			AmountMilliSatoshi:     "100000000msat",
			AmountSent:             100001001,
			AmountSentMilliSatoshi: "100001001msat",
			CreatedAt:              1561390572,
			Status:                 "complete",
			PaymentPreimage:        "9540d98095fd7f37687ebb7759e733934234d4f934e34433d4998a37de3733ee",
		}
		assert.Equal(t, expected, event)
	})

	msg := `{"jsonrpc":"2.0","method":"sendpay_success","params":{"sendpay_success":{"id":1,"payment_hash":"5c85bf402b87d4860f4a728e2e58a2418bda92cd7aea0ce494f11670cfbfb206","destination":"035d2b1192dfba134e10e540875d366ebc8bc353d5aa766b80c090b39c3a5d885d","msatoshi":100000000,"amount_msat":"100000000msat","msatoshi_sent":100001001,"amount_sent_msat":"100001001msat","created_at":1561390572,"status":"complete","payment_preimage":"9540d98095fd7f37687ebb7759e733934234d4f934e34433d4998a37de3733ee"}}}`

	runTest(t, plugin, msg+"\n\n", "")
}

func TestSubscription_SendPayFailure(t *testing.T) {
	var wg sync.WaitGroup
	defer await(t, &wg)

	wg.Add(1)
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.SubscribeSendPayFailure(func(event *glightning.SendPayFailure) {
		defer wg.Done()
		expected := &glightning.SendPayFailure{
			Code:    204,
			Message: "failed: WIRE_UNKNOWN_NEXT_PEER (reply from remote)",
			Data: glightning.SendPayFailureData{
				Id:                     2,
				PaymentHash:            "9036e3bdbd2515f1e653cb9f22f8e4c49b73aa2c36e937c926f43e33b8db8851",
				Destination:            "035d2b1192dfba134e10e540875d366ebc8bc353d5aa766b80c090b39c3a5d885d",
				MilliSatoshi:           100000000,
				AmountMilliSatoshi:     "100000000msat",
				AmountSent:             100001001,
				AmountSentMilliSatoshi: "100001001msat",
				CreatedAt:              1561395134,
				Status:                 "failed",
				ErringIndex:            1,
				FailCode:               16394,
				FailCodeName:           "WIRE_UNKNOWN_NEXT_PEER",
				ErringNode:             "022d223620a359a47ff7f7ac447c85c46c923da53389221a0054c11c1e3ca31d59",
				ErringChannel:          "103x2x1",
				ErringDirection:        0,
			},
		}
		assert.Equal(t, expected, event)
	})

	msg := `{"jsonrpc":"2.0","method":"sendpay_failure","params":{"sendpay_failure": {
    "code": 204,
    "message": "failed: WIRE_UNKNOWN_NEXT_PEER (reply from remote)",
    "data": {
      "id": 2,
      "payment_hash": "9036e3bdbd2515f1e653cb9f22f8e4c49b73aa2c36e937c926f43e33b8db8851",
      "destination": "035d2b1192dfba134e10e540875d366ebc8bc353d5aa766b80c090b39c3a5d885d",
      "msatoshi": 100000000,
      "amount_msat": "100000000msat",
      "msatoshi_sent": 100001001,
      "amount_sent_msat": "100001001msat",
      "created_at": 1561395134,
      "status": "failed",
      "erring_index": 1,
      "failcode": 16394,
      "failcodename": "WIRE_UNKNOWN_NEXT_PEER",
      "erring_node": "022d223620a359a47ff7f7ac447c85c46c923da53389221a0054c11c1e3ca31d59",
      "erring_channel": "103x2x1",
      "erring_direction": 0
    }}}}`
	runTest(t, plugin, msg+"\n\n", "")
}

func TestSubscription_Warning(t *testing.T) {
	var wg sync.WaitGroup
	defer await(t, &wg)

	wg.Add(1)
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.SubscribeWarnings(func(event *glightning.Warning) {
		defer wg.Done()
		expected := &glightning.Warning{
			Level:  "warn",
			Time:   "1565639989.291189188",
			Source: "lightningd(23822):",
			Log:    "Unable to estimate ECONOMICAL/100 fee",
		}
		assert.Equal(t, expected, event)
	})

	msg := `{"jsonrpc":"2.0","method":"warning","params":{"warning":{"level":"warn","time":"1565639989.291189188","source":"lightningd(23822):","log":"Unable to estimate ECONOMICAL/100 fee"}}}`

	runTest(t, plugin, msg+"\n\n", "")
}

func TestSubscription_Forwarding(t *testing.T) {
	var wg sync.WaitGroup
	defer await(t, &wg)

	wg.Add(1)
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.SubscribeForwardings(func(event *glightning.Forwarding) {
		defer wg.Done()
		expected := &glightning.Forwarding{
			InChannel:       "103x2x1",
			OutChannel:      "110x1x0",
			MilliSatoshiIn:  100001001,
			InMsat:          "100001001msat",
			MilliSatoshiOut: 100000000,
			OutMsat:         "100000000msat",
			Fee:             1001,
			FeeMsat:         "1001msat",
			Status:          "local_failed",
			PaymentHash:     "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			FailCode:        16392,
			FailReason:      "WIRE_PERMANENT_CHANNEL_FAILURE",
			ReceivedTime:    1560696343.052,
		}
		assert.Equal(t, expected, event)
	})

	msg := `{"jsonrpc":"2.0","method":"forward_event","params":{"forward_event":{"payment_hash":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","in_channel":"103x2x1","out_channel":"110x1x0","in_msatoshi":100001001,"in_msat":"100001001msat","out_msatoshi":100000000,"out_msat":"100000000msat","fee":1001,"fee_msat":"1001msat","status":"local_failed","failcode":16392,"failreason":"WIRE_PERMANENT_CHANNEL_FAILURE","received_time":1560696343.052}}}`

	runTest(t, plugin, msg+"\n\n", "")
}
func TestSubscription_ChannelOpened(t *testing.T) {
	var wg sync.WaitGroup
	defer await(t, &wg)

	wg.Add(1)
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.SubscribeChannelOpened(func(event *glightning.ChannelOpened) {
		defer wg.Done()
		expected := &glightning.ChannelOpened{
			PeerId:          "026bbfba23a5a0034181ec46bfe99eb03f135f765eeaf89cc7c84f4daeb7289462",
			FundingSatoshis: "100000000msat",
			FundingTxId:     "db31fc18891b5d75207051f2dbea94d01ed14939d2a61cc4cd5f88e7bd42aa71",
			FundingLocked:   true,
		}
		assert.Equal(t, expected, event)
	})

	msg := `{"jsonrpc":"2.0","method":"channel_opened","params":{"channel_opened":{"id":"026bbfba23a5a0034181ec46bfe99eb03f135f765eeaf89cc7c84f4daeb7289462","amount":"100000000msat","funding_txid":"db31fc18891b5d75207051f2dbea94d01ed14939d2a61cc4cd5f88e7bd42aa71","funding_locked":true}}}`

	runTest(t, plugin, msg+"\n\n", "")
}

func TestSubscription_Connected(t *testing.T) {
	var wg sync.WaitGroup
	defer await(t, &wg)

	wg.Add(1)
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.SubscribeConnect(func(event *glightning.ConnectEvent) {
		defer wg.Done()
		expected := &glightning.ConnectEvent{
			PeerId: "02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6",
			Address: glightning.Address{
				Type: "ipv4",
				Addr: "127.0.0.1",
				Port: 9090,
			},
		}
		assert.Equal(t, expected.PeerId, event.PeerId)
		assert.Equal(t, expected.Address, event.Address)
	})

	msg := `{"jsonrpc":"2.0","method":"connect","params":{"id":"02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6","address":{"type":"ipv4","address":"127.0.0.1","port":9090}}}`

	runTest(t, plugin, msg+"\n\n", "")
}

func TestSubscription_Disconnected(t *testing.T) {
	var wg sync.WaitGroup
	defer await(t, &wg)

	wg.Add(1)
	initFn := getInitFunc(t, func(t *testing.T, options map[string]string, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.SubscribeDisconnect(func(event *glightning.DisconnectEvent) {
		defer wg.Done()
		expected := &glightning.DisconnectEvent{
			PeerId: "02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6",
		}
		assert.Equal(t, expected.PeerId, event.PeerId)
	})

	msg := `{"jsonrpc":"2.0","method":"disconnect","params":{"id":"02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6"}}`

	runTest(t, plugin, msg+"\n\n", "")
}

func await(t *testing.T, wg *sync.WaitGroup) {
	awaitWithTimeout(t, wg, 1*time.Second)
}

func awaitWithTimeout(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		wg.Wait()
		c <- struct{}{}
	}()

	select {
	case <-c:
		// continue
	case <-time.After(timeout):
		t.FailNow()
	}
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
			t.Log(err)
			t.FailNow()
		}
	}(progIn, progOut)

	// call the method
	// would using a client implementation be nice here?
	testOut.Write([]byte(inputMsg))

	// Allow early escape for notifications
	if expectedMsg == "" {
		return
	}

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

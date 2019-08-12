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
				PeerId:         "02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6",
				Addr:           "127.0.0.1:58366",
				GlobalFeatures: "",
				LocalFeatures:  "aa",
			}
			assert.Equal(t, expected, event.Peer)
			return event.Continue(), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"peer_connected","params":{"peer":{"id":"02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6","addr":"127.0.0.1:58366","globalfeatures":"","localfeatures":"aa"}}}`
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
				PeerId:         "02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6",
				Addr:           "127.0.0.1:58366",
				GlobalFeatures: "",
				LocalFeatures:  "aa",
			}
			assert.Equal(t, expected, event.Peer)
			return event.Disconnect("there is a problem"), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"peer_connected","params":{"peer":{"id":"02c0114aac5ea2bce7759eb48d5aa75129700c1eb7fe6cc8743968a202f26505d6","addr":"127.0.0.1:58366","globalfeatures":"","localfeatures":"aa"}}}`
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

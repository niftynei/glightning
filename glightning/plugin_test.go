package glightning_test

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/niftynei/glightning/glightning"
	"github.com/niftynei/glightning/jrpc2"
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
	greeting, err := hi.plugin.GetOption("greeting")
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf("Hello, %s", greeting), nil
}

func getInitFunc(t *testing.T, testFn func(t *testing.T, opt map[string]glightning.Option, config *glightning.Config)) func(*glightning.Plugin, map[string]glightning.Option, *glightning.Config) {
	return func(plugin *glightning.Plugin, options map[string]glightning.Option, config *glightning.Config) {
		testFn(t, options, config)
	}
}

func nullInitFunc(plugin *glightning.Plugin, options map[string]glightning.Option, config *glightning.Config) {
	// does nothing
}

func OnPeerConnect(event *glightning.PeerConnectedEvent) (*glightning.PeerConnectedResponse, error) {
	return nil, nil
}

func OnDbWrite(event *glightning.DbWriteEvent) (*glightning.DbWriteResponse, error) {
	return event.Fail(), nil
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

	option := glightning.NewOption("greeting", "How you'd like to be called", "Mary")

	initTestFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
		option.Val = "Jenny"
		assert.Equal(t, option, options["greeting"])
		assert.Equal(t, "rpc.file", config.RpcFile)
		assert.Equal(t, "dirforlightning", config.LightningDir)
		assert.Equal(t, true, config.Startup)
		assert.Equal(t, "testnet", config.Network)
	})
	plugin := glightning.NewPlugin(initTestFn)
	plugin.RegisterOption(option)
	plugin.RegisterMethod(glightning.NewRpcMethod(NewHiMethod(plugin), "Send a greeting."))

	initJson := "{\"jsonrpc\":\"2.0\",\"method\":\"init\",\"params\":{\"options\":{\"greeting\":\"Jenny\"},\"configuration\":{\"rpc-file\":\"rpc.file\",\"startup\":true,\"network\":\"testnet\",\"lightning-dir\":\"dirforlightning\"}},\"id\":1}\n\n"

	expectedJson := "{\"jsonrpc\":\"2.0\",\"result\":\"ok\",\"id\":1}"
	runTest(t, plugin, initJson, expectedJson)
}

func TestMissingOptionRpcCall(t *testing.T) {
	initTestFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterMethod(glightning.NewRpcMethod(NewHiMethod(plugin), "Send a greeting."))
	plugin.RegisterOption(glightning.NewOption("greeting", "How you'd like to be called", "Mary"))
	plugin.SubscribeConnect(HandleConnect)
	plugin.SetDynamic(true)

	msg := "{\"jsonrpc\":\"2.0\",\"method\":\"getmanifest\",\"id\":\"aloha\"}\n\n"
	resp := "{\"jsonrpc\":\"2.0\",\"result\":{\"options\":[{\"name\":\"greeting\",\"type\":\"string\",\"default\":\"Mary\",\"description\":\"How you'd like to be called\"}],\"rpcmethods\":[{\"name\":\"hi\",\"description\":\"Send a greeting.\",\"usage\":\"\"}],\"dynamic\":true,\"subscriptions\":[\"connect\"],\"featurebits\":{}},\"id\":\"aloha\"}"
	runTest(t, plugin, msg, resp)
}

func TestManifestWithHooks(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	resp := `{"jsonrpc":"2.0","result":{"options":[],"rpcmethods":[],"dynamic":true,"hooks":["db_write","peer_connected","invoice_payment","openchannel","htlc_accepted"],"featurebits":{}},"id":"aloha"}`
	runTest(t, plugin, msg, resp)
}

func TestHook_DbWriteOk(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		DbWrite: func(event *glightning.DbWriteEvent) (*glightning.DbWriteResponse, error) {
			assert.Equal(t, 3, len(event.Writes))
			writesExp := []string{
				"BEGIN TRANSACTION;",
				"UPDATE vars SET val='2' WHERE name='bip32_max_index';",
				"COMMIT;",
			}
			assert.Equal(t, writesExp, event.Writes)
			return event.Continue(), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"db_write","params":{"writes":["BEGIN TRANSACTION;","UPDATE vars SET val='2' WHERE name='bip32_max_index';","COMMIT;"]}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"continue"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_DbWriteFail(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		DbWrite: func(event *glightning.DbWriteEvent) (*glightning.DbWriteResponse, error) {
			assert.Equal(t, 3, len(event.Writes))
			writesExp := []string{
				"BEGIN TRANSACTION;",
				"UPDATE vars SET val='2' WHERE name='bip32_max_index';",
				"COMMIT;",
			}
			assert.Equal(t, writesExp, event.Writes)
			return event.Fail(), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"db_write","params":{"writes":["BEGIN TRANSACTION;","UPDATE vars SET val='2' WHERE name='bip32_max_index';","COMMIT;"]}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"fail"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_PeerConnectedOk(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		HtlcAccepted: func(event *glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
			expected := &glightning.HtlcAcceptedEvent{
				Onion: glightning.Onion{
					Payload: "0000000000000000000000000000c3500000014b000000000000000000000000",
					PerHop: &glightning.PerHop{
						Realm:                      "00",
						ShortChannelId:             "0x0x0",
						ForwardAmountMilliSatoshis: "50000msat",
						OutgoingCltvValue:          331,
					},
					NextOnion:    "0003ff512b805f80ba69052342482401cda5f4986ed8600316ba8caf72b4fcc5826f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000171bd5ff2190087d1572def9e35dd7c14e9ab3e06ce797b0480c9ccc8fb00a7d2390daf2fcead26d6775d7feac22f21fce0353b7fe77491401d18f7d69379c336d0000000000000000000000000000000000000000000000000000000000000000",
					SharedSecret: "eb5ab3e3db045e589597687e0eba89a98af0d19fd1967e1f24feb6f2814cb9c5",
				},
				Htlc: glightning.HtlcOffer{
					AmountMilliSatoshi: "50000msat",
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

func TestHook_HtlcAcceptTlvHop(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		HtlcAccepted: func(event *glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
			expected := &glightning.HtlcAcceptedEvent{
				Onion: glightning.Onion{
					Payload:        "1202030186a204018406080000680000010000",
					Type:           "tlv",
					NextOnion:      "0002674db4d6f1b9c1bcbb8567eaf89f6f34fb2900f166ec7290ddc4f5390f2954786df2109e796d901b04e03dfa7087a0cf98601d94b85ee7ebc17b2e0bfa04c14b6a9077ae1c4f6dd790b4a92b3ea06846f345052d0733c2d6fe7bb95f3e763aa066f4101c3e9bf77b4d3e7965c572630a8cf662452b16c0a26f8646a0020aa225efb201fba354157a93b08232a6300fbd175108dba41e7ed5e882930fd23c820176ccbb295b38ea90342f87cad58eb51e95cdea0cfcb749efe690cb38d3b0d4864804980ebffc2b3bcc988396bbaca07acf5215230b4810975e07e31160affebb0f31e375f9c3a4f0c87c27bafc086c59e76e2047816d4640df2f0c5b460b9252569177b0a900cbfb3802df7b560c5a95454f9c2c792c5cbdeb4397f96892fa586a857d4061d86d23eb2363df18f659f8e52aa42924f3b71d58c02dc226a9c04f013fd43babe1af235accf49cefdd2c77226baad6e6ae48e36e94940bc6f9492b5494e6a02bbe2f68a9f6b44572f8bc8a9d1a8af78a95d155674e5068d81070634520daeeda9c75f17720825b8baaa2c285911aa44df66853a9e754f56f048a9e1c5fb26a44767f1648191b79578ff601b430588388d635bdf0fc39552a823891f781f81463209e0c99622d2c3bed5d6d9ccb30143ce7d69940e0b21bfb60a126d77700ff6e751ffecb238e63be30306a6ec4e746b605e6ac6b287c7f3effe0e9644bdc5c3e5e80de2f468cf90ffba99a6424371333a72c661449984ce50b95749a8fbb2c3caf6acf833aca5d391f754406a0916d38a07ae8041b0797d91113bfcfc7b872437232c4a8a24c989bae2b3bc3caa3d316cf66ff250c583a7807c4720c34dfb1f8c75fa4baf8e85f4476e7d3c851e5c350f6845db1d0652c0ecf577571197804bc8f87b095727eb0d4696398aa3c811895c5f0284748f3da18f4dbcfcb61651b94b2c3bab9888ef3b80d8cfd83bb526aef34b4e114dbaa0446cb06210c838f14e2e13ff243df9f9f5a1f99c8e5e3288e48d880aa791a08a45daf0e5c43303148d29f690c8b278dcd7a1f48ee7e06ac3a83e6a2427e7d44cbe4330697ff96a7a695e21bcdd083fa72c885c989ea9043a55e5492f678a92eba07cbc6ffe107ea37ffd8dea34c7ae4e6b8a91e503e08548db9e3c58d04d4fc5b2a45feb51345c799a8c2a5f1d25d628712b6c50129ac4e30f827600c6230fc65d2dc499f1a8192829ce2d43e0371e94309673652621ca75b537ba57db0060d34e3667117a9028ce68541bc7410a62732ae8f5709566c7830bc1f71dbb97d016c8d8601644539375b6c757cd3701c4dbd21108551ab99cff4d92e2fefa8f27abb0628629f51468ca86613e0f06d94f2ff7f1ca3cdc5d0476c10d334d3c966832f93428b4cf3a001fbe1dccd497a4e2931d3042f27fd20a197ec491a81054e473b42b3655eff453e7655d9efe7db50d8224219d57fd053c0ee3d8517a65b04bf5f0f1de0fb0002bf75ccb82d69f7aaffbe813d712b893c764157378907481d19ac9b236de5c38d311b60854a3b5b7c0b7810125083102bff455cb6d4b6c2e6b8d834d9c0e13c9c79cf63198965b6dc55326d4119d668dacde955c0b6e4c061a8d3ad8a963e7a1c22637c38d132e85a18666084654dbd759334224c95d67c09896234c328a030f4988828eaebd22065a2e41e871aa47257862f80e598a82e4efde539de3cad1d786318562e1dee36399d0fea06e01e346dafd18d644ea5a8e2c66681cd622bb7f4e0dc099ba360aa95a782821f4a723036993c81348626af4bb0463075e9a0b6fb4b78c4bdff9cfc974f02c0d1f7d82739db6940b3131cea338320776a1bd553fafae3ef1c5fde743e74286a2b105a55f332bcd3cef611f04fd105b28e1fe6e3de13eb8e41cb785e0eda0ddda7641d838048a5",
					SharedSecret:   "90f681e4fdb8626ac4953c7f5dc035cc6318ece9ec78ed3fb931446c4644b5a6",
					ForwardAmount:  "100002msat",
					OutgoingCltv:   132,
					ShortChannelId: "104x1x0",
				},
				Htlc: glightning.HtlcOffer{
					AmountMilliSatoshi: "100004msat",
					CltvExpiry:         138,
					CltvExpiryRelative: 23,
					PaymentHash:        "b929d8ae3fa7a61c1e3dc6eff5dbfc201e242e6c7286442380520e0c5e6d0e0c",
				},
			}
			assert.Equal(t, expected.Onion, event.Onion)
			assert.Equal(t, expected.Htlc, event.Htlc)
			return event.Continue(), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"htlc_accepted","params":{"onion":{"payload":"1202030186a204018406080000680000010000","type":"tlv","short_channel_id":"104x1x0","forward_amount":"100002msat","outgoing_cltv_value":132,"next_onion":"0002674db4d6f1b9c1bcbb8567eaf89f6f34fb2900f166ec7290ddc4f5390f2954786df2109e796d901b04e03dfa7087a0cf98601d94b85ee7ebc17b2e0bfa04c14b6a9077ae1c4f6dd790b4a92b3ea06846f345052d0733c2d6fe7bb95f3e763aa066f4101c3e9bf77b4d3e7965c572630a8cf662452b16c0a26f8646a0020aa225efb201fba354157a93b08232a6300fbd175108dba41e7ed5e882930fd23c820176ccbb295b38ea90342f87cad58eb51e95cdea0cfcb749efe690cb38d3b0d4864804980ebffc2b3bcc988396bbaca07acf5215230b4810975e07e31160affebb0f31e375f9c3a4f0c87c27bafc086c59e76e2047816d4640df2f0c5b460b9252569177b0a900cbfb3802df7b560c5a95454f9c2c792c5cbdeb4397f96892fa586a857d4061d86d23eb2363df18f659f8e52aa42924f3b71d58c02dc226a9c04f013fd43babe1af235accf49cefdd2c77226baad6e6ae48e36e94940bc6f9492b5494e6a02bbe2f68a9f6b44572f8bc8a9d1a8af78a95d155674e5068d81070634520daeeda9c75f17720825b8baaa2c285911aa44df66853a9e754f56f048a9e1c5fb26a44767f1648191b79578ff601b430588388d635bdf0fc39552a823891f781f81463209e0c99622d2c3bed5d6d9ccb30143ce7d69940e0b21bfb60a126d77700ff6e751ffecb238e63be30306a6ec4e746b605e6ac6b287c7f3effe0e9644bdc5c3e5e80de2f468cf90ffba99a6424371333a72c661449984ce50b95749a8fbb2c3caf6acf833aca5d391f754406a0916d38a07ae8041b0797d91113bfcfc7b872437232c4a8a24c989bae2b3bc3caa3d316cf66ff250c583a7807c4720c34dfb1f8c75fa4baf8e85f4476e7d3c851e5c350f6845db1d0652c0ecf577571197804bc8f87b095727eb0d4696398aa3c811895c5f0284748f3da18f4dbcfcb61651b94b2c3bab9888ef3b80d8cfd83bb526aef34b4e114dbaa0446cb06210c838f14e2e13ff243df9f9f5a1f99c8e5e3288e48d880aa791a08a45daf0e5c43303148d29f690c8b278dcd7a1f48ee7e06ac3a83e6a2427e7d44cbe4330697ff96a7a695e21bcdd083fa72c885c989ea9043a55e5492f678a92eba07cbc6ffe107ea37ffd8dea34c7ae4e6b8a91e503e08548db9e3c58d04d4fc5b2a45feb51345c799a8c2a5f1d25d628712b6c50129ac4e30f827600c6230fc65d2dc499f1a8192829ce2d43e0371e94309673652621ca75b537ba57db0060d34e3667117a9028ce68541bc7410a62732ae8f5709566c7830bc1f71dbb97d016c8d8601644539375b6c757cd3701c4dbd21108551ab99cff4d92e2fefa8f27abb0628629f51468ca86613e0f06d94f2ff7f1ca3cdc5d0476c10d334d3c966832f93428b4cf3a001fbe1dccd497a4e2931d3042f27fd20a197ec491a81054e473b42b3655eff453e7655d9efe7db50d8224219d57fd053c0ee3d8517a65b04bf5f0f1de0fb0002bf75ccb82d69f7aaffbe813d712b893c764157378907481d19ac9b236de5c38d311b60854a3b5b7c0b7810125083102bff455cb6d4b6c2e6b8d834d9c0e13c9c79cf63198965b6dc55326d4119d668dacde955c0b6e4c061a8d3ad8a963e7a1c22637c38d132e85a18666084654dbd759334224c95d67c09896234c328a030f4988828eaebd22065a2e41e871aa47257862f80e598a82e4efde539de3cad1d786318562e1dee36399d0fea06e01e346dafd18d644ea5a8e2c66681cd622bb7f4e0dc099ba360aa95a782821f4a723036993c81348626af4bb0463075e9a0b6fb4b78c4bdff9cfc974f02c0d1f7d82739db6940b3131cea338320776a1bd553fafae3ef1c5fde743e74286a2b105a55f332bcd3cef611f04fd105b28e1fe6e3de13eb8e41cb785e0eda0ddda7641d838048a5","shared_secret":"90f681e4fdb8626ac4953c7f5dc035cc6318ece9ec78ed3fb931446c4644b5a6"},"htlc":{"amount":"100004msat","cltv_expiry":138,"cltv_expiry_relative":23,"payment_hash":"b929d8ae3fa7a61c1e3dc6eff5dbfc201e242e6c7286442380520e0c5e6d0e0c"}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"continue"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_HtlcAcceptTlvEndpoint(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
		t.Error("Should not have called init when calling get manifest")
	})
	plugin := glightning.NewPlugin(initFn)
	plugin.RegisterHooks(&glightning.Hooks{
		HtlcAccepted: func(event *glightning.HtlcAcceptedEvent) (*glightning.HtlcAcceptedResponse, error) {
			expected := &glightning.HtlcAcceptedEvent{
				Onion: glightning.Onion{
					Payload:           "2d02030186a2040184082344428360c90c7aa6c6838c4f2105c42486afca9e99990adecb80643219ae68310186a2",
					Type:              "tlv",
					NextOnion:         "0002a6ec00763ee3dc03d5b4d1d5a7232957be558a1de2f21484c5d6141d55e3a5ba9f41ce201ac737abd36bc9deff156109c614526c1ebb746f928e86ad21b5332f2623744a2035990ca70a38d6a4361a4e0c20a26633a9bdeaec0142fb618eda15c19c5bc84b89f9ebef517bcf739d720d45e1b9cc0c6324ce6494e0f111f699d0dfdf6ade8e3f2badc8175dd1537a22ef64741b65d09c460d86bf3ec777969e292495c20350fb58f70d563e8b5ece48f5c4a155f595ddb1fa4576bf01e03128bfc0c0d67df13d78739a51d95d5a673a2245fccc57225a70c39377baa1015ec13a7c0460621eb1de9ad98587af66bb3e69bec7a9fb75a99e0d4f3d7c92f0c4a8d04a84a2d83fb03f830b01bca191b980a0a4d509536d2a4c2697ba2384639e6a9611b30031c312d93f9fa5977424b4d42d005a3a9d41d6ab361bfd7816440b03410b9c8800c213cf481a7e4b2f4d7edd971eb03be2f6ff0969a1f3c38e36f20f97779066d08ed80710e86c467669369513f82292f61840dc91cdf7a6541d0958a571420b6f53888d9d2af5e023adab876b445945c3ac4778ae380a3b33b1cb9ee1db6299ea12f9a3c263da4c4e811af9ead92428d784d612eab61d3ca1e2c20a32da17c8797850ef29657b441050f607057e3e9a2b9ec7549783dc617ef08ded3a1e5aaf515863d4e12c3208820d126e275299dae58c25a1c21c3c35283e9f5bfb468221e39f43ac0650ebfa8cc2a0060dbaa6847008650f2590bba7cff8ff7802b9f36a3bedcf57d5c198eaaacbc88d8ae943fe86315fe8df865c0a3d5a2787dd3a7cbc9140de51a18f4e009d4e91eb365162c715413ab551323c563347a16278c36a0b4d5f1507577e0d0d162ca80720187021903f3908e5e792832def3058a3ac1afea3535d35a0d85497fea7181cdf23e79a5ef1fe422d8da068b8b3c20a5f1c90b68a780e4199b3a2d4053055af03a981b39bc1660b34edfb93e232165ca584876adc7aba204666eae8eb5d8040c17b09adb853fc7ce7ae5257d0e244dbca59aea6c787a9aad980e273a5f99064d429342ac6175678e34d2e3004a83f1ba51bb25cb701272818487dcf4ff7e19e4d8d415f0c0e83c4140294bec3a3948ba1f9852cf7fc03ee2d6b6736cd87864a6afbe85b990ca75757cd95593d0bfca37c1c0d6acb998abc296ebbc37735750a6555ec0c90076b65b83e83ad29532e669a3e496ff4de23597097e4267842423fc658f3264d2dd8d39c661aff6688c6cc6b4704c81871193c8ca7d08e3520f54f0c0386b2e4cd4901c64503623a0978c6938e464912e82dffd4c605eeacf1d649968e4977b6eca81df645ddde7ce7c77efedffb804a558913cda31797f2aa4cf071600620e36cf0aeeabb697595b0d8d32690065b18bd010c6957575ce1a80fb36d76d7f214c80bce34226ad1c2244fc0b806092c4984d1213675f922f8d8a70321d5890aee58fee8f826c6a02f1fc319934228a41511e279dc384670d7b6154dd55997b81d302f60acbec8194f8d3197fe9ae5a8e6884468b654dc4290abecdddc1fd6a1dc3af62002f49164cd5d0eb341a4281feaa184e2ffc9b5d3bf4860492fa5f583e86cb71af0d6dd9cfef9aa3002c960d11e2faf6b38070a9ee18a053abe12abaa6239d922cdcbde66ac8fc09c0aaf3e721a9f1c0f7133905549ae4e4852fccad8fb11157bb801f60483a31a70237a63b7d9f01852a86355bbd7a4d9c19f8f5ea3debb3e94180621a76ffe70c593ced77e1289199af9c0e3df5c0b70fd3aa0c431fad05c21658230e910ff5d212d1fe53a7a05dd8080879f7ec263a7f24f2b21c6368663584129f10a50d36a52544fef8125e0276bfbe45ab1faf017ef361f14ee7b0000000000000000000000000000000000000000000000000000000000000000",
					SharedSecret:      "c30185ff88b15ddf5b3fdeedccd0373673b9b62b52141c5308934e99dc30714f",
					ForwardAmount:     "100002msat",
					OutgoingCltv:      132,
					TotalMilliSatoshi: "100002msat",
					PaymentSecret:     "44428360c90c7aa6c6838c4f2105c42486afca9e99990adecb80643219ae6831",
				},
				Htlc: glightning.HtlcOffer{
					AmountMilliSatoshi: "100002msat",
					CltvExpiry:         132,
					CltvExpiryRelative: 17,
					PaymentHash:        "b929d8ae3fa7a61c1e3dc6eff5dbfc201e242e6c7286442380520e0c5e6d0e0c",
				},
			}
			assert.Equal(t, expected.Onion, event.Onion)
			assert.Equal(t, expected.Htlc, event.Htlc)
			return event.Continue(), nil
		},
	})

	msg := `{"jsonrpc":"2.0","id":"aloha","method":"htlc_accepted","params":{"onion":{"payload":"2d02030186a2040184082344428360c90c7aa6c6838c4f2105c42486afca9e99990adecb80643219ae68310186a2","type":"tlv","forward_amount":"100002msat","outgoing_cltv_value":132,"total_msat":"100002msat","payment_secret":"44428360c90c7aa6c6838c4f2105c42486afca9e99990adecb80643219ae6831","next_onion":"0002a6ec00763ee3dc03d5b4d1d5a7232957be558a1de2f21484c5d6141d55e3a5ba9f41ce201ac737abd36bc9deff156109c614526c1ebb746f928e86ad21b5332f2623744a2035990ca70a38d6a4361a4e0c20a26633a9bdeaec0142fb618eda15c19c5bc84b89f9ebef517bcf739d720d45e1b9cc0c6324ce6494e0f111f699d0dfdf6ade8e3f2badc8175dd1537a22ef64741b65d09c460d86bf3ec777969e292495c20350fb58f70d563e8b5ece48f5c4a155f595ddb1fa4576bf01e03128bfc0c0d67df13d78739a51d95d5a673a2245fccc57225a70c39377baa1015ec13a7c0460621eb1de9ad98587af66bb3e69bec7a9fb75a99e0d4f3d7c92f0c4a8d04a84a2d83fb03f830b01bca191b980a0a4d509536d2a4c2697ba2384639e6a9611b30031c312d93f9fa5977424b4d42d005a3a9d41d6ab361bfd7816440b03410b9c8800c213cf481a7e4b2f4d7edd971eb03be2f6ff0969a1f3c38e36f20f97779066d08ed80710e86c467669369513f82292f61840dc91cdf7a6541d0958a571420b6f53888d9d2af5e023adab876b445945c3ac4778ae380a3b33b1cb9ee1db6299ea12f9a3c263da4c4e811af9ead92428d784d612eab61d3ca1e2c20a32da17c8797850ef29657b441050f607057e3e9a2b9ec7549783dc617ef08ded3a1e5aaf515863d4e12c3208820d126e275299dae58c25a1c21c3c35283e9f5bfb468221e39f43ac0650ebfa8cc2a0060dbaa6847008650f2590bba7cff8ff7802b9f36a3bedcf57d5c198eaaacbc88d8ae943fe86315fe8df865c0a3d5a2787dd3a7cbc9140de51a18f4e009d4e91eb365162c715413ab551323c563347a16278c36a0b4d5f1507577e0d0d162ca80720187021903f3908e5e792832def3058a3ac1afea3535d35a0d85497fea7181cdf23e79a5ef1fe422d8da068b8b3c20a5f1c90b68a780e4199b3a2d4053055af03a981b39bc1660b34edfb93e232165ca584876adc7aba204666eae8eb5d8040c17b09adb853fc7ce7ae5257d0e244dbca59aea6c787a9aad980e273a5f99064d429342ac6175678e34d2e3004a83f1ba51bb25cb701272818487dcf4ff7e19e4d8d415f0c0e83c4140294bec3a3948ba1f9852cf7fc03ee2d6b6736cd87864a6afbe85b990ca75757cd95593d0bfca37c1c0d6acb998abc296ebbc37735750a6555ec0c90076b65b83e83ad29532e669a3e496ff4de23597097e4267842423fc658f3264d2dd8d39c661aff6688c6cc6b4704c81871193c8ca7d08e3520f54f0c0386b2e4cd4901c64503623a0978c6938e464912e82dffd4c605eeacf1d649968e4977b6eca81df645ddde7ce7c77efedffb804a558913cda31797f2aa4cf071600620e36cf0aeeabb697595b0d8d32690065b18bd010c6957575ce1a80fb36d76d7f214c80bce34226ad1c2244fc0b806092c4984d1213675f922f8d8a70321d5890aee58fee8f826c6a02f1fc319934228a41511e279dc384670d7b6154dd55997b81d302f60acbec8194f8d3197fe9ae5a8e6884468b654dc4290abecdddc1fd6a1dc3af62002f49164cd5d0eb341a4281feaa184e2ffc9b5d3bf4860492fa5f583e86cb71af0d6dd9cfef9aa3002c960d11e2faf6b38070a9ee18a053abe12abaa6239d922cdcbde66ac8fc09c0aaf3e721a9f1c0f7133905549ae4e4852fccad8fb11157bb801f60483a31a70237a63b7d9f01852a86355bbd7a4d9c19f8f5ea3debb3e94180621a76ffe70c593ced77e1289199af9c0e3df5c0b70fd3aa0c431fad05c21658230e910ff5d212d1fe53a7a05dd8080879f7ec263a7f24f2b21c6368663584129f10a50d36a52544fef8125e0276bfbe45ab1faf017ef361f14ee7b0000000000000000000000000000000000000000000000000000000000000000","shared_secret":"c30185ff88b15ddf5b3fdeedccd0373673b9b62b52141c5308934e99dc30714f"},"htlc":{"amount":"100002msat","cltv_expiry":132,"cltv_expiry_relative":17,"payment_hash":"b929d8ae3fa7a61c1e3dc6eff5dbfc201e242e6c7286442380520e0c5e6d0e0c"}}}`
	resp := `{"jsonrpc":"2.0","result":{"result":"continue"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_HtlcAcceptResolve(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	resp := `{"jsonrpc":"2.0","result":{"result":"continue"},"id":"aloha"}`
	runTest(t, plugin, msg+"\n\n", resp)
}

func TestHook_InvoicePaymentFail(t *testing.T) {
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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
	initFn := getInitFunc(t, func(t *testing.T, options map[string]glightning.Option, config *glightning.Config) {
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

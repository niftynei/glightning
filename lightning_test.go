package golight_test

import (
	"testing"
	"github.com/niftynei/golight"
	"github.com/stretchr/testify/assert"
	"net"
	"os"
	"io"
	"io/ioutil"
	"bufio"
	"fmt"
)

func TestDecodePay(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)

	bolt11 := "lnbc2500u1pvjluezpp5qqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqypqdq5xysxxatsyp3k7enxv4jsxqzpuaztrnwngzn3kdzw5hydlzf03qdgm2hdq27cqv3agm2awhz5se903vruatfhq77w3ls4evs3ch9zw97j25emudupq63nyw24cg27h2rspfj9srp"
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"decodepay\",\"params\":{\"bolt11\":\"lnbc2500u1pvjluezpp5qqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqypqdq5xysxxatsyp3k7enxv4jsxqzpuaztrnwngzn3kdzw5hydlzf03qdgm2hdq27cqv3agm2awhz5se903vruatfhq77w3ls4evs3ch9zw97j25emudupq63nyw24cg27h2rspfj9srp\"},\"id\":1}"
	resp := wrapResult(1, `{ "currency": "bc", "created_at": 1496314658, "expiry": 60, "payee": "03e7156ae33b0a208d0744199163177e909e80176e55d97a2f221ede0f934dd9ad", "msatoshi": 250000000, "description": "1 cup of coffee", "min_final_cltv_expiry": 9, "payment_hash": "0001020304050607080900010203040506070809000102030405060708090102", "signature": "3045022100e89639ba6814e36689d4b91bf125f10351b55da057b00647a8dabaeb8a90c95f0220160f9d5a6e0f79d1fc2b964238b944e2fa4aa677c6f020d466472ab842bd750e" } `)
	go runServerSide(t, req, resp, replyQ, requestQ)
	decodedBolt, err := lightning.DecodePay(bolt11, "")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &golight.DecodedBolt11{
		Currency: "bc",
		CreatedAt: uint64(1496314658),
		Expiry: uint64(60),
		Payee: "03e7156ae33b0a208d0744199163177e909e80176e55d97a2f221ede0f934dd9ad",
		MilliSatoshis: 250000000,
		Description: "1 cup of coffee",
		MinFinalCltvExpiry: 9,
		PaymentHash: "0001020304050607080900010203040506070809000102030405060708090102",
		Signature: "3045022100e89639ba6814e36689d4b91bf125f10351b55da057b00647a8dabaeb8a90c95f0220160f9d5a6e0f79d1fc2b964238b944e2fa4aa677c6f020d466472ab842bd750e",


	}, decodedBolt)
}

func TestDecodePayWithDescAndFallbacks(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)

	bolt11 := "lnbc20m1pvjluezpp5qqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqqqsyqcyq5rqwzqfqypqhp58yjmdan79s6qqdhdzgynm4zwqd5d7xmw5fk98klysy043l2ahrqsfpp3qjmp7lwpagxun9pygexvgpjdc4jdj85fr9yq20q82gphp2nflc7jtzrcazrra7wwgzxqc8u7754cdlpfrmccae92qgzqvzq2ps8pqqqqqqpqqqqq9qqqvpeuqafqxu92d8lr6fvg0r5gv0heeeqgcrqlnm6jhphu9y00rrhy4grqszsvpcgpy9qqqqqqgqqqqq7qqzqj9n4evl6mr5aj9f58zp6fyjzup6ywn3x6sk8akg5v4tgn2q8g4fhx05wf6juaxu9760yp46454gpg5mtzgerlzezqcqvjnhjh8z3g2qqdhhwkj"
	desc := "One piece of chocolate cake, one icecream cone, one pickle, one slice of swiss cheese, one slice of salami, one lollypop, one piece of cherry pie, one sausage, one cupcake, and one slice of watermelon"
	req := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"method\":\"decodepay\",\"params\":{\"bolt11\":\"%s\",\"description\":\"%s\"},\"id\":1}", bolt11, desc)
	resp := wrapResult(1, `{
  "currency": "bc",
  "created_at": 1496314658,
  "expiry": 3600,
  "payee": "03e7156ae33b0a208d0744199163177e909e80176e55d97a2f221ede0f934dd9ad",
  "msatoshi": 2000000000,
  "description_hash": "3925b6f67e2c340036ed12093dd44e0368df1b6ea26c53dbe4811f58fd5db8c1",
  "min_final_cltv_expiry": 9,
  "fallbacks": [
    {
      "type": "P2PKH",
      "addr": "1RustyRX2oai4EYYDpQGWvEL62BBGqN9T",
      "hex": "76a91404b61f7dc1ea0dc99424464cc4064dc564d91e8988ac"
    }
  ],
  "routes": [
    [
      {
        "pubkey": "029e03a901b85534ff1e92c43c74431f7ce72046060fcf7a95c37e148f78c77255",
        "short_channel_id": "66051:263430:1800",
        "fee_base_msat": 1,
        "fee_proportional_millionths": 20,
        "cltv_expiry_delta": 3
      },
      {
        "pubkey": "039e03a901b85534ff1e92c43c74431f7ce72046060fcf7a95c37e148f78c77255",
        "short_channel_id": "197637:395016:2314",
        "fee_base_msat": 2,
        "fee_proportional_millionths": 30,
        "cltv_expiry_delta": 4
      }
    ]
  ],
  "payment_hash": "0001020304050607080900010203040506070809000102030405060708090102",
  "signature": "304502210091675cb3fad8e9d915343883a49242e074474e26d42c7ed914655689a80745530220733e8e4ea5ce9b85f69e40d755a55014536b12323f8b220600c94ef2b9c51428"
} `)
	go runServerSide(t, req, resp, replyQ, requestQ)
	decodedBolt, err := lightning.DecodePay(bolt11, desc)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &golight.DecodedBolt11{
		Currency: "bc",
		CreatedAt: uint64(1496314658),
		Expiry: uint64(3600),
		Payee: "03e7156ae33b0a208d0744199163177e909e80176e55d97a2f221ede0f934dd9ad",
		MilliSatoshis: 2000000000,
		DescriptionHash: "3925b6f67e2c340036ed12093dd44e0368df1b6ea26c53dbe4811f58fd5db8c1",
		MinFinalCltvExpiry: 9,
		PaymentHash: "0001020304050607080900010203040506070809000102030405060708090102",
		Signature: "304502210091675cb3fad8e9d915343883a49242e074474e26d42c7ed914655689a80745530220733e8e4ea5ce9b85f69e40d755a55014536b12323f8b220600c94ef2b9c51428",
		Fallbacks: []golight.Fallback{
			golight.Fallback{
				Type: "P2PKH",
				Address: "1RustyRX2oai4EYYDpQGWvEL62BBGqN9T",
				Hex: "76a91404b61f7dc1ea0dc99424464cc4064dc564d91e8988ac",
			},
		},
		Routes: [][]golight.BoltRoute{
			[]golight.BoltRoute{
			golight.BoltRoute{
				Pubkey: "029e03a901b85534ff1e92c43c74431f7ce72046060fcf7a95c37e148f78c77255",
				ShortChannelId: "66051:263430:1800",
				FeeBaseMilliSatoshis: uint64(1),
				FeeProportionalMillionths: uint64(20),
				CltvExpiryDelta: uint(3),
			},
			golight.BoltRoute{
				Pubkey: "039e03a901b85534ff1e92c43c74431f7ce72046060fcf7a95c37e148f78c77255",
				ShortChannelId: "197637:395016:2314",
				FeeBaseMilliSatoshis: uint64(2),
				FeeProportionalMillionths: uint64(30),
				CltvExpiryDelta: uint(4),
			},
			},
		},
	}, decodedBolt)
}

func TestConnect(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)

	peerId := "02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636"
	host := "104.131.77.55"
	port := uint(6666)
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"connect\",\"params\":{\"host\":\"104.131.77.55\",\"id\":\"02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636\",\"port\":6666},\"id\":1}"
	resp := wrapResult(1, `{ "id" : "02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636" }`)
	go runServerSide(t, req, resp, replyQ, requestQ)
	id, err := lightning.Connect(peerId,host,port)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, peerId, id)
}

func TestPing(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)

	peerId := "02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636"
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"ping\",\"params\":{\"id\":\"02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636\",\"len\":128,\"pongbytes\":128},\"id\":1}"
	resp := wrapResult(1, `{ "totlen": 132 }`)
	go runServerSide(t, req, resp, replyQ, requestQ)
	pong, err := lightning.Ping(peerId)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &golight.Pong{
		TotalLen: 132,
	}, pong)
}

func TestPingWithLen(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)

	peerId := "02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636"
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"ping\",\"params\":{\"id\":\"02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636\",\"len\":20,\"pongbytes\":230},\"id\":1}"
	resp := wrapResult(1, `{ "totlen": 234}`)
	go runServerSide(t, req, resp, replyQ, requestQ)
	pong, err := lightning.PingWithLen(peerId, 20, 230)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &golight.Pong{
		TotalLen: 234,
	}, pong)
}

func TestNewAddr(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)

	req := "{\"jsonrpc\":\"2.0\",\"method\":\"newaddr\",\"params\":{\"addresstype\":\"p2sh-segwit\"},\"id\":1}"
	resp := wrapResult(1, `{ "address": "3LfQdff5doR791QNzjn5KdPkFfFn3dmYpc" } `)
	go runServerSide(t, req, resp, replyQ, requestQ)
	addr, err := lightning.NewAddressOfType(golight.P2SHSegwit)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "3LfQdff5doR791QNzjn5KdPkFfFn3dmYpc", addr)

	req = "{\"jsonrpc\":\"2.0\",\"method\":\"newaddr\",\"params\":{\"addresstype\":\"bech32\"},\"id\":2}"
	resp = wrapResult(2, `{ "address": "bc1q4va8cea0ye7hr8f6rwmug7r2rlkvc7lz93zqmh" } `)
	go runServerSide(t, req, resp, replyQ, requestQ)
	addr, err = lightning.NewAddressOfType(golight.Bech32)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "bc1q4va8cea0ye7hr8f6rwmug7r2rlkvc7lz93zqmh", addr)
}

func TestFeeRate(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)

	// what i expect the lightning rpc to generate
	expectedRequest := "{\"jsonrpc\":\"2.0\",\"method\":\"feerates\",\"params\":{\"style\":\"perkb\"},\"id\":1}"
	// json the server will respond with
	reply := "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{ \"perkb\": { \"urgent\": 3328, \"normal\": 1012, \"slow\": 1012, \"min_acceptable\": 1012, \"max_acceptable\": 33280 }, \"onchain_fee_estimates\": { \"opening_channel_satoshis\": 177, \"mutual_close_satoshis\": 170, \"unilateral_close_satoshis\": 497 }}}"

	// queue request & response
	go runServerSide(t, expectedRequest, reply, replyQ, requestQ)
	rates, err := lightning.FeeRates(golight.SatPerKiloByte)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &golight.FeeRateEstimate{
		Style: golight.SatPerKiloByte,
		Details: &golight.FeeRateDetails{
			Urgent: 3328,
			Normal: 1012,
			Slow: 1012,
			MinAcceptable: 1012,
			MaxAcceptable: 33280,
		},
		OnchainEstimate: &golight.OnchainEstimate{
			OpeningChannelSatoshis: 177,
			MutualCloseSatoshis: 170,
			UnilateralCloseSatoshis: 497,
		},
		Warning: "",
	}, rates)

	expectedRequest = "{\"jsonrpc\":\"2.0\",\"method\":\"feerates\",\"params\":{\"style\":\"perkw\"},\"id\":2}"

	reply = "{ \"jsonrpc\":\"2.0\", \"id\":2,\"result\":{\"perkw\": { \"urgent\": 832, \"normal\": 253, \"slow\": 253, \"min_acceptable\": 253, \"max_acceptable\": 8320 }, \"onchain_fee_estimates\": { \"opening_channel_satoshis\": 177, \"mutual_close_satoshis\": 170, \"unilateral_close_satoshis\": 497 }}}"

	// queue request & response
	go runServerSide(t, expectedRequest, reply, replyQ, requestQ)
	rates, err = lightning.FeeRates(golight.SatPerKiloSipa)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &golight.FeeRateEstimate{
		Style: golight.SatPerKiloSipa,
		Details: &golight.FeeRateDetails{
			Urgent: 832,
			Normal: 253,
			Slow: 253,
			MinAcceptable: 253,
			MaxAcceptable: 8320,
		},
		OnchainEstimate: &golight.OnchainEstimate{
			OpeningChannelSatoshis: 177,
			MutualCloseSatoshis: 170,
			UnilateralCloseSatoshis: 497,
		},
		Warning: "",
	}, rates)
}

func runServerSide(t *testing.T, expectedRequest, reply string, replyQ, requestQ chan []byte) {
	// take the request off the requestQ
	request := <-requestQ
	assert.Equal(t, expectedRequest, string(request))
	// send the reply
	replyQ <- []byte(reply + "\n\n")
}

// Set up lightning to talk over a test socket
func startupServer(t *testing.T) (lightning *golight.Lightning, requestQ, replyQ chan []byte) {
	tmpfile, err := ioutil.TempFile("", "rpc.socket")
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(tmpfile.Name())

	requestQueue := make(chan []byte)
	replyQueue := make(chan []byte)
	ok := make(chan bool)

	go func(socket string, t *testing.T, requestQueue, replyQueue chan []byte, ok chan bool) {
		ln, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		for {
			ok<-true
			inconn, err := ln.Accept()
			if err != nil {
				t.Fatal(err)
			}
			go listen(inconn, requestQueue, t)
			go writer(inconn, replyQueue, t)
		}
	}(tmpfile.Name(), t, requestQueue, replyQueue, ok)

	// block until the socket is listening
	<-ok

	lightning = golight.NewLightning()
	lightning.StartUp("", tmpfile.Name())
	return lightning, requestQueue, replyQueue
}

func listen(in io.Reader, requestQueue chan []byte, t *testing.T) {
	scanner := bufio.NewScanner(in)
	buf := make([]byte, 1024)
	scanner.Buffer(buf, 10*1024*1024)
	scanner.Split(scanDoubleNewline)
	for scanner.Scan() {
		requestQueue <- scanner.Bytes()
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func scanDoubleNewline(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' && (i+1) < len(data) && data[i+1] == '\n' {
			return i + 2, data[:i], nil
		}
	}
	return 0, nil, nil
}

func writer(outPipe io.Writer, replyQueue chan []byte, t *testing.T) {
	out := bufio.NewWriter(outPipe)
	twoNewlines := []byte("\n\n")
	for reply := range replyQueue {
		println(string(reply))
		reply = append(reply, twoNewlines...)
		out.Write(reply)
		out.Flush()
	}
}

func wrapResult(id int, result string) string {
	return fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":%s}", id, result)
}

package glightning_test

import (
	"bufio"
	"fmt"
	"github.com/niftynei/glightning/glightning"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net"
	"os"
	"testing"
)

func TestListPeers(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"listpeers","params":{},"id":1}`
	resp := wrapResult(1, `{                                                                                                                                                         
  "peers": [                                                                                                                                              
    {
      "id": "02e3cd7849f177a46f137ae3bfc1a08fc6a90bf4026c74f83c1ecc8430c282fe96",
      "connected": true,
      "netaddr": [
        "0.0.0.0:6677"
      ],
      "global_features": "11",
      "local_features": "8a",
      "globalfeatures": "11",
      "localfeatures": "8a",
      "channels": [
        {
          "state": "CHANNELD_NORMAL",
          "scratch_txid": "cd13ba846709958bfd073155283c3493f08f7db1bb4ef199c014559e5505d18d",
          "owner": "lightning_channeld",
          "short_channel_id": "355x1x0",
	  "direction": 1,
          "channel_id": "5415f1347cf12f30222c5968c59a4744e78ee39f0361e19b6ce2996cce4e1538",
          "funding_txid": "38154ece6c99e26c9be161039fe38ee744479ac568592c22302ff17c34f11554",
	  "private": true,
	  "funding_allocation_msat": {
            "03d3b9f07da45df23f61b3b28eaab84fa024d6d0d0a0c3bbbcca97c3e60e2114b4": 0,
            "028286c0714b0b390096e15615ecd9354ca19021c00ecc0e9dd800636346e04764": 1000000000
          },
          "msatoshi_to_us": 16777215000,
          "msatoshi_to_us_min": 16777215000,
          "msatoshi_to_us_max": 16777215000,
          "msatoshi_total": 16777215000,
          "dust_limit_satoshis": 546,
          "max_htlc_value_in_flight_msat": 18446744073709551615,
          "their_channel_reserve_satoshis": 167773,
          "our_channel_reserve_satoshis": 167773,
          "spendable_msatoshi": 16609442000,
          "htlc_minimum_msat": 10,
          "their_to_self_delay": 6,
          "our_to_self_delay": 6,
          "max_accepted_htlcs": 483,
          "status": [
            "CHANNELD_NORMAL:Funding transaction locked. Channel announced."
          ],
          "in_payments_offered": 110,
          "in_msatoshi_offered": 123,
          "in_payments_fulfilled": 123,
          "in_msatoshi_fulfilled": 123,
          "out_payments_offered": 123,
          "out_msatoshi_offered": 123,
          "out_payments_fulfilled": 123,
          "out_msatoshi_fulfilled": 123,
          "htlcs": [
            {
              "direction": "out",
              "id": 1,
              "msatoshi": 1437433749,
              "expiry": 556832,
              "payment_hash": "3525b49c055604a7997512f866694b6154987a32cc60e1c374113246d38bd5ad",
              "state": "SENT_REMOVE_ACK_COMMIT"
            }
          ]
        }
      ]
    }
  ]
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	peers, err := lightning.ListPeers()
	if err != nil {
		t.Fatal(err)
	}

	fundingAlloc := make(map[string]uint64)
	fundingAlloc["03d3b9f07da45df23f61b3b28eaab84fa024d6d0d0a0c3bbbcca97c3e60e2114b4"] = uint64(0)
	fundingAlloc["028286c0714b0b390096e15615ecd9354ca19021c00ecc0e9dd800636346e04764"] = uint64(1000000000)
	htlcs := []*glightning.Htlc{
		&glightning.Htlc{
			Direction:    "out",
			Id:           1,
			MilliSatoshi: 1437433749,
			Expiry:       556832,
			PaymentHash:  "3525b49c055604a7997512f866694b6154987a32cc60e1c374113246d38bd5ad",
			State:        "SENT_REMOVE_ACK_COMMIT",
		},
	}
	expected := []glightning.Peer{
		glightning.Peer{
			Id:             "02e3cd7849f177a46f137ae3bfc1a08fc6a90bf4026c74f83c1ecc8430c282fe96",
			Connected:      true,
			NetAddresses:   []string{"0.0.0.0:6677"},
			GlobalFeatures: "11",
			LocalFeatures:  "8a",
			Channels: []glightning.PeerChannel{
				glightning.PeerChannel{
					State:                            "CHANNELD_NORMAL",
					ScratchTxId:                      "cd13ba846709958bfd073155283c3493f08f7db1bb4ef199c014559e5505d18d",
					Owner:                            "lightning_channeld",
					ShortChannelId:                   "355x1x0",
					ChannelDirection:                 1,
					ChannelId:                        "5415f1347cf12f30222c5968c59a4744e78ee39f0361e19b6ce2996cce4e1538",
					FundingTxId:                      "38154ece6c99e26c9be161039fe38ee744479ac568592c22302ff17c34f11554",
					Private:                          true,
					FundingAllocations:               fundingAlloc,
					MilliSatoshiToUs:                 16777215000,
					MilliSatoshiToUsMin:              16777215000,
					MilliSatoshiToUsMax:              16777215000,
					MilliSatoshiTotal:                16777215000,
					DustLimitSatoshi:                 546,
					MaxHtlcValueInFlightMilliSatoshi: 18446744073709551615,
					TheirChannelReserveSatoshi:       167773,
					OurChannelReserveSatoshi:         167773,
					SpendableMilliSatoshi:            16609442000,
					HtlcMinMilliSatoshi:              10,
					TheirToSelfDelay:                 6,
					OurToSelfDelay:                   6,
					MaxAcceptedHtlcs:                 483,
					Status: []string{
						"CHANNELD_NORMAL:Funding transaction locked. Channel announced.",
					},
					InPaymentsOffered:        110,
					InMilliSatoshiOffered:    123,
					InPaymentsFulfilled:      123,
					InMilliSatoshiFulfilled:  123,
					OutPaymentsOffered:       123,
					OutMilliSatoshiOffered:   123,
					OutPaymentsFulfilled:     123,
					OutMilliSatoshiFulfilled: 123,
					Htlcs:                    htlcs,
				},
			},
		},
	}
	assert.Equal(t, expected, peers)
}

func TestListForwards(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"listforwards","params":{},"id":1}`
	resp := wrapResult(1, `{
  "forwards": [
    {
      "in_channel": "233x1x0",
      "out_channel": "263x1x0",
      "in_msatoshi": 10001,
      "out_msatoshi": 10000,
      "fee": 1,
      "status": "settled"
    }
  ]
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	forwards, err := lightning.ListForwards()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.Forwarding{
		glightning.Forwarding{
			InChannel:       "233x1x0",
			OutChannel:      "263x1x0",
			MilliSatoshiIn:  10001,
			MilliSatoshiOut: 10000,
			Fee:             1,
			Status:          "settled",
		},
	}, forwards)
}

func TestListPayments(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"listpayments","params":{},"id":1}`
	resp := wrapResult(1, `{                       
  "payments": [                                                      
    {          
      "id": 1,       
      "payment_hash": "3d8705ad509bb52ee01047a4ced0cd4099da92507674e5452d19271f29df2993",
      "destination": "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
      "msatoshi": 10000,                
      "msatoshi_sent": 10001,
      "created_at": 1546480001,
      "status": "complete",
      "payment_preimage": "1ca5dd46bb09fdb03cbb888800f8d18954da991c5368a37cd3d62968ae5bf089"
    }                                                                                    
  ]                                                                                       
} `)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	payments, err := lightning.ListPaymentsAll()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.PaymentFields{
		glightning.PaymentFields{
			Id:               1,
			PaymentHash:      "3d8705ad509bb52ee01047a4ced0cd4099da92507674e5452d19271f29df2993",
			Destination:      "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
			MilliSatoshi:     10000,
			MilliSatoshiSent: 10001,
			CreatedAt:        1546480001,
			Status:           "complete",
			PaymentPreimage:  "1ca5dd46bb09fdb03cbb888800f8d18954da991c5368a37cd3d62968ae5bf089",
		},
	}, payments)
}

func TestPay(t *testing.T) {
	bolt11 := "lnbcrt3u1pwz67h2pp5h694gdd2suutuv2cpscucarmcgmarjpla9rd5vuwu8rtlzkgtgfqdpzvehhygr8dahkgueqv9hxggrnv4e8v6trv5cqp2rzjq0ashz3etfsqsj2xatuce766s84qzrsrql40x696y8nad08sunwyzqqpquqqqqgqqqqqqqqpqqqqqzsqqcvwxa6a3uu2ue80wflztg9ed27vtwu9k6ymtl03yxswnej5qzdw99ndmhwueuckg2ua2g8hfqf0l3mxvn9azs2u6qx0ag3hxye9x6e9qqv29cq5"

	req := `{"jsonrpc":"2.0","method":"pay","params":{"bolt11":"lnbcrt3u1pwz67h2pp5h694gdd2suutuv2cpscucarmcgmarjpla9rd5vuwu8rtlzkgtgfqdpzvehhygr8dahkgueqv9hxggrnv4e8v6trv5cqp2rzjq0ashz3etfsqsj2xatuce766s84qzrsrql40x696y8nad08sunwyzqqpquqqqqgqqqqqqqqpqqqqqzsqqcvwxa6a3uu2ue80wflztg9ed27vtwu9k6ymtl03yxswnej5qzdw99ndmhwueuckg2ua2g8hfqf0l3mxvn9azs2u6qx0ag3hxye9x6e9qqv29cq5"},"id":1}`
	resp := wrapResult(1, `{
  "id": 5,
  "payment_hash": "be8b5435aa8738be31580c31cc747bc237d1c83fe946da338ee1c6bf8ac85a12",
  "destination": "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
  "msatoshi": 300000,
  "msatoshi_sent": 301080,
  "created_at": 1546484611,
  "status": "complete",
  "payment_preimage": "b368340fc5fb5839beaaf59885efa6636557715746be26601cddf876a2bc489b",
  "description": "for goods and service",
  "getroute_tries": 1,
  "sendpay_tries": 1,
  "route": [
    {
      "id": "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
      "channel": "233x1x0",
      "msatoshi": 301080,
      "delay": 16
    },
    {
      "id": "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
      "channel": "263x1x0",
      "msatoshi": 301076,
      "delay": 10
    }
  ],
  "failures": [
  {
      "message": "reply from remote",
      "type": "FAIL_PAYMENT_REPLY",
      "erring_index": 1,
      "failcode": 4103,
      "erring_node": "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
      "erring_channel": "263x1x0",
      "channel_update": "01028f8dc4547391f45988ddb2c46844eacae0cae02e11129087dcbbc27292084c3e430d2827641db78a4be825b3a92ff1690cc0ae236accde60cbb313d2c2bf2d7406226e46111a0b59caaf126043eb5bbf28c34f3a5e332a1fc7b2b73cf188910f00010700000100005c2d7f07010300060000000000000000000000010000000a000000003a699d00",
      "route": [
        {
          "id": "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
          "channel": "233x1x0",
          "msatoshi": 300660,
          "delay": 16
        },
        {
          "id": "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
          "channel": "263x1x0",
          "msatoshi": 300656,
          "delay": 10
        }
      ]
    }
  ]
} `)
	lightning, requestQ, replyQ := startupServer(t)
	// confirm that we're using non-timeout path
	lightning.SetTimeout(0)
	go runServerSide(t, req, resp, replyQ, requestQ)
	payment, err := lightning.PayBolt(bolt11)
	if err != nil {
		t.Fatal(err)
	}
	paymentFields := &glightning.PaymentFields{
		Id:               5,
		PaymentHash:      "be8b5435aa8738be31580c31cc747bc237d1c83fe946da338ee1c6bf8ac85a12",
		Destination:      "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
		MilliSatoshi:     300000,
		MilliSatoshiSent: 301080,
		CreatedAt:        1546484611,
		Status:           "complete",
		PaymentPreimage:  "b368340fc5fb5839beaaf59885efa6636557715746be26601cddf876a2bc489b",
		Description:      "for goods and service",
	}
	route := []glightning.RouteHop{
		glightning.RouteHop{
			Id:             "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
			ShortChannelId: "233x1x0",
			MilliSatoshi:   301080,
			Delay:          16,
		},
		glightning.RouteHop{
			Id:             "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
			ShortChannelId: "263x1x0",
			MilliSatoshi:   301076,
			Delay:          10,
		},
	}
	failroute := []glightning.RouteHop{
		glightning.RouteHop{
			Id:             "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
			ShortChannelId: "233x1x0",
			MilliSatoshi:   300660,
			Delay:          16,
		},
		glightning.RouteHop{
			Id:             "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
			ShortChannelId: "263x1x0",
			MilliSatoshi:   300656,
			Delay:          10,
		},
	}
	failures := []glightning.PayFailure{
		glightning.PayFailure{
			Message:       "reply from remote",
			Type:          "FAIL_PAYMENT_REPLY",
			ErringIndex:   1,
			FailCode:      4103,
			ErringNode:    "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
			ErringChannel: "263x1x0",
			ChannelUpdate: "01028f8dc4547391f45988ddb2c46844eacae0cae02e11129087dcbbc27292084c3e430d2827641db78a4be825b3a92ff1690cc0ae236accde60cbb313d2c2bf2d7406226e46111a0b59caaf126043eb5bbf28c34f3a5e332a1fc7b2b73cf188910f00010700000100005c2d7f07010300060000000000000000000000010000000a000000003a699d00",
			Route:         failroute,
		},
	}
	expect := &glightning.PaymentSuccess{*paymentFields, 1, 1, route, failures}
	assert.Equal(t, expect, payment)
}

func TestWaitSendPay(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"waitsendpay","params":{"payment_hash":"37ef7c6ff62d5a2fbce1940ab2f4de2785045b922f93944b73f7bc5123ed698f"},"id":1}`
	resp := wrapResult(1, `{
  "id": 4,
  "payment_hash": "37ef7c6ff62d5a2fbce1940ab2f4de2785045b922f93944b73f7bc5123ed698f",
  "destination": "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
  "msatoshi": 10000,
  "msatoshi_sent": 10001,
  "created_at": 1546483736,
  "status": "complete",
  "payment_preimage": "eb7608df66f66d34c688b90346b8fcd904170b10278d797b608cc1168317458d"
}`)
	paymentHash := "37ef7c6ff62d5a2fbce1940ab2f4de2785045b922f93944b73f7bc5123ed698f"
	lightning, requestQ, replyQ := startupServer(t)
	// confirm that we're using non-timeout path
	lightning.SetTimeout(0)
	go runServerSide(t, req, resp, replyQ, requestQ)
	payment, err := lightning.WaitSendPay(paymentHash, 0)
	if err != nil {
		t.Fatal(err)
	}
	paymentFields := &glightning.PaymentFields{
		Id:               4,
		PaymentHash:      paymentHash,
		Destination:      "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
		MilliSatoshi:     10000,
		MilliSatoshiSent: 10001,
		CreatedAt:        1546483736,
		Status:           "complete",
		PaymentPreimage:  "eb7608df66f66d34c688b90346b8fcd904170b10278d797b608cc1168317458d",
	}
	assert.Equal(t, paymentFields, payment)

}

func TestWaitSendPayError(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"waitsendpay","params":{"payment_hash":"37ef7c6ff62d5a2fbce1940ab2f4de2785045b922f93944b73f7bc5123ed698f"},"id":1}`
	resp := wrapError(1, 204, "failed: WIRE_TEMPORARY_CHANNEL_FAILURE", `{"erring_index": 2, "failcode": 4107,
  "erring_node": "038863cf8ab91046230f561cd5b386cbff8309fa02e3f0c3ed161a3aeb64a643b9",
  "erring_channel": "1451409x38x0",
  "erring_direction": 1,
  "channel_update": "0102fc0d7e4831887e04c5abce42f4860869ab984a037c49fd43f16aef81cc42de4075092f0c24e6c8febd42faafb41ffe48f974d1cdb5dc4dc3cebe4eec41881f6043497fd7f826957108f4a30fd9cec3aeba79972084e90ead01ea33090000000016259100002600005c4ab84e0100009000000000000003e8000003e80000000100000003e7fffc18"}`)
	paymentHash := "37ef7c6ff62d5a2fbce1940ab2f4de2785045b922f93944b73f7bc5123ed698f"
	lightning, requestQ, replyQ := startupServer(t)
	// confirm that we're using non-timeout path
	lightning.SetTimeout(0)
	go runServerSide(t, req, resp, replyQ, requestQ)
	_, err := lightning.WaitSendPay(paymentHash, 0)
	if err == nil {
		t.Fatal("Expected error, got nothing")
	}
	payErr, ok := err.(*glightning.PaymentError)
	if !ok {
		t.Fatal(err)
	}
	assert.Equal(t, payErr.Error(), "204:failed: WIRE_TEMPORARY_CHANNEL_FAILURE")
	assert.Equal(t, payErr.Message, "failed: WIRE_TEMPORARY_CHANNEL_FAILURE")
	assert.Equal(t, payErr.Code, 204)
	errData := &glightning.PaymentErrorData{
		ErringIndex:     2,
		FailCode:        4107,
		ErringNode:      "038863cf8ab91046230f561cd5b386cbff8309fa02e3f0c3ed161a3aeb64a643b9",
		ErringChannel:   "1451409x38x0",
		ErringDirection: 1,
		ChannelUpdate:   "0102fc0d7e4831887e04c5abce42f4860869ab984a037c49fd43f16aef81cc42de4075092f0c24e6c8febd42faafb41ffe48f974d1cdb5dc4dc3cebe4eec41881f6043497fd7f826957108f4a30fd9cec3aeba79972084e90ead01ea33090000000016259100002600005c4ab84e0100009000000000000003e8000003e80000000100000003e7fffc18",
	}
	assert.Equal(t, *errData, payErr.Data)

}

func TestSendPay(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"sendpay","params":{"payment_hash":"3d8705ad509bb52ee01047a4ced0cd4099da92507674e5452d19271f29df2993","route":[{"id":"03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41","channel":"233x1x0","msatoshi":10001,"delay":15},{"id":"023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82","channel":"263x1x0","msatoshi":10000,"delay":9}]},"id":1}`
	resp := wrapResult(1, `{
  "message": "Monitor status with listpayments or waitsendpay",
  "id": 1,
  "payment_hash": "3d8705ad509bb52ee01047a4ced0cd4099da92507674e5452d19271f29df2993",
  "destination": "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
  "msatoshi": 10000,
  "msatoshi_sent": 10001,
  "created_at": 1546480001,
  "status": "pending"
}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	route := []glightning.RouteHop{
		glightning.RouteHop{
			Id:             "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
			ShortChannelId: "233x1x0",
			MilliSatoshi:   uint64(10001),
			Delay:          15,
		},
		glightning.RouteHop{
			Id:             "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
			ShortChannelId: "263x1x0",
			MilliSatoshi:   uint64(10000),
			Delay:          9,
		},
	}
	paymentHash := "3d8705ad509bb52ee01047a4ced0cd4099da92507674e5452d19271f29df2993"
	invoice, err := lightning.SendPay(route, paymentHash, "", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	paymentFields := &glightning.PaymentFields{
		Id:               1,
		PaymentHash:      "3d8705ad509bb52ee01047a4ced0cd4099da92507674e5452d19271f29df2993",
		Destination:      "023d0e0719af06baa4aac6a1fc8d291b66e00b0a79c6282ed584ce27742f542a82",
		MilliSatoshi:     10000,
		MilliSatoshiSent: 10001,
		CreatedAt:        1546480001,
		Status:           "pending",
	}
	result := &glightning.SendPayResult{
		"Monitor status with listpayments or waitsendpay",
		*paymentFields,
	}
	assert.Equal(t, result, invoice)
}

func TestWaitAnyInvoice(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"waitanyinvoice","params":{"lastpay_index":1},"id":1}`
	resp := wrapResult(1, `{    
  "label": "bagatab",                                                            
  "bolt11": "lnbcrt100n1pwz6a8wpp5249mj72sysuemctra4gsmexjec066g2ra7qkkp2rwvuzxuyhhesqdq8v3jhxccxqp9cqp2rzjq0ashz3etfsqsj2xatuce766s84qzrsrql40x696y8nad08sunwyzqqpquqqqqgqqqqqqqqpqqqqqzsqqc9ua2tv4kqglsgxnt7l2lcrdajc4juwhtl3jkqqvdnzqfyth5lefx25n0ef8emstfxm4v6dcx8s5ae8ef0ug64nquwdxv9zduggxr8lgpg9m473",
  "payment_hash": "554bb9795024399de163ed510de4d2ce1fad2143ef816b05437338237097be60",
  "msatoshi": 10000,
  "status": "paid",
  "pay_index": 2,
  "msatoshi_received": 10000, 
  "paid_at": 1546482927,
  "description": "desc",
  "expires_at": 1546482931              
}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	invoice, err := lightning.WaitAnyInvoice(1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.CompletedInvoice{
		Label:                "bagatab",
		Bolt11:               "lnbcrt100n1pwz6a8wpp5249mj72sysuemctra4gsmexjec066g2ra7qkkp2rwvuzxuyhhesqdq8v3jhxccxqp9cqp2rzjq0ashz3etfsqsj2xatuce766s84qzrsrql40x696y8nad08sunwyzqqpquqqqqgqqqqqqqqpqqqqqzsqqc9ua2tv4kqglsgxnt7l2lcrdajc4juwhtl3jkqqvdnzqfyth5lefx25n0ef8emstfxm4v6dcx8s5ae8ef0ug64nquwdxv9zduggxr8lgpg9m473",
		PaymentHash:          "554bb9795024399de163ed510de4d2ce1fad2143ef816b05437338237097be60",
		Status:               "paid",
		Description:          "desc",
		ExpiresAt:            1546482931,
		MilliSatoshiReceived: 10000,
		MilliSatoshi:         10000,
		PayIndex:             2,
		PaidAt:               1546482927,
	}, invoice)
}

func TestWaitInvoice(t *testing.T) {

	req := `{"jsonrpc":"2.0","method":"waitinvoice","params":{"label":"gab"},"id":1}`
	resp := wrapResult(1, `{                   
  "label": "gab",
  "bolt11": "lnbcrt100n1pwz66vqpp58krstt2snw6jacqsg7jva5xdgzva4yjswe6w23fdryn372wl9xfsdq8v3jhxccxqp9cqp2rzjq0ashz3etfsqsj2xatuce766s84qzrsrql40x696y8nad08sunwyzqqpquqqqqgqqqqqqqqpqqqqqzsqqcrffyde0s43ylmkypcduqrg7vh2423x6usl4jwyw6jxlsqz2r3s39jqqns2c5wp6lgjffuvlfpwvzkfcp898ea4edvt4tak78qrq3n3qq8mjwlg",
  "payment_hash": "3d8705ad509bb52ee01047a4ced0cd4099da92507674e5452d19271f29df2993",
  "msatoshi": 10000,    
  "status": "paid",
  "pay_index": 1,
  "msatoshi_received": 10000,
  "paid_at": 1546480002,
  "description": "desc",                                                                                                                                                                     
  "expires_at": 1546480005                                                                                                                                                                   
} `)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	invoice, err := lightning.WaitInvoice("gab")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.CompletedInvoice{
		Label:                "gab",
		Bolt11:               "lnbcrt100n1pwz66vqpp58krstt2snw6jacqsg7jva5xdgzva4yjswe6w23fdryn372wl9xfsdq8v3jhxccxqp9cqp2rzjq0ashz3etfsqsj2xatuce766s84qzrsrql40x696y8nad08sunwyzqqpquqqqqgqqqqqqqqpqqqqqzsqqcrffyde0s43ylmkypcduqrg7vh2423x6usl4jwyw6jxlsqz2r3s39jqqns2c5wp6lgjffuvlfpwvzkfcp898ea4edvt4tak78qrq3n3qq8mjwlg",
		PaymentHash:          "3d8705ad509bb52ee01047a4ced0cd4099da92507674e5452d19271f29df2993",
		Status:               "paid",
		Description:          "desc",
		ExpiresAt:            1546480005,
		MilliSatoshiReceived: 10000,
		MilliSatoshi:         10000,
		PayIndex:             1,
		PaidAt:               1546480002,
	}, invoice)
}

func TestDeleteInvoice(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"delinvoice","params":{"label":"uniq","status":"expired"},"id":1}`
	resp := wrapResult(1, `{
  "label": "uniq",
  "bolt11": "lnbcrt10p1pwz6k92pp5qgfu5fzu5g77enmz5e9znz5c3wly94huwcsywyffx2xzl23uedaqdq8v3jhxccxqzxgcqp28685h6tlq0lnz3yueqxhtdhqqq7mrwr6mv9j94zdhxpxfg3cd6y4pum736hwve4wq2pmgswkj7apnxcnu8yn89ve0vrhmt6g0jsxfkcqa5uxfj",
  "payment_hash": "0213ca245ca23deccf62a64a298a988bbe42d6fc7620471129328c2faa3ccb7a",
  "msatoshi": 1,
  "status": "expired",
  "description": "desc",
  "expires_at": 1546475890
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	invoices, err := lightning.DeleteInvoice("uniq", "expired")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.Invoice{
		Label:       "uniq",
		Bolt11:      "lnbcrt10p1pwz6k92pp5qgfu5fzu5g77enmz5e9znz5c3wly94huwcsywyffx2xzl23uedaqdq8v3jhxccxqzxgcqp28685h6tlq0lnz3yueqxhtdhqqq7mrwr6mv9j94zdhxpxfg3cd6y4pum736hwve4wq2pmgswkj7apnxcnu8yn89ve0vrhmt6g0jsxfkcqa5uxfj",
		PaymentHash: "0213ca245ca23deccf62a64a298a988bbe42d6fc7620471129328c2faa3ccb7a",
		Status:      "expired",
		Description: "desc",
		ExpiresAt:   1546475890,
	}, invoices)
}

func TestListInvoices(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"listinvoices","params":{},"id":1}`
	resp := wrapResult(1, `{
  "invoices": [
    {
      "label": "label",
      "bolt11": "lnbcrt1pwz646mpp59plmhlzsnz0yu6twf2mtjmydt40zlle2fzlkkkdzlmxqgqeha2gsdq8v3jhxccxqzxgcqp2vj8dqhg6yyzrvcd7kfwu4svh6k44mv5uy6wetpwfyxav504rthkxhxll2d9e4dwcm7xzpsxy9l9aulpmskepqad2x8vz82krme8zevgq3utwgq",
      "payment_hash": "287fbbfc50989e4e696e4ab6b96c8d5d5e2fff2a48bf6b59a2fecc040337ea91",
      "status": "expired",
      "description": "desc",
      "expires_at": 1546475555
    },
    {
      "label": "uniq",
      "bolt11": "lnbcrt10p1pwz6k92pp5qgfu5fzu5g77enmz5e9znz5c3wly94huwcsywyffx2xzl23uedaqdq8v3jhxccxqzxgcqp28685h6tlq0lnz3yueqxhtdhqqq7mrwr6mv9j94zdhxpxfg3cd6y4pum736hwve4wq2pmgswkj7apnxcnu8yn89ve0vrhmt6g0jsxfkcqa5uxfj",
      "payment_hash": "0213ca245ca23deccf62a64a298a988bbe42d6fc7620471129328c2faa3ccb7a",
      "msatoshi": 1,
      "status": "expired",
      "description": "desc",
      "expires_at": 1546475890
    }
  ]
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	invoices, err := lightning.ListInvoices()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.Invoice{
		glightning.Invoice{
			Label:       "label",
			Bolt11:      "lnbcrt1pwz646mpp59plmhlzsnz0yu6twf2mtjmydt40zlle2fzlkkkdzlmxqgqeha2gsdq8v3jhxccxqzxgcqp2vj8dqhg6yyzrvcd7kfwu4svh6k44mv5uy6wetpwfyxav504rthkxhxll2d9e4dwcm7xzpsxy9l9aulpmskepqad2x8vz82krme8zevgq3utwgq",
			PaymentHash: "287fbbfc50989e4e696e4ab6b96c8d5d5e2fff2a48bf6b59a2fecc040337ea91",
			Status:      "expired",
			Description: "desc",
			ExpiresAt:   1546475555,
		},
		glightning.Invoice{
			Label:       "uniq",
			Bolt11:      "lnbcrt10p1pwz6k92pp5qgfu5fzu5g77enmz5e9znz5c3wly94huwcsywyffx2xzl23uedaqdq8v3jhxccxqzxgcqp28685h6tlq0lnz3yueqxhtdhqqq7mrwr6mv9j94zdhxpxfg3cd6y4pum736hwve4wq2pmgswkj7apnxcnu8yn89ve0vrhmt6g0jsxfkcqa5uxfj",
			PaymentHash: "0213ca245ca23deccf62a64a298a988bbe42d6fc7620471129328c2faa3ccb7a",
			Status:      "expired",
			Description: "desc",
			ExpiresAt:   1546475890,
		},
	}, invoices)
}

func TestGetInvoice(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"listinvoices","params":{"label":"uniq"},"id":1}`
	resp := wrapResult(1, `{
  "invoices": [
    {
      "label": "uniq",
      "bolt11": "lnbcrt10p1pwz6k92pp5qgfu5fzu5g77enmz5e9znz5c3wly94huwcsywyffx2xzl23uedaqdq8v3jhxccxqzxgcqp28685h6tlq0lnz3yueqxhtdhqqq7mrwr6mv9j94zdhxpxfg3cd6y4pum736hwve4wq2pmgswkj7apnxcnu8yn89ve0vrhmt6g0jsxfkcqa5uxfj",
      "payment_hash": "0213ca245ca23deccf62a64a298a988bbe42d6fc7620471129328c2faa3ccb7a",
      "msatoshi": 1,
      "status": "expired",
      "description": "desc",
      "expires_at": 1546475890
    }
  ]
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	invoices, err := lightning.GetInvoice("uniq")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.Invoice{
		glightning.Invoice{
			Label:       "uniq",
			Bolt11:      "lnbcrt10p1pwz6k92pp5qgfu5fzu5g77enmz5e9znz5c3wly94huwcsywyffx2xzl23uedaqdq8v3jhxccxqzxgcqp28685h6tlq0lnz3yueqxhtdhqqq7mrwr6mv9j94zdhxpxfg3cd6y4pum736hwve4wq2pmgswkj7apnxcnu8yn89ve0vrhmt6g0jsxfkcqa5uxfj",
			PaymentHash: "0213ca245ca23deccf62a64a298a988bbe42d6fc7620471129328c2faa3ccb7a",
			Status:      "expired",
			Description: "desc",
			ExpiresAt:   1546475890,
		},
	}, invoices)
}

func TestInvoice(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"invoice","params":{"description":"desc","expiry":200,"exposeprivatechannels":true,"label":"uniq","msatoshi":"1"},"id":1}`
	resp := wrapResult(1, `{
  "payment_hash": "0213ca245ca23deccf62a64a298a988bbe42d6fc7620471129328c2faa3ccb7a",
  "expires_at": 1546475890,
  "bolt11": "lnbcrt10p1pwz6k92pp5qgfu5fzu5g77enmz5e9znz5c3wly94huwcsywyffx2xzl23uedaqdq8v3jhxccxqzxgcqp28685h6tlq0lnz3yueqxhtdhqqq7mrwr6mv9j94zdhxpxfg3cd6y4pum736hwve4wq2pmgswkj7apnxcnu8yn89ve0vrhmt6g0jsxfkcqa5uxfj",
  "warning_capacity": "No channels have sufficient incoming capacity"
} `)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	invoice, err := lightning.CreateInvoice(uint64(1), "uniq", "desc", uint32(200), nil, "", true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.Invoice{
		PaymentHash:     "0213ca245ca23deccf62a64a298a988bbe42d6fc7620471129328c2faa3ccb7a",
		ExpiresAt:       1546475890,
		Bolt11:          "lnbcrt10p1pwz6k92pp5qgfu5fzu5g77enmz5e9znz5c3wly94huwcsywyffx2xzl23uedaqdq8v3jhxccxqzxgcqp28685h6tlq0lnz3yueqxhtdhqqq7mrwr6mv9j94zdhxpxfg3cd6y4pum736hwve4wq2pmgswkj7apnxcnu8yn89ve0vrhmt6g0jsxfkcqa5uxfj",
		WarningCapacity: "No channels have sufficient incoming capacity",
	}, invoice)
}

func TestInvoiceAny(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"invoice","params":{"description":"desc","expiry":200,"exposeprivatechannels":false,"label":"label","msatoshi":"any"},"id":1}`
	resp := wrapResult(1, `{
  "payment_hash": "287fbbfc50989e4e696e4ab6b96c8d5d5e2fff2a48bf6b59a2fecc040337ea91",
  "expires_at": 1546475555,
  "bolt11": "lnbcrt1pwz646mpp59plmhlzsnz0yu6twf2mtjmydt40zlle2fzlkkkdzlmxqgqeha2gsdq8v3jhxccxqzxgcqp2vj8dqhg6yyzrvcd7kfwu4svh6k44mv5uy6wetpwfyxav504rthkxhxll2d9e4dwcm7xzpsxy9l9aulpmskepqad2x8vz82krme8zevgq3utwgq",
  "warning_capacity": "No channels have sufficient incoming capacity"
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	invoice, err := lightning.CreateInvoiceAny("label", "desc", uint32(200), nil, "", false)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.Invoice{
		PaymentHash:     "287fbbfc50989e4e696e4ab6b96c8d5d5e2fff2a48bf6b59a2fecc040337ea91",
		ExpiresAt:       1546475555,
		Bolt11:          "lnbcrt1pwz646mpp59plmhlzsnz0yu6twf2mtjmydt40zlle2fzlkkkdzlmxqgqeha2gsdq8v3jhxccxqzxgcqp2vj8dqhg6yyzrvcd7kfwu4svh6k44mv5uy6wetpwfyxav504rthkxhxll2d9e4dwcm7xzpsxy9l9aulpmskepqad2x8vz82krme8zevgq3utwgq",
		WarningCapacity: "No channels have sufficient incoming capacity",
	}, invoice)
}

func TestGetRouteSimple(t *testing.T) {
	id := "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41"
	req := `{"jsonrpc":"2.0","method":"getroute","params":{"cltv":9,"fuzzpercent":5,"id":"03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41","msatoshi":300000,"riskfactor":99},"id":1}`
	resp := wrapResult(1, `{
  "route": [
    {
      "id": "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
      "channel": "233x1x0",
      "msatoshi": 300000,
      "delay": 9
    }
  ]
}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	route, err := lightning.GetRouteSimple(id, 300000, 99)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.RouteHop{
		glightning.RouteHop{
			Id:             "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
			ShortChannelId: "233x1x0",
			MilliSatoshi:   300000,
			Delay:          9,
		},
	}, route)
}

func TestGetRoute(t *testing.T) {
	id := "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41"
	fromId := "02e9ce22855694b3dea98d78512c3e73c198c98553912cd04b53d1563b40f661da"

	req := `{"jsonrpc":"2.0","method":"getroute","params":{"cltv":32,"exclude":["1020x222x1/1"],"fromid":"02e9ce22855694b3dea98d78512c3e73c198c98553912cd04b53d1563b40f661da","fuzzpercent":10,"id":"03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41","maxhops":10,"msatoshi":300000,"riskfactor":99},"id":1}`
	resp := wrapResult(1, `{
  "route": [
    {
      "id": "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
      "channel": "233x1x0",
      "msatoshi": 300000,
      "delay": 32 
    }
  ]
}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	route, err := lightning.GetRoute(id, 300000, 99, 32, fromId, 10.0, []string{"1020x222x1/1"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.RouteHop{
		glightning.RouteHop{
			Id:             "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
			ShortChannelId: "233x1x0",
			MilliSatoshi:   300000,
			Delay:          32,
		},
	}, route)

}

func TestWithdraw(t *testing.T) {
	addr := "bcrt1qx5yjs8y4vm929ykzpmm8r7yxwakyvjwmyc5mkm"
	req := `{"jsonrpc":"2.0","method":"withdraw","params":{"destination":"bcrt1qx5yjs8y4vm929ykzpmm8r7yxwakyvjwmyc5mkm","feerate":"125perkb","satoshi":"500000"},"id":1}`
	resp := wrapResult(1, `{
  "tx": "020000000001012a62fd17c6b13b7d89df7bbceb9baa79ab937223887c9c69b05fefc9288a2d640000000000ffffffff0250c30000000000001600143509281c9566caa292c20ef671f886776c4649db9d3aff00000000001600142e7dfaf485fba60010bfb37c99fc93b8bb42ad0202483045022100b99e4231fcf98dc2f94d88094b63dc12fc0ba7c125dc78df1f7a50bfca726b8a02204639577a20f39830d63dfefcfc85f134f0d8128c55a2833775bb906957a0fa86012103d5aea229d81a06e576dfcf71db13670422b22dd1093cddc01269a5596fb0c7d100000000",
  "txid": "f80423d5daed70d31585e597d8e1c0d191a5f2d8050a11dee730f7727c5abd9c"
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	feerate := glightning.NewFeeRate(glightning.SatPerKiloByte, 125)
	result, err := lightning.Withdraw(addr, glightning.NewAmount(500000), feerate, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.WithdrawResult{
		Tx:   "020000000001012a62fd17c6b13b7d89df7bbceb9baa79ab937223887c9c69b05fefc9288a2d640000000000ffffffff0250c30000000000001600143509281c9566caa292c20ef671f886776c4649db9d3aff00000000001600142e7dfaf485fba60010bfb37c99fc93b8bb42ad0202483045022100b99e4231fcf98dc2f94d88094b63dc12fc0ba7c125dc78df1f7a50bfca726b8a02204639577a20f39830d63dfefcfc85f134f0d8128c55a2833775bb906957a0fa86012103d5aea229d81a06e576dfcf71db13670422b22dd1093cddc01269a5596fb0c7d100000000",
		TxId: "f80423d5daed70d31585e597d8e1c0d191a5f2d8050a11dee730f7727c5abd9c",
	}, result)
}

func TestWithdrawAll(t *testing.T) {
	addr := "2MzpEvkwrYfuUFiPQdWHDBSFCw8zipNkYBz"
	req := `{"jsonrpc":"2.0","method":"withdraw","params":{"destination":"2MzpEvkwrYfuUFiPQdWHDBSFCw8zipNkYBz","feerate":"125perkb","satoshi":"all"},"id":1}`
	resp := wrapResult(1, `{
  "tx": "020000000001012a62fd17c6b13b7d89df7bbceb9baa79ab937223887c9c69b05fefc9288a2d640000000000ffffffff0250c30000000000001600143509281c9566caa292c20ef671f886776c4649db9d3aff00000000001600142e7dfaf485fba60010bfb37c99fc93b8bb42ad0202483045022100b99e4231fcf98dc2f94d88094b63dc12fc0ba7c125dc78df1f7a50bfca726b8a02204639577a20f39830d63dfefcfc85f134f0d8128c55a2833775bb906957a0fa86012103d5aea229d81a06e576dfcf71db13670422b22dd1093cddc01269a5596fb0c7d100000000",
  "txid": "f80423d5daed70d31585e597d8e1c0d191a5f2d8050a11dee730f7727c5abd9c"
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	feerate := glightning.NewFeeRate(glightning.SatPerKiloByte, 125)
	result, err := lightning.Withdraw(addr, glightning.NewAllAmount(), feerate, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.WithdrawResult{
		Tx:   "020000000001012a62fd17c6b13b7d89df7bbceb9baa79ab937223887c9c69b05fefc9288a2d640000000000ffffffff0250c30000000000001600143509281c9566caa292c20ef671f886776c4649db9d3aff00000000001600142e7dfaf485fba60010bfb37c99fc93b8bb42ad0202483045022100b99e4231fcf98dc2f94d88094b63dc12fc0ba7c125dc78df1f7a50bfca726b8a02204639577a20f39830d63dfefcfc85f134f0d8128c55a2833775bb906957a0fa86012103d5aea229d81a06e576dfcf71db13670422b22dd1093cddc01269a5596fb0c7d100000000",
		TxId: "f80423d5daed70d31585e597d8e1c0d191a5f2d8050a11dee730f7727c5abd9c",
	}, result)
}

func TestTxPrepare(t *testing.T) {
	destination := "bcrt1qeyyk6sl5pr49ycpqyckvmttus5ttj25pd0zpvg"
	amount := glightning.NewAmount(100000)
	feerate := glightning.NewFeeRate(glightning.SatPerKiloSipa, 243)
	minConf := uint16(1)
	req := `{"jsonrpc":"2.0","method":"txprepare","params":{"destination":"bcrt1qeyyk6sl5pr49ycpqyckvmttus5ttj25pd0zpvg","feerate":"243perkw","minconf":1,"satoshi":"100000"},"id":1}`
	resp := wrapResult(1, `{
   "unsigned_tx" : "0200000001060528291e1039a5a2e071ab88ffca8cb9655481f62108dff2e87a1aa139b6450000000000ffffffff02a086010000000000160014c9096d43f408ea526020262ccdad7c8516b92a81d86a042a01000000160014e1cfb78798b16dd8f0b05b540f853d07ac5c555200000000",
   "txid" : "cec03e956f3761624f176d62428d9e2cd51cb923258e00e17a34fc49b0da6dde"
}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	result, err := lightning.PrepareTx(destination, amount, feerate, &minConf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, &glightning.TxResult{
		Tx:   "0200000001060528291e1039a5a2e071ab88ffca8cb9655481f62108dff2e87a1aa139b6450000000000ffffffff02a086010000000000160014c9096d43f408ea526020262ccdad7c8516b92a81d86a042a01000000160014e1cfb78798b16dd8f0b05b540f853d07ac5c555200000000",
		TxId: "cec03e956f3761624f176d62428d9e2cd51cb923258e00e17a34fc49b0da6dde",
	}, result)
}

func TestTxSend(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"txsend","params":{"txid":"c139ff2ce1c1e1056429c1527262d56da2be096559f554e061da18ee72d5c5ed"},"id":1}`
	resp := wrapResult(1, `{
   "unsigned_tx" : "0200000001f56ad611189c96c9ae9499d61872e590a3ba4d55760f7663b0642d81c2b1880d0000000000ffffffff02a086010000000000160014c9096d43f408ea526020262ccdad7c8516b92a81d86a042a010000001600146ea01d6c5aaa643076902d1c8b026e9eb47b32c000000000",
   "txid" : "c139ff2ce1c1e1056429c1527262d56da2be096559f554e061da18ee72d5c5ed"
}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	result, err := lightning.SendTx("c139ff2ce1c1e1056429c1527262d56da2be096559f554e061da18ee72d5c5ed")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, &glightning.TxResult{
		Tx:   "0200000001f56ad611189c96c9ae9499d61872e590a3ba4d55760f7663b0642d81c2b1880d0000000000ffffffff02a086010000000000160014c9096d43f408ea526020262ccdad7c8516b92a81d86a042a010000001600146ea01d6c5aaa643076902d1c8b026e9eb47b32c000000000",
		TxId: "c139ff2ce1c1e1056429c1527262d56da2be096559f554e061da18ee72d5c5ed",
	}, result)
}

func TestTxDiscard(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"txdiscard","params":{"txid":"c139ff2ce1c1e1056429c1527262d56da2be096559f554e061da18ee72d5c5ed"},"id":1}`
	resp := wrapResult(1, `{
   "unsigned_tx" : "0200000001f56ad611189c96c9ae9499d61872e590a3ba4d55760f7663b0642d81c2b1880d0000000000ffffffff02a086010000000000160014c9096d43f408ea526020262ccdad7c8516b92a81d86a042a010000001600146ea01d6c5aaa643076902d1c8b026e9eb47b32c000000000",
   "txid" : "c139ff2ce1c1e1056429c1527262d56da2be096559f554e061da18ee72d5c5ed"
}`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	result, err := lightning.DiscardTx("c139ff2ce1c1e1056429c1527262d56da2be096559f554e061da18ee72d5c5ed")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, &glightning.TxResult{
		Tx:   "0200000001f56ad611189c96c9ae9499d61872e590a3ba4d55760f7663b0642d81c2b1880d0000000000ffffffff02a086010000000000160014c9096d43f408ea526020262ccdad7c8516b92a81d86a042a010000001600146ea01d6c5aaa643076902d1c8b026e9eb47b32c000000000",
		TxId: "c139ff2ce1c1e1056429c1527262d56da2be096559f554e061da18ee72d5c5ed",
	}, result)
}

func TestClose(t *testing.T) {
	id := "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41"
	req := `{"jsonrpc":"2.0","method":"close","params":{"id":"03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41"},"id":1}`
	resp := wrapResult(1, `{
  "tx": "02000000015c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c0000000000ffffffff0195feff000000000016001449a59c8b2c806e554858127df08ed4aadf361b4600000000",
  "txid": "642d8a28c9ef5fb0699c7c88237293ab79aa9bebbc7bdf897d3bb1c617fd622a",
  "type": "mutual"
}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	result, err := lightning.CloseNormal(id)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.CloseResult{
		Tx:   "02000000015c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c0000000000ffffffff0195feff000000000016001449a59c8b2c806e554858127df08ed4aadf361b4600000000",
		TxId: "642d8a28c9ef5fb0699c7c88237293ab79aa9bebbc7bdf897d3bb1c617fd622a",
		Type: "mutual",
	}, result)
}

func TestListFunds(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"listfunds","params":{},"id":1}`
	resp := wrapResult(1, `{
  "outputs": [
    {
      "txid": "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c",
      "output": 1,
      "value": 983222176,
      "address": "bcrt1qm9f2tleu0r9zcj8a3c454crfnzra69nwvp5frw",
      "status": "confirmed"
    }
  ],
  "channels": [
    {
      "peer_id": "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
      "short_channel_id": "103x1x0",
      "channel_sat": 16777215,
      "channel_total_sat": 16777215,
      "funding_txid": "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c"
    }
  ]
} `)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	funds, err := lightning.ListFunds()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.FundsResult{
		Outputs: []*glightning.FundOutput{
			&glightning.FundOutput{
				TxId:    "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c",
				Output:  1,
				Value:   uint64(983222176),
				Address: "bcrt1qm9f2tleu0r9zcj8a3c454crfnzra69nwvp5frw",
				Status:  "confirmed",
			},
		},
		Channels: []*glightning.FundingChannel{
			&glightning.FundingChannel{
				Id:                  "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41",
				ShortChannelId:      "103x1x0",
				ChannelSatoshi:      16777215,
				ChannelTotalSatoshi: 16777215,
				FundingTxId:         "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c",
			},
		},
	}, funds)
}

func TestDisconnect(t *testing.T) {
	id := "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41"

	req := fmt.Sprintf(`{"jsonrpc":"2.0","method":"disconnect","params":{"force":false,"id":"%s"},"id":%d}`, id, 1)
	resp := wrapResult(1, `{}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	err := lightning.Disconnect(id, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFundChannel(t *testing.T) {
	id := "03fb0b8a395a60084946eaf98cfb5a81ea010e0307eaf368ba21e7d6bcf0e4dc41"
	amount := 90000

	req := fmt.Sprintf(`{"jsonrpc":"2.0","method":"fundchannel","params":{"announce":false,"feerate":"500perkw","id":"%s","satoshi":"%d"},"id":%d}`, id, amount, 1)
	resp := wrapResult(1, `{
  "tx": "0200000000010153bcd4cfabb72750bb8d16fc711c91b30215957549a0a93370f50475fa9457570100000000ffffffff02ffffff0000000000220020b2c1a13de4a5926ed48601626e281a171d0cdb548fddc4e7cc8cdc9d982a2368a0c79a3a00000000160014d952a5ff3c78ca2c48fd8e2b4ae0699887dd166e0247304402206a781a53902e6526686b9ecc79f7287d372d11614dc094789f05f843458e703e022041cc1f5f7e2526d415b44563c80b80d9ce808310eaf8a73fbead45a0238b01e0012102cf978ae73c98d6e8e73b384b217c13180fd75cd867f5d9daf19624ecebf5fc0a00000000",
  "txid": "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c",
  "channel_id": "5c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c"
}
`)
	sats := glightning.NewAmount(amount)
	feeRate := glightning.NewFeeRate(glightning.SatPerKiloSipa, 500)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	result, err := lightning.FundChannelExt(id, sats, feeRate, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.FundChannelResult{
		FundingTx:   "0200000000010153bcd4cfabb72750bb8d16fc711c91b30215957549a0a93370f50475fa9457570100000000ffffffff02ffffff0000000000220020b2c1a13de4a5926ed48601626e281a171d0cdb548fddc4e7cc8cdc9d982a2368a0c79a3a00000000160014d952a5ff3c78ca2c48fd8e2b4ae0699887dd166e0247304402206a781a53902e6526686b9ecc79f7287d372d11614dc094789f05f843458e703e022041cc1f5f7e2526d415b44563c80b80d9ce808310eaf8a73fbead45a0238b01e0012102cf978ae73c98d6e8e73b384b217c13180fd75cd867f5d9daf19624ecebf5fc0a00000000",
		FundingTxId: "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c",
		ChannelId:   "5c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c",
	}, result)

	// Run again, but with different fee rate
	resp = wrapResult(2, `{
  "tx": "0200000000010153bcd4cfabb72750bb8d16fc711c91b30215957549a0a93370f50475fa9457570100000000ffffffff02ffffff0000000000220020b2c1a13de4a5926ed48601626e281a171d0cdb548fddc4e7cc8cdc9d982a2368a0c79a3a00000000160014d952a5ff3c78ca2c48fd8e2b4ae0699887dd166e0247304402206a781a53902e6526686b9ecc79f7287d372d11614dc094789f05f843458e703e022041cc1f5f7e2526d415b44563c80b80d9ce808310eaf8a73fbead45a0238b01e0012102cf978ae73c98d6e8e73b384b217c13180fd75cd867f5d9daf19624ecebf5fc0a00000000",
  "txid": "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c",
  "channel_id": "5c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c"
}
`)
	sats = &glightning.SatoshiAmount{Amount: uint64(amount)}
	feeRate = glightning.NewFeeRateByDirective(glightning.SatPerKiloByte, glightning.Urgent)
	req = fmt.Sprintf(`{"jsonrpc":"2.0","method":"fundchannel","params":{"announce":true,"feerate":"urgent","id":"%s","satoshi":"%d"},"id":%d}`, id, amount, 2)
	go runServerSide(t, req, resp, replyQ, requestQ)
	_, err = lightning.FundChannelExt(id, sats, feeRate, true, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Run again, using 'all' satoshis
	resp = wrapResult(3, `{
  "tx": "0200000000010153bcd4cfabb72750bb8d16fc711c91b30215957549a0a93370f50475fa9457570100000000ffffffff02ffffff0000000000220020b2c1a13de4a5926ed48601626e281a171d0cdb548fddc4e7cc8cdc9d982a2368a0c79a3a00000000160014d952a5ff3c78ca2c48fd8e2b4ae0699887dd166e0247304402206a781a53902e6526686b9ecc79f7287d372d11614dc094789f05f843458e703e022041cc1f5f7e2526d415b44563c80b80d9ce808310eaf8a73fbead45a0238b01e0012102cf978ae73c98d6e8e73b384b217c13180fd75cd867f5d9daf19624ecebf5fc0a00000000",
  "txid": "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c",
  "channel_id": "5c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c"
}
`)
	req = fmt.Sprintf(`{"jsonrpc":"2.0","method":"fundchannel","params":{"announce":true,"feerate":"300perkb","id":"%s","satoshi":"%s"},"id":%d}`, id, "all", 3)
	go runServerSide(t, req, resp, replyQ, requestQ)
	sats = &glightning.SatoshiAmount{SendAll: true}
	feeRate = glightning.NewFeeRate(glightning.SatPerKiloByte, uint(300))
	_, err = lightning.FundChannelExt(id, sats, feeRate, true, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Run again, using private channel
	resp = wrapResult(4, `{
  "tx": "0200000000010153bcd4cfabb72750bb8d16fc711c91b30215957549a0a93370f50475fa9457570100000000ffffffff02ffffff0000000000220020b2c1a13de4a5926ed48601626e281a171d0cdb548fddc4e7cc8cdc9d982a2368a0c79a3a00000000160014d952a5ff3c78ca2c48fd8e2b4ae0699887dd166e0247304402206a781a53902e6526686b9ecc79f7287d372d11614dc094789f05f843458e703e022041cc1f5f7e2526d415b44563c80b80d9ce808310eaf8a73fbead45a0238b01e0012102cf978ae73c98d6e8e73b384b217c13180fd75cd867f5d9daf19624ecebf5fc0a00000000",
  "txid": "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c",
  "channel_id": "5c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c"
}
`)
	sats = &glightning.SatoshiAmount{SendAll: true}
	feeRate = glightning.NewFeeRateByDirective(glightning.SatPerKiloByte, glightning.Urgent)
	req = fmt.Sprintf(`{"jsonrpc":"2.0","method":"fundchannel","params":{"announce":false,"feerate":"urgent","id":"%s","satoshi":"all"},"id":%d}`, id, 4)
	go runServerSide(t, req, resp, replyQ, requestQ)
	_, err = lightning.FundChannelExt(id, sats, feeRate, false, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartFundChannel(t *testing.T) {
	id := "0334b7c8e723c00aedb6aaab0988619a6929f0039275ac195185efbadad1a343f9"
	sats := uint64(100000)
	feeRate := glightning.NewFeeRateByDirective(glightning.SatPerKiloByte, glightning.Urgent)
	req := fmt.Sprintf(`{"jsonrpc":"2.0","method":"fundchannel_start","params":{"announce":true,"feerate":"urgent","id":"%s","satoshi":%d},"id":%d}`, id, sats, 1)
	resp := wrapResult(1, `{"funding_address" : "bcrt1qc4p5fwkgznrrlml5z4hy0xwauys8nlsxsca2zn2ew2wez27hlyequp6sff"}
`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	result, err := lightning.StartFundChannel(id, sats, true, feeRate)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "bcrt1qc4p5fwkgznrrlml5z4hy0xwauys8nlsxsca2zn2ew2wez27hlyequp6sff", result)
}

func TestCompleteFundChannel(t *testing.T) {

	id := "0334b7c8e723c00aedb6aaab0988619a6929f0039275ac195185efbadad1a343f9"
	txid := "7c158044dd655057ea344924e135f8c5e5cffa8f583ccd81650f2b82057f0b5c"
	req := fmt.Sprintf(`{"jsonrpc":"2.0","method":"fundchannel_complete","params":{"id":"%s","txid":"%s","txout":0},"id":%d}`, id, txid, 1)
	resp := wrapResult(1, `{
   "channel_id" : "5c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c",
   "commitments_secured" : true
}
`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	result, err := lightning.CompleteFundChannel(id, txid, uint16(0))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "5c0b7f05822b0f6581cd3c588ffacfe5c5f835e1244934ea575065dd4480157c", result)
}

func TestCancelFundChannel(t *testing.T) {
	id := "0334b7c8e723c00aedb6aaab0988619a6929f0039275ac195185efbadad1a343f9"
	req := fmt.Sprintf(`{"jsonrpc":"2.0","method":"fundchannel_cancel","params":{"id":"%s"},"id":%d}`, id, 1)
	resp := wrapResult(1, `{
   "cancelled" : "Channel open canceled by RPC"
}`)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	result, err := lightning.CancelFundChannel(id)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, result)
}

func TestStop(t *testing.T) {
	req := `{"jsonrpc":"2.0","method":"stop","params":{},"id":1}`
	resp := wrapResult(1, `"Shutting down"`)
	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, req, resp, replyQ, requestQ)
	stopmsg, err := lightning.Stop()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Shutting down", stopmsg)

}

func TestListChannelsBySource(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)
	source := "0399a287c8bcc11e8547f2d9cbcceccab0b74c1a07803b482d1d450233ddd447a6"
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"listchannels\",\"params\":{\"source\":\"0399a287c8bcc11e8547f2d9cbcceccab0b74c1a07803b482d1d450233ddd447a6\"},\"id\":1}"
	resp := wrapResult(1, `{
  "channels": [
    {
      "source": "02308c54b63e2c1375a52ce6ca27b171188f99e7c274eaf14be396289d93fb6003",
      "destination": "034143d1a45cb9bcb912eab97facf4a971098385c4701753d6bc40e52192d0c04f",
      "short_channel_id": "556297x2967x0",
      "public": true,
      "satoshis": 500000,
      "message_flags": 0,
      "channel_flags": 0,
      "flags": 0,
      "active": true,
      "last_update": 1546213724,
      "base_fee_millisatoshi": 1000,
      "fee_per_millionth": 1,
      "delay": 144
    },
    {
      "source": "034143d1a45cb9bcb912eab97facf4a971098385c4701753d6bc40e52192d0c04f",
      "destination": "02308c54b63e2c1375a52ce6ca27b171188f99e7c274eaf14be396289d93fb6003",
      "short_channel_id": "556297x2967x0",
      "public": true,
      "satoshis": 500000,
      "message_flags": 0,
      "channel_flags": 1,
      "flags": 1,
      "active": true,
      "last_update": 1546213449,
      "base_fee_millisatoshi": 1000,
      "fee_per_millionth": 1,
      "delay": 144
    }
  ]
}`)
	go runServerSide(t, req, resp, replyQ, requestQ)
	_, err := lightning.ListChannelsBySource(source)
	if err != nil {
		t.Fatal(err)
	}
}

func TestListChannels(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)
	scid := "556297x2967x0"
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"listchannels\",\"params\":{\"short_channel_id\":\"556297x2967x0\"},\"id\":1}"
	resp := wrapResult(1, `{
  "channels": [
    {
      "source": "02308c54b63e2c1375a52ce6ca27b171188f99e7c274eaf14be396289d93fb6003",
      "destination": "034143d1a45cb9bcb912eab97facf4a971098385c4701753d6bc40e52192d0c04f",
      "short_channel_id": "556297x2967x0",
      "public": true,
      "satoshis": 500000,
      "message_flags": 0,
      "channel_flags": 0,
      "flags": 0,
      "active": true,
      "last_update": 1546213724,
      "base_fee_millisatoshi": 1000,
      "fee_per_millionth": 1,
      "delay": 144
    },
    {
      "source": "034143d1a45cb9bcb912eab97facf4a971098385c4701753d6bc40e52192d0c04f",
      "destination": "02308c54b63e2c1375a52ce6ca27b171188f99e7c274eaf14be396289d93fb6003",
      "short_channel_id": "556297x2967x0",
      "public": true,
      "satoshis": 500000,
      "message_flags": 0,
      "channel_flags": 1,
      "flags": 1,
      "active": true,
      "last_update": 1546213449,
      "base_fee_millisatoshi": 1000,
      "fee_per_millionth": 1,
      "delay": 144
    }
  ]
}`)
	go runServerSide(t, req, resp, replyQ, requestQ)
	channels, err := lightning.GetChannel(scid)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.Channel{
		glightning.Channel{
			Source:              "02308c54b63e2c1375a52ce6ca27b171188f99e7c274eaf14be396289d93fb6003",
			Destination:         "034143d1a45cb9bcb912eab97facf4a971098385c4701753d6bc40e52192d0c04f",
			ShortChannelId:      "556297x2967x0",
			IsPublic:            true,
			Satoshis:            500000,
			MessageFlags:        uint(0),
			ChannelFlags:        uint(0),
			IsActive:            true,
			LastUpdate:          uint(1546213724),
			BaseFeeMillisatoshi: 1000,
			FeePerMillionth:     uint64(1),
			Delay:               uint(144),
		},
		glightning.Channel{
			Source:              "034143d1a45cb9bcb912eab97facf4a971098385c4701753d6bc40e52192d0c04f",
			Destination:         "02308c54b63e2c1375a52ce6ca27b171188f99e7c274eaf14be396289d93fb6003",
			ShortChannelId:      "556297x2967x0",
			IsPublic:            true,
			Satoshis:            500000,
			MessageFlags:        uint(0),
			ChannelFlags:        uint(1),
			IsActive:            true,
			LastUpdate:          uint(1546213449),
			BaseFeeMillisatoshi: 1000,
			FeePerMillionth:     uint64(1),
			Delay:               uint(144),
		},
	}, channels)
}

func TestListNodes(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)
	nodeId := "02befaace6e8970aaca34eafe85f30f988e374628ec279d94e7eca8b574b738eb4"
	req := `{"jsonrpc":"2.0","method":"listnodes","params":{"id":"02befaace6e8970aaca34eafe85f30f988e374628ec279d94e7eca8b574b738eb4"},"id":1}`
	resp := wrapResult(1, ` {"nodes": [
    {    
      "nodeid": "02befaace6e8970aaca34eafe85f30f988e374628ec279d94e7eca8b574b738eb4",
      "alias": "LightningBerry [LND]",        
      "color": "68f442",
      "last_timestamp": 1542574678,
      "globalfeatures": "",
      "global_features": "",
      "addresses": [
        {
          "type": "ipv4",                    
          "address": "84.219.199.67",                                                                                             
          "port": 9735                  
        }
      ]     
    }
  ]                                                                                  
} `)
	go runServerSide(t, req, resp, replyQ, requestQ)
	nodes, err := lightning.GetNode(nodeId)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.Node{
		glightning.Node{
			Id:             "02befaace6e8970aaca34eafe85f30f988e374628ec279d94e7eca8b574b738eb4",
			Alias:          "LightningBerry [LND]",
			Color:          "68f442",
			LastTimestamp:  uint(1542574678),
			GlobalFeatures: "",
			Addresses: []glightning.Address{
				glightning.Address{
					Type: "ipv4",
					Addr: "84.219.199.67",
					Port: 9735,
				},
			},
		},
	}, nodes)
}

func TestGetInfo(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"getinfo\",\"params\":{},\"id\":1}"
	resp := wrapResult(1, `{ "id": "020631b6c35d614ebdf8856bfd2ccb5099337588b1b56453d5d7567654d6710b92", "alias": "LATENTNET-v0.6.2-291-g91c9ce7", "color": "020631", "num_peers": 2, "num_pending_channels": 3, "num_active_channels": 1, "num_inactive_channels": 8, "address": [ ], "binding": [ { "type": "ipv6", "address": "::", "port": 9735 }, { "type": "ipv4", "address": "0.0.0.0", "port": 9735 } ], "version": "v0.6.2-291-g91c9ce7", "blockheight": 556302, "network": "bitcoin", "msatoshi_fees_collected": 300 }`)
	go runServerSide(t, req, resp, replyQ, requestQ)
	info, err := lightning.GetInfo()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.NodeInfo{
		Id:                   "020631b6c35d614ebdf8856bfd2ccb5099337588b1b56453d5d7567654d6710b92",
		Alias:                "LATENTNET-v0.6.2-291-g91c9ce7",
		Color:                "020631",
		PeerCount:            2,
		PendingChannelCount:  3,
		ActiveChannelCount:   1,
		InactiveChannelCount: 8,
		Addresses:            []glightning.Address{},
		Binding: []glightning.AddressInternal{
			glightning.AddressInternal{
				Type: "ipv6",
				Addr: "::",
				Port: 9735,
			},
			glightning.AddressInternal{
				Type: "ipv4",
				Addr: "0.0.0.0",
				Port: 9735,
			},
		},
		Version:                    "v0.6.2-291-g91c9ce7",
		Blockheight:                556302,
		Network:                    "bitcoin",
		FeesCollectedMilliSatoshis: 300,
	}, info)
}

func TestGetLog(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"getlog\",\"params\":{\"level\":\"info\"},\"id\":1}"
	resp := wrapResult(1, `{"created_at":"1546200491.277516996", "bytes_used":6445039,"bytes_max":20971520,"log":[{"type": "UNUSUAL","time": "4709.811937439","source": "lightningd(9383):", "log": "bitcoin-cli: finished bitcoin-cli getblockhash 556283 (12250 ms)"},{"type": "SKIPPED","num_skipped": 89},{"type": "INFO","time": "5688.218267611","source": "lightningd(9383):","log": "lightning_openingd-02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636 chan #1: Peer connection lost"}]}`)
	go runServerSide(t, req, resp, replyQ, requestQ)
	logresp, err := lightning.GetLog(glightning.Info)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.LogResponse{
		CreatedAt: "1546200491.277516996",
		BytesUsed: uint64(6445039),
		BytesMax:  uint64(20971520),
		Logs: []glightning.Log{
			glightning.Log{
				Type:    "UNUSUAL",
				Time:    "4709.811937439",
				Source:  "lightningd(9383):",
				Message: "bitcoin-cli: finished bitcoin-cli getblockhash 556283 (12250 ms)",
			},
			glightning.Log{
				Type:       "SKIPPED",
				NumSkipped: uint(89),
			},
			glightning.Log{
				Type:    "INFO",
				Time:    "5688.218267611",
				Source:  "lightningd(9383):",
				Message: "lightning_openingd-02cca6c5c966fcf61d121e3a70e03a1cd9eeeea024b26ea666ce974d43b242e636 chan #1: Peer connection lost",
			},
		},
	}, logresp)
}

func TestHelp(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)
	resp := wrapResult(1, `{"help": [{"command": "feerates style","description": "Return feerate estimates, either satoshi-per-kw ({style} perkw) or satoshi-per-kb ({style} perkb).","verbose": "HELP! Please contribute a description for this json_command!"},{"command": "connect id [host] [port]","description": "Connect to {id} at {host} (which can end in ':port' if not default). {id} can also be of the form id@host","verbose": "HELP! Please contribute a description for this json_command!"}]}`)
	req := "{\"jsonrpc\":\"2.0\",\"method\":\"help\",\"params\":{},\"id\":1}"
	go runServerSide(t, req, resp, replyQ, requestQ)
	help, err := lightning.Help()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []glightning.Command{
		glightning.Command{
			NameAndUsage: "feerates style",
			Description:  "Return feerate estimates, either satoshi-per-kw ({style} perkw) or satoshi-per-kb ({style} perkb).",
			Verbose:      "HELP! Please contribute a description for this json_command!",
		},
		glightning.Command{
			NameAndUsage: "connect id [host] [port]",
			Description:  "Connect to {id} at {host} (which can end in ':port' if not default). {id} can also be of the form id@host",
			Verbose:      "HELP! Please contribute a description for this json_command!",
		},
	}, help)
}

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
	assert.Equal(t, &glightning.DecodedBolt11{
		Currency:           "bc",
		CreatedAt:          uint64(1496314658),
		Expiry:             uint64(60),
		Payee:              "03e7156ae33b0a208d0744199163177e909e80176e55d97a2f221ede0f934dd9ad",
		MilliSatoshis:      250000000,
		Description:        "1 cup of coffee",
		MinFinalCltvExpiry: 9,
		PaymentHash:        "0001020304050607080900010203040506070809000102030405060708090102",
		Signature:          "3045022100e89639ba6814e36689d4b91bf125f10351b55da057b00647a8dabaeb8a90c95f0220160f9d5a6e0f79d1fc2b964238b944e2fa4aa677c6f020d466472ab842bd750e",
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
	assert.Equal(t, &glightning.DecodedBolt11{
		Currency:           "bc",
		CreatedAt:          uint64(1496314658),
		Expiry:             uint64(3600),
		Payee:              "03e7156ae33b0a208d0744199163177e909e80176e55d97a2f221ede0f934dd9ad",
		MilliSatoshis:      2000000000,
		DescriptionHash:    "3925b6f67e2c340036ed12093dd44e0368df1b6ea26c53dbe4811f58fd5db8c1",
		MinFinalCltvExpiry: 9,
		PaymentHash:        "0001020304050607080900010203040506070809000102030405060708090102",
		Signature:          "304502210091675cb3fad8e9d915343883a49242e074474e26d42c7ed914655689a80745530220733e8e4ea5ce9b85f69e40d755a55014536b12323f8b220600c94ef2b9c51428",
		Fallbacks: []glightning.Fallback{
			glightning.Fallback{
				Type:    "P2PKH",
				Address: "1RustyRX2oai4EYYDpQGWvEL62BBGqN9T",
				Hex:     "76a91404b61f7dc1ea0dc99424464cc4064dc564d91e8988ac",
			},
		},
		Routes: [][]glightning.BoltRoute{
			[]glightning.BoltRoute{
				glightning.BoltRoute{
					Pubkey:                    "029e03a901b85534ff1e92c43c74431f7ce72046060fcf7a95c37e148f78c77255",
					ShortChannelId:            "66051:263430:1800",
					FeeBaseMilliSatoshis:      uint64(1),
					FeeProportionalMillionths: uint64(20),
					CltvExpiryDelta:           uint(3),
				},
				glightning.BoltRoute{
					Pubkey:                    "039e03a901b85534ff1e92c43c74431f7ce72046060fcf7a95c37e148f78c77255",
					ShortChannelId:            "197637:395016:2314",
					FeeBaseMilliSatoshis:      uint64(2),
					FeeProportionalMillionths: uint64(30),
					CltvExpiryDelta:           uint(4),
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
	id, err := lightning.Connect(peerId, host, port)
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
	assert.Equal(t, &glightning.Pong{
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
	assert.Equal(t, &glightning.Pong{
		TotalLen: 234,
	}, pong)
}

func TestNewAddr(t *testing.T) {
	lightning, requestQ, replyQ := startupServer(t)

	req := "{\"jsonrpc\":\"2.0\",\"method\":\"newaddr\",\"params\":{\"addresstype\":\"p2sh-segwit\"},\"id\":1}"
	resp := wrapResult(1, `{ "address": "3LfQdff5doR791QNzjn5KdPkFfFn3dmYpc" } `)
	go runServerSide(t, req, resp, replyQ, requestQ)
	addr, err := lightning.NewAddressOfType(glightning.P2SHSegwit)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "3LfQdff5doR791QNzjn5KdPkFfFn3dmYpc", addr)

	req = "{\"jsonrpc\":\"2.0\",\"method\":\"newaddr\",\"params\":{\"addresstype\":\"bech32\"},\"id\":2}"
	resp = wrapResult(2, `{ "address": "bc1q4va8cea0ye7hr8f6rwmug7r2rlkvc7lz93zqmh" } `)
	go runServerSide(t, req, resp, replyQ, requestQ)
	addr, err = lightning.NewAddressOfType(glightning.Bech32)
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
	rates, err := lightning.FeeRates(glightning.SatPerKiloByte)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.FeeRateEstimate{
		Style: glightning.SatPerKiloByte,
		Details: &glightning.FeeRateDetails{
			Urgent:        3328,
			Normal:        1012,
			Slow:          1012,
			MinAcceptable: 1012,
			MaxAcceptable: 33280,
		},
		OnchainEstimate: &glightning.OnchainEstimate{
			OpeningChannelSatoshis:  177,
			MutualCloseSatoshis:     170,
			UnilateralCloseSatoshis: 497,
		},
		Warning: "",
	}, rates)

	expectedRequest = "{\"jsonrpc\":\"2.0\",\"method\":\"feerates\",\"params\":{\"style\":\"perkw\"},\"id\":2}"

	reply = "{ \"jsonrpc\":\"2.0\", \"id\":2,\"result\":{\"perkw\": { \"urgent\": 832, \"normal\": 253, \"slow\": 253, \"min_acceptable\": 253, \"max_acceptable\": 8320 }, \"onchain_fee_estimates\": { \"opening_channel_satoshis\": 177, \"mutual_close_satoshis\": 170, \"unilateral_close_satoshis\": 497 }}}"

	// queue request & response
	go runServerSide(t, expectedRequest, reply, replyQ, requestQ)
	rates, err = lightning.FeeRates(glightning.SatPerKiloSipa)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.FeeRateEstimate{
		Style: glightning.SatPerKiloSipa,
		Details: &glightning.FeeRateDetails{
			Urgent:        832,
			Normal:        253,
			Slow:          253,
			MinAcceptable: 253,
			MaxAcceptable: 8320,
		},
		OnchainEstimate: &glightning.OnchainEstimate{
			OpeningChannelSatoshis:  177,
			MutualCloseSatoshis:     170,
			UnilateralCloseSatoshis: 497,
		},
		Warning: "",
	}, rates)
}

func TestPlugins(t *testing.T) {

	lightning, requestQ, replyQ := startupServer(t)
	pluginList := `{"plugins":[{"name":"autoclean","active":true},{"name":"pay","active":true},{"name":"plugin_example","active":true}]}`
	reqTemplate := "{\"jsonrpc\":\"2.0\",\"method\":\"plugin\",\"params\":{%s\"subcommand\":\"%s\"},\"id\":%d}"
	expected := []glightning.PluginInfo{
		glightning.PluginInfo{"autoclean", true},
		glightning.PluginInfo{"pay", true},
		glightning.PluginInfo{"plugin_example", true},
	}

	// test "list"
	go runServerSide(t,
		fmt.Sprintf(reqTemplate, "", "list", 1),
		wrapResult(1, pluginList),
		replyQ, requestQ)
	plugins, err := lightning.ListPlugins()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, plugins)

	// test "rescan"
	go runServerSide(t,
		fmt.Sprintf(reqTemplate, "", "rescan", 2),
		wrapResult(2, pluginList),
		replyQ, requestQ)
	plugins, err = lightning.RescanPlugins()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, plugins)

	// test "start"
	go runServerSide(t,
		fmt.Sprintf(reqTemplate, "\"plugin\":\"name\",", "start", 3),
		wrapResult(3, pluginList),
		replyQ, requestQ)
	plugins, err = lightning.StartPlugin("name")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, plugins)

	// test "stop"
	go runServerSide(t,
		fmt.Sprintf(reqTemplate, "\"plugin\":\"name\",", "stop", 4),
		wrapResult(4, pluginList),
		replyQ, requestQ)
	plugins, err = lightning.StopPlugin("name")
	if err != nil {
		t.Fatal(err)
	}

	// test plugin start-dir
	assert.Equal(t, expected, plugins)
	go runServerSide(t,
		fmt.Sprintf(reqTemplate, "\"directory\":\"dir\",", "start-dir", 5),
		wrapResult(5, pluginList),
		replyQ, requestQ)
	plugins, err = lightning.SetPluginStartDir("dir")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, plugins)
}

func TestLimitedFeeRates(t *testing.T) {
	request := "{\"jsonrpc\":\"2.0\",\"method\":\"feerates\",\"params\":{\"style\":\"perkw\"},\"id\":1}"
	reply := wrapResult(1, `{ "perkw": { "min_acceptable": 253, "max_acceptable": 4294967295 }, "warning": "Some fee estimates unavailable: bitcoind startup?" } `)

	lightning, requestQ, replyQ := startupServer(t)
	go runServerSide(t, request, reply, replyQ, requestQ)
	rates, err := lightning.FeeRates(glightning.SatPerKiloSipa)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &glightning.FeeRateEstimate{
		Style: glightning.SatPerKiloSipa,
		Details: &glightning.FeeRateDetails{
			MinAcceptable: 253,
			MaxAcceptable: 4294967295,
		},
		Warning: "Some fee estimates unavailable: bitcoind startup?",
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
func startupServer(t *testing.T) (lightning *glightning.Lightning, requestQ, replyQ chan []byte) {
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
			ok <- true
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

	lightning = glightning.NewLightning()
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
		reply = append(reply, twoNewlines...)
		out.Write(reply)
		out.Flush()
	}
}

func wrapError(id, code int, message, data string) string {
	return fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"error\":{\"code\":%d,\"message\":\"%s\",\"data\":%s}}", id, code, message, data)
}

func wrapResult(id int, result string) string {
	return fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"result\":%s}", id, result)
}

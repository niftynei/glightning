package main

import (
	"fmt"
	"github.com/niftynei/glightning/glightning"
	"sync"
)

func main() {
	lone := glightning.NewLightning()
	lone.StartUp("lightning-rpc", "/tmp/clight-1")
	id := "03a13a469bae4785e27fae24e7664e648cfdb976b97f95c694dea5e55e7d302846"
	sats := glightning.NewAmount(600000)
	feerate := glightning.NewFeeRateByDirective(glightning.SatPerKiloSipa, glightning.Urgent)
	result, err := lone.FundChannelExt(id, sats, feerate, true)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", result)
}

func paybolt() {
	bolt11 := "lnbcrt3u1pwz6lkfpp52tu7g3q4eht0mzjqsw2s8lstwq0vrhzl6xjvx73uxlsf3z93avzqdqdv35hxctnw3jhycqp2rzjq0ashz3etfsqsj2xatuce766s84qzrsrql40x696y8nad08sunwyzqqpquqqqqgqqqqqqqqpqqqqqzsqqcv7w6lzehxng32p8dy4qa4a285gaa6jda6ffzzp0zwg2dvdq2sr7naz2yz7nvz6jshecakws67fscxn3rrfva0t6q998jwy4awejf2msqzrp3u4"

	lone := glightning.NewLightning()
	lone.StartUp("lightning-rpc", "/tmp/clight-1")

	success, err := lone.PayBolt(bolt11)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", success)
}

func payInvoice() {
	lone := glightning.NewLightning()
	ltwo := glightning.NewLightning()

	lone.StartUp("lightning-rpc", "/tmp/clight-1")
	ltwo.StartUp("lightning-rpc", "/tmp/clight-3")

	satoshi := uint64(10000)

	invoiceLabel := "ayc"
	invoice, err := ltwo.CreateInvoice(satoshi, invoiceLabel, "desc", uint32(5), nil, "")
	fmt.Printf("Invoice one is %s\n", invoice.PaymentHash)
	invoiceTwo, err := ltwo.CreateInvoice(satoshi, invoiceLabel+"ab", "desc", uint32(5), nil, "")
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func(lone *glightning.Lightning, paymentHash string) {
		defer wg.Done()
		info, err := ltwo.GetInfo()
		if err != nil {
			panic(err)
		}
		route, err := lone.GetRouteSimple(info.Id, satoshi, 5)
		if err != nil {
			panic(err)
		}
		result, err := lone.SendPay(route, paymentHash, "", 0)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", result)
		sent, err := lone.WaitSendPay(paymentHash, 0)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", sent)
	}(lone, invoiceTwo.PaymentHash)
	paidInvoice, err := ltwo.WaitAnyInvoice(3)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", paidInvoice.Label)
	wg.Wait()
}

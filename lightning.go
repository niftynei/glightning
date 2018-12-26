package golight

import (
	"fmt"
	"reflect"
	"github.com/niftynei/golight/jrpc2"
)

// This file's the one that holds all the objects for the 
// c-lightning RPC commands 

type Lightning struct {
	client jrpc2.Client
}

func RegisterLightningCmds(p *Plugin) {
	// todo: this hashtag crying lollol
}

type ListPeersRequest struct {
	PeerId string	`json:"id,omitempty"`
	Level string	`json:"level,omitempty"`
}

func (r *ListPeersRequest) Name() string {
	return "listpeers"
}

// Show current peer {peerId}. If {level} is set, include logs.
func (l *Lightning) GetPeer(peerId, level string) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&ListPeersRequest{peerId,level}, result)
	return result, err
}

// Show current peers, if {level} is set, include logs.
func (l *Lightning) ListPeers(level string) (interface{}, error) {
	return l.GetPeer("", level)
}


type ListNodeRequest struct {
	NodeId	string	`json:"id,omitempty"`
}

func (ln *ListNodeRequest) Name() string {
	return "listnodes"
}

// Get all nodes in our local network view, filter on node {id},
// if provided
func (l *Lightning) GetNode(nodeId string) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&ListNodeRequest{nodeId}, &result)
	return result, err
}

// List all nodes in our local network view
func (l *Lightning) ListNodes() (interface{}, error) {
	return l.GetNode("")
}

type RouteRequest struct {
	PeerId string		`json:"id"`
	MilliSatoshis uint64	`json:"msatoshi"`
	RiskFactor float32	`json:"riskfactor"`
	Cltv	uint		`json:"cltv"`
	FromId	string		`json:"fromid,omitempty"`
	FuzzPercent float32	`json:"fuzzpercent"`
	Seed	string		`json:"seed,omitempty'`
}

func (rr *RouteRequest) Name() string {
	return "getroute"
}

// Show route to {id} for {msatoshis}, using a {riskfactor} and optional
// {cltv} value (defaults to 9). If specified, search from {fromId} otherwise
// use current node as the source. Randomize the route with up to {fuzzpercent}
// (0.0 -> 100.0, default 5.0) using {seed} as an arbitrary-size string seed.
func (l *Lightning) GetRoute(peerId string, msats uint64, riskfactor float32, cltv uint, fromId string, fuzzpercent float32, seed string) (interface{}, error) {
	if peerId == "" {
		return nil, fmt.Errorf("Must provide a peerId to route to")
	}

	if msats == 0 {
		return nil, fmt.Errorf("No value set for payment. (`msatoshis` is equal to zero).")
	}

	if riskfactor <= 0 || riskfactor >= 100 {
		return nil, fmt.Errorf("The risk factor must set above 0 and beneath 100")
	}

	if fuzzpercent == 0 {
		fuzzpercent = 5.0
	} else if fuzzpercent < 0 || fuzzpercent > 100 {
		return nil, fmt.Errorf("The `fuzzpercent` value must be between 0 and 100")
	}

	if cltv == 0 {
		cltv = 9
	}

	var result interface{}
	err := l.client.Request(&RouteRequest{
		PeerId: peerId,
		MilliSatoshis: msats,
		RiskFactor: riskfactor,
		Cltv: cltv,
		FromId: fromId,
		FuzzPercent: fuzzpercent,
		Seed: seed,
	}, &result)
	return result, err
}

type ListChannelRequest struct {
	ShortChannelId string	`json:"short_channel_id"`
}

func (lc *ListChannelRequest) Name() string {
	return "listchannels"
}

// Get channel by {shortChanId}
func (l *Lightning) GetChannel(shortChanId string) (interface{}, error) {
// todo: type for short chan id?
	var result interface{}
	err := l.client.Request(&ListChannelRequest{shortChanId}, result)
	return result, err
}

func (l *Lightning) ListChannels() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&ListChannelRequest{}, result)
	return result, err
}

type InvoiceRequest struct {
	MilliSatoshis string	`json:"msatoshi"`
	Label string	`json:"label"`
	Description string	`json:"description"`
	ExpirySeconds uint32	`json:"expiry,omitempty"`
	Fallbacks []string	`json:"fallbacks,omitempty"`
	PreImage string	`json:"preimage,omitempty"`
}

func (ir *InvoiceRequest) Name() string {
	return "invoice"
}

// Creates an invoice with a value of "any", that can be paid with any amount
func (l *Lightning) CreateInvoiceAny(label, description string, expirySeconds uint32, fallbacks []string, preimage string) (interface{}, error) {
	return createInvoice(l, "any", label, description, expirySeconds, fallbacks, preimage)
}

// Creates an invoice with a value of `msat`. Label and description must be set.
//
// The 'label' is a unique string or number (which is treated as a string); it is
// never revealed to other nodes, but it can be used to query the status of this 
// invoice.
//
// The 'description' is a short description of purpose of payment. It is encoded 
// into the invoice. Must be UTF-8, cannot use '\n' JSON escape codes.
//
// The 'expiry' is optionally the number of seconds the invoice is valid for.
// Defaults to 3600 (1 hour).
// 
// 'fallbacks' is one or more fallback addresses to include in the invoice. They
// should be ordered from most preferred to least. Noe that these are not
// currently tracked to fulfill the invoice.
//
// The 'preimage' is a 64-digit hex string to be used as payment preimage for 
// the created invoice. By default, c-lightning will generate a secure 
// pseudorandom preimage seeded from an appropriate entropy source on your 
// system. **NOTE**: if you specify the 'preimage', you are responsible for 
// both ensuring that a suitable psuedorandom generator with sufficient entropy 
// was used in its creation and keeping it secret. 
// This parameter is an advanced feature intended for use with cutting-edge 
// cryptographic protocols and should not be used unless explicitly needed.
func (l *Lightning) CreateInvoice(msat uint64, label, description string, expirySeconds uint32, fallbacks []string, preimage string) (interface{}, error) {

	if msat <= 0 {
		return nil, fmt.Errorf("No value set for invoice. (`msat` is less than or equal to zero).")
	}
	return createInvoice(l, string(msat), label, description, expirySeconds, fallbacks, preimage)

}

func createInvoice(l *Lightning, msat, label, description string, expirySeconds uint32, fallbacks []string, preimage string) (interface{}, error) {

	if label == "" {
		return nil, fmt.Errorf("Must set a label on an invoice")
	}
	if description == "" {
		return nil, fmt.Errorf("Must set a description on an invoice")
	}

	var result interface{}
	err := l.client.Request(&InvoiceRequest{
		MilliSatoshis: msat,
		Label: label,
		Description: description,
		ExpirySeconds: expirySeconds,
		Fallbacks: fallbacks,
		PreImage: preimage,
	}, result)
	return result, err
}

type ListInvoiceRequest struct {
	Label string	`json:"label,omitempty"`
}

func (r *ListInvoiceRequest) Name() string {
	return "listinvoices"
}

// List all invoices
func (l *Lightning) ListInvoices() (interface{}, error) {
	return l.GetInvoice("")
}

// Show invoice {label}.
func (l *Lightning) GetInvoice(label string) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&ListInvoiceRequest{label}, result)
	return result, err
}

type DeleteInvoiceRequest struct {
	Label string	`json:"label"`
	Status string	`json:"status"`
}

func (r *DeleteInvoiceRequest) Name() string {
	return "delinvoice"
}

// Delete unpaid invoice {label} with {status}
func (l *Lightning) DeleteInvoice(label, status string) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DeleteInvoiceRequest{label,status}, result)
	return result, err
}

type WaitAnyInvoiceRequest struct {
	LastPayIndex uint	`json:"lastpay_index"`
}

func (r *WaitAnyInvoiceRequest) Name() string {
	return "waitanyinvoice"
}

// Waits until an invoice is paid, then returns a single entry. 
// Will not return or provide any invoices paid prior to or including 
// the lastPayIndex.
// 
// The 'pay index' is a monotonically-increasing number assigned to
// an invoice when it gets paid. The first valid 'pay index' is 1.
func (l *Lightning) WaitAnyInvoice(lastPayIndex uint) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&WaitAnyInvoiceRequest{lastPayIndex}, result)
	return result, err
}

type WaitInvoiceRequest struct {
	Label string	`json:"label"`
}

func (r *WaitInvoiceRequest) Name() string {
	return "waitinvoice"
}

func (l *Lightning) WaitInvoice(label string) (interface{}, error) {
	if label == "" {
		return nil, fmt.Errorf("Must call wait invoice with a label")
	}

	var result interface{}
	err := l.client.Request(&WaitInvoiceRequest{label}, result)
	return result, err
}

type DecodePayRequest struct {
	Bolt11 string	`json:"bolt11"`
	Description string	`json:"description,omitempty"`
}

func (r *DecodePayRequest) Name() string {
	return "decodepay"
}

// Decode the {bolt11}, using the provided 'description' if necessary.*
//
// * This is only necesary if the bolt11 includes a description hash.
// The provided description must match the included hash.
func (l *Lightning) DecodePay(bolt11, desc string) (interface{}, error) {
	if bolt11 == "" {
		return nil, fmt.Errorf("Must call decode pay with a bolt11")
	}

	var result interface{}
	err := l.client.Request(&DecodePayRequest{bolt11, desc}, result)
	return result, err
}

type HelpRequest struct {}

func (r *HelpRequest) Name() string {
	return "help"
}

// Show available c-lightning RPC commands
func (l *Lightning) Help() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&HelpRequest{}, result)
	return result, err
}

type StopRequest struct {}

func (r *StopRequest) Name() string {
	return "stop"
}

// Shut down the c-lightning process
func (l *Lightning) Stop() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&StopRequest{}, result)
	return result, err
}

type LogRequest struct {
	Level string	`json:"level,omitempty"`
}

func (r *LogRequest) Name() string {
	return "getlog"
}

// Show logs, with optional log {level} (info|unusual|debug|io)
// todo: use enum for levels 
func (l *Lightning) GetLog(level string) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&LogRequest{level}, result)
	return result, err
}

type DevRHashRequest struct {
	Secret string	`json:"secret"`
}

func (r *DevRHashRequest) Name() string {
	return "dev-rhash"
}

// Show SHA256 of {secret}
func (l *Lightning) DevHash(secret string) (interface{}, error) {
	if secret == "" {
		return nil, fmt.Errorf("Must pass in a valid secret to hash")
	}

	var result interface{}
	err := l.client.Request(&DevRHashRequest{secret}, result)
	return result, err
}

type DevCrashRequest struct {}

func (r *DevCrashRequest) Name() string {
	return "dev-crash"
}

// Crash lightningd by calling fatal()
func (l *Lightning) DevCrash() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DevCrashRequest{}, result)
	return result, err
}

type DevQueryShortChanIdsRequest struct {
	PeerId string	`json:"id"`
	ShortChanIds []string	`json:"scids"`
}

func (r *DevQueryShortChanIdsRequest) Name() string {
	return "dev-query-scids"
}

// Ask a peer for a particular set of short channel ids
func (l *Lightning) DevQueryShortChanIds(peerId string, shortChanIds []string) (interface{}, error) {
	if peerId == "" {
		return nil, fmt.Errorf("Must provide a peer id")
	}

	if len(shortChanIds) == 0 {
		return nil, fmt.Errorf("Must specify short channel ids to query for")
	}

	var result interface{}
	err := l.client.Request(&DevQueryShortChanIdsRequest{peerId, shortChanIds}, result)
	return result, err
}

type GetInfoRequest struct {}

func (r *GetInfoRequest) Name() string {
	return "getinfo"
}

func (l *Lightning) GetInfo() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&GetInfoRequest{}, result)
	return result, err
}

type SendPayRequest struct {
	Route interface{} `json:"route"`
	PaymentHash string `json:"payment_hash"`
	Desc string	`json:"description,omitempty"`
	MilliSatoshis uint64 `json:"msatoshi,omitempty"`
}

func (r *SendPayRequest) Name() string {
	return "sendpay"
}

// Send along {route} in return for preimage of {paymentHash}
//  Description and msat are optional.
// Generally a client would call GetRoute to resolve a route, then 
// use SendPay to send it.  If it fails, it would call GetRoute again
// to retry.
//
// Response will occur when payment is on its way to the destination.
// Does not wati for a definitive success or failure. Use 'waitsendpay'
// to poll or wait for definite success or failure.
//
// 'description', if provided, will be returned in 'waitsendpay' and
// 'listpayments' results.
//
// 'msat', if provided, is the amount that will be recorded as the target
// payment value. If not specified, it will be the final amount to the 
// destination (specified in route).  If specified, then the final amount
// at the destination must be from the specified 'msat' to twice that
// value, inclusive. This is inteded to obscure payments by overpaying
// slightly at the destination -- the acutal target paymnt is what 
// should be specified as the 'msat' argument.
// 
// Once a payment has succeeded, calls to 'SendPay' with the same 
// 'paymentHash' but a different 'msat' or destination will fail; this
// prevents accidental multiple payments. Calls with the same 'paymentHash',
// 'msat' and destination as a previous successful payment will return
// immediately with a success, even if the route is different.
func (l *Lightning) SendPay(route interface{}, paymentHash, description string, msat uint64) (interface{}, error) {
	if paymentHash == "" {
		return nil, fmt.Errorf("Must specify a paymentHash to pay")
	}
	// todo: an actual 'route' object specification
	if reflect.ValueOf(route).IsNil() {
		return nil, fmt.Errorf("Must specify a route to send payment along")
	}

	var result interface{}
	err := l.client.Request(&SendPayRequest{
		Route: route,
		PaymentHash: paymentHash,
		Desc: description,
		MilliSatoshis: msat,
	}, result)
	return result, err
}

type WaitSendPayRequest struct {
	PaymentHash string	`json:"payment_hash"`
	Timeout uint		`json:"timeout"`
}

func (r *WaitSendPayRequest) Name() string {
	return "waitsendpay"
}

// Polls or waits for the status of an outgoing payment that was 
// initiated by a previous 'SendPay' invocation.
//
// May provide a 'timeout, in seconds. When provided, will return a
// 200 error code (payment still in progress) if timeout elapses
// before the payment is definitively concluded (success or fail).
// If no 'timeout' is provided, the call waits indefinitely.
func (l *Lightning) WaitSendPay(paymentHash string, timeout uint) (interface{}, error) {
	if paymentHash == "" {
		return nil, fmt.Errorf("Must provide a payment hash to pay")
	}

	var result interface{}
	err := l.client.Request(&WaitSendPayRequest{paymentHash, timeout}, result)
	return result, err

}

type PayRequest struct {
	Bolt11 string	`json:"bolt11"`
	MilliSatoshi uint64	`json:"msatoshi,omitempty"`
	Desc string	`json:"description,omitempty"`
	RiskFactor float32	`json:"riskfactor,omitempty"`
	MaxFeePercent float32	`json:"maxfeeprecent,omitempty"`
	RetryFor uint	`json:"retry_for,omitempty"`
	MaxDelay uint	`json:"maxdelay,omitempty"`
	ExemptFee bool	`json:"exemptfee,omitempty"`
}

func (r *PayRequest) Name() string {
	return "pay"
}

func (l *Lightning) PayBolt(bolt11 string) (interface{}, error) {
	return l.Pay(&PayRequest{
		Bolt11: bolt11,
	})
}

// Send payment as specified by 'Bolt11' with 'MilliSatoshi'
// (Millisatoshis amount is ignored if the 'Bolt11' includes an amount).
//
// 'description' is required if the 'bolt11' includes a description hash.
// 
// 'riskfactor' is optional, defaults to 1.0
// Briefly, the 'riskfactor' is the estimated annual cost of your funds
// being stuck (as a percentage), multiplied by the percent change of 
// each node failing. Ex: 1% chance of node failure and a 20% annual cost
// would give you a risk factor of 20. c-lightning defaults to 1.0
//
// 'MaxFeePercent' is the max percentage of a payment that can be paid 
// in fees. c-lightning defaults to 0.5.
// 
// 'ExemptFee' can be used for tiny paymetns which would otherwise be 
// dominated by the fee leveraged by forwarding nodes. Setting 'ExemptFee' 
// allows 'MaxFeePercent' check to be skipped on fees that are smaller than
// 'ExemptFee'. c-lightning default is 5000 millisatoshi.
//
// c-lightning will keep finding routes and retrying payment until it succeeds
// or the given 'RetryFor' seconds have elapsed.  Note that the command may 
// stop retrying while payment is pending. You can continuing monitoring
// payment status with the ListPayments or WaitSendPay. 'RetryFor' defaults
// to 60 seconds.
// 
// 'MaxDelay' is used when determining whether a route incurs an acceptable 
// delay. A route will not be used if the estimated delay is above this.
// Defaults to the configured locktime max (--max-locktime-blocks)
// Units is in blocks.
func (l *Lightning) Pay(req *PayRequest) (interface{}, error) {
	if req.Bolt11 == "" {
		return nil, fmt.Errorf("Must supply a Bolt11 to pay")
	}
	if req.RiskFactor < 0 {
		return nil, fmt.Errorf("Risk factor must be postiive %f", req.RiskFactor)
	}
	if req.MaxFeePercent < 0 || req.MaxFeePercent > 100 {
		return nil, fmt.Errorf("MaxFeePercent must be a percentage. %f", req.MaxFeePercent)
	}
	var result interface{}
	err := l.client.Request(req, result)
	return result, err
}

type ListPaymentRequest struct {
	Bolt11 string	`json:"bolt11,omitempty"`
	PaymentHash string	`json:"payment_hash,omitempty"`
}

func (r *ListPaymentRequest) Name() string {
	return "listpayments"
}

// Show outgoing payments, regarding {bolt11}
func (l *Lightning) ListPayments(bolt11 string) (interface{}, error) {
	return l.listPayments(&ListPaymentRequest{
		Bolt11: bolt11,
	})
}

// Show outgoing payments, regarding {paymentHash}
func (l *Lightning) ListPaymentsHash(paymentHash string) (interface{}, error) {
	return l.listPayments(&ListPaymentRequest{
		PaymentHash: paymentHash,
	})
}

func (l *Lightning) listPayments(req *ListPaymentRequest) (interface{}, error) {
	var result interface{}
	err := l.client.Request(req, result)
	return result, err
}

type ConnectRequest struct {
	PeerId string	`json:"id"`
	Host string	`json:"host"`
	Port uint	`json:"port"`
}

func (r *ConnectRequest) Name() string {
	return "connect"
}

// Connect to {peerId} at {host}:{port}
func (l *Lightning) Connect(peerId, host string, port uint) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&ConnectRequest{peerId,host,port}, result)
	return result, err
}

type FundChannelRequest struct {
	Id string	`json:"id"`
	Satoshi uint64	`json:"satoshi"`
	FeeRate float32	`json:"feerate,omitempty"`
	Announce bool	`json:"announce,omitempty"`
}

func (r *FundChannelRequest) Name() string {
	return "fundchannel"
}

// Fund channel with node {id} using {satoshi} satoshis, with feerate of {feerate}. Uses
// default feerate if unset. 
// If announce is false, channel announcements will not be sent.
func (l *Lightning) FundChannel(id string, satoshis uint64, feerate float32, announce bool) (interface{}, error) {
	if feerate < 0 {
		return nil, fmt.Errorf("Feerate must be positive %f", feerate)
	}
	var result interface{}
	err := l.client.Request(&FundChannelRequest{id,satoshis,feerate,announce}, result)
	return result, err
}

type CloseRequest struct {
	PeerId string	`json:"id"`
	Force bool	`json:"force,omitempty"`
	Timeout uint	`json:"timeout,omitempty"`
}

func (r *CloseRequest) Name() string {
	return "close"
}

// Close the channel with peer {id}, timing out with {timeout} seconds. 
// If unspecified, times out in 30 seconds. 
// 
// If {force} is set, and close attempt times out, the channel will be closed
// unilaterally from our side.
//
// Can pass either peer id or channel id as {id} field.
func (l *Lightning) Close(id string, force bool, timeout uint) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&CloseRequest{id, force, timeout}, result)
	return result, err
}

type DevSignLastTxRequest struct {
	PeerId string	`json:"id"`
}

func (r *DevSignLastTxRequest) Name() string {
	return "dev-sign-last-tx"
}

// Sign and show the last commitment transaction with peer {peerId}
func (l *Lightning) DevSignLastTx(peerId string) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DevSignLastTxRequest{peerId}, result)
	return result, err
}

type DevFailRequest struct {
	PeerId string	`json:"id"`
}

func (r *DevFailRequest) Name() string {
	return "dev-fail"
}

// Fail with peer {id}
func (l *Lightning) DevFail(peerId string) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DevFailRequest{peerId}, result)
	return result, err
}

type DevReenableCommitRequest struct {
	PeerId string	`json:"id"`
}

func (r *DevReenableCommitRequest) Name() string {
	return "dev-reenable-commit"
}

// Re-enable the commit timer on peer {id}
func (l *Lightning) DevReenableCommit(id string) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DevReenableCommitRequest{id}, result)
	return result, err
}

type PingRequest struct {
	Id string	`json:"id"`
	Len uint	`json:"len"`
	PongBytes uint	`json:"pongbytes"`
}

func (r *PingRequest) Name() string {
	return "ping"
}

// Send {peerId} a ping of size 128, asking for 128 bytes in response
func (l *Lightning) Ping(peerId string) (interface{}, error) {
	return l.PingWithLen(peerId, 128, 128)
}

// Send {peerId} a ping of length {pingLen} asking for bytes {pongByteLen}
func (l *Lightning) PingWithLen(peerId string, pingLen, pongByteLen uint) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&PingRequest{peerId, pingLen, pongByteLen}, result)
	return result, err
}

type DevMemDumpRequest struct { }

func (r *DevMemDumpRequest) Name() string {
	return "dev-memdump"
}

// Show memory objects currently in use
func (l *Lightning) DevMemDump() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DevMemDumpRequest{}, result)
	return result, err
}

type DevMemLeakRequest struct {}

func (r *DevMemLeakRequest) Name() string {
	return "dev-memleak"
}

// Show unreferenced memory objects
func (l *Lightning) DevMemLeak() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DevMemLeakRequest{}, result)
	return result, err
}

type WithdrawRequest struct {
	Destination string	`json:"destination"`
	Satoshi string		`json:"satoshi"`
	FeeRate string		`json:"feerate,omitempty"`
}

type SatoshiAmount struct {
	Amount uint64
	SendAll bool
}

func (s *SatoshiAmount) String() string {
	if s.SendAll {
		return "all"
	}
	return string(s.Amount)
}

type FeeDirective int
const (
	Normal FeeDirective = iota
	Urgent
	Slow
)

func (f FeeDirective) String() string {
	return []string{
		"normal",
		"urgent",
		"slow",
	}[f]
}

type FeeRateStyle int

const (
	SatPerKiloByte FeeRateStyle = iota
	SatPerKiloSipa
)

type FeeRate struct {
	Rate uint
	Style FeeRateStyle
	Directive FeeDirective
}

func (r FeeRateStyle) String() string {
	return []string{"perkb","perkw"}[r]
}

func (f *FeeRate) String() string {
	if f.Rate > 0 {
		return string(f.Rate) + f.Style.String()
	}
	// defaults to 'normal'
	return f.Directive.String()
}

func (r *WithdrawRequest) Name() string {
	return "withdraw"
}

// Withdraw sends funds from c-lightning's internal wallet to the
// address specified in {destination}. Address can be of any Bitcoin
// accepted type, including bech32.
//
// {satoshi} is the amount to be withdrawn from the wallet.
// 
// {feerate} is an optional feerate to use. Can be either a directive
// (urgent, normal, or slow) or a number with an optional suffix.
// 'perkw' means the number is interpreted as satoshi-per-kilosipa (weight)
// and 'perkb' means it is interpreted bitcoind-style as satoshi-per-kilobyte.
// Omitting the suffix is equivalent to 'perkb'
// If not set, {feerate} defaults to 'normal'.
func (l *Lightning) Withdraw(destination string, amount *SatoshiAmount, feerate *FeeRate) (interface{}, error) {
	if amount == nil || (amount.Amount == 0 && !amount.SendAll) {
		return nil, fmt.Errorf("Must set satoshi amount to send")
	}
	request := &WithdrawRequest{
		Destination: destination,
		Satoshi: amount.String(),
	}
	if feerate != nil {
		request.FeeRate = feerate.String()
	}
	var result interface{}
	err := l.client.Request(request, &result)
	return result, err
}

type NewAddrRequest struct {
	AddressType string	`json:"addresstype,omitempty"`
}

func (r *NewAddrRequest) Name() string {
	return "newaddr"
}

type AddressType int

const (
	Bech32 AddressType = iota
	P2SHSegwit
)

func (a AddressType) String() string {
	return []string{"bech32", "p2sh-segwit"}[a]
}

// Get new Bech32 address for the internal wallet.
func (l *Lightning) NewAddr() (interface{}, error) {
	return l.NewAddressOfType(Bech32)
}

// Get new address of type {addrType} of the internal wallet.
func (l *Lightning) NewAddressOfType(addrType AddressType) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&NewAddrRequest{addrType.String()}, result)
	return result, err
}

type ListFundsRequest struct {}

func (r *ListFundsRequest) Name() string {
	return "listfunds"
}

// Show funds available for opening channels
func (l *Lightning) ListFunds() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&ListFundsRequest{}, result)
	return result, err
}

type ListForwardsRequest struct {}

func (r *ListForwardsRequest) Name() string {
	return "listforwards"
}

// List all forwarded payments and their information
func (l *Lightning) ListForwards() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&ListForwardsRequest{}, result)
	return result, err
}

type DevRescanOutputsRequest struct {}

func (r *DevRescanOutputsRequest) Name() string {
	return "dev-rescan-outputs"
}

// Synchronize the state of our funds with bitcoind
func (l *Lightning) DevRescanOutputs() (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DevRescanOutputsRequest{}, result)
	return result, err
}

type DevForgetChannelRequest struct {
	PeerId string	`json:"id"`
	Force bool	`json:"force"`
}

func (r *DevForgetChannelRequest) Name() string {
	return "dev-forget-channel"
}

// Forget channel with id {peerId}. Optionally {force} if has active channel.
// Caution, this might lose you funds.
func (l *Lightning) DevForgetChannel(peerId string, force bool) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DevForgetChannelRequest{peerId, force}, result)
	return result, err
}

type DisconnectRequest struct {
	PeerId string	`json:"id"`
	Force bool	`json:"force"`
}

func (r *DisconnectRequest) Name() string {
	return "disconnect"
}

// Disconnect from peer with {peerId}. Optionally {force} if has active channel.
func (l *Lightning) Disconnect(peerId string, force bool) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&DisconnectRequest{peerId, force}, result)
	return result, err
}

type FeeRatesRequest struct {
	Style string	`json:"style"`
}

func (r *FeeRatesRequest) Name() string {
	return "feerates"
}

// Return feerate estimates, either satoshi-per-kw or satoshi-per-kb {style}
func (l *Lightning) FeeRates(style FeeRateStyle) (interface{}, error) {
	var result interface{}
	err := l.client.Request(&FeeRatesRequest{style.String()}, result)
	return result, err
}

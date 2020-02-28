package glightning

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/niftynei/glightning/jrpc2"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
)

type Subscription string
type Hook string

const (
	_Connect        Subscription = "connect"
	_Disconnect     Subscription = "disconnect"
	_InvoicePaid    Subscription = "invoice_payment"
	_ChannelOpened  Subscription = "channel_opened"
	_Warning        Subscription = "warning"
	_Forward        Subscription = "forward_event"
	_SendPaySuccess Subscription = "sendpay_success"
	_SendPayFailure Subscription = "sendpay_failure"
	_PeerConnected  Hook         = "peer_connected"
	_DbWrite        Hook         = "db_write"
	_InvoicePayment Hook         = "invoice_payment"
	_OpenChannel    Hook         = "openchannel"
	_HtlcAccepted   Hook         = "htlc_accepted"
)

// This hook is called whenever a peer has connected and successfully completed
//   the cryptographic handshake. The parameters have the following structure if
//   there is a channel with the peer:
type PeerConnectedEvent struct {
	Peer PeerEvent `json:"peer"`
	hook func(*PeerConnectedEvent) (*PeerConnectedResponse, error)
}

type PeerEvent struct {
	PeerId   string `json:"id"`
	Addr     string `json:"addr"`
	Features string `json:"features"`
}

type _PeerConnectedResult string

const _PcDisconnect _PeerConnectedResult = "disconnect"
const _PcContinue _PeerConnectedResult = "continue"

type PeerConnectedResponse struct {
	Result       _PeerConnectedResult `json:"result"`
	ErrorMessage string               `json:"error_message,omitempty"`
}

func (pc *PeerConnectedEvent) New() interface{} {
	return &PeerConnectedEvent{
		hook: pc.hook,
	}
}

func (pc *PeerConnectedEvent) Name() string {
	return string(_PeerConnected)
}

func (pc *PeerConnectedEvent) Call() (jrpc2.Result, error) {
	return pc.hook(pc)
}

func (pc *PeerConnectedEvent) Continue() *PeerConnectedResponse {
	return &PeerConnectedResponse{
		Result: _PcContinue,
	}
}

func (pc *PeerConnectedEvent) Disconnect(errMsg string) *PeerConnectedResponse {
	return &PeerConnectedResponse{
		Result:       _PcDisconnect,
		ErrorMessage: errMsg,
	}
}

// Note that this Hook is called before the plugin is initialized.
// A plugin that registers for this hook may not register for any other
// hooks.
//
// Any response but "true" will cause lightningd to error without committing
// to the database.
type DbWriteEvent struct {
	Writes []string `json:"writes"`
	hook   func(*DbWriteEvent) (bool, error)
}

func (dbw *DbWriteEvent) New() interface{} {
	return &DbWriteEvent{
		hook: dbw.hook,
	}
}

func (dbw *DbWriteEvent) Name() string {
	return string(_DbWrite)
}

func (dbw *DbWriteEvent) Call() (jrpc2.Result, error) {
	return dbw.hook(dbw)
}

type Payment struct {
	Label         string `json:"label"`
	PreImage      string `json:"preimage"`
	MilliSatoshis string `json:"msat"`
}

type InvoicePaymentResponse struct {
	FailureCode *uint16 `json:"failure_code,omitempty"`
}

type InvoicePaymentEvent struct {
	Payment Payment `json:"payment"`
	hook    func(*InvoicePaymentEvent) (*InvoicePaymentResponse, error)
}

func (ip *InvoicePaymentEvent) New() interface{} {
	return &InvoicePaymentEvent{
		hook: ip.hook,
	}
}

func (ip *InvoicePaymentEvent) Name() string {
	return string(_InvoicePayment)
}

func (ip *InvoicePaymentEvent) Call() (jrpc2.Result, error) {
	return ip.hook(ip)
}

func (ip *InvoicePaymentEvent) Continue() *InvoicePaymentResponse {
	return &InvoicePaymentResponse{}
}

func (ip *InvoicePaymentEvent) Fail(failureCode uint16) *InvoicePaymentResponse {
	return &InvoicePaymentResponse{
		FailureCode: &failureCode,
	}
}

type OpenChannelEvent struct {
	OpenChannel OpenChannel `json:"openchannel"`
	hook        func(*OpenChannelEvent) (*OpenChannelResponse, error)
}

type OpenChannel struct {
	PeerId                            string `json:"id"`
	FundingSatoshis                   string `json:"funding_satoshis"`
	PushMilliSatoshis                 string `json:"push_msat"`
	DustLimitSatoshis                 string `json:"dust_limit_satoshis"`
	MaxHtlcValueInFlightMilliSatoshis string `json:"max_htlc_value_in_flight_msat"`
	ChannelReserveSatoshis            string `json:"channel_reserve_satoshis"`
	HtlcMinimumMillisatoshis          string `json:"htlc_minimum_msat"`
	FeeratePerKw                      int    `json:"feerate_per_kw"`
	ToSelfDelay                       int    `json:"to_self_delay"`
	MaxAcceptedHtlcs                  int    `json:"max_accepted_htlcs"`
	ChannelFlags                      int    `json:"channel_flags"`
	ShutdownScriptPubkey              string `json:"shutdown_scriptpubkey"`
}

type OpenChannelResult string

const OcReject OpenChannelResult = "reject"
const OcContinue OpenChannelResult = "continue"

type OpenChannelResponse struct {
	Result OpenChannelResult `json:"result"`
	// Only allowed if result is "reject"
	// Sent back to peer.
	Message string `json:"error_message,omitempty"`
	CloseToAddress string `json:"close_to,omitempty"`
}

func (oc *OpenChannelEvent) New() interface{} {
	return &OpenChannelEvent{
		hook: oc.hook,
	}
}

func (oc *OpenChannelEvent) Name() string {
	return string(_OpenChannel)
}

func (oc *OpenChannelEvent) Call() (jrpc2.Result, error) {
	return oc.hook(oc)
}

func (oc *OpenChannelEvent) Reject(errorMessage string) *OpenChannelResponse {
	return &OpenChannelResponse{
		Result:  OcReject,
		Message: errorMessage,
	}
}

func (oc *OpenChannelEvent) Continue() *OpenChannelResponse {
	return &OpenChannelResponse{
		Result: OcContinue,
	}
}

func (oc *OpenChannelEvent) ContinueWithCloseTo(address string) *OpenChannelResponse {
	return &OpenChannelResponse{
		Result: OcContinue,
		CloseToAddress: address,
	}
}

// The `htlc_accepted` hook is called whenever an incoming HTLC is accepted, and
// its result determines how `lightningd` should treat that HTLC.
//
// Warning: `lightningd` will replay the HTLCs for which it doesn't have a final
//   verdict during startup. This means that, if the plugin response wasn't
//   processed before the HTLC was forwarded, failed, or resolved, then the plugin
//   may see the same HTLC again during startup. It is therefore paramount that the
//   plugin is idempotent if it talks to an external system.
type HtlcAcceptedEvent struct {
	Onion Onion     `json:"onion"`
	Htlc  HtlcOffer `json:"htlc"`
	hook  func(*HtlcAcceptedEvent) (*HtlcAcceptedResponse, error)
}

type Onion struct {
	Payload      string `json:"payload"`
	NextOnion    string `json:"next_onion"`
	SharedSecret string `json:"shared_secret"`
	PerHop       PerHop `json:"per_hop_v0"`
}

type PerHop struct {
	Realm                      string `json:"realm"`
	ShortChannelId             string `json:"short_channel_id"`
	ForwardAmountMilliSatoshis string `json:"forward_amount"`
	OutgoingCltvValue          int    `json:"outgoing_cltv_value"`
}

type HtlcOffer struct {
	Amount             string `json:"amount"`
	CltvExpiry         int    `json:"cltv_expiry"`
	CltvExpiryRelative int    `json:"cltv_expiry_relative"`
	PaymentHash        string `json:"payment_hash"`
}

type HtlcAcceptedResult string

const (
	_HcContinue HtlcAcceptedResult = "continue"
	_HcFail     HtlcAcceptedResult = "fail"
	_HcResolve  HtlcAcceptedResult = "resolve"
)

type HtlcAcceptedResponse struct {
	Result HtlcAcceptedResult `json:"result"`
	// Only allowed if result is 'fail'
	FailureCode *uint16 `json:"failure_code,omitempty"`
	// Only allowed if result is 'resolve'
	PaymentKey string `json:"payment_key,omitempty"`
}

func (ha *HtlcAcceptedEvent) New() interface{} {
	return &HtlcAcceptedEvent{
		hook: ha.hook,
	}
}

func (ha *HtlcAcceptedEvent) Name() string {
	return string(_HtlcAccepted)
}

func (ha *HtlcAcceptedEvent) Call() (jrpc2.Result, error) {
	return ha.hook(ha)
}

func (ha *HtlcAcceptedEvent) Continue() *HtlcAcceptedResponse {
	return &HtlcAcceptedResponse{
		Result: _HcContinue,
	}
}

func (ha *HtlcAcceptedEvent) Fail(failCode uint16) *HtlcAcceptedResponse {
	return &HtlcAcceptedResponse{
		Result:      _HcFail,
		FailureCode: &failCode,
	}
}

func (ha *HtlcAcceptedEvent) Resolve(paymentKey string) *HtlcAcceptedResponse {
	return &HtlcAcceptedResponse{
		Result:     _HcResolve,
		PaymentKey: paymentKey,
	}
}

type ConnectEvent struct {
	PeerId  string  `json:"id"`
	Address Address `json:"address"`
	cb      func(*ConnectEvent)
}

func (e *ConnectEvent) Name() string {
	return string(_Connect)
}

func (e *ConnectEvent) New() interface{} {
	return &ConnectEvent{
		cb: e.cb,
	}
}

func (e *ConnectEvent) Call() (jrpc2.Result, error) {
	e.cb(e)
	return nil, nil
}

type DisconnectEvent struct {
	PeerId string `json:"id"`
	cb     func(d *DisconnectEvent)
}

func (e *DisconnectEvent) Name() string {
	return string(_Disconnect)
}

func (e *DisconnectEvent) New() interface{} {
	return &DisconnectEvent{
		cb: e.cb,
	}
}

func (e *DisconnectEvent) Call() (jrpc2.Result, error) {
	e.cb(e)
	return nil, nil
}

type InvoicePaidEvent struct {
	Payment Payment `json:"invoice_payment"`
	cb      func(e *Payment)
}

func (e *InvoicePaidEvent) Name() string {
	return string(_InvoicePaid)
}

func (e *InvoicePaidEvent) New() interface{} {
	return &InvoicePaidEvent{
		cb: e.cb,
	}
}

func (e *InvoicePaidEvent) Call() (jrpc2.Result, error) {
	e.cb(&e.Payment)
	return nil, nil
}

type ChannelOpenedEvent struct {
	ChannelOpened ChannelOpened `json:"channel_opened"`
	cb            func(e *ChannelOpened)
}

type ChannelOpened struct {
	PeerId          string `json:"id"`
	FundingSatoshis string `json:"amount"`
	FundingTxId     string `json:"funding_txid"`
	FundingLocked   bool   `json:"funding_locked"`
}

func (e *ChannelOpenedEvent) Name() string {
	return string(_ChannelOpened)
}

func (e *ChannelOpenedEvent) New() interface{} {
	return &ChannelOpenedEvent{
		cb: e.cb,
	}
}

func (e *ChannelOpenedEvent) Call() (jrpc2.Result, error) {
	e.cb(&e.ChannelOpened)
	return nil, nil
}

type ForwardEvent struct {
	Forward *Forwarding `json:"forward_event"`
	cb      func(*Forwarding)
}

func (e *ForwardEvent) Name() string {
	return string(_Forward)
}

func (e *ForwardEvent) New() interface{} {
	return &ForwardEvent{
		cb: e.cb,
	}
}

func (e *ForwardEvent) Call() (jrpc2.Result, error) {
	e.cb(e.Forward)
	return nil, nil
}

type SendPaySuccess struct {
	Id                     uint   `json:"id"`
	PaymentHash            string `json:"payment_hash"`
	Destination            string `json:"destination"`
	MilliSatoshi           uint64 `json:"msatoshi"`
	AmountMilliSatoshi     string `json:"amount_msat"`
	AmountSent             uint64 `json:"msatoshi_sent"`
	AmountSentMilliSatoshi string `json:"amount_sent_msat"`
	CreatedAt              uint64 `json:"created_at"`
	Status                 string `json:"status"`
	PaymentPreimage        string `json:"payment_preimage"`
}

type SendPaySuccessEvent struct {
	SendPaySuccess *SendPaySuccess `json:"sendpay_success"`
	cb             func(*SendPaySuccess)
}

func (e *SendPaySuccessEvent) Name() string {
	return string(_SendPaySuccess)
}

func (e *SendPaySuccessEvent) New() interface{} {
	return &SendPaySuccessEvent{
		cb: e.cb,
	}
}

func (e *SendPaySuccessEvent) Call() (jrpc2.Result, error) {
	e.cb(e.SendPaySuccess)
	return nil, nil
}

type SendPayFailureData struct {
	Id                     int    `json:"id"`
	PaymentHash            string `json:"payment_hash"`
	Destination            string `json:"destination"`
	MilliSatoshi           uint64 `json:"msatoshi"`
	AmountMilliSatoshi     string `json:"amount_msat"`
	AmountSent             uint64 `json:"msatoshi_sent"`
	AmountSentMilliSatoshi string `json:"amount_sent_msat"`
	Status                 string `json:"status"`
	CreatedAt              uint64 `json:"created_at"`
	ErringIndex            uint64 `json:"erring_index"`
	FailCode               int    `json:"failcode"`
	ErringNode             string `json:"erring_node"`
	ErringChannel          string `json:"erring_channel"`
	ErringDirection        int    `json:"erring_direction"`
	FailCodeName           string `json:"failcodename"`
}

type SendPayFailure struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    SendPayFailureData `json:"data"`
}

type SendPayFailureEvent struct {
	SendPayFailure *SendPayFailure `json:"sendpay_failure"`
	cb             func(*SendPayFailure)
}

func (e *SendPayFailureEvent) Name() string {
	return string(_SendPayFailure)
}

func (e *SendPayFailureEvent) New() interface{} {
	return &SendPayFailureEvent{
		cb: e.cb,
	}
}

func (e *SendPayFailureEvent) Call() (jrpc2.Result, error) {
	e.cb(e.SendPayFailure)
	return nil, nil
}

type WarnEvent struct {
	Warning Warning `json:"warning"`
	cb      func(*Warning)
}

type Warning struct {
	Level  string `json:"level"`
	Time   string `json:"time"`
	Source string `json:"source"`
	Log    string `json:"log"`
}

func (e *WarnEvent) Name() string {
	return string(_Warning)
}

func (e *WarnEvent) New() interface{} {
	return &WarnEvent{
		cb: e.cb,
	}
}

func (e *WarnEvent) Call() (jrpc2.Result, error) {
	e.cb(&e.Warning)
	return nil, nil
}

type Option struct {
	Name        string
	Default     string
	description string
	Val         string
}

func NewOption(name, description, defaultValue string) *Option {
	return &Option{
		Name:        name,
		Default:     defaultValue,
		description: description,
	}
}

func (o *Option) Description() string {
	if o.description != "" {
		return o.description
	}

	return "A g-lightning plugin option"
}

func (o *Option) Set(value string) {
	o.Val = value
}

func (o *Option) Value() string {
	return o.Val
}

func (o *Option) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Default     string `json:"default"`
		Description string `json:"description"`
		Category    string `json:"category,omitempty"`
	}{
		Name:        o.Name,
		Type:        "string", // all options are type string atm
		Default:     o.Default,
		Description: o.Description(),
	})
}

const FormatSimple string = "simple"

type RpcMethod struct {
	Method   jrpc2.ServerMethod
	Desc     string
	LongDesc string
	Category string
}

func NewRpcMethod(method jrpc2.ServerMethod, desc string) *RpcMethod {
	return &RpcMethod{
		Method: method,
		Desc:   desc,
	}
}

func (r *RpcMethod) Description() string {
	if r.Desc != "" {
		return r.Desc
	}

	return "A g-lightning RPC method."
}

func (r *RpcMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name     string `json:"name"`
		Desc     string `json:"description"`
		Usage    string `json:"usage"`
		LongDesc string `json:"long_description,omitempty"`
		Category string `json:"category,omitempty"`
	}{
		Name:     r.Method.Name(),
		Desc:     r.Description(),
		LongDesc: r.LongDesc,
		Usage:    getUsageList(r.Method),
		Category: r.Category,
	})
}

type GetManifestMethod struct {
	plugin *Plugin
}

func (gm *GetManifestMethod) New() interface{} {
	method := &GetManifestMethod{}
	method.plugin = gm.plugin
	return method
}

func NewManifestRpcMethod(p *Plugin) *RpcMethod {
	return &RpcMethod{
		Method: &GetManifestMethod{
			plugin: p,
		},
		Desc: "Generate manifest for plugin",
	}
}

type Manifest struct {
	Options       []*Option    `json:"options"`
	RpcMethods    []*RpcMethod `json:"rpcmethods"`
	Dynamic       bool         `json:"dynamic"`
	Subscriptions []string     `json:"subscriptions,omitempty"`
	Hooks         []Hook       `json:"hooks,omitempty"`
}

func (gm GetManifestMethod) Name() string {
	return "getmanifest"
}

// Don't include 'built-in' methods in manifest list
func isBuiltInMethod(name string) bool {
	return name == "getmanifest" ||
		name == "init"
}

// Builds the manifest object that's returned from the
// `getmanifest` method.
func (gm GetManifestMethod) Call() (jrpc2.Result, error) {
	m := &Manifest{}
	m.RpcMethods = make([]*RpcMethod, 0, len(gm.plugin.methods))
	for _, rpc := range gm.plugin.methods {
		if !isBuiltInMethod(rpc.Method.Name()) {
			m.RpcMethods = append(m.RpcMethods, rpc)
		}
	}

	m.Options = make([]*Option, len(gm.plugin.options))
	i := 0
	for _, option := range gm.plugin.options {
		m.Options[i] = option
		i++
	}
	m.Subscriptions = make([]string, len(gm.plugin.subscriptions))
	for i, sub := range gm.plugin.subscriptions {
		m.Subscriptions[i] = sub
	}
	m.Hooks = make([]Hook, len(gm.plugin.hooks))
	for i, hook := range gm.plugin.hooks {
		m.Hooks[i] = hook
	}

	m.Dynamic = gm.plugin.dynamic

	return m, nil
}

type Config struct {
	LightningDir string `json:"lightning-dir"`
	RpcFile      string `json:"rpc-file"`
	Startup      bool   `json:"startup,omitempty"`
	Network      string `json:"network"`
}

type InitMethod struct {
	Options       map[string]string `json:"options"`
	Configuration *Config           `json:"configuration"`
	plugin        *Plugin
}

func NewInitRpcMethod(p *Plugin) *RpcMethod {
	return &RpcMethod{
		Method: &InitMethod{
			plugin: p,
		},
	}
}

func (im InitMethod) New() interface{} {
	method := &InitMethod{}
	method.plugin = im.plugin
	return method
}

func (im InitMethod) Name() string {
	return "init"
}

func (im InitMethod) Call() (jrpc2.Result, error) {
	// fill in options
	for name, value := range im.Options {
		option, exists := im.plugin.options[name]
		if !exists {
			log.Printf("No option %s registered on this plugin", name)
			continue
		}
		opt := option
		opt.Set(value)
	}
	// stash the config...
	im.plugin.Config = im.Configuration
	im.plugin.initialized = true

	// call init hook
	im.plugin.initFn(im.plugin, im.plugin.getOptionSet(), im.Configuration)

	// Result of `init` is currently discarded by c-light
	return "ok", nil
}

type LogNotification struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func (r *LogNotification) Name() string {
	return "log"
}

func (p *Plugin) Log(message string, level LogLevel) {
	for _, line := range strings.Split(message, "\n") {
		p.server.Notify(&LogNotification{level.String(), line})
	}
}

// Map for registering hooks. Not the *most* elegant but
//   it'll do for now.
type Hooks struct {
	PeerConnected  func(*PeerConnectedEvent) (*PeerConnectedResponse, error)
	DbWrite        func(*DbWriteEvent) (bool, error)
	InvoicePayment func(*InvoicePaymentEvent) (*InvoicePaymentResponse, error)
	OpenChannel    func(*OpenChannelEvent) (*OpenChannelResponse, error)
	HtlcAccepted   func(*HtlcAcceptedEvent) (*HtlcAcceptedResponse, error)
}

func (p *Plugin) RegisterHooks(hooks *Hooks) error {
	if hooks.DbWrite != nil {
		err := p.server.Register(&DbWriteEvent{
			hook: hooks.DbWrite,
		})
		if err != nil {
			return err
		}
		p.hooks = append(p.hooks, _DbWrite)
	}
	if hooks.PeerConnected != nil {
		err := p.server.Register(&PeerConnectedEvent{
			hook: hooks.PeerConnected,
		})
		if err != nil {
			return err
		}
		p.hooks = append(p.hooks, _PeerConnected)
	}
	if hooks.InvoicePayment != nil {
		err := p.server.Register(&InvoicePaymentEvent{
			hook: hooks.InvoicePayment,
		})
		if err != nil {
			return err
		}
		p.hooks = append(p.hooks, _InvoicePayment)
	}
	if hooks.OpenChannel != nil {
		err := p.server.Register(&OpenChannelEvent{
			hook: hooks.OpenChannel,
		})
		if err != nil {
			return err
		}
		p.hooks = append(p.hooks, _OpenChannel)
	}
	if hooks.HtlcAccepted != nil {
		err := p.server.Register(&HtlcAcceptedEvent{
			hook: hooks.HtlcAccepted,
		})
		if err != nil {
			return err
		}
		p.hooks = append(p.hooks, _HtlcAccepted)
	}
	return nil
}

type Plugin struct {
	server        *jrpc2.Server
	options       map[string]*Option
	methods       map[string]*RpcMethod
	hooks         []Hook
	subscriptions []string
	initialized   bool
	initFn        func(plugin *Plugin, options map[string]string, c *Config)
	Config        *Config
	stopped       bool
	dynamic       bool
}

func NewPlugin(initHandler func(p *Plugin, o map[string]string, c *Config)) *Plugin {
	plugin := &Plugin{}
	plugin.server = jrpc2.NewServer()
	plugin.options = make(map[string]*Option)
	plugin.methods = make(map[string]*RpcMethod)
	plugin.initFn = initHandler
	plugin.dynamic = true
	return plugin
}

func (p *Plugin) Start(in, out *os.File) error {
	p.checkForMonkeyPatch()
	// register the init & getmanifest commands
	p.RegisterMethod(NewManifestRpcMethod(p))
	p.RegisterMethod(NewInitRpcMethod(p))

	return p.server.StartUp(in, out)
}

func (p *Plugin) Stop() {
	p.stopped = true
	p.server.Shutdown()
}

// Remaps stdout to print logs to c-lightning via notifications
func (p *Plugin) checkForMonkeyPatch() {
	_, isLN := os.LookupEnv("LIGHTNINGD_PLUGIN")
	if !isLN {
		return
	}

	// Use a logfile instead
	filename, _ := os.LookupEnv("GOLIGHT_DEBUG_LOGFILE")
	if filename != "" {
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("Unable to open log file for writing: " + err.Error())
			return
		}
		log.SetFlags(log.Ltime | log.Lshortfile)
		log.SetOutput(f)
		return
	}
	// otherwise we send things out
	// pipe logs out...
	in, out := io.Pipe()
	go func(in io.Reader, plugin *Plugin) {
		// everytime we get a new message, log it thru c-lightning
		scanner := bufio.NewScanner(in)
		for scanner.Scan() && !plugin.stopped {
			plugin.Log(scanner.Text(), Info)
		}
		if err := scanner.Err(); err != nil {
			log.Fatal("can't print out to std err, killing..." + err.Error())
		}
	}(in, p)
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetOutput(out)
}

func (p *Plugin) RegisterMethod(m *RpcMethod) error {
	err := p.server.Register(m.Method)
	if err != nil {
		return err
	}
	err = p.registerRpcMethod(m)
	if err != nil {
		p.server.Unregister(m.Method)
	}
	return err
}

func (p *Plugin) registerRpcMethod(rpc *RpcMethod) error {
	if rpc == nil || rpc.Method == nil {
		return fmt.Errorf("Can't register an empty rpc method")
	}
	m := rpc.Method
	if _, exists := p.methods[m.Name()]; exists {
		return fmt.Errorf("Method `%s` already registered", m.Name())
	}
	p.methods[m.Name()] = rpc
	return nil
}

func (p *Plugin) UnregisterMethod(rpc *RpcMethod) error {
	// potentially munges the error code from server
	// but we don't really care as long as the method
	// is no longer registered either place.
	err := p.unregisterMethod(rpc)
	if err != nil || rpc.Method != nil {
		err = p.server.Unregister(rpc.Method)
	}
	return err
}

func (p *Plugin) unregisterMethod(rpc *RpcMethod) error {
	if rpc == nil || rpc.Method == nil {
		return fmt.Errorf("Can't unregister an empty method")
	}
	m := rpc.Method
	if _, exists := p.methods[m.Name()]; !exists {
		return fmt.Errorf("Can't unregister, method %s is unknown", m.Name())
	}
	delete(p.methods, m.Name())
	return nil
}

func (p *Plugin) RegisterOption(o *Option) error {
	if o == nil {
		return fmt.Errorf("Can't register an empty option")
	}
	if _, exists := p.options[o.Name]; exists {
		return fmt.Errorf("Option `%s` already registered", o.Name)
	}
	p.options[o.Name] = o
	return nil
}

func (p *Plugin) UnregisterOption(o *Option) error {
	if o == nil {
		return fmt.Errorf("Can't remove an empty option")
	}
	if _, exists := p.options[o.Name]; !exists {
		return fmt.Errorf("No %s option registered", o.Name)
	}
	delete(p.options, o.Name)
	return nil
}

func (p *Plugin) GetOption(name string) (*Option, error) {
	opt := p.options[name]
	if opt == nil {
		return nil, errors.New(fmt.Sprintf("Option '%s' not found", name))
	}
	return opt, nil
}

func (p *Plugin) GetOptionValue(name string) (string, error) {
	opt, err := p.GetOption(name)
	if err != nil {
		return "", err
	}
	return opt.Val, nil
}

func (p *Plugin) getOptionSet() map[string]string {
	options := make(map[string]string, len(p.options))
	for key, option := range p.options {
		options[key] = option.Value()
	}
	return options
}

func (p *Plugin) SubscribeConnect(cb func(c *ConnectEvent)) {
	p.subscribe(&ConnectEvent{
		cb: cb,
	})
}

func (p *Plugin) SubscribeDisconnect(cb func(c *DisconnectEvent)) {
	p.subscribe(&DisconnectEvent{
		cb: cb,
	})
}

func (p *Plugin) SubscribeInvoicePaid(cb func(c *Payment)) {
	p.subscribe(&InvoicePaidEvent{
		cb: cb,
	})
}

func (p *Plugin) SubscribeChannelOpened(cb func(c *ChannelOpened)) {
	p.subscribe(&ChannelOpenedEvent{
		cb: cb,
	})
}

func (p *Plugin) SubscribeWarnings(cb func(c *Warning)) {
	p.subscribe(&WarnEvent{
		cb: cb,
	})
}

func (p *Plugin) SubscribeSendPaySuccess(cb func(c *SendPaySuccess)) {
	p.subscribe(&SendPaySuccessEvent{
		cb: cb,
	})
}

func (p *Plugin) SubscribeSendPayFailure(cb func(c *SendPayFailure)) {
	p.subscribe(&SendPayFailureEvent{
		cb: cb,
	})
}

func (p *Plugin) SubscribeForwardings(cb func(c *Forwarding)) {
	p.subscribe(&ForwardEvent{
		cb: cb,
	})
}

func (p *Plugin) subscribe(subscription jrpc2.ServerMethod) {
	p.server.Register(subscription)
	p.subscriptions = append(p.subscriptions, subscription.Name())
}

func (p *Plugin) SetDynamic(d bool) {
	p.dynamic = d
}

// Returns a list of params for this call, wrap
// optional (i.e. omitempty) marked params with []
func getUsageList(method jrpc2.ServerMethod) string {
	var sb strings.Builder

	v := reflect.Indirect(reflect.ValueOf(method))
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)
		if !fieldVal.CanInterface() {
			continue
		}
		tag, _ := fieldType.Tag.Lookup("json")
		// Ignore ignored fields
		if tag == "-" {
			continue
		}
		optional := strings.Contains(tag, "omitempty")
		if i := strings.Index(tag, ","); i > -1 {
			tag = tag[:i]
		}

		if sb.Len() > 0 {
			sb.WriteRune(' ')
		}
		if optional {
			sb.WriteRune('[')
		}
		sb.WriteString(tag)
		if optional {
			sb.WriteRune(']')
		}
	}

	return sb.String()
}

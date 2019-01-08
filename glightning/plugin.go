package glightning

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/niftynei/golight/jrpc2"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
)

type Subscription string

const (
	Connect    Subscription = "connect"
	Disconnect Subscription = "disconnect"
)

type ConnectEvent struct {
	PeerId  string  `json:"id"`
	Address Address `json:"address"`
	cb      func(*ConnectEvent)
}

func (e *ConnectEvent) Name() string {
	return string(Connect)
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
	return string(Disconnect)
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
		Default     string `json:"default,omitempty"`
		Description string `json:"description"`
	}{
		Name:        o.Name,
		Type:        "string", // all options are type string atm
		Default:     o.Default,
		Description: o.Description(),
	})
}

type RpcMethod struct {
	Method   jrpc2.ServerMethod
	Desc     string
	LongDesc string
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
		Name     string   `json:"name"`
		Desc     string   `json:"description"`
		Params   []string `json:"params,omitempty"`
		LongDesc string   `json:"long_description,omitempty"`
	}{
		Name:     r.Method.Name(),
		Desc:     r.Description(),
		LongDesc: r.LongDesc,
		Params:   getParamList(r.Method),
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
	Subscriptions []string     `json:"subscriptions,omitempty"`
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

	return m, nil
}

type Config struct {
	LightningDir string `json:"lightning-dir"`
	RpcFile      string `json:"rpc-file"`
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

type Plugin struct {
	server        *jrpc2.Server
	options       map[string]*Option
	methods       map[string]*RpcMethod
	subscriptions []string
	initialized   bool
	initFn        func(plugin *Plugin, options map[string]string, c *Config)
	Config        *Config
	stopped       bool
}

func NewPlugin(initHandler func(p *Plugin, o map[string]string, c *Config)) *Plugin {
	plugin := &Plugin{}
	plugin.server = jrpc2.NewServer()
	plugin.options = make(map[string]*Option)
	plugin.methods = make(map[string]*RpcMethod)
	plugin.initFn = initHandler
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
		f, err := os.OpenFile("plugin.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
		fmt.Errorf("Can't unregister, method %s is unknown", m.Name())
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

func (p *Plugin) GetOption(name string) *Option {
	return p.options[name]
}

func (p *Plugin) GetOptionValue(name string) string {
	return p.GetOption(name).Val
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

func (p *Plugin) subscribe(subscription jrpc2.ServerMethod) {
	p.server.Register(subscription)
	p.subscriptions = append(p.subscriptions, subscription.Name())
}

func getParamList(method jrpc2.ServerMethod) []string {
	paramList := make([]string, 0)
	v := reflect.Indirect(reflect.ValueOf(method))

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() {
			continue
		}
		paramList = append(paramList, field.Type().Name())
	}
	return paramList
}

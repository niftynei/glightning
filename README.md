# golight: a c-lightning plugin and RPC driver

golight is a driver for the Lightning Network protocol implemenation [c-lightning](https://github.com/ElementsProject/lightning).

It offers an RPC client for calling lightning commands and a Plugin infrastructure, for creating your own c-lightning commands and registering for subscriptions.

More details on c-lightning plugins can be found in the [c-lightning docs](https://github.com/ElementsProject/lightning/blob/master/doc/plugins.md)


## Plugins: How to Use
`golight` builds upon the method paradigm established in `jrpc2`. Options, RpcMethods, and Subscriptions all must be registered on the plugin prior to start in order to be included in your manifest. 

RpcMethods and Subscriptons are both a form of `jrpc2.Method`. 

### Adding a startup option

c-lightning plugins allow you to specify command line options that you can set
when you startup lightningd.

For example, here's how you'd set the `name` option

```
$ ./lightningd --network=testnet --name=Ginger
```

Here's how to register an option with the `golight` plugin.

```
// The last value is the default value. This option will default to 'Mary' 
// if not set.
option := golight.NewOption("name", "How you'd like to be called", "Mary")
plugin.RegisterOption(option)

```

### Creating a new RpcMethod

`RpcMethods` are `jrpc2.ServerMethod`s with a few extra fields. These fields are
added when you create the new RpcMethod via the golight helper. Here's an example:

```
// Set up the struct for the RPC method you'd like to add to c-lightning
type Hello struct {
	// This struct has no params
}

func (h *Hello) New() interface{} {
	return &Hello{}
}

// The command line invocation of this command will be 
// ./cli/lightning-cli say-hi
func (h *Hello) Name() string {
	return "say-hi"
}

// This is what gets run when you call the command.
func (h *Hello) Call() (jrpc2.Result, error) {
	// Here we're using an option value in the result
	name := plugin.GetOptionValue("name")
	return fmt.Sprintf("Howdy %s!", name), nil
}
```

Once your function has been defined via a struct, you need to register it with 
the plugin. You'll also need to provide a description and an optional 'long 
description', with more details on how the command can be used. These details
will be shown when you call the c-lightning `help` command.

```
plugin := golight.NewPlugin(initfn)

// The second parameter here is the short command description
rpcHello := golight.NewRpcMethod(&Hello{}, "Say hello!")
rpcHello.LongDesc = "Say hello! To whom you'll be greeting is set by the 'name' options, passed in at startup."
plugin.RegisterMethod(rpcHello)
```


### Beta: Subscribing to a notification stream

Subscriptions are currently in beta, but they will allow your plugin to receive
a notification every time an event matching the notification type occurs
within c-lightning. The two beta notifications are `connect` and `disconnect`.

`golight` provides a beta harness for the provided subscriptions. This requires
creating a new struct that includes the 'harness' for the subscription you want 
to provide a call for. By way of example, here's how you'd create a `connect`
callback and register it with the plugin.

```
type Connect struct {
	golight.ConnectSubscription
}

func (c *Connect) Call() (jrpc2.Result, error) {
	log.Printf("Peer %s connected", c.PeerId)
	return nil, nil
}

func main() {
	plugin := golight.NewPlugin(initfn)
	plugin.Subscribe(&Connect{})
}

```



### Callback from Init

After your plugin's manifest has been parsed by c-lightning, c-lightning will call your plugin's Init method. `golight` registers this for you automatically. You need to supply the `NewPlugin` method with a callback function that will trigger once the plugin has been initialized.

The init function has the following signature:

```
func onInit(plugin *golight.Plugin, options map[string]string, config *golight.Config)
```

### Wiring into Lightning's RPC

The `init` command call will pass back to your plugin a Config object, which includes a lightning-rpc filename and the lightning directory. You can pass this
information directly into `golight`'s `Lightning.StartUp` to create a working
RPC connection. e.g.

```
go lightning.StartUp(config.RpcFile, config.LightningDir)
```

After the connection has been opened, `lightning.IsUp()` will flip to `true`.
You can make any calls provided on the Lightning RPC then. 

```
	lightning.StartUp(config.RpcFile, config.LightningDir)
	channels, _ := lightning.ListChannels()
	log.Printf("You know about %d channels", len(channels))
```

## Logging as a c-lightning Plugin
The c-lightning plugin subsystem uses stdin and stdout as its communication pipes. As most logging would interfere with normal operation of the plugin `golight` overrides the `log` package to pipe all logs to c-lightning. When developing a plugin, it is best practice to use the `log` library write all print statements, so as not to interfere with normal operation of the plugin.


## Work in Progress
Please note that `golight` is currently a work in progress. Although most of the RPC commands are provided, they have *not* all been tested yet. You will probably run into bugs when attempting to use them. They're in the process of being tested.

Futher, the API, provided as is and is subject to revision without warning.

The author hereby acknowledges the boilerplate-y nature of the method definitions, as currently provided.

### Lightning RPC

The lightning RPC functionality is currently untested, except for the following commands. All others are provided as is (and will be tested ... soon).

- listchannels  
- listpeers  

The following RPC non-dev methods need to be added

- delexpiredinvoice  
- autocleaninvoice  

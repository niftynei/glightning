# glightning: a c-lightning plugin driver and RPC client

[![CircleCI](https://circleci.com/gh/niftynei/glightning.svg?style=svg)](https://circleci.com/gh/niftynei/glightning)

glightning is a driver for the Lightning Network protocol implemenation [c-lightning](https://github.com/ElementsProject/lightning).

It offers an RPC client for calling lightning commands and a Plugin infrastructure, for creating your own c-lightning commands and registering for subscriptions.

More details on c-lightning plugins can be found in the [c-lightning docs](https://github.com/ElementsProject/lightning/blob/master/doc/PLUGINS.md)


## Plugins: How to Use
`glightning` builds upon the method paradigm established in [`jrpc2`](jrpc2/README.md). Options, RpcMethods, and Subscriptions all must be registered on the plugin prior to start in order to be included in your manifest. 

RpcMethods and Subscriptons are both a form of `jrpc2.Method`. 

### Adding a startup option

c-lightning plugins allow you to specify command line options that you can set
when you startup lightningd.

For example, here's how you'd set the `name` option

```
$ ./lightningd --network=testnet --name=Ginger
```

Here's how to register an option with the `glightning` plugin.

```
// The last value is the default value. This option will default to 'Mary' 
// if not set.
option := glightning.NewOption("name", "How you'd like to be called", "Mary")
plugin.RegisterOption(option)

```

### Creating a new RpcMethod

`RpcMethods` are `jrpc2.ServerMethod`s with a few extra fields. These fields are
added when you create the new RpcMethod via the glightning helper. Here's an example:

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
plugin := glightning.NewPlugin(initfn)

// The second parameter here is the short command description
rpcHello := glightning.NewRpcMethod(&Hello{}, "Say hello!")
rpcHello.LongDesc = "Say hello! To whom you'll be greeting is set by the 'name' options, passed in at startup."
plugin.RegisterMethod(rpcHello)
```


### Subscribing to a notification stream

Subscriptions allow your plugin to receive a notification every time an 
event matching the notification type occurs within c-lightning. 

The two currently supported notifications are `connect` and `disconnect`.

By way of example, here's how you'd create a `connect` callback and 
register it with the plugin.

```
func OnConnect(e *glightning.ConnectEvent) {
    log.Printf("Connected to %s\n", e.PeerId)
}

func main() {
	plugin := glightning.NewPlugin(initfn)
	plugin.SubscribeConnect(OnConnect)
}

```


### Callback from Init

After your plugin's manifest has been parsed by c-lightning, c-lightning will call your plugin's Init method. `glightning` registers this for you automatically. You need to supply the `NewPlugin` method with a callback function that will trigger once the plugin has been initialized.

The init function has the following signature:

```
func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config)
```

### Wiring into Lightning's RPC

The `init` command call will pass back to your plugin a Config object, which includes a lightning-rpc filename and the lightning directory. You can pass this
information directly into `glightning`'s `Lightning.StartUp` to create a working
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

### Dynamic plugin loading and unloading

Plugins can be configured to be dynamically controlled through the CLI/RPC.  By default a plugin loaded at startup will be stoppable.  This behavior can be overridden by calling the plugin's `SetDynamic` command.

```
func main() {
	plugin := glightning.NewPlugin(initfn)
	plugin.SetDynamic(false)
}
```

will disable management with the [plugin control](https://github.com/ElementsProject/lightning/blob/master/doc/lightning-plugin.7.txt) feature.


## Logging as a c-lightning Plugin
The c-lightning plugin subsystem uses stdin and stdout as its communication pipes. As most logging would interfere with normal operation of the plugin `glightning` overrides the `log` package to pipe all logs to c-lightning. When developing a plugin, it is best practice to use the `log` library write all print statements, so as not to interfere with normal operation of the plugin.

You can override this by providing a logfile to write to via the environment variable `GOLIGHT_DEBUG_LOGFILE`. See [plugin debugging](#plugin_debugging).


### Plugin Debugging

`glightning` provides a few environment variables to help with debugging.

`GOLIGHT_DEBUG_LOGFILE`: If set, will log to the file named in this variable. Otherwise, sends logs back to c-lightning to be added to its internal log buffer.

`GOLIGHT_DEBUG_IO`: Logs all json messages sent and received from c-lightning. Must be used in conjunction with `GOLIGHT_DEBUG_LOGFILE` to avoid creating a log loop.

Example usage: 

```
$ GOLIGHT_DEBUG_IO=1 GOLIGHT_DEBUG_LOGFILE=plugin.log lightupfg --plugin=/path/to/plugin/exec
```


## Work in Progress
Please note that `glightning` is currently a work in progress. 

Futher, the API, provided as is and is subject to revision without warning.

The author hereby acknowledges the boilerplate-y nature of the method definitions, as currently provided.

### Lightning RPC

The following RPC non-dev methods need to be added

- delexpiredinvoice  
- autocleaninvoice  

Note that most of the 'dev' commands aren't well tested, and that many of them require you to set various flags at compile or configuration time (of lightningd) in order to use them. You'll need to at least have configured your c-lightning build into developer mode, ie:

```
cwd/lightning$ ./configure --enable-developer
```


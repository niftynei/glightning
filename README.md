# glightning: a c-lightning plugin driver and RPC client

[![CircleCI](https://circleci.com/gh/niftynei/glightning.svg?style=svg)](https://circleci.com/gh/niftynei/glightning)

glightning is a driver for the Lightning Daemon [c-lightning](https://github.com/ElementsProject/lightning).

It offers an RPC client for calling lightning commands and a framework for writing 
Go-native [plugins](https://github.com/ElementsProject/lightning/blob/master/doc/PLUGINS.md)


## Plugins: How to Use
For a complete example of the Hooks, Options, Subscriptions and Methods see the examples in [examples/plugin](examples/plugin/plugin_example.go)

Below is a quick primer on each of these options.


### Options

c-lightning plugins allow you to specify command line options that you can set
when you startup lightningd.

For example, here's how you'd set the `name` option

```
$ ./lightningd --network=testnet --name=Ginger
```

Here's how to register an option with the `glightning` plugin.

```
// The last value is the default value, e.g. this option will default to 'Mary' 
// if not set.
option := glightning.NewOption("name", "How you'd like to be called", "Mary")
plugin.RegisterOption(option)

```

### Creating a new Method

You can create your own RPC methods to add additional functionality to a clightning node by registering
new methods!

Each method definition requires a struct that implements the jrpc2.ServerMethod interface.
There are three method on this interface: 

   - Name, which returns the callable name of the function, 
   - New, which returns a new copy of this method struct, and
   - Call, which executes when this method is called.


Here's a quick example

```
// Set up the struct for the RPC method you'd like to add to c-lightning
type Hello struct {
	// This method takes no parameters, so there's no fields here
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
	// Here we're using an startup option value in the result
	name := plugin.GetOptionValue("name")
	return fmt.Sprintf("Howdy %s!", name), nil
}
```

A method should be registered so c-lightning knows about it. These are 
defined before starting up the plugin.

```
plugin := glightning.NewPlugin(initfn)

// The second parameter here is the short command description
rpcHello := glightning.NewRpcMethod(&Hello{}, "Say hello!")
rpcHello.LongDesc = "Say hello! To whom you'll be greeting is set by the 'name' options, passed in at startup."
plugin.RegisterMethod(rpcHello)
```


### Subscriptions

Subscriptions allow your plugin to receive a notification every time an 
event matching the notification type occurs within c-lightning. 

Here's how you'd create a `connect` callback and register it with 
the plugin.

    func OnConnect(e *glightning.ConnectEvent) {
        log.Printf("Connected to %s\n", e.PeerId)
    }
    
    func main() {
    	plugin := glightning.NewPlugin(initfn)
	plugin.SubscribeConnect(OnConnect)
    }


### Initializing a Plugin

c-lightning will call your plugin's Init method when it's started. `glightning` registers this for you automatically. 
You need to supply the `NewPlugin` method with a callback function that will trigger once the plugin has been initialized.

The init function has the following signature:

```
func onInit(plugin *glightning.Plugin, options map[string]glightning.Option, config *glightning.Config)
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

Plugins can be configured to be dynamically controlled through the CLI/RPC.  
By default a plugin loaded at startup will be stoppable.  This behavior can be 
overridden by calling the plugin's `SetDynamic` command.

```
func main() {
	plugin := glightning.NewPlugin(initfn)
	plugin.SetDynamic(false)
}
```

will disable management with the [plugin control](https://github.com/ElementsProject/lightning/blob/master/doc/lightning-plugin.7.txt) feature.


## Logging as a c-lightning Plugin

The c-lightning plugin subsystem uses stdin and stdout as its communication pipes. As most logging would 
interfere with normal operation of the plugin `glightning` overrides the `log` package to pipe all 
logs to c-lightning. When developing a plugin, it is best practice to use the `log` library write 
all print statements, so as not to interfere with normal operation of the plugin.

You can override this by providing a logfile to write to via the environment variable 
`GOLIGHT_DEBUG_LOGFILE`. See [plugin debugging](#plugin_debugging).


### Plugin Debugging

`glightning` provides a few environment variables to help with debugging.

`GOLIGHT_DEBUG_LOGFILE`: If set, will log to the file named in this variable. Otherwise, sends logs back to c-lightning to be added to its internal log buffer.

`GOLIGHT_DEBUG_IO_IN`: Logs any incoming IO messages. Useful if you want to log incoming messages without specifying a debug log file.

`GOLIGHT_DEBUG_IO`: Logs all json messages sent and received from c-lightning. Must be used in conjunction with `GOLIGHT_DEBUG_LOGFILE` to avoid creating a log loop.


Example usage: 

```
$ GOLIGHT_DEBUG_IO=1 GOLIGHT_DEBUG_LOGFILE=plugin.log lightningd --plugin=/path/to/plugin/exec --daemon 
```


#### Using Strict Mode

glightning is meant to be used with the corresponding c-lightning version. Updates to the 
c-lightning RPC API are reflected in each new glightning version.

Support for older versions of lightningd is coming soon TM

The default configuration of glightning is such that it works `allow-deprecated-apis=true`, which is the 
default setting on c-lightning; however if you wish to run it 'strict mode', such that glightning 
will fail if an RPC response includes unexpected parameters, you should set the environment 
variable`GOLIGHT_STRICT_MODE=1` and the c-lightning startup flag `allow-deprecated-apis=false`.


### Dev Commands

Note that most of the 'dev' commands aren't well tested, and that many of them require you to set 
various flags at compile or configuration time (of lightningd) in order to use them. You'll 
need to at least have configured your c-lightning build into developer mode, ie:

```
cwd/lightning$ ./configure --enable-developer && make
```

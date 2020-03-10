# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Note that versioning matches clightning's to signal interoperablility.
Any bugfixes will increment the version at the 4th place eg. a bugfix release
for 0.8.0 will be noted as 0.8.0.1, etc

All additions from the clightning CHANGELOG also apply, this just documents 

## [0.8.1]
- glightning: includes a 'bitcoin backend' helper for writing swappable bitcoin sources for 
  clightning. note that the API on this is currently in flux. there's a complete
  re-implementation of the packaged bitcoind backend that clightning ships with, `bcli`,
  included as an example in examples/plugin/btc/plugin_btc.go
- glightning: SatoshiAmount has been renamed 'Sat'
- glightning: NewAmount() has been renamed 'NewSat'
- gbitcoin: now prints IO logs when GOLIGHT_DEBUG_IO flagged on
- gbitcoin: the following RPC calls have been added
   - getblockchaininfo
   - getblockhash
   - getblock (raw/verbose `0` mode only)
   - estimatesmartfee
   - gettxout
- glightning: Peer.Features are now a 'Hexed' obj (string->\*(Hexed))
- glightning: Node.Features are now a 'Hexed' obj (string->\*(Hexed))
- glightning: GetNode returns a \*Node, not list of nodes ([]Node -> \*Node)
- glightning: new method CreateInvoiceExposing, accepts list of short channel ids
- glightning: new method Invoice, shorthand for getting an invoice
- glightning: new method WaitAnyInvoiceTimeout
- glightning: DecodedBolt new field 'Features'
- glightning: new method FundPrivateChannel
- glightning: new method FundChannelAtFee (convenient for providing a feerate)
- glightning: new method FundPrivateChannelAtFee (convenient for providing a feerate)
- glightning: method FundChannelExt now also requires a pushMsat amount
- glightning: Withdraw and WithdrawWithUtxos were affected by the SatoshiAmount->Sat renaming
- glightning: StopPlugin now returns a string, not a list of PluginInfo ([]PluginInfo -> string)
- glightning: DbWriteEvent hook object now includes a field `DataVersion`
- glightning: DbWrite's hook now requires a \*DbWriteResponse (bool -> \*DbWriteResponse)
- glightning: DbWriteEvent now provides two methods 'Continue' and 'Fail'
- glightning: new method Plugin.AddInitFeatures
- glightning: new method Plugin.AddNodeFeatures
- glightning: new method Plugin.AddInvoiceFeatures
- glightning: new method Plugin.RegisterNewOption
- glightning: new type MSat, for representing millisatoshi amounts. rpc results will begin to natively
  parse results as this type in a future release.
- fixed::glightning: allowhigh fees parses correctly
- jrpc2: actually parses hexstrings correctly into []bytes (a hexstring is assumed if 
  the data is a string and the destination field is a []byte)


## [0.8.0] 
- lightning: SatPerKiloByte,SatsPerKiloWeight updated to PerKb/PerKw respectively
- jrpc2: parsers now properly handles []byte and json.RawMessage fields in method objects
- bitcoin: now includes a very incomplete Bitcoin RPC client. endpoints implemented: 
   - ping
   - getnewaddress
   - generatetoaddress
   - sendtoaddress
   - createrawtransaction
   - fundrawtranaction
   - sendrawtransaction
   - decoderawtransaction
- lightning: GetPeer now only requires a peerid, log parameter removed to `GetPeerWithLogs`
- lightning: GetChannel now returns an array of pointers ([]Channel -> []\*Channel)
- lightning: GetChannelsBySource now returns an array of pointers ([]Channel -> []\*Channel)
- lightning: ListChannels now returns an array of pointers ([]Channel -> []\*Channel)
- lightning: ListInvoices now returns an array of pointers ([]Invoice -> []\*Invoice)
- lightning: GetInvoice now returns a invoice pointer ([]Invoice -> \*Invoice)
- lightning: WaitAnyInvoice now returns Invoice; CompletedInvoice has been removed
- lightning: WaitInvoice now returns Invoice; CompletedInvoice has been removed
- lightning: SendPay now also needs a paymentSecret and partId
- lightning: NEW WaitSendPayPart method
- lightning: the `txout` parameter on CompleteFundChannel is now uint32 (uint16->uint32)
- lightning: new convenience method for getting a SatoshiAmount `NewAmt` (takes a uint64)


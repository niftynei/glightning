# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Note that versioning matches clightning's to signal interoperablility.
Any bugfixes will increment the version at the 4th place eg. a bugfix release
for 0.8.0 will be noted as 0.8.0.1, etc

All additions from the clightning CHANGELOG also apply, this just documents 

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


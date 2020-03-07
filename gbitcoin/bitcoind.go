package gbitcoin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/niftynei/glightning/jrpc2"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

// taken from bitcoind
const defaultClientTimeout int = 900
const defaultRpcHost string = "http://localhost"

const debug bool = false

func isDebug() bool {
	return debug
}

type Bitcoin struct {
	isUp           bool
	httpClient     *http.Client
	port           uint
	host           string
	bitcoinDir     string
	requestCounter int64
	username       string
	password       string
}

func NewBitcoin(username, password string) *Bitcoin {
	bt := &Bitcoin{}

	tr := &http.Transport{
		MaxIdleConns:    20,
		IdleConnTimeout: time.Duration(defaultClientTimeout) * time.Second,
	}
	bt.httpClient = &http.Client{Transport: tr}
	bt.username = username
	bt.password = password
	return bt
}

func (b *Bitcoin) Endpoint() string {
	return b.host + ":" + strconv.Itoa(int(b.port))
}

func (b *Bitcoin) SetTimeout(secs uint) {
	tr := &http.Transport{
		MaxIdleConns:    20,
		IdleConnTimeout: time.Duration(secs) * time.Second,
	}
	b.httpClient = &http.Client{Transport: tr}
}

func (b *Bitcoin) StartUp(host, bitcoinDir string, port uint) {
	if host == "" {
		b.host = defaultRpcHost
	} else {
		b.host = host
	}

	b.port = port
	b.bitcoinDir = bitcoinDir

	for {
		up, err := b.Ping()
		if up {
			break
		}
		if isDebug() {
			log.Print(err)
		}
	}
}

// Blocking!
func (b *Bitcoin) request(m jrpc2.Method, resp interface{}) error {

	id := b.NextId()
	mr := &jrpc2.Request{id, m}
	jbytes, err := json.Marshal(mr)
	if err != nil {
		return err
	}

	if _, ok := os.LookupEnv("GOLIGHT_DEBUG_IO"); ok {
		log.Println(string(jbytes))
	}

	req, err := http.NewRequest("POST", b.Endpoint(), bytes.NewBuffer(jbytes))
	if err != nil {
		return err
	}

	req.Header.Set("Host", b.host)
	req.Header.Set("Connection", "close")
	req.SetBasicAuth(b.username, b.password)
	req.Header.Set("Content-Type", "application/json")

	rezp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer rezp.Body.Close()

	switch rezp.StatusCode {
	case http.StatusUnauthorized:
		return errors.New("Authorization failed: Incorrect user or password")
	case http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError:
		// do nothing
	default:
		if rezp.StatusCode > http.StatusBadRequest {
			return errors.New(fmt.Sprintf("server returned HTTP error %d", rezp.StatusCode))
		} else if rezp.ContentLength == 0 {
			return errors.New("no response from server")
		}
	}

	var rawResp jrpc2.RawResponse
	if _, ok := os.LookupEnv("GOLIGHT_DEBUG_IO"); ok {
		data, err := ioutil.ReadAll(rezp.Body)
		if err != nil {
			log.Printf("err response, %s", err)
			return err
		}
		log.Println(string(data))
		err = json.Unmarshal(data, &rawResp)
	} else {
		decoder := json.NewDecoder(rezp.Body)
		err = decoder.Decode(&rawResp)
	}

	if err != nil {
		return err
	}

	if rawResp.Error != nil {
		return rawResp.Error
	}

	return json.Unmarshal(rawResp.Raw, resp)
}

type PingRequest struct{}

func (r *PingRequest) Name() string {
	return "ping"
}

func (b *Bitcoin) Ping() (bool, error) {
	var result string
	err := b.request(&PingRequest{}, &result)
	return err == nil, err
}

type GetNewAddressRequest struct {
	Label       string `json:"label,omitempty"`
	AddressType string `json:"address_type,omitempty"`
}

type AddrType int

const (
	Bech32 AddrType = iota
	P2shSegwit
	Legacy
)

func (a AddrType) String() string {
	return []string{"bech32", "p2sh-segwit", "legacy"}[a]
}

func (r *GetNewAddressRequest) Name() string {
	return "getnewaddress"
}

func (b *Bitcoin) GetNewAddress(addrType AddrType) (string, error) {
	var result string
	err := b.request(&GetNewAddressRequest{
		AddressType: addrType.String(),
	}, &result)
	return result, err
}

type GenerateToAddrRequest struct {
	NumBlocks uint   `json:"nblocks"`
	Address   string `json:"address"`
	MaxTries  uint   `json:"maxtries,omitempty"`
}

func (r *GenerateToAddrRequest) Name() string {
	return "generatetoaddress"
}

func (b *Bitcoin) GenerateToAddress(address string, numBlocks uint) ([]string, error) {
	var resp []string
	err := b.request(&GenerateToAddrRequest{
		NumBlocks: numBlocks,
		Address:   address,
	}, &resp)
	return resp, err
}

type SendToAddrReq struct {
	Address               string `json:"address"`
	Amount                string `json:"amount"`
	Comment               string `json:"comment,omitempty"`
	CommentTo             string `json:"comment_to,omitempty"`
	SubtractFeeFromAmount bool   `json:"subtractfeefromamount,omitempty"`
	Replaceable           bool   `json:"replaceable,omitempty"`
	ConfirmationTarget    uint   `json:"conf_target,omitempty"`
	FeeEstimateMode       string `json:"estimate_mode,omitempty"`
}

func (r *SendToAddrReq) Name() string {
	return "sendtoaddress"
}

func (b *Bitcoin) SendToAddress(address, amount string) (string, error) {
	var result string
	err := b.request(&SendToAddrReq{
		Address: address,
		Amount:  amount,
	}, &result)
	return result, err
}

type TxIn struct {
	TxId     string `json:"txid"`
	Vout     uint   `json:"vout"`
	Sequence uint   `json:"sequence,omitempty"`
}

type TxOut struct {
	Address string
	Satoshi uint64
}

func (o *TxOut) Marshal() []byte {
	// we need to convert the satoshi into bitcoin
	// FIXME: check uint64 to float64 conversion
	amt := float64(o.Satoshi) / math.Pow(10, 8)
	log.Printf(`{"%s":"%f"`, o.Address, amt)
	return []byte(fmt.Sprintf(`{"%s":"%f"}`, o.Address, amt))
}

// Because we're using a weird JSON marshaler for parameter packing
// we encode the outputs before passing them along as a request (instead
// of writing a custom json Marshaler)
func stringifyOutputs(outs []*TxOut) []json.RawMessage {
	results := make([]json.RawMessage, len(outs))

	for i := 0; i < len(outs); i++ {
		results[i] = json.RawMessage(outs[i].Marshal())
	}

	return results
}

type CreateRawTransactionReq struct {
	Ins         []*TxIn           `json:"inputs"`
	Outs        []json.RawMessage `json:"outputs"`
	Locktime    *uint32           `json:"locktime,omitempty"`
	Replaceable *bool             `json:"replaceable,omitempty"`
}

func (r *CreateRawTransactionReq) Name() string {
	return "createrawtransaction"
}

func (b *Bitcoin) CreateRawTx(ins []*TxIn, outs []*TxOut, locktime *uint32, replaceable *bool) (string, error) {
	if len(outs) == 0 {
		return "", errors.New("Must provide at least one output")
	}

	// bitcoind requires at least an empty array
	if ins == nil {
		ins = make([]*TxIn, 0)
	}
	request := &CreateRawTransactionReq{
		Ins:         ins,
		Outs:        stringifyOutputs(outs),
		Locktime:    locktime,
		Replaceable: replaceable,
	}

	var resp string
	err := b.request(request, &resp)
	return resp, err
}

type FundRawOptions struct {
	ChangeAddress   string `json:"changeAddress,omitempty"`
	ChangePosition  *uint  `json:"changePosition,omitempty"`
	ChangeType      string `json:"change_type,omitempty"`
	IncludeWatching *bool  `json:"includeWatching,omitempty"`
	LockUnspents    *bool  `json:"lockUnspents,omitempty"`
	FeeRate         string `json:"feeRate,omitempty"`
	// The fee will be equally deducted from the amount of each specified output.
	// Those recipients will receive less bitcoins than you enter in their
	//   corresponding amount field.
	// If no outputs are specified here, the sender pays the fee.
	// array values: The zero-based output index to deduct fee from,
	//   before a change output is added.
	SubtractFeeFromOutputs []uint `json:"subtractFeeFromOutputs,omitempty"`
	Replaceable            *bool  `json:"replaceable,omitempty"`
	ConfirmationTarget     uint   `json:"conf_target,omitempty"`
	EstimateMode           string `json:"estimate_mode,omitempty"`
}

type FundRawTransactionReq struct {
	TxString  string          `json:"hexstring"`
	Options   *FundRawOptions `json:"options,omitempty"`
	IsWitness *bool           `json:"iswitness,omitempty"`
}

func (r *FundRawTransactionReq) Name() string {
	return "fundrawtransaction"
}

type FundRawResult struct {
	TxString string  `json:"hex"`
	Fee      float64 `json:"fee"`
	// Position of the added change output, or -1
	ChangePosition int `json:"chanepos"`
}

func (f *FundRawResult) HasChange() bool {
	return f.ChangePosition != -1
}

// Defaults to a segwit transaction
func (b *Bitcoin) FundRawTx(txstring string) (*FundRawResult, error) {
	return b.FundRawWithOptions(txstring, nil, nil)
}

func (b *Bitcoin) FundRawWithOptions(txstring string, options *FundRawOptions, iswitness *bool) (*FundRawResult, error) {
	var resp FundRawResult
	err := b.request(&FundRawTransactionReq{
		TxString:  txstring,
		Options:   options,
		IsWitness: iswitness,
	}, &resp)
	return &resp, err
}

type SendRawTransactionReq struct {
	TxString      string `json:"hexstring"`
	AllowHighFees *bool   `json:"allowhighfees,omitempty"`
}

func (r *SendRawTransactionReq) Name() string {
	return "sendrawtransaction"
}

func (b *Bitcoin) SendRawTx(txstring string) (string, error) {
	var result string
	err := b.request(&SendRawTransactionReq{
		TxString:      txstring,
	}, &result)
	return result, err
}

type DecodeRawTransactionReq struct {
	TxString  string `json:"hexstring"`
	IsWitness *bool  `json:"iswitness,omitempty"`
}

func (r *DecodeRawTransactionReq) Name() string {
	return "decoderawtransaction"
}

type Tx struct {
	TxId        string      `json:"txid"`
	Hash        string      `json:"hash"`
	Size        uint        `json:"size"`
	VirtualSize uint        `json:"vsize"`
	Weight      uint        `json:"weight"`
	Version     uint        `json:"version"`
	Locktime    uint32      `json:"locktime"`
	Inputs      []*TxInput  `json:"vin"`
	Outputs     []*TxOutput `json:"vout"`
}

type TxInput struct {
	TxId            string   `json:"txid"`
	Vout            uint     `json:"vout"`
	ScriptSignature *Script  `json:"scriptSig"`
	TxInWitness     []string `json:"txinwitness,omitempty"`
	Sequence        uint     `json:"sequence"`
}

type Script struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

type TxOutput struct {
	// The value in BTC
	Value        float64    `json:"value"`
	Index        uint       `json:"n"`
	ScriptPubKey *OutScript `json:"scriptPubKey"`
}

type OutScript struct {
	Script
	RequiredSigs uint     `json:"reqSigs"`
	Type         string   `json:"type"`
	Addresses    []string `json:"addresses"`
}

func (tx *Tx) FindOutputIndex(address string) (uint32, error) {
	for i := 0; i < len(tx.Outputs); i++ {
		out := tx.Outputs[i]
		if out.ScriptPubKey == nil {
			continue
		}
		for j := 0; j < len(out.ScriptPubKey.Addresses); j++ {
			if out.ScriptPubKey.Addresses[j] == address {
				return uint32(i), nil
			}
		}
	}

	return 0, errors.New(fmt.Sprintf("%s not found", address))
}

func (b *Bitcoin) DecodeRawTx(txstring string) (*Tx, error) {
	var resp Tx
	err := b.request(&DecodeRawTransactionReq{
		TxString: txstring,
	}, &resp)

	return &resp, err
}

// for now, use a counter as the id for requests
func (b *Bitcoin) NextId() *jrpc2.Id {
	val := atomic.AddInt64(&b.requestCounter, 1)
	return jrpc2.NewIdAsInt(val)
}

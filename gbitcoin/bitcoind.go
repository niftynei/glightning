package gbitcoin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/niftynei/glightning/jrpc2"
	"log"
	"net/http"
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
	isUp bool
	httpClient *http.Client
	port uint
	host string
	bitcoinDir string
	requestCounter int64
	username string
	password string
}

func NewBitcoin(username, password string) *Bitcoin {
	bt := &Bitcoin{}

	tr := &http.Transport{
		MaxIdleConns: 20,
		IdleConnTimeout: time.Duration(defaultClientTimeout) * time.Second,
	}
	bt.httpClient = &http.Client{ Transport: tr }
	bt.username = username
	bt.password = password
	return bt
}

func (b *Bitcoin) Endpoint() string {
	return b.host + ":" + strconv.Itoa(int(b.port))
}

func (b *Bitcoin) SetTimeout(secs uint) {
	tr := &http.Transport{
		MaxIdleConns: 20,
		IdleConnTimeout: time.Duration(secs) * time.Second,
	}
	b.httpClient = &http.Client{ Transport: tr }
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
			break;
		}
		if isDebug() {
			log.Print(err)
		}
	}
}

// Blocking!
func (b *Bitcoin) request(m jrpc2.Method, resp interface{}) error {

	id := b.NextId()
	mr := &jrpc2.Request{ id, m }
	jbytes, err := json.Marshal(mr)
	if err != nil {
		return err
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

	switch (rezp.StatusCode) {
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
	decoder := json.NewDecoder(rezp.Body)
	err = decoder.Decode(&rawResp)
	if err != nil {
		return err
	}

	if rawResp.Error != nil {
		return rawResp.Error
	}

	return json.Unmarshal(rawResp.Raw, resp)
}

type PingRequest struct {}

func (r *PingRequest) Name() string {
	return "ping"
}

func (b *Bitcoin) Ping() (bool, error) {
	var result string
	err := b.request(&PingRequest{}, &result)
	return err == nil, err
}

type GetNewAddressRequest struct {
	Label string `json:"label,omitempty"`
	AddressType string `json:"address_type,omitempty"`
}

type AddrType int

const (
	Bech32 AddrType = iota
	P2shSegwit
	Legacy
)

func (a AddrType) String() string {
	return []string{"bech32","p2sh-segwit","legacy"}[a]
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
	NumBlocks uint `json:"nblocks"`
	Address string `json:"address"`
	MaxTries uint `json:"maxtries,omitempty"`
}

func (r *GenerateToAddrRequest) Name() string {
	return "generatetoaddress"
}

func (b *Bitcoin) GenerateToAddress(address string, numBlocks uint) ([]string, error) {
	var resp []string
	err := b.request(&GenerateToAddrRequest {
		NumBlocks: numBlocks,
		Address: address,
	}, &resp)
	return resp, err
}

type SendToAddrReq struct {
	Address string `json:"address"`
	Amount string `json:"amount"`
	Comment string `json:"comment,omitempty"`
	CommentTo string `json:"comment_to,omitempty"`
	SubtractFeeFromAmount bool `json:"subtractfeefromamount,omitempty"`
	Replaceable bool `json:"replaceable,omitempty"`
	ConfirmationTarget uint `json:"conf_target,omitempty"`
	FeeEstimateMode string `json:"estimate_mode,omitempty"`
}

func (r *SendToAddrReq) Name() string {
	return "sendtoaddress"
}

func (b *Bitcoin) SendToAddress(address, amount string) (string, error) {
	var result string
	err := b.request(&SendToAddrReq{
		Address: address,
		Amount: amount,
	}, &result)
	return result, err
}

// for now, use a counter as the id for requests
func (b *Bitcoin) NextId() *jrpc2.Id {
	val := atomic.AddInt64(&b.requestCounter, 1)
	return jrpc2.NewIdAsInt(val)
}

package jrpc2_test

import (
	"encoding/json"
	"fmt"
	"github.com/niftynei/glightning/jrpc2"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

//// This section (below) is for method json marshalling,
// with special emphasis on how the parameters get marshalled
// and unmarshalled to/from 'Method' objects
type HelloMethod struct {
	First  int64 `json:"first"`
	Second int64 `json:"second"`
}

type HelloResult struct {
	Result int64
}

func (hm HelloMethod) New() interface{} {
	return &HelloMethod{}
}

func (hm HelloMethod) Call() (jrpc2.Result, error) {
	return &HelloResult{hm.First + hm.Second}, nil
}

func (hm HelloMethod) Name() string {
	return "hello"
}

type fun func(string, int64)

func TestJsonId(t *testing.T) {
	s := jrpc2.NewServer()
	s.Register(&EmptyMethod{})

	jsonNullId := `{"id":null,"jsonrpc":"2.0","method":"empty"}`
	var req jrpc2.Request
	err := s.Unmarshal([]byte(jsonNullId), &req)
	assert.Nil(t, err)
	assert.Nil(t, req.Id)

	jsonIntId := `{"id":123409398493,"jsonrpc":"2.0","method":"empty"}`
	errOne := s.Unmarshal([]byte(jsonIntId), &req)
	assert.Nil(t, errOne)
	assert.Equal(t, "123409398493", req.Id.Val())

	jsonStrId := `{"id":"akak","jsonrpc":"2.0","method":"empty"}`
	errTwo := s.Unmarshal([]byte(jsonStrId), &req)
	assert.Nil(t, errTwo)
	assert.Equal(t, "akak", req.Id.Val())

	jsonInvalidStr := `{"id":"akak,"jsonrpc":"2.0","method":"empty"}`
	errInvalid := s.Unmarshal([]byte(jsonInvalidStr), &req)
	assert.NotNil(t, errInvalid)

	jsonFloatId := `{"id":193.392,"jsonrpc":"2.0","method":"empty"}`
	errTree := s.Unmarshal([]byte(jsonFloatId), &req)
	assert.NotNil(t, errTree)

	jsonObjId := `{"id":{"method":"empty"},"jsonrpc":"2.0","method":"empty"}`
	errFour := s.Unmarshal([]byte(jsonObjId), &req)
	assert.NotNil(t, errFour)

	jsonArrId := `{"id":[1,2,3],"jsonrpc":2.0",name":"empty"}`
	errFive := s.Unmarshal([]byte(jsonArrId), &req)
	assert.NotNil(t, errFive)
}

type EmptyMethod struct{}

type EmptyResult struct{}

func (hm EmptyMethod) New() interface{} {
	return &EmptyMethod{}
}

func (hm EmptyMethod) Name() string {
	return "empty"
}

func (e EmptyMethod) Call() (jrpc2.Result, error) {
	return "", nil
}

func TestParamParsing(t *testing.T) {
	requestJsonObjParams := `{"jsonrpc":"2.0","method":"hello","params":{"first":2,"second":3},"id":123493}`
	requestJsonArrParams := `{"id":null,"params":[2,3],"jsonrpc":"2.0","method":"hello"}`
	requestJsonNoParams := `{"id":123,"jsonrpc":"2.0","method":"empty"}`

	s := jrpc2.NewServer()
	s.Register(&HelloMethod{})

	var req jrpc2.Request
	err := s.Unmarshal([]byte(requestJsonObjParams), &req)
	assert.Nil(t, err)

	hello := req.Method.(*HelloMethod)
	assert.Equal(t, int64(2), hello.First, "First should be set")
	assert.Equal(t, int64(3), hello.Second, "Second should be set")

	// since we're 'hardcoded' to using obj params...
	js, codedErr := json.Marshal(&req)
	assert.Nil(t, codedErr)
	assert.Equal(t, requestJsonObjParams, string(js))

	err = s.Unmarshal([]byte(requestJsonArrParams), &req)
	assert.Nil(t, err)
	hello = req.Method.(*HelloMethod)
	assert.Equal(t, int64(2), hello.First, "First should be set")
	assert.Equal(t, int64(3), hello.Second, "Second should be set")

	s.Register(&EmptyMethod{})
	err = s.Unmarshal([]byte(requestJsonNoParams), &req)
	assert.Nil(t, err)
}

func TestJsonUnmarshal(t *testing.T) {
	requestJson := `{"id":123493,"method":"hello","params":{"first":202,"second":3},"jsonrpc":"2.0"}`
	s := jrpc2.NewServer()
	s.Register(&HelloMethod{})

	var result jrpc2.Request
	err := s.Unmarshal([]byte(requestJson), &result)
	assert.Nil(t, err, "No error returned from unmarshaler")

	assert.Equal(t, "hello", result.Method.Name(), "Method name should be 'hello'")
	ans, _ := result.Method.(jrpc2.ServerMethod).Call()
	assert.Equal(t, int64(205), ans.(*HelloResult).Result, "Hello method should add")
}

func TestSimpleNamedParamParsing(t *testing.T) {
	first := int64(2)
	second := int64(3)
	hm := HelloMethod{first, second}
	params := jrpc2.GetNamedParams(&hm)

	hm2 := &HelloMethod{}
	jrpc2.ParseNamedParams(hm2, params)
	assert.Equal(t, first, hm2.First, "The named param First should be two")
	assert.Equal(t, second, hm2.Second, "The named param Second should be three")
}

type Outer struct {
	Method HelloMethod `json:"method"`
}

func (o Outer) Name() string {
	return "outer"
}

func TestStructNamedParamParsing(t *testing.T) {
	first := int64(2)
	second := int64(3)
	out := &Outer{HelloMethod{first, second}}
	params := jrpc2.GetNamedParams(out)

	outTwo := &Outer{}
	jrpc2.ParseNamedParams(outTwo, params)
	assert.Equal(t, outTwo.Method.First, first, "Outer.Method.First should be set")
	assert.Equal(t, outTwo.Method.Second, second, "Outer.Method.Second should be set")
}

type Inside struct {
	Blah  string `json:"blah"`
	Tadah string `json:"tadah"`
}

type OuterP struct {
	Method *HelloMethod `json:"method"`
	Inline string       `json:"inline"`
	Inside *Inside      `json:"inside"`
}

func (o OuterP) New() interface{} {
	return &OuterP{}
}

func (o OuterP) Name() string {
	return "outer"
}

func (o OuterP) Call() (jrpc2.Result, error) {
	return "", nil
}

func TestPtrsNamedParamParsing(t *testing.T) {
	first := int64(2)
	second := int64(3)
	str := "outero"
	out := &OuterP{&HelloMethod{first, second}, str, &Inside{"hi", "bye"}}
	params := jrpc2.GetNamedParams(out)

	assert.Equal(t, reflect.TypeOf(params["method"]), reflect.TypeOf(&HelloMethod{}))
	assert.Equal(t, first, params["method"].(*HelloMethod).First)

	outJson, err := json.Marshal(&jrpc2.Request{
		Id:     nil,
		Method: *out,
	})
	assert.Nil(t, err, "Problem parsing outer request")

	s := jrpc2.NewServer()
	s.Register(out)
	var req jrpc2.Request
	err = s.Unmarshal(outJson, &req)
	assert.Nil(t, err, "Problem parsing outer request")

	outer := req.Method.(*OuterP)
	assert.Equal(t, first, outer.Method.First, "Outer.Method.First should be set")
	assert.Equal(t, second, outer.Method.Second, "Outer.Method.Second should be set")
	assert.Equal(t, str, outer.Inline, "Outer.Inline should be set")
}

func TestNilPtrInterior(t *testing.T) {
	first := int64(2)
	second := int64(3)
	str := "outero"
	out := &OuterP{&HelloMethod{first, second}, str, &Inside{"hi", "bye"}}

	// set pointer to nil heheh
	out.Method = nil
	outJson, err := json.Marshal(&jrpc2.Request{
		Id:     nil,
		Method: *out,
	})
	assert.Nil(t, err, "Problem parsing outer request")

	s := jrpc2.NewServer()
	s.Register(out)
	var req jrpc2.Request
	err = s.Unmarshal(outJson, &req)
	assert.Nil(t, err, "Problem parsing outer request")

	outer := req.Method.(*OuterP)
	assert.Nil(t, outer.Method)
	assert.Equal(t, str, outer.Inline, "Outer.Inline should be set")
}

type A struct {
	B int64 `json:"b"`
}

type B struct {
	Inner A `json:"inner"`
}

func (o B) New() interface{} {
	return &B{}
}

func (o B) Name() string {
	return "struct"
}

func (o B) Call() (jrpc2.Result, error) {
	return "b", nil
}

func TestStructFilledIn(t *testing.T) {
	four := int64(4)
	a := &A{four}
	ab := &B{*a}
	outJson, err := json.Marshal(&jrpc2.Request{
		Id:     jrpc2.NewId("abc"),
		Method: *ab,
	})
	assert.Nil(t, err, "Problem parsing struct request")

	s := jrpc2.NewServer()
	s.Register(ab)
	var req jrpc2.Request
	err = s.Unmarshal(outJson, &req)
	assert.Nil(t, err, "Problem parsing struct request")

	bc := req.Method.(*B)
	assert.Equal(t, four, bc.Inner.B)
}

type C struct {
	Inner []*A `json:"inner"`
}

func (o C) New() interface{} {
	return &C{}
}

func (o C) Name() string {
	return "slice"
}

func (o C) Call() (jrpc2.Result, error) {
	return "", nil
}

func TestSliceFilled(t *testing.T) {
	c := &C{}
	c.Inner = make([]*A, 3)
	for i := range c.Inner {
		c.Inner[i] = &A{int64(i)}
	}

	outJson, err := json.Marshal(&jrpc2.Request{
		Id:     jrpc2.NewId("ccc"),
		Method: *c,
	})
	assert.Nil(t, err, "Problem parsing slice request")

	s := jrpc2.NewServer()
	s.Register(c)
	var req jrpc2.Request
	err = s.Unmarshal(outJson, &req)
	assert.Nil(t, err, "Problem parsing slice request")

	unC := req.Method.(*C)
	assert.Equal(t, int64(1), unC.Inner[1].B)
}

type D struct {
	Inner map[string]A `json:"inner"`
}

func (o D) New() interface{} {
	return &D{}
}

func (o D) Name() string {
	return "map"
}

func (o D) Call() (jrpc2.Result, error) {
	return "", nil
}

func TestMapFilled(t *testing.T) {
	d := &D{}
	d.Inner = make(map[string]A)
	d.Inner["one"] = A{int64(1)}
	d.Inner["two"] = A{int64(2)}

	outJson, err := json.Marshal(&jrpc2.Request{
		Id:     jrpc2.NewId("dmap"),
		Method: *d,
	})
	assert.Nil(t, err, "Problem marshalling map request")

	s := jrpc2.NewServer()
	s.Register(d)
	var req jrpc2.Request
	err = s.Unmarshal(outJson, &req)
	assert.Nil(t, err, "Problem unmarshalling map request")

	unD := req.Method.(*D)
	assert.Equal(t, int64(1), unD.Inner["one"].B)
}

type E struct {
	Inner map[string]string `json:"inner"`
}

func (o E) New() interface{} {
	return &E{}
}

func (o E) Name() string {
	return "mapprim"
}

func (o E) Call() (jrpc2.Result, error) {
	return "", nil
}

func TestMapFilledPrimitive(t *testing.T) {
	e := &E{}
	e.Inner = make(map[string]string)
	e.Inner["one"] = "one_"
	e.Inner["two"] = "two_"

	outJson, err := json.Marshal(&jrpc2.Request{
		Id:     jrpc2.NewId("dmapsimple"),
		Method: *e,
	})
	assert.Nil(t, err, "Problem marshalling map request")

	s := jrpc2.NewServer()
	s.Register(e)
	var req jrpc2.Request
	err = s.Unmarshal(outJson, &req)
	assert.Nil(t, err, "Problem unmarshalling map request")

	unE := req.Method.(*E)
	assert.Equal(t, "one_", unE.Inner["one"])

}

type Anon struct {
	Value string `json:"value"`
}

type WithAnon struct {
	Anon
	Field string `json:"field"`
}

func (o WithAnon) New() interface{} {
	return &WithAnon{}
}

func (o WithAnon) Name() string {
	return "with-anon"
}

func (o WithAnon) Call() (jrpc2.Result, error) {
	return "ok", nil
}

func TestAnonField(t *testing.T) {
	wa := &WithAnon{}
	wa.Value = "hello"
	wa.Field = "yep"

	outJson, err := json.Marshal(&jrpc2.Request{
		Id:     jrpc2.NewId("abcde"),
		Method: *wa,
	})
	s := jrpc2.NewServer()
	s.Register(wa)
	var req jrpc2.Request
	err = s.Unmarshal(outJson, &req)
	assert.Nil(t, err)

	unWa := req.Method.(*WithAnon)
	assert.Equal(t, "yep", unWa.Field)
	assert.Equal(t, "hello", unWa.Value)
}

/// now tests for the Result side of things
// this is a bit less involved than the Method parameter
// parsing, since we can effectively pass the marshalling
// responsibility to the Call method on the function, i think

type ArbitraryData struct {
	Item  A
	Map   map[string]A
	Ptr   *A
	Slice []A
	Name  string
}

func TestResponseMarshalling(t *testing.T) {
	merp := make(map[string]A)
	merp["one"] = A{2}
	slis := make([]A, 1)
	slis[0] = A{4}
	arbitData := ArbitraryData{
		Item:  A{1},
		Map:   merp,
		Ptr:   &A{3},
		Slice: slis,
		Name:  "arbit",
	}
	arbitBytes, err := json.Marshal(arbitData)
	erro := &jrpc2.Error{
		Code:    2,
		Message: "omg",
		Data:    arbitBytes,
	}
	resp := &jrpc2.Response{
		Id:    jrpc2.NewId("id"),
		Error: erro,
	}

	out, err := json.Marshal(resp)
	assert.Nil(t, err, "Marshalling error with arbitrary data")
	assert.Equal(t, `{"jsonrpc":"2.0","error":{"code":2,"message":"omg","data":{"Item":{"b":1},"Map":{"one":{"b":2}},"Ptr":{"b":3},"Slice":[{"b":4}],"Name":"arbit"}},"id":"id"}`, string(out))

	var uResp jrpc2.Response
	errTwo := json.Unmarshal(out, &uResp)
	assert.Nil(t, errTwo, "Unmarshalling response")

	var data ArbitraryData
	errThree := json.Unmarshal(uResp.Error.Data, &data)
	assert.Nil(t, errThree, "Unmarshalling extra data")
	assert.Nil(t, uResp.Result, "expected nil result")
	assert.Equal(t, "arbit", data.Name)
}

func TestResponseResult(t *testing.T) {
	out := []byte(`{"jsonrpc":"2.0","id":"29ak","result":"answer"}`)
	var uResp jrpc2.Response
	errTwo := json.Unmarshal(out, &uResp)
	assert.Nil(t, errTwo, "unmarshalling result")

	assert.Equal(t, "answer", uResp.Result.(string))
}

func TestResponseRawResult(t *testing.T) {
	out := []byte(`{"jsonrpc":"2.0","id":"29ak","result":"answer"}`)
	var uResp jrpc2.RawResponse
	err := json.Unmarshal(out, &uResp)
	assert.Nil(t, err, "unmarshalling raw result")

	var str string
	errTwo := json.Unmarshal(uResp.Raw, &str)
	assert.Nil(t, errTwo)
	assert.Equal(t, "answer", str)
}

func TestReponseFancyRaw(t *testing.T) {
	initResponse := &jrpc2.Response{
		Id:     jrpc2.NewId("hello"),
		Result: &A{2},
	}
	out, err := json.Marshal(initResponse)
	assert.Nil(t, err)

	var uResp jrpc2.RawResponse
	errTwo := json.Unmarshal(out, &uResp)
	assert.Nil(t, errTwo)

	var a A
	errThree := json.Unmarshal(uResp.Raw, &a)
	assert.Nil(t, errThree)
	assert.Equal(t, int64(2), a.B)
}

func TestInvalidRawResult(t *testing.T) {
	out := []byte(`{"jsonrpc":"2.0","id":"29ak"}`)
	var uResp jrpc2.RawResponse
	errTwo := json.Unmarshal(out, &uResp)
	assert.NotNil(t, errTwo, "unmarshalling raw result")
	assert.Equal(t, "Must send either a result or an error in a response", errTwo.Error())
}

/// starting with the server because it feels more tractable
// from an implementation perspective

// what needs to happen?
// the server receives an inbound method call
//  the server validates that it's a valid method call
//    - if not the server returns an error
//  the server calls the method and returns the result

// step two, add channeled event loop
// step three, wire it into a socket

type SubtractMethod struct {
	Subtrahend int
	Minuend    int
}

func (s SubtractMethod) New() interface{} {
	return &SubtractMethod{}
}

func (s SubtractMethod) Call() (jrpc2.Result, error) {
	return s.Minuend - s.Subtrahend, nil
}

func (s SubtractMethod) Name() string {
	return "subtract"
}

type ErroringMethod struct{}

func (e ErroringMethod) New() interface{} {
	return &ErroringMethod{}
}
func (e ErroringMethod) Call() (jrpc2.Result, error) {
	return nil, fmt.Errorf("You've got yourself an error")
}

func (e ErroringMethod) Name() string {
	return "error"
}

func TestInboundServer(t *testing.T) {
	sub := &SubtractMethod{5, 2}
	// when the server gets an inbound message, it makes the call
	// and then returns an answer
	// for now, let's pretend this is synchronous
	resp := jrpc2.Execute(nil, sub)
	assert.Nil(t, resp.Error)
	assert.Equal(t, -3, resp.Result.(int))
}

func TestErrorMethod(t *testing.T) {
	method := &ErroringMethod{}
	resp := jrpc2.Execute(nil, method)
	assert.Nil(t, resp.Result)
	assert.Equal(t, "You've got yourself an error", resp.Error.Message)
}

func TestServerRegistry(t *testing.T) {
	server := jrpc2.NewServer()
	method := &ErroringMethod{}

	err := server.Register(method)
	assert.Nil(t, err)
	err = server.Register(method)
	assert.Equal(t, "Method already registered", err.Error())

	err_ := server.UnregisterByName(method.Name())
	assert.Nil(t, err_)
	err_ = server.Unregister(method)
	assert.Equal(t, "Method not registered", err_.Error())
}

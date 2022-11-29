package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/niftynei/glightning/glightning"
	"github.com/niftynei/glightning/jrpc2"
)

type RpcMethodWithParams struct {
	Required string `json:"required"`
	Optional string `json:"optional,omitempty"` // Add 'omitempty' to mark optional
}

func (r *RpcMethodWithParams) Name() string {
	return "new-method"
}

func (r *RpcMethodWithParams) New() interface{} {
	return &RpcMethodWithParams{}
}

func (r *RpcMethodWithParams) Call() (jrpc2.Result, error) {
	if r.Required == "" {
		return nil, errors.New("Missing required parameter")
	}
	return fmt.Sprintf("Called! %s [%s]", r.Required, r.Optional), nil
}

var plugin *glightning.Plugin

func main() {
	plugin = glightning.NewPlugin(onInit)

	rpcMethod := glightning.NewRpcMethod(&RpcMethodWithParams{}, "Example rpc method")
	rpcMethod.LongDesc = "An example rpc method, to try out"
	rpcMethod.Category = "test"

	plugin.RegisterMethod(rpcMethod)

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]glightning.Option, config *glightning.Config) {
	log.Printf("Initialized!")
}

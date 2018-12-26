package jrpc2_test

import (
	"github.com/niftynei/golight/jrpc2"
	"github.com/stretchr/testify/assert"
	"testing"
	"os"
)

type ClientSubtract struct {
	Minuend int
	Subtrahend int
}

func (s *ClientSubtract) Name() string {
	return "subtract"
}

func TestClientParsing(t *testing.T) {
	// set up a server with a function we can call
	s, in, out := setupServer(t)
	s.Register(&Subtract{}) // defined in jsonrpc_cases_test.go
	// setup client
	client := jrpc2.NewClient()
	go client.StartUp(in, out)

	answer, err := subtract(client, 8, 2)
	assert.Nil(t, err)
	assert.Equal(t, 6, answer)
}

func TestClientNoMethod(t *testing.T) {
	// set up a server with a function we can call
	_, in, out := setupServer(t)
	// setup client
	client := jrpc2.NewClient()
	go client.StartUp(in, out)

	answer, err := subtract(client, 8, 2)
	assert.Equal(t, "-32601:Method not found", err.Error())
	assert.Equal(t, 0, answer)
}

func subtract(client *jrpc2.Client, minuend, subtrahend int) (int, error) {
	var response int
	err := client.Request(&ClientSubtract{minuend,subtrahend}, &response)
	return response, err
}

func setupServer(t *testing.T) (server *jrpc2.Server, in, out *os.File) {
	serverIn, out, err := os.Pipe()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	in, serverOut, err := os.Pipe()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	server = jrpc2.NewServer()
	go server.StartUp(serverIn, serverOut)
	return server, in, out
}

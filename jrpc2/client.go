package jrpc2

import (
	"errors"
)
// a client needs to be able to ...
// - 'call' a method which is really...
// - fire off a request 
// - receive a result back (& match that result to outbound request)
// bonus round: 
//    - send and receive in batches 
type Client struct {
	registry map[string]Method
}

func NewClient() *Client {
	client := &Client{}
	client.registry = make(map[string]Method)
	return client
}

// todo: this
func (c *Client) Connect() {
}

// Isses an RPC call. Is blocking.
// todo: take care of an id for this 
// todo: provide an interface to parse into?
func (c *Client) Request(m Method, resp interface{}) (error) {
	// todo: send the request out over the wire,
	// with an appropriate id. i think what we really want
	// is to create a channel/sending mapping that lets
	// us do this in some fancy blocking/async manner

	// when the response comes back, it will either have an error,
	// that we should parse into an 'error' (depending on the code?)
	// or a raw response, that we should json map into the 
	// provided resp (interface)
	return nil
}

// We need to register client functions
// so that the parser can do the right thing
// with them (i think..)
func (c *Client) Register(method Method) error {
	name := method.Name()
	if _, exists := c.registry[name]; exists {
		return errors.New("Method already registered")
	}

	c.registry[name] = method
	return nil
}

func (c *Client) Unregister(method Method) error {
	if _, exists := c.registry[method.Name()]; !exists {
		return errors.New("Method not registered")
	}
	delete(c.registry, method.Name())
	return nil
}



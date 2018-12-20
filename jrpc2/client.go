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

// methods a client needs.. ??
func (c *Client) Send() {
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



package jrpc2

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

// method type to register on the server side
type ServerMethod interface {
	Method
	New() interface{}
	Call() (Result, error)
}

// a server needs to be able to
// - send back a response (with the right id)
// bonus round:
//   - respond to batched requests
type Server struct {
	registry map[string]ServerMethod
	outQueue chan *Response
	shutdown bool
}

func NewServer() *Server {
	server := &Server{}
	server.registry = make(map[string]ServerMethod)
	server.outQueue = make(chan *Response)
	server.shutdown = false
	return server
}

// Listen through a file socket
func (s *Server) StartUpSingle(in string) {
	ln, err := net.Listen("unix", in)
	if err != nil {
		log.Fatalf("Unable to listen on file socket %s", err.Error())
		return
	}
	defer ln.Close()
	for !s.shutdown {
		inConn, err := ln.Accept()
		if err != nil {
			log.Print(err.Error())
			continue
		}
		go func() {
			s.listen(inConn)
		}()
		go func() {
			defer inConn.Close()
			s.setupWriteQueue(inConn)
		}()
	}
}

func (s *Server) StartUp(in, out *os.File) error {
	go s.setupWriteQueue(out)
	return s.listen(in)
}

func (s *Server) Shutdown() {
	s.shutdown = true
	close(s.outQueue)
}

func scanDoubleNewline(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' && (i+1) < len(data) && data[i+1] == '\n' {
			return i + 2, data[:i], nil
		}
	}
	// this trashes anything left over in
	// the buffer if we're at EOF, with no /n/n present
	return 0, nil, nil
}

func (s *Server) listen(in io.Reader) error {
	// use a scanner to read in messages.
	// since we're mapping this pretty 'strongly'
	// to c-lightning's plugin system,
	// we use the double newline character
	// to break out new messages
	scanner := bufio.NewScanner(in)
	scanner.Split(scanDoubleNewline)
	for scanner.Scan() && !s.shutdown {
		msg := scanner.Bytes()
		// todo: send this over a channel
		// for processing, so the number
		// of things we process at once
		// is more easy to control
		go processMsg(s, msg)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func (s *Server) setupWriteQueue(outWriter io.Writer) {
	out := bufio.NewWriter(outWriter)
	twoNewlines := []byte("\n\n")
	for response := range s.outQueue {
		data, err := json.Marshal(response)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		// append two newlines to the outgoing message
		data = append(data, twoNewlines...)
		out.Write(data)
		out.Flush()
	}
}

func processMsg(s *Server, data []byte) {
	// read is done. time to figure out what we've gotten
	if len(data) == 0 {
		s.outQueue <- (&Response{
			Error: &Error{
				Code:    InvalidRequest,
				Message: "Invalid Request",
			},
		})
		return
	}

	// right now we don't handle arrays of requests...
	// todo: infra for batches (ie use wait group)
	if data[0] == '[' {
		s.outQueue <- &Response{
			Error: &Error{
				Code:    InternalErr,
				Message: "This server can't handle batch requests",
			},
		}
		return
	}

	// parse the received buffer into a request object
	var request Request
	err := s.Unmarshal(data, &request)
	if err != nil {
		s.outQueue <- &Response{
			Id: err.Id,
			Error: &Error{
				Code:    err.Code,
				Message: err.Msg,
			},
		}
		return
	}

	// ok we've successfully gotten the method call out..
	s.outQueue <- Execute(request.Id, request.Method.(ServerMethod))
}

func Execute(id *Id, method ServerMethod) *Response {
	result, err := method.Call()
	resp := &Response{
		Id: id,
	}
	if err != nil {
		// todo: data object for errors?
		resp.Error = constructError(err)
	} else {
		resp.Result = result
	}

	return resp
}

func (s *Server) Register(method ServerMethod) error {
	name := method.Name()
	if _, exists := s.registry[name]; exists {
		return errors.New("Method already registered")
	}

	s.registry[name] = method
	return nil
}

func (s *Server) GetMethodMap() []ServerMethod {
	list := make([]ServerMethod, len(s.registry))
	i := 0
	for _, v := range s.registry {
		list[i] = v
		i++
	}
	return list
}

func (s *Server) UnregisterByName(name string) error {
	if _, exists := s.registry[name]; !exists {
		return errors.New("Method not registered")
	}
	delete(s.registry, name)
	return nil
}

func (s *Server) Unregister(method ServerMethod) error {
	return s.UnregisterByName(method.Name())
}

func constructError(err error) *Error {
	// todo: specify return data?
	return &Error{
		Code:    -1,
		Message: err.Error(),
	}
}

func (s *Server) Unmarshal(data []byte, r *Request) *CodedError {
	type Alias Request
	raw := &struct {
		Version string          `json:"jsonrpc"`
		Params  json.RawMessage `json:"params,omitempty"`
		Name    string          `json:"method"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return NewError(nil, ParseError, "Parse error")
	}
	if raw.Version != specVersion {
		return NewError(raw.Id, InvalidRequest, fmt.Sprintf(`Invalid version, expected "%s" got "%s"`, specVersion, raw.Version))
	}
	if raw.Name == "" {
		return NewError(raw.Id, InvalidRequest, "`method` cannot be empty")
	}

	stashedMethod, ok := s.registry[raw.Name]
	if !ok {
		return NewError(raw.Id, MethodNotFound, fmt.Sprintf("Method not found"))
	}

	// New method of the given type
	method := stashedMethod.New()
	r.Method = method.(Method)

	// figure out what kind of params we've got: named, an array, or empty
	if len(raw.Params) == 0 {
		return nil
	}
	var obj interface{}
	err = json.Unmarshal(raw.Params, &obj)
	if err != nil {
		return NewError(raw.Id, ParseError, "Parse error")
	}
	switch obj.(type) {
	case []interface{}:
		err = ParseParamArray(r.Method, obj.([]interface{}))
	case map[string]interface{}:
		err = ParseNamedParams(r.Method, obj.(map[string]interface{}))
	default:
		err = NewError(raw.Id, InvalidParams, "Invalid params")
	}

	// set the id for an error created in a subroutine
	if err != nil {
		codedErr, ok := err.(CodedError)
		if ok {
			codedErr.Id = raw.Id
			return &codedErr
		}
		return NewError(raw.Id, InvalidParams, err.Error())
	}
	return nil
}

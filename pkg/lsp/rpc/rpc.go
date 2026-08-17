// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/
package rpc

import (
	"bufio"
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strconv"
)

const JSON_RPCVersion = "2.0"

// Things that can be sent/received
// ========

type Message interface {
	lspMsg()
}

type baseMessage struct {
	JSONRPC string `json:"jsonrpc"` // Must be "2.0"
}

func (*baseMessage) lspMsg() {}

type (
	Method string
	ID     = *Union2[int, string]
)

type Request struct {
	baseMessage
	Id     ID     `json:"id"`               // integer | string. The request id.
	Method Method `params:"method"`         // The method to be invoked.
	Params any    `json:"params,omitempty"` // object | array | null. The method's params.
}
type Response struct {
	baseMessage
	// The request id.
	Id ID `json:"id,omitempty"` // integer | string | null
	/**
	 * The result of a request. This member is REQUIRED on success.
	 * This member MUST NOT exist if there was an error invoking the method.
	 */
	Result any `json:"result,omitempty"` // LSPAny | null
	/**
	 * The error object in case a request fails.
	 */
	Error *ResponseError `json:"error,omitempty"`
}
type Notification struct {
	baseMessage
	Method Method `json:"method"`
	Params any    `json:"params"` // array | object | null
}

type ResponseError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// Encoding/Decoding
// =======

func Encode(msg any) ([]byte, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return fmt.Appendf(nil, "Content-Length: %d\r\n\r\n%s", len(b), b), nil
}

func Decode(b []byte) (Message, error) {
	// Using [fmt.Sscanf] creates issues when decoding Klar source code in strings
	header, body, ok := bytes.Cut(b, []byte("\r\n\r\n"))
	if !ok {
		return nil, errors.New("message is missing header")
	}
	ctLen, err := strconv.Atoi(string(header[len("Content-Length: "):]))
	if err != nil {
		return nil, fmt.Errorf("failed to parse content length: %v", err)
	}
	body = body[:ctLen]
	// Go's JSON-RPC 2.0 implementation also uses a combined wire type
	// https://github.com/golang/tools/blob/master/internal/jsonrpc2_v2/wire.go
	var baseMsg struct {
		JSONRPC string         `json:"jsonrpc"`
		Id      ID             `json:"id"`
		Method  Method         `json:"method"`
		Params  jsontext.Value `json:"params"`
		Result  jsontext.Value `json:"result"`
		Error   *ResponseError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &baseMsg); err != nil {
		return nil, err
	}
	switch {
	case baseMsg.JSONRPC != JSON_RPCVersion:
		return nil, fmt.Errorf(
			"'jsonrpc' must be %q, got %q", JSON_RPCVersion, baseMsg.JSONRPC,
		)
	case baseMsg.Result != nil, baseMsg.Error != nil:
		return &Response{
			Id:     baseMsg.Id,
			Result: baseMsg.Result,
			Error:  baseMsg.Error,
		}, nil
	case baseMsg.Id.IsNil():
		// A notification is like a request, but no id
		return &Notification{
			Method: baseMsg.Method,
			Params: baseMsg.Params,
		}, nil
	default:
		return &Request{
			Id:     baseMsg.Id,
			Method: baseMsg.Method,
			Params: baseMsg.Params,
		}, nil
	}
}

var _ bufio.SplitFunc = Split

func Split(data []byte, _atEOF bool) (advance int, token []byte, err error) {
	header, body, found := bytes.Cut(data, []byte("\r\n\r\n"))
	if !found {
		return 0, nil, nil
	}
	ctLenBytes := header[len("Content-Length: "):]
	ctLen, err := strconv.Atoi(string(ctLenBytes))
	switch {
	case err != nil:
		return 0, nil, err
	case len(body) < ctLen:
		return 0, nil, nil // We need more
	default:
		totalLen := len(header) + 4 /* \r\n\r\n */ + ctLen
		return totalLen, data[:totalLen], nil
	}
}

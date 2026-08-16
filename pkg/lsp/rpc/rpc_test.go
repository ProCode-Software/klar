package rpc

import (
	"fmt"
	"testing"
)

type encodingExample struct {
	Testing bool `json:"testing"`
}

func TestEncode(t *testing.T) {
	exp := "Content-Length: 16\r\n\r\n" + `{"testing":true}`
	got, err := Encode(encodingExample{Testing: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != exp {
		t.Errorf("expected %#q, got %#q", exp, got)
	}
}

func TestDecode(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"x"}`
	in := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	msg, err := Decode([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	req, ok := msg.(*Request)
	if !ok {
		t.Errorf("expected *Request, got %T", msg)
	}
	if req.Id == nil || req.Id.Value != int(1) {
		t.Errorf("expected id 1, got %q", req.Id)
	}
	if req.Method != "x" {
		t.Errorf("expected method x, got %s", req.Method)
	}
}

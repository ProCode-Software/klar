package lsp

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/pkg/lsp"
	"github.com/ProCode-Software/klar/pkg/lsp/rpc"
)

func (s *Server) Listen() {
	scanner := bufio.NewScanner(s.in)
	scanner.Split(rpc.Split)
	s.Debug("Server started")
	for scanner.Scan() {
		msg, err := rpc.Decode(scanner.Bytes())
		if err != nil {
			s.Error("Error while decoding", slog.Any("error", err))
			s.sendError("Decode error: "+err.Error(), nil, rpc.ParseError)
			continue
		}
		switch msg := msg.(type) {
		case *rpc.Request:
			s.handleRequest(msg)
		case *rpc.Response:
		case *rpc.Notification:
		default:
			s.Error(fmt.Sprintf("invalid rpc.Message type: %T", msg))
		}
	}
	if err := scanner.Err(); err != nil {
		s.Error("Error while reading", slog.Any("error", err))
	}
}

// sendResponse sends a successful response to the output stream.
func (s *Server) sendResponse(result any, id rpc.ID) (ok bool) {
	// TODO: Do we need an ID?
	msg := &rpc.Response{
		Result: result,
		Id:     id,
	}
	msg.JSONRPC = rpc.JSON_RPCVersion
	b, err := rpc.Encode(msg)
	if err == nil {
		_, err = s.out.Write(b)
	}
	if err != nil {
		s.Error("Error while sending response", slog.Any("error", err))
		return false
	}
	return true
}

func (s *Server) sendError(msg string, id rpc.ID, code rpc.ErrorCode) {
	rpcMsg := &rpc.Response{
		Id:     id,
		Result: nil,
		Error:  &rpc.ResponseError{Code: code, Message: msg},
	}
	rpcMsg.JSONRPC = rpc.JSON_RPCVersion
	b, err := rpc.Encode(rpcMsg)
	if err == nil {
		_, err = s.out.Write(b)
	}
	if err != nil {
		s.Error("Error while sending error", slog.Any("error", err))
	} else {
		s.Debug("Error response sent")
	}
}

func decodeParams[T any](s *Server, id rpc.ID, val jsontext.Value) (T, bool) {
	var result T
	if err := json.Unmarshal(val, &result); err != nil {
		s.Error("Failed to decode message params", slog.Any("error", err))
		s.sendError(
			fmt.Sprintf("Failed to decode message params: %v", err),
			id, rpc.ParseError,
		)
		return result, false
	}
	return result, true
}

func (s *Server) handleRequest(req *rpc.Request) {
	var params jsontext.Value
	if req.Params != nil {
		params = req.Params.(jsontext.Value)
	}
	s.Info("Got request", slog.Any("method", req.Method))
	switch req.Method {
	case "initialize":
		s.Debug(string(params))
		params, ok := decodeParams[*lsp.InitializeParams](s, req.Id, params)
		if !ok {
			return
		}
		s.sendResponse(lsp.InitializeResult{
			Capabilities: s.getCapabilities(params),
			ServerInfo: &lsp.ServerInfo{
				Name:    "KlarLS",
				Version: cli.KlarVersion,
			},
		}, req.Id)
		s.Info("Server initialized", slog.String("clientName", params.ClientInfo.Name))
	}
}

func (s *Server) getCapabilities(init *lsp.InitializeParams) *lsp.ServerCapabilities {
	caps := &lsp.ServerCapabilities{
		// Get notifications when files are opened/closed. The compiler currently doesn't
		// support file-level incremental parsing (yet), so we must sync the whole file.
		TextDocumentSync: &rpc.Union2[lsp.TextDocumentSyncOptions, lsp.TextDocumentSyncKind]{
			lsp.TextDocumentSyncOptions{
				OpenClose: new(true),
				Change:    new(lsp.TextDocumentSyncKindFull),
			},
		},
		PositionEncoding:   nil, // Set below. If not, it should be utf-16
		CompletionProvider: &lsp.CompletionOptions{},
		CodeActionProvider: &rpc.Union2[bool, lsp.CodeActionOptions]{},
		Workspace:          &lsp.WorkspaceOptions{},
		NotebookDocumentSync: &rpc.Union2[
			lsp.NotebookDocumentSyncOptions, lsp.NotebookDocumentSyncRegistrationOptions,
		]{false}, // Jupyter notebooks not supported yet
		HoverProvider: &rpc.Union2[bool, lsp.HoverOptions]{lsp.HoverOptions{}},
	}
	// Determine the position encoding the server should use.
	//
	// The lexer encodes positions in UTF-32 (codepoints/runes). If the client
	// supports it, prefer that. Otherwise, prefer uft8 if supported. There are
	// better Go APIs for UTF-8 than UTF-16.
	preferredEncoding := [...]lsp.PositionEncodingKind{
		lsp.PositionEncodingKindUTF32, lsp.PositionEncodingKindUTF8,
		lsp.PositionEncodingKindUTF16,
	}
	clientEncodings := init.Capabilities.General.PositionEncodings
	switch len(clientEncodings) {
	case 0:
		// Per the spec: If no positions are provided, assume utf16
		caps.PositionEncoding = new(lsp.UTF16)
	case 1:
		caps.PositionEncoding = &clientEncodings[0]
	default:
		for _, pref := range preferredEncoding {
			if slices.Contains(clientEncodings, pref) {
				caps.PositionEncoding = &pref
				break
			}
		}
	}
	return caps
}

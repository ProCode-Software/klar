package lsp

import (
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"log/slog"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/pkg/lsp"
	"github.com/ProCode-Software/klar/pkg/lsp/rpc"
)

func (s *Server) decodeParams[T any](id rpc.ID, val jsontext.Value) (T, bool) {
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
		params, ok := s.decodeParams[*lsp.InitializeParams](req.Id, params)
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
	case "textDocument/documentColor":
		params, ok := s.decodeParams[lsp.DocumentColorParams](req.Id, params)
		if !ok {
			return
		}
		s.documentColor(req.Id, params.TextDocument)
	case "textDocument/diagnostic":
		params, ok := s.decodeParams[lsp.DocumentDiagnosticParams](req.Id, params)
		if !ok {
			return
		}
		s.documentDiagnostic(req.Id, params)
	default:
		s.Warn("Unhandled request", slog.String("method", string(req.Method)))
		// Spec: [JSONRPC] requires that every request sends a response back
		s.sendError("Not implemented", req.Id, rpc.MethodNotFound)
	}
}

func (s *Server) handleNotification(not *rpc.Notification) {
	var params jsontext.Value
	if not.Params != nil {
		params = not.Params.(jsontext.Value)
	}
	switch not.Method {
	case "initialized": // Nothing to do
	case "textDocument/didOpen":
		params, ok := s.decodeParams[lsp.DidOpenTextDocumentParams](nil, params)
		if !ok {
			return
		}
		s.didOpen(params)
	case "textDocument/didChange":
		params, ok := s.decodeParams[lsp.DidChangeTextDocumentParams](nil, params)
		if !ok {
			return
		}
		// TODO: Cancel previous compilation if still compiling
		s.didChange(params)
	default:
		s.Warn("Unhandled notification", slog.String("method", string(not.Method)))
	}
}

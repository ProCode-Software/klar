package lsp

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/pkg/lsp"
	"github.com/ProCode-Software/klar/pkg/lsp/rpc"
)

type Server struct {
	in  io.Reader // Usually stdin
	out io.Writer // Usually stdout

	// File store
	pkgs map[string]*Package
	fs   *FileSystem
	*slog.Logger
}

func (s *Server) Listen() {
	scanner := bufio.NewScanner(s.in)
	scanner.Split(rpc.Split)
	s.Debug("Server started")
	var isShutDown bool
	for scanner.Scan() {
		msg, err := rpc.Decode(scanner.Bytes())
		if err != nil {
			s.Error("Error while decoding", slog.Any("error", err))
			s.sendError("Decode error: "+err.Error(), nil, rpc.ParseError)
			continue
		}
		switch msg := msg.(type) {
		case *rpc.Request:
			// Lifecycle methods
			switch msg.Method {
			case "shutdown":
				isShutDown = true
				// Spec: Params: null
				s.sendResponse(nil, msg.Id)
				continue
			case "exit":
				// Spec: The server should exit with success code 0 if the shutdown
				// request has been received before; otherwise with error code 1.
				if !isShutDown {
					cli.Exit(1)
				} else {
					cli.Exit(0)
				}
			}
			s.handleRequest(msg)
		case *rpc.Response:
		case *rpc.Notification:
			s.handleNotification(msg)
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
	if result == nil {
		// We have to use a [jsontext.Value] instead of nil so the result
		// isn't omitted (by omitzero)
		result = jsontext.Value("null")
	}
	msg := &rpc.Response{
		JSONRPC: rpc.JSONRPCVersion,
		Result:  result,
		Id:      id,
	}
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
		JSONRPC: rpc.JSONRPCVersion,
		Id:      id,
		Result:  nil,
		Error:   &rpc.ResponseError{Code: code, Message: msg},
	}
	b, err := rpc.Encode(rpcMsg)
	if err == nil {
		_, err = s.out.Write(b)
	}
	if err != nil {
		s.Error("Error while sending error", slog.Any("error", err))
	}
}

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
	default:
		s.Warn("Unhandled request", slog.String("method", string(req.Method)))
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
		s.compilePackages()
	case "textDocument/didChange":
		params, ok := s.decodeParams[lsp.DidChangeTextDocumentParams](nil, params)
		if !ok {
			return
		}
		// TODO: Cancel previous compilation if still compiling
		s.didChange(params)
		s.compilePackages()
	default:
		s.Warn("Unhandled notification", slog.String("method", string(not.Method)))
	}
}

func (s *Server) getCapabilities(init *lsp.InitializeParams) *lsp.ServerCapabilities {
	caps := &lsp.ServerCapabilities{
		// Get notifications when files are opened/closed. The compiler currently doesn't
		// support file-level incremental parsing (yet), so we must sync the whole file.
		TextDocumentSync: &rpc.Union2[lsp.TextDocumentSyncOptions, lsp.TextDocumentSyncKind]{
			lsp.TextDocumentSyncOptions{
				OpenClose: new(true),
				Change:    new(lsp.TextDocumentSyncFull),
			},
		},
		DocumentFormattingProvider: &rpc.Union2[bool, lsp.DocumentFormattingOptions]{true},
		DocumentRangeFormattingProvider: &rpc.Union2[bool, lsp.DocumentRangeFormattingOptions]{
			true,
		},
		DefinitionProvider: &rpc.Union2[bool, lsp.DefinitionOptions]{true},
		SignatureHelpProvider: &lsp.SignatureHelpOptions{
			TriggerCharacters:   []string{"(", ",", ":"},
			RetriggerCharacters: []string{")"},
		},
		ReferencesProvider: &rpc.Union2[bool, lsp.ReferenceOptions]{true},
		PositionEncoding:   nil, // Set below. If not, it should be utf-16
		CompletionProvider: &lsp.CompletionOptions{
			TriggerCharacters: []string{".", "-", "@"},
			CompletionItem:    nil, /* &lsp.ServerCompletionItemOptions{
				LabelDetailsSupport: new(true),
			} */
		},
		ColorProvider: &rpc.Union3[
			bool, lsp.DocumentColorOptions, lsp.DocumentColorRegistrationOptions,
		]{true},
		SemanticTokensProvider: &rpc.Union2[
			lsp.SemanticTokensOptions, lsp.SemanticTokensRegistrationOptions,
		]{lsp.SemanticTokensOptions{
			Full: &rpc.Union2[bool, lsp.SemanticTokensFullDelta]{true},
		}},
		CodeActionProvider:     &rpc.Union2[bool, lsp.CodeActionOptions]{},
		Workspace:              &lsp.WorkspaceOptions{},
		DocumentSymbolProvider: &rpc.Union2[bool, lsp.DocumentSymbolOptions]{true},
		CodeLensProvider:       &lsp.CodeLensOptions{},
		RenameProvider: &rpc.Union2[bool, lsp.RenameOptions]{lsp.RenameOptions{
			PrepareProvider: new(true),
		}},
		NotebookDocumentSync: nil, // Jupyter notebooks not supported yet
		HoverProvider:        &rpc.Union2[bool, lsp.HoverOptions]{true},
		FoldingRangeProvider: &rpc.Union3[bool,
			lsp.FoldingRangeOptions, lsp.FoldingRangeRegistrationOptions]{true},
		TypeDefinitionProvider: &rpc.Union3[bool,
			lsp.TypeDefinitionOptions, lsp.TypeDefinitionRegistrationOptions]{true},
		DiagnosticProvider: &rpc.Union2[lsp.DiagnosticOptions,
			lsp.DiagnosticRegistrationOptions]{lsp.DiagnosticOptions{
			WorkspaceDiagnostics:  false,
			InterFileDependencies: true,
			Identifier:            "Klar", // TODO: Do we need this?
		}},
		ImplementationProvider: &rpc.Union3[bool,
			lsp.ImplementationOptions, lsp.ImplementationRegistrationOptions]{true},
		WorkspaceSymbolProvider: &rpc.Union2[bool, lsp.WorkspaceSymbolOptions]{true},
	}
	// Determine the position encoding the server should use.
	//
	// The lexer encodes positions in UTF-32 (codepoints/runes). If the client
	// supports it, prefer that. Otherwise, prefer uft8 if supported. There are
	// better Go APIs for UTF-8 than UTF-16.
	preferredEncoding := [...]lsp.PositionEncodingKind{
		lsp.PositionEncodingUTF32, lsp.PositionEncodingUTF8, lsp.PositionEncodingUTF16,
	}
	clientEncodings := init.Capabilities.General.PositionEncodings
	switch len(clientEncodings) {
	case 0:
		// Per the spec: If no positions are provided, assume utf16
		caps.PositionEncoding = new(lsp.PositionEncodingUTF16)
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

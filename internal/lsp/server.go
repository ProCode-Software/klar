package lsp

import (
	"bufio"
	"context"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"log/slog"
	"slices"

	"github.com/ProCode-Software/klar/internal/build"
	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/pkg/lsp"
	"github.com/ProCode-Software/klar/pkg/lsp/rpc"
)

type Server struct {
	in  io.Reader // Usually stdin
	out io.Writer // Usually stdout

	// File store
	pkgs    map[string]*Package
	fs      *FileSystem
	modules map[string]*Module

	compiler *build.Compiler // Shared compiler
	caps     *capabilities   // Capabilities decided on after initialization
	*slog.Logger
}

func NewServer(in io.Reader, out io.Writer, l *slog.Logger) *Server {
	s := &Server{
		in: in, out: out,
		Logger:  l,
		fs:      &FileSystem{Files: make(map[string]*File)},
		pkgs:    make(map[string]*Package),
		modules: make(map[string]*Module),
	}
	s.initCompiler()
	return s
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
			switch {
			case msg.Method == "shutdown":
				isShutDown = true
				s.sendResponse(nil, msg.Id) // Spec: Params: null
				// The server still has to wait for an 'exit' notification
				continue
			case isShutDown:
				// Spec: If a server receives requests after a shutdown request
				// those requests should error with [rpc.InvalidRequest].
				s.sendError("Server is shut down", msg.Id, rpc.InvalidRequest)
			default:
				s.handleRequest(msg)
			}
		case *rpc.Response:
		case *rpc.Notification:
			if msg.Method == "exit" {
				// Spec: The server should exit with success code 0 if the shutdown
				// request has been received before; otherwise with error code 1.
				if !isShutDown {
					cli.Exit(1)
				} else {
					cli.Exit(0)
				}
			}
			s.handleNotification(msg)
		default:
			s.Error(fmt.Sprintf("invalid rpc.Message type: %T", msg))
		}
	}
	if err := scanner.Err(); err != nil {
		s.Error("Error while reading", slog.Any("error", err))
	}
}

func (s *Server) logEnabled() bool {
	return s.Logger.Enabled(context.Background(), slog.LevelInfo)
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

func (s *Server) sendNotification(method rpc.Method, params any) (ok bool) {
	if params == nil {
		params = jsontext.Value("null")
	}
	msg := &rpc.Notification{
		JSONRPC: rpc.JSONRPCVersion,
		Method:  method,
		Params:  params,
	}
	b, err := rpc.Encode(msg)
	if err == nil {
		_, err = s.out.Write(b)
	}
	if err != nil {
		s.Error("Error while sending notification", slog.Any("error", err))
		return false
	}
	return true
}

func (s *Server) showMessageToUser(typ lsp.MessageType, msg string) {
	s.sendNotification("window/showMessage", lsp.ShowMessageParams{
		Type: typ, Message: msg,
	})
}

func (s *Server) getCapabilities(init *lsp.InitializeParams) *lsp.ServerCapabilities {
	caps := &lsp.ServerCapabilities{
		// Get notifications when files are opened/closed. The compiler currently doesn't
		// support file-level incremental parsing (yet), so we must sync the whole file.
		TextDocumentSync: &rpc.Union2[lsp.TextDocumentSyncOptions, lsp.TextDocumentSyncKind]{
			lsp.TextDocumentSyncOptions{
				OpenClose: new(true),
				Change:    new(lsp.TextDocumentSyncFull),
				Save: &rpc.Union2[bool, lsp.SaveOptions]{
					lsp.SaveOptions{IncludeText: new(false)},
				},
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
		PositionEncoding:   new(lsp.PositionEncodingUTF16), // Set below
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
		CallHierarchyProvider: &rpc.Union3[bool,
			lsp.CallHierarchyOptions, lsp.CallHierarchyRegistrationOptions]{true},
		TypeHierarchyProvider: nil, // Type hierarchy isn't applicable for Klar
		CodeActionProvider:    &rpc.Union2[bool, lsp.CodeActionOptions]{},
		Workspace: &lsp.WorkspaceOptions{
			WorkspaceFolders: &lsp.WorkspaceFoldersServerCapabilities{
				Supported:           new(true),
				ChangeNotifications: &rpc.Union2[string, bool]{true},
			},
			// TODO: Do we need any of these? Gopls has didCreate only
			FileOperations:      nil,
			TextDocumentContent: nil, // TODO: Re-evaluate later
		},
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
		// Commented out until implemented. It breaks my editor if enabled
		// and not sending responses.
		// WorkspaceSymbolProvider: &rpc.Union2[bool, lsp.WorkspaceSymbolOptions]{true},
		DeclarationProvider: &rpc.Union3[bool,
			lsp.DeclarationOptions, lsp.DeclarationRegistrationOptions]{true},
		DocumentLinkProvider:      &lsp.DocumentLinkOptions{},
		DocumentHighlightProvider: &rpc.Union2[bool, lsp.DocumentHighlightOptions]{true},
		InlayHintProvider: &rpc.Union3[bool,
			lsp.InlayHintOptions, lsp.InlayHintRegistrationOptions]{true},
		ExecuteCommandProvider: &lsp.ExecuteCommandOptions{
			Commands: nil, // TODO
		},
		Experimental: nil,
	}
	// Determine the position encoding the server should use.
	//
	// The lexer encodes positions in UTF-32 (codepoints/runes). If the client
	// supports it, prefer that. Otherwise, prefer UTF-8 if supported. There are
	// better Go APIs for UTF-8 than UTF-16.
	preferredEncoding := [...]lsp.PositionEncodingKind{
		lsp.PositionEncodingUTF32, lsp.PositionEncodingUTF8, lsp.PositionEncodingUTF16,
	}
	var clientEncodings []lsp.PositionEncodingKind
	if init.Capabilities.General != nil {
		clientEncodings = init.Capabilities.General.PositionEncodings
	}
	switch len(clientEncodings) {
	case 0:
		// Per the spec: If no positions are provided, assume utf16 (set above)
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
	// Code action support. Spec: CodeActionOptions may only be specified if the
	// client states that it supports `codeActionLiteralSupport` in its initial
	// `initialize` request.
	var codeActionProv any = true
	if tdCaps := init.Capabilities.TextDocument; tdCaps != nil &&
		tdCaps.CodeAction != nil && tdCaps.CodeAction.CodeActionLiteralSupport != nil {
		codeActionProv = lsp.CodeActionOptions{
			ResolveProvider: new(true),
			Documentation:   nil, // TODO: I don't understand the use of this
			CodeActionKinds: []lsp.CodeActionKind{
				// All code action kinds KlarLS supports
				lsp.CodeActionEmpty, lsp.CodeActionQuickFix, lsp.CodeActionRefactor,
				lsp.CodeActionRefactorExtract, lsp.CodeActionRefactorInline,
				lsp.CodeActionRefactorMove, lsp.CodeActionRefactorRewrite,
				lsp.CodeActionSource, lsp.CodeActionSourceOrganizeImports,
			},
		}
	}
	caps.CodeActionProvider.Value = codeActionProv
	// KlarLS is implemented using pull diagnostics (only)
	if init.Capabilities.TextDocument.Diagnostic == nil {
		// Client doesn't support pull diagnostics (uses publishDiagnostics instead)
		s.Error("Client doesn't support pull diagnostics")
	}

	s.caps = &capabilities{
		posEncoding: *caps.PositionEncoding,
	}
	return caps
}

type capabilities struct {
	posEncoding lsp.PositionEncodingKind
}

package lsp

import "github.com/ProCode-Software/klar/pkg/lsp/rpc"

type ClientCapabilities struct {
	// Workspace specific client capabilities.
	Workspace *WorkspaceClientCapabilities `json:"workspace,omitempty"`
	// Text document specific client capabilities.
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	/**
	 * Capabilities specific to the notebook document support.
	 * @since 3.17.0
	 */
	NotebookDocument *NotebookDocumentClientCapabilities `json:"notebookDocument,omitempty"`
	// Window specific client capabilities.
	Window *WindowClientCapabilities `json:"window,omitempty"`
	/**
	 * General client capabilities.
	 * @since 3.16.0
	 */
	General *GeneralClientCapabilities `json:"general,omitempty"`
	// Experimental client capabilities.
	Experimental any `json:"experimental,omitempty"`
}

type TextDocumentClientCapabilities struct {
	// Defines which synchronization capabilities the client supports.
	Synchronization *TextDocumentSyncClientCapabilities `json:"synchronization,omitempty"`
	/**
	 * Defines which filters the client supports.
	 * @since 3.18.0
	 */
	Filters *TextDocumentFilterClientCapabilities `json:"filters,omitempty"`
	// Capabilities specific to the `textDocument/completion` request.
	Completion *CompletionClientCapabilities `json:"completion,omitempty"`
	// Capabilities specific to the `textDocument/hover` request.
	Hover *HoverClientCapabilities `json:"hover,omitempty"`
	// Capabilities specific to the `textDocument/signatureHelp` request.
	SignatureHelp *SignatureHelpClientCapabilities `json:"signatureHelp,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/declaration` request.
	 * @since 3.14.0
	 */
	Declaration *DeclarationClientCapabilities `json:"declaration,omitempty"`
	// Capabilities specific to the `textDocument/definition` request.
	Definition *DefinitionClientCapabilities `json:"definition,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/typeDefinition` request.
	 * @since 3.6.0
	 */
	TypeDefinition *TypeDefinitionClientCapabilities `json:"typeDefinition,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/implementation` request.
	 * @since 3.6.0
	 */
	Implementation *ImplementationClientCapabilities `json:"implementation,omitempty"`
	// Capabilities specific to the `textDocument/references` request.
	References *ReferenceClientCapabilities `json:"references,omitempty"`
	// Capabilities specific to the `textDocument/documentHighlight` request.
	DocumentHighlight *DocumentHighlightClientCapabilities `json:"documentHighlight,omitempty"`
	// Capabilities specific to the `textDocument/documentSymbol` request.
	DocumentSymbol *DocumentSymbolClientCapabilities `json:"documentSymbol,omitempty"`
	// Capabilities specific to the `textDocument/codeAction` request.
	CodeAction *CodeActionClientCapabilities `json:"codeAction,omitempty"`
	// Capabilities specific to the `textDocument/codeLens` request.
	CodeLens *CodeLensClientCapabilities `json:"codeLens,omitempty"`
	// Capabilities specific to the `textDocument/documentLink` request.
	DocumentLink *DocumentLinkClientCapabilities `json:"documentLink,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/documentColor` and the
	 * `textDocument/colorPresentation` request.
	 * @since 3.6.0
	 */
	ColorProvider *DocumentColorClientCapabilities `json:"colorProvider,omitempty"`
	// Capabilities specific to the `textDocument/formatting` request.
	Formatting *DocumentFormattingClientCapabilities `json:"formatting,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/rangeFormatting` and
	 * `textDocument/rangesFormatting requests.
	 */
	RangeFormatting *DocumentRangeFormattingClientCapabilities `json:"rangeFormatting,omitempty"`
	// Capabilities specific to the `textDocument/onTypeFormatting` request.
	OnTypeFormatting *DocumentOnTypeFormattingClientCapabilities `json:"onTypeFormatting,omitempty"`
	// Capabilities specific to the `textDocument/rename` request.
	Rename *RenameClientCapabilities `json:"rename,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/publishDiagnostics`
	 * notification.
	 */
	PublishDiagnostics *PublishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/foldingRange` request.
	 * @since 3.10.0
	 */
	FoldingRange *FoldingRangeClientCapabilities `json:"foldingRange,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/selectionRange` request.
	 * @since 3.15.0
	 */
	SelectionRange *SelectionRangeClientCapabilities `json:"selectionRange,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/linkedEditingRange` request.
	 * @since 3.16.0
	 */
	LinkedEditingRange *LinkedEditingRangeClientCapabilities `json:"linkedEditingRange,omitempty"`
	/**
	 * Capabilities specific to the various call hierarchy requests.
	 * @since 3.16.0
	 */
	CallHierarchy *CallHierarchyClientCapabilities `json:"callHierarchy,omitempty"`
	/**
	 * Capabilities specific to the various semantic token requests.
	 * @since 3.16.0
	 */
	SemanticTokens *SemanticTokensClientCapabilities `json:"semanticTokens,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/moniker` request.
	 * @since 3.16.0
	 */
	Moniker *MonikerClientCapabilities `json:"moniker,omitempty"`
	/**
	 * Capabilities specific to the various type hierarchy requests.
	 * @since 3.17.0
	 */
	TypeHierarchy *TypeHierarchyClientCapabilities `json:"typeHierarchy,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/inlineValue` request.
	 * @since 3.17.0
	 */
	InlineValue *InlineValueClientCapabilities `json:"inlineValue,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/inlayHint` request.
	 * @since 3.17.0
	 */
	InlayHint *InlayHintClientCapabilities `json:"inlayHint,omitempty"`
	/**
	 * Capabilities specific to the diagnostic pull model.
	 * @since 3.17.0
	 */
	Diagnostic *DiagnosticClientCapabilities `json:"diagnostic,omitempty"`
	/**
	 * Capabilities specific to the `textDocument/inlineCompletion` request.
	 * @since 3.18.0
	 */
	InlineCompletion *InlineCompletionClientCapabilities `json:"inlineCompletion,omitempty"`
}

type NotebookDocumentClientCapabilities struct {
	/**
	 * Capabilities specific to notebook document synchronization
	 * @since 3.17.0
	 */
	Synchronization NotebookDocumentSyncClientCapabilities `json:"synchronization"`
}

type WindowClientCapabilities struct {
	/**
	 * It indicates whether the client supports server initiated
	 * progress using the `window/workDoneProgress/create` request.
	 * The capability also controls Whether client supports handling
	 * of progress notifications. If set servers are allowed to report a
	 * `workDoneProgress` property in the request specific server
	 * capabilities.
	 * @since 3.15.0
	 */
	WorkDoneProgress *bool `json:"workDoneProgress,omitempty"`
	/**
	 * Capabilities specific to the showMessage request.
	 * @since 3.16.0
	 */
	ShowMessage *ShowMessageRequestClientCapabilities `json:"showMessage,omitempty"`
	/**
	 * Capabilities specific to the showDocument request.
	 * @since 3.16.0
	 */
	ShowDocument *ShowDocumentClientCapabilities `json:"showDocument,omitempty"`
}

type GeneralClientCapabilities struct {
	/**
	 * Client capability that signals how the client
	 * handles stale requests (e.g. a request
	 * for which the client will not process the response
	 * anymore since the information is outdated).
	 * @since 3.17.0
	 */
	StaleRequestSupport *StaleRequestSupportOptions `json:"staleRequestSupport,omitempty"`
	/**
	 * Client capabilities specific to regular expressions.
	 * @since 3.16.0
	 */
	RegularExpressions *RegularExpressionsClientCapabilities `json:"regularExpressions,omitempty"`
	/**
	 * Client capabilities specific to the client's markdown parser.
	 * @since 3.16.0
	 */
	Markdown *MarkdownClientCapabilities `json:"markdown,omitempty"`
	/**
	 * The position encodings supported by the client. Client and server
	 * have to agree on the same position encoding to ensure that offsets
	 * (e.g. character position in a line) are interpreted the same on both
	 * sides.
	 * To keep the protocol backwards compatible the following applies: if
	 * the value 'utf-16' is missing from the array of position encodings
	 * servers can assume that the client supports UTF-16. UTF-16 is
	 * therefore a mandatory encoding.
	 * If omitted it defaults to ['utf-16'].
	 * Implementation considerations: since the conversion from one encoding
	 * into another requires the content of the file / line the conversion
	 * is best done where the file is read which is usually on the server
	 * side.
	 * @since 3.17.0
	 */
	PositionEncodings []PositionEncodingKind `json:"positionEncodings,omitempty"`
}

type TextDocumentFilterClientCapabilities struct {
	/**
	 * The client supports Relative Patterns.
	 * @since 3.18.0
	 */
	RelativePatternSupport *bool `json:"relativePatternSupport,omitempty"`
}

type WorkspaceClientCapabilities struct {
	/**
	 * The client supports applying batch edits
	 * to the workspace by supporting the request
	 * 'workspace/applyEdit'
	 */
	ApplyEdit *bool `json:"applyEdit,omitempty"`
	// Capabilities specific to `WorkspaceEdit`s
	WorkspaceEdit *WorkspaceEditClientCapabilities `json:"workspaceEdit,omitempty"`
	/**
	 * Capabilities specific to the `workspace/didChangeConfiguration`
	 * notification.
	 */
	DidChangeConfiguration *DidChangeConfigurationClientCapabilities `json:"didChangeConfiguration,omitempty"`
	/**
	 * Capabilities specific to the `workspace/didChangeWatchedFiles`
	 * notification.
	 */
	DidChangeWatchedFiles *DidChangeWatchedFilesClientCapabilities `json:"didChangeWatchedFiles,omitempty"`
	// Capabilities specific to the `workspace/symbol` request.
	Symbol *WorkspaceSymbolClientCapabilities `json:"symbol,omitempty"`
	// Capabilities specific to the `workspace/executeCommand` request.
	ExecuteCommand *ExecuteCommandClientCapabilities `json:"executeCommand,omitempty"`
	/**
	 * The client has support for workspace folders.
	 * @since 3.6.0
	 */
	WorkspaceFolders *bool `json:"workspaceFolders,omitempty"`
	/**
	 * The client supports `workspace/configuration` requests.
	 * @since 3.6.0
	 */
	Configuration *bool `json:"configuration,omitempty"`
	/**
	 * Capabilities specific to the semantic token requests scoped to the
	 * workspace.
	 * @since 3.16.0
	 */
	SemanticTokens *SemanticTokensWorkspaceClientCapabilities `json:"semanticTokens,omitempty"`
	/**
	 * Capabilities specific to the code lens requests scoped to the
	 * workspace.
	 * @since 3.16.0
	 */
	CodeLens *CodeLensWorkspaceClientCapabilities `json:"codeLens,omitempty"`
	/**
	 * The client has support for file requests/notifications.
	 * @since 3.16.0
	 */
	FileOperations *FileOperationClientCapabilities `json:"fileOperations,omitempty"`
	/**
	 * Client workspace capabilities specific to inline values.
	 * @since 3.17.0
	 */
	InlineValue *InlineValueWorkspaceClientCapabilities `json:"inlineValue,omitempty"`
	/**
	 * Client workspace capabilities specific to inlay hints.
	 * @since 3.17.0
	 */
	InlayHint *InlayHintWorkspaceClientCapabilities `json:"inlayHint,omitempty"`
	/**
	 * Client workspace capabilities specific to diagnostics.
	 * @since 3.17.0.
	 */
	Diagnostics *DiagnosticWorkspaceClientCapabilities `json:"diagnostics,omitempty"`
	/**
	 * Capabilities specific to the folding range requests
	 * scoped to the workspace.
	 * @since 3.18.0
	 */
	FoldingRange *FoldingRangeWorkspaceClientCapabilities `json:"foldingRange,omitempty"`
	/**
	 * Capabilities specific to the `workspace/textDocumentContent`
	 * request.
	 * @since 3.18.0
	 */
	TextDocumentContent *TextDocumentContentClientCapabilities `json:"textDocumentContent,omitempty"`
}

type FileOperationClientCapabilities struct {
	/**
	 * Whether the client supports dynamic registration for
	 * file requests/notifications.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
	// The client has support for sending didCreateFiles notifications.
	DidCreate *bool `json:"didCreate,omitempty"`
	// The client has support for sending willCreateFiles requests.
	WillCreate *bool `json:"willCreate,omitempty"`
	// The client has support for sending didRenameFiles notifications.
	DidRename *bool `json:"didRename,omitempty"`
	// The client has support for sending willRenameFiles requests.
	WillRename *bool `json:"willRename,omitempty"`
	// The client has support for sending didDeleteFiles notifications.
	DidDelete *bool `json:"didDelete,omitempty"`
	// The client has support for sending willDeleteFiles requests.
	WillDelete *bool `json:"willDelete,omitempty"`
}

type StaleRequestSupportOptions struct {
	// The client will actively cancel the request.
	Cancel bool `json:"cancel"`
	/**
	 * The list of requests for which the client
	 * will retry the request if it receives a
	 * response with error code `ContentModified`
	 */
	RetryOnContentModified []string `json:"retryOnContentModified"`
}

type ServerCapabilities struct {
	/**
	 * The position encoding the server picked from the encodings offered
	 * by the client via the client capability `general.positionEncodings`.
	 * If the client didn't provide any position encodings the only valid
	 * value that a server can return is 'utf-16'.
	 * If omitted it defaults to 'utf-16'.
	 * @since 3.17.0
	 */
	PositionEncoding *PositionEncodingKind `json:"positionEncoding,omitempty"`
	/**
	 * Defines how text documents are synced. Is either a detailed structure
	 * defining each notification or for backwards compatibility the
	 * TextDocumentSyncKind number. If omitted it defaults to
	 * `TextDocumentSyncKind.None`.
	 */
	TextDocumentSync *rpc.Union2[TextDocumentSyncOptions, TextDocumentSyncKind] `json:"textDocumentSync,omitempty"`
	/**
	 * Defines how notebook documents are synced.
	 * @since 3.17.0
	 */
	NotebookDocumentSync *rpc.Union2[NotebookDocumentSyncOptions, NotebookDocumentSyncRegistrationOptions] `json:"notebookDocumentSync,omitempty"`
	// The server provides completion support.
	CompletionProvider *CompletionOptions `json:"completionProvider,omitempty"`
	// The server provides hover support.
	HoverProvider *rpc.Union2[bool, HoverOptions] `json:"hoverProvider,omitempty"`
	// The server provides signature help support.
	SignatureHelpProvider *SignatureHelpOptions `json:"signatureHelpProvider,omitempty"`
	/**
	 * The server provides go to declaration support.
	 * @since 3.14.0
	 */
	DeclarationProvider any `json:"declarationProvider,omitempty"` // boolean | DeclarationOptions | DeclarationRegistrationOptions
	// The server provides goto definition support.
	DefinitionProvider *rpc.Union2[bool, DefinitionOptions] `json:"definitionProvider,omitempty"`
	/**
	 * The server provides goto type definition support.
	 * @since 3.6.0
	 */
	TypeDefinitionProvider any `json:"typeDefinitionProvider,omitempty"` // boolean | TypeDefinitionOptions | TypeDefinitionRegistrationOptions
	/**
	 * The server provides goto implementation support.
	 * @since 3.6.0
	 */
	ImplementationProvider any `json:"implementationProvider,omitempty"` // boolean | ImplementationOptions | ImplementationRegistrationOptions
	// The server provides find references support.
	ReferencesProvider *rpc.Union2[bool, ReferenceOptions] `json:"referencesProvider,omitempty"`
	// The server provides document highlight support.
	DocumentHighlightProvider *rpc.Union2[
		bool, DocumentHighlightOptions,
	] `json:"documentHighlightProvider,omitempty"`
	// The server provides document symbol support.
	DocumentSymbolProvider *rpc.Union2[bool, DocumentSymbolOptions] `json:"documentSymbolProvider,omitempty"`
	/**
	 * The server provides code actions. The `CodeActionOptions` return type is
	 * only valid if the client signals code action literal support via the
	 * property `textDocument.codeAction.codeActionLiteralSupport`.
	 */
	CodeActionProvider *rpc.Union2[bool, CodeActionOptions] `json:"codeActionProvider,omitempty"`
	// The server provides code lens.
	CodeLensProvider *CodeLensOptions `json:"codeLensProvider,omitempty"`
	// The server provides document link support.
	DocumentLinkProvider *DocumentLinkOptions `json:"documentLinkProvider,omitempty"`
	/**
	 * The server provides color provider support.
	 * @since 3.6.0
	 */
	ColorProvider any `json:"colorProvider,omitempty"` // boolean | DocumentColorOptions | DocumentColorRegistrationOptions
	// The server provides document formatting.
	DocumentFormattingProvider *rpc.Union2[bool, DocumentFormattingOptions] `json:"documentFormattingProvider,omitempty"`
	// The server provides document range formatting.
	DocumentRangeFormattingProvider *rpc.Union2[
		bool, DocumentRangeFormattingOptions,
	] `json:"documentRangeFormattingProvider,omitempty"`
	// The server provides document formatting on typing.
	DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`
	/**
	 * The server provides rename support. RenameOptions may only be
	 * specified if the client states that it supports
	 * `prepareSupport` in its initial `initialize` request.
	 */
	RenameProvider *rpc.Union2[bool, RenameOptions] `json:"renameProvider,omitempty"`
	/**
	 * The server provides folding provider support.
	 * @since 3.10.0
	 */
	FoldingRangeProvider any `json:"foldingRangeProvider,omitempty"` // boolean | FoldingRangeOptions | FoldingRangeRegistrationOptions
	// The server provides execute command support.
	ExecuteCommandProvider *ExecuteCommandOptions `json:"executeCommandProvider,omitempty"`
	/**
	 * The server provides selection range support.
	 * @since 3.15.0
	 */
	SelectionRangeProvider any `json:"selectionRangeProvider,omitempty"` // boolean | SelectionRangeOptions | SelectionRangeRegistrationOptions
	/**
	 * The server provides linked editing range support.
	 * @since 3.16.0
	 */
	LinkedEditingRangeProvider any `json:"linkedEditingRangeProvider,omitempty"` // boolean | LinkedEditingRangeOptions | LinkedEditingRangeRegistrationOptions
	/**
	 * The server provides call hierarchy support.
	 * @since 3.16.0
	 */
	CallHierarchyProvider any `json:"callHierarchyProvider,omitempty"` // boolean | CallHierarchyOptions | CallHierarchyRegistrationOptions
	/**
	 * The server provides semantic tokens support.
	 * @since 3.16.0
	 */
	SemanticTokensProvider *rpc.Union2[
		SemanticTokensOptions, SemanticTokensRegistrationOptions,
	] `json:"semanticTokensProvider,omitempty"`
	/**
	 * Whether server provides moniker support.
	 * @since 3.16.0
	 */
	MonikerProvider any `json:"monikerProvider,omitempty"` // boolean | MonikerOptions | MonikerRegistrationOptions
	/**
	 * The server provides type hierarchy support.
	 * @since 3.17.0
	 */
	TypeHierarchyProvider any `json:"typeHierarchyProvider,omitempty"` // boolean | TypeHierarchyOptions | TypeHierarchyRegistrationOptions
	/**
	 * The server provides inline values.
	 * @since 3.17.0
	 */
	InlineValueProvider any `json:"inlineValueProvider,omitempty"` // boolean | InlineValueOptions | InlineValueRegistrationOptions
	/**
	 * The server provides inlay hints.
	 * @since 3.17.0
	 */
	InlayHintProvider any `json:"inlayHintProvider,omitempty"` // boolean | InlayHintOptions | InlayHintRegistrationOptions
	/**
	 * The server has support for pull model diagnostics.
	 * @since 3.17.0
	 */
	DiagnosticProvider *rpc.Union2[
		DiagnosticOptions, DiagnosticRegistrationOptions,
	] `json:"diagnosticProvider,omitempty"`
	// The server provides workspace symbol support.
	WorkspaceSymbolProvider *rpc.Union2[
		bool, WorkspaceSymbolOptions,
	] `json:"workspaceSymbolProvider,omitempty"`
	/**
	 * The server provides inline completions.
	 * @since 3.18.0
	 */
	InlineCompletionProvider *rpc.Union2[bool, InlineCompletionOptions] `json:"inlineCompletionProvider,omitempty"`
	// Workspace specific server capabilities
	Workspace *WorkspaceOptions `json:"workspace,omitempty"`
	// Experimental server capabilities.
	Experimental any `json:"experimental,omitempty"`
}

type WorkspaceOptions struct {
	/**
	 * The server supports workspace folder.
	 * @since 3.6.0
	 */
	WorkspaceFolders *WorkspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
	/**
	 * The server is interested in notifications/requests for operations on files.
	 * @since 3.16.0
	 */
	FileOperations *FileOperationOptions `json:"fileOperations,omitempty"`
	/**
	 * The server supports the `workspace/textDocumentContent` request.
	 * @since 3.18.0
	 */
	TextDocumentContent *rpc.Union2[
		TextDocumentContentOptions, TextDocumentContentRegistrationOptions,
	] `json:"textDocumentContent,omitempty"`
}

type FileOperationOptions struct {
	// The server is interested in receiving didCreateFiles notifications.
	DidCreate *FileOperationRegistrationOptions `json:"didCreate,omitempty"`
	// The server is interested in receiving willCreateFiles requests.
	WillCreate *FileOperationRegistrationOptions `json:"willCreate,omitempty"`
	// The server is interested in receiving didRenameFiles notifications.
	DidRename *FileOperationRegistrationOptions `json:"didRename,omitempty"`
	// The server is interested in receiving willRenameFiles requests.
	WillRename *FileOperationRegistrationOptions `json:"willRename,omitempty"`
	// The server is interested in receiving didDeleteFiles file notifications.
	DidDelete *FileOperationRegistrationOptions `json:"didDelete,omitempty"`
	// The server is interested in receiving willDeleteFiles file requests.
	WillDelete *FileOperationRegistrationOptions `json:"willDelete,omitempty"`
}

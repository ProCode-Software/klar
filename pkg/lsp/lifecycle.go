package lsp

// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.18/specification/#initializeParams
type InitializeParams struct {
	WorkDoneProgressParams
	/**
	 * The process Id of the parent process that started the server. Is null if
	 * the process has not been started by another process. If the parent
	 * process is not alive then the server should exit (see exit notification)
	 * its process.
	 */
	ProcessID int `json:"processId,omitzero"` // integer | null
	/**
	 * Information about the client
	 *
	 * @since 3.15.0
	 */
	ClientInfo *ClientInfo `json:"clientInfo,omitempty"`
	/**
	 * The locale the client is currently showing the user interface
	 * in. This must not necessarily be the locale of the operating system.
	 *
	 * Uses IETF language tags as the value's syntax
	 * (See https://en.wikipedia.org/wiki/IETF_language_tag)
	 *
	 * @since 3.16.0
	 */
	Locale string `json:"locale,omitempty"`
	/**
	 * The rootPath of the workspace. Is null if no folder is open.
	 *
	 * @deprecated in favour of `rootUri`.
	 */
	RootPath string `json:"rootPath,omitzero"` // string | null
	// User provided initialization options.
	InitializationOptions any `json:"initializationOptions,omitempty"`
	// The capabilities provided by the client (editor or tool)
	Capabilities *ClientCapabilities `json:"capabilities"`
	// The initial trace setting. If omitted trace is disabled ('off').
	Trace TraceValue `json:"trace,omitempty"`
	/**
	 * The workspace folders configured in the client when the server starts.
	 * This property is only available if the client supports workspace folders.
	 * It can be `null` if the client supports workspace folders but none are
	 * configured.
	 *
	 * @since 3.6.0
	 */
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders,omitzero"` // WorkspaceFolder[] | null
}

/**
 * Information about the client
 *
 * @since 3.15.0
 */
type ClientInfo struct {
	// The name of the client as defined by the client.
	Name string `json:"name"`
	// The client's version as defined by the client.
	Version string `json:"version,omitempty"`
}

type InitializeResult struct {
	// The capabilities the language server provides.
	Capabilities *ServerCapabilities `json:"capabilities"`
	/**
	 * Information about the server.
	 *
	 * @since 3.15.0
	 */
	ServerInfo *ServerInfo `json:"serverInfo,omitempty"`
}

/**
 * Information about the server
 *
 * @since 3.15.0
 */
type ServerInfo struct {
	// The name of the server as defined by the server.
	Name string `json:"name"`
	// The server's version as defined by the server.
	Version string `json:"version,omitempty"`
}

/**
 * Known error codes for an `InitializeErrorCodes`;
 */
type InitializeErrorCodes int

type InitializeError struct {
	/**
	 * Indicates whether the client execute the following retry logic:
	 * (1) show the message provided by the ResponseError to the user
	 * (2) user selects retry or cancel
	 * (3) if user selected retry the initialize method is sent again.
	 */
	Retry bool `json:"retry"`
}

type TraceValue string

const (
	TraceOff      TraceValue = "off"
	TraceMessages TraceValue = "messages"
	TraceVerbose  TraceValue = "verbose"
)

type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

package rpc

type ErrorCode int32

// Defined by JSON-RPC
const (
	ParseError     ErrorCode = -32700
	InvalidRequest ErrorCode = -32600
	MethodNotFound ErrorCode = -32601
	InvalidParams  ErrorCode = -32602
	InternalError  ErrorCode = -32603
)

const (
	/**
	 * This is the start range of JSON-RPC reserved error codes.
	 * It doesn't denote a real error code. No LSP error codes should
	 * be defined between the start and end range. For backwards
	 * compatibility the `ServerNotInitialized` and the `UnknownErrorCode`
	 * are left in the range.
	 *
	 * @since 3.16.0
	 */
	JSONRPCReservedErrorRangeStart ErrorCode = -32099
	/** @deprecated use jsonrpcReservedErrorRangeStart */
	ServerErrorStart = JSONRPCReservedErrorRangeStart

	/**
	 * Error code indicating that a server received a notification or
	 * request before the server received the `initialize` request.
	 */
	ServerNotInitialized ErrorCode = -32002
	UnknownErrorCode     ErrorCode = -32001

	/**
	 * This is the end range of JSON-RPC reserved error codes.
	 * It doesn't denote a real error code.
	 *
	 * @since 3.16.0
	 */
	JSONRPCReservedErrorRangeEnd ErrorCode = -32000
	/** @deprecated use jsonrpcReservedErrorRangeEnd */
	ServerErrorEnd ErrorCode = JSONRPCReservedErrorRangeEnd

	/**
	 * This is the start range of LSP reserved error codes.
	 * It doesn't denote a real error code.
	 *
	 * @since 3.16.0
	 */
	LSPReservedErrorRangeStart ErrorCode = -32899

	/**
	 * A request failed but it was syntactically correct, e.g the
	 * method name was known and the parameters were valid. The error
	 * message should contain human readable information about why
	 * the request failed.
	 *
	 * @since 3.17.0
	 */
	RequestFailed ErrorCode = -32803

	/**
	 * The server cancelled the request. This error code should
	 * only be used for requests that explicitly support being
	 * server cancellable.
	 *
	 * @since 3.17.0
	 */
	ServerCancelled ErrorCode = -32802

	/**
	 * The server detected that the content of a document got
	 * modified outside normal conditions. A server should
	 * NOT send this error code if it detects a content change
	 * in its unprocessed messages. The result even computed
	 * on an older state might still be useful for the client.
	 *
	 * If a client decides that a result is not of any use anymore
	 * the client should cancel the request.
	 */
	ContentModified ErrorCode = -32801

	/**
	 * The client has canceled a request and a server has detected
	 * the cancel.
	 */
	RequestCancelled ErrorCode = -32800

	/**
	 * This is the end range of LSP reserved error codes.
	 * It doesn't denote a real error code.
	 *
	 * @since 3.16.0
	 */
	LSPReservedErrorRangeEnd ErrorCode = -32800
)

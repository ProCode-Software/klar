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
)

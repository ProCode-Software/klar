package lsp

type TextDocumentSyncKind int

const (
	/**
	 * Documents should not be synced at all.
	 */
	SyncNone TextDocumentSyncKind = 0
	/**
	 * Documents are synced by always sending the full content
	 * of the document.
	 */
	SyncFull TextDocumentSyncKind = 1
	/**
	 * Documents are synced by sending the full content on open.
	 * After that only incremental updates to the document are
	 * sent.
	 */
	SyncIncremental TextDocumentSyncKind = 2
)

type TextDocumentSyncOptions struct {
	/**
	 * Open and close notifications are sent to the server. If omitted open
	 * close notifications should not be sent.
	 */
	OpenClose *bool `json:"openClose,omitempty"`

	/**
	 * Change notifications are sent to the server. See
	 * TextDocumentSyncKind.None, TextDocumentSyncKind.Full and
	 * TextDocumentSyncKind.Incremental. If omitted it defaults to
	 * TextDocumentSyncKind.None.
	 */
	Change *TextDocumentSyncKind `json:"change,omitempty"`
}

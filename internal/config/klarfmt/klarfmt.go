package klarfmt

type Config struct {
	// Rules for rewriting syntax in Klar files. If set to 'false', KlarRewrite
	// will be disabled.
	Rewrite *RewriteRules
	// Options for formatting Klon files.
	Klon *KlonOptions
	// Maximum amount of characters on a line before wrapping. Set to 'none' for
	// no limit. The minimum limit is `10`.
	LineLimit int
	// The quote style to use for strings.
	QuoteStyle QuoteStyle
	// Options for contextual string quoting. Set to `false` to disable.
	AdvancedQuoting *AdvancedQuotingOptions
	// Options for normalizing comments.
	Comments *CommentOptions
	// Files to exclude from formatting. Glob patterns are supported.
	Exclude []string
}

type KlonOptions struct {
	// Sort objects by key A-Z order. For 'glas.pack' manifests, different
	// sorting rules are used.
	SortObjects bool
	// Preserve up to 1 provided newline between object keys. If disabled, the
	// formatter will always print entries without newlines between.
	PreserveNewlines bool
}

type QuoteStyle int

const (
	SingleQuote QuoteStyle = iota
	DoubleQuote
	Backtick
)

type RewriteRules struct{}

type AdvancedQuotingOptions struct {
	// Use single quotes for strings with a single character.
	SingleQuoteForChars bool
	// Switch to a different quoting style to avoid escaping the current quote character.
	AvoidEscaping bool
	// Use the wrap-string format for strings that exceed the line limit.
	WrapString bool
	// Prefer string interpolations ("{...}") rather than concatenation ('...' + '...').
	// If enabled, concatenation of more than 2 strings will be replaced with an
	// interpolated string.
	//
	// Note: Casts to `String(...)` won't be removed; use KlarRewrite to simplify them.
	PreferInterpolation bool
}

type CommentStyle uint8

const (
	InputComment CommentStyle = iota
	LineComment
	BlockComment
)

type CommentOptions struct {
	// The comment style to use for multiline comments, including consecutive
	// line comments. This also applies to doc comments.
	Multiline CommentStyle
	// The comment style to use for top-level doc comments.
	Doc CommentStyle
	// The comment style to use for the module doc comment ('///' or '/**'
	// at the top of the file).
	Module CommentStyle
}

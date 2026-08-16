package lsp

/**
 * A type indicating how positions are encoded,
 * specifically what column offsets mean.
 *
 * @since 3.17.0
 */
type PositionEncodingKind string

/**
 * A set of predefined position encoding kinds.
 *
 * @since 3.17.0
 */
const (

	/**
	 * Character offsets count UTF-8 code units (i.e. bytes).
	 */
	UTF8 PositionEncodingKind = "utf-8"

	/**
	 * Character offsets count UTF-16 code units.
	 *
	 * This is the default and must always be supported
	 * by servers.
	 */
	UTF16 PositionEncodingKind = "utf-16"

	/**
	 * Character offsets count UTF-32 code units.
	 *
	 * Implementation note: these are the same as Unicode code points,
	 * so this `PositionEncodingKind` may also be used for an
	 * encoding-agnostic representation of character offsets.
	 */
	UTF32 PositionEncodingKind = "utf-32"
)

// Package char provides cached characters for repeating.
package char

var (
	Spaces       = []byte("                                ")
	SingleQuotes = []byte("''''''''''''''''''''''''''''''''")
	DoubleQuotes = []byte(`""""""""""""""""""""""""""""""""`)
	Backticks    = []byte("````````````````````````````````")
	Slashes      = []byte("////////////////////////////////")
	Length       = len(Spaces) // 32

	QuoteMap = map[byte][]byte{
		'"': DoubleQuotes, '\'': SingleQuotes, '/': Slashes, '`': Backticks, ' ': Spaces,
	}
)

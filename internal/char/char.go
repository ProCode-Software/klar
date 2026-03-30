// Package char provides cached characters for repeating.
package char

import (
	"bytes"
	"slices"
)

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

func Repeat(r byte, n int) []byte {
	rep := QuoteMap[r]
	if rep != nil && n <= len(rep) {
		return rep[:n]
	}
	// More than 32 chars
	arr := make([]byte, n)
	copy(arr, rep)
	for i := len(rep); i < n; i++ {
		arr[i] = r
	}
	QuoteMap[r] = slices.Clone(arr)
	return arr
}

func RepeatRune(r rune, n int) []byte {
	return bytes.Repeat([]byte(string(r)), n)
}


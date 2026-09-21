package codegen

import (
	"math/rand/v2"

	"github.com/ProCode-Software/klar/internal/ir/jsir"
)

func randomName(n int) string {
	res := make([]byte, n)
	for i := range n {
		if num := rand.N(52); num >= 26 {
			res[i] = byte('A' + num - 26)
		} else {
			res[i] = byte('a' + num)
		}
	}
	return string(res)
}

// generateLetter generates a letter name for use as an identifier.
// The result will never be a JavaScript keyword.
func generateLetter(i int) string {
	// TODO: Make this return 'aa', 'ab', ... after 'z' (use math.Log)
	ln := i/26 + 1
	res := make([]byte, ln)
	for j := range ln {
		res[j] = byte('a' + i%26)
		if len(res) >= 2 {
			// Choose a different name if it collides with a keyword like 'if'
			differentLetter := 'a' + i%26
			for {
				if _, ok := jsir.JSKeywords[string(res)]; !ok {
					break
				}
				differentLetter++
				// If we somehow go above Z, try again with capital letters.
				// This should never happen. Please let us know if otherwise.
				if differentLetter > 'z' {
					differentLetter = 'A'
				}
				res[j] = byte(differentLetter)
			}
		}
	}
	return string(res)
}

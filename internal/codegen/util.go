package codegen

import "math/rand/v2"

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

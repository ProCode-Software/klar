package ansi

import "fmt"

func BackgroundBit8(c int, s string) string {
	if DisableColor {
		return ""
	}
	return fmt.Sprintf("\x1b[48;5;%dm", c) + s + CodeReset
}

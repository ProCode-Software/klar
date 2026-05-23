package ansi

import "strings"

// space is either "38" (foreground) or "48" (background).
func gradient(space string, text string, colors ...[3]int) string {
	esc := func(color [3]int) string {
		return RGBSpace(space, color[0], color[1], color[2])
	}
	if DisableColor || text == "" || len(colors) == 0 {
		return text
	}
	chars := []rune(text)
	charCt := len(chars)
	if len(colors) == 1 || charCt == 1 {
		return esc(colors[0]) + text + CodeReset
	}

	var (
		b         strings.Builder
		textSteps = charCt - 1
		colorCt   = len(colors)
	)
	for i, r := range chars {
		// Find the two adjacent colors to interpolate between
		var (
			colorOffset = i * (colorCt - 1)
			colorI      = colorOffset / textSteps
			colorRatio  = colorOffset % textSteps
		)
		if colorI >= colorCt-1 {
			colorI = colorCt - 2
			colorRatio = textSteps
		}
		currColor := colors[colorI]
		nextColor := colors[colorI+1]

		// Interpolate each RGB channel
		interpolated := [3]int{
			currColor[0] + (nextColor[0]-currColor[0])*colorRatio/textSteps,
			currColor[1] + (nextColor[1]-currColor[1])*colorRatio/textSteps,
			currColor[2] + (nextColor[2]-currColor[2])*colorRatio/textSteps,
		}
		b.WriteString(esc(interpolated))
		b.WriteRune(r)
	}
	// Reset at the end
	b.WriteString(CodeReset)
	return b.String()
}

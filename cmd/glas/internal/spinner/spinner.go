package spinner

import (
	"fmt"
	"time"

	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

func Circle(text string, done <-chan struct{}) {
	// Colourful palette
	/* colorRanges := [...][2]int{
		{161, 160}, {204, 202}, // Red
		{178, 181}, {211, 208}, {214, 217}, // Orange
		{184, 187},                               // Yellow
		{36, 39}, {42, 45}, {48, 51}, {148, 151}, // Green
		{73, 75}, {109, 111}, {116, 117}, // Blue
		{96, 99},   // Purple
		{165, 160}, // Purple - Red
	} */
	// Muted palette
	colorRanges := [...][2]int{
		{204, 202}, {178, 181}, {148, 153}, // Red, Yellow, Green - Blue
		{110, 111}, {105, 105}, // Blue
		{171, 169}, {213, 208}, // Purple, Purple - Red
	}
	groupI, color := 0, colorRanges[0][0]
	nextColor := func() int {
		currColor := color
		group := colorRanges[groupI]
		groupStart, groupEnd := group[0], group[1]
		switch {
		case color == groupEnd:
			// End of group
			groupI = (groupI + 1) % len(colorRanges)
			color = colorRanges[groupI][0]
		case groupStart < groupEnd:
			color++
		default:
			color-- // Backwards group
		}
		return currColor
	}
	const first, count = '◐', 6
	var i int
	for {
		select {
		case <-done:
			return
		default:
			i = (i + 1) % count
			r := rune(first + i)
			fmt.Printf("%s %s\r", ansi.Bit8(nextColor(), string(r)), text)
			time.Sleep(time.Millisecond * 75)
		}
	}
}

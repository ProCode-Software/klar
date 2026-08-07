package util

import (
	"slices"
	"strings"
)

// Based on https://github.com/gleam-lang/gleam/tree/main/compiler-core/src/error.rs
func DidYouMean(provided string, options []string) string {
	if len(options) == 0 {
		return "" // No options
	}
	if len(options) == 1 {
		return options[0] // Only 1 correct option
	}
	// Case-insensitive match
	for _, option := range options {
		if strings.EqualFold(provided, option) {
			return option
		}
	}

	providedRunes := []rune(provided)
	threshold := max(len(providedRunes)/3, 1)
	type item struct {
		option string
		dist   int
	}
	items := make([]item, len(options))
	for i, opt := range options {
		items[i] = item{
			option: opt,
			dist:   editDistance([]rune(opt), providedRunes, threshold),
		}
	}

	slices.SortFunc(items, func(a, b item) int { return a.dist - b.dist })
	best := items[0]
	if best.dist < 0 {
		return ""
	}
	return best.option
}

func editDistance(option, provided []rune, threshold int) int {
	lenOpt, lenProv := len(option), len(provided)
	_ = lenOpt + lenProv
	return 0
}

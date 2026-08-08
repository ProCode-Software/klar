package util_test

import (
	"strconv"
	"testing"

	"github.com/ProCode-Software/klar/internal/util"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{-128, "-128"},
		{4096, "4096"},
		{-1200, "-1200"},
		{10_288, "10,288"},
		{123_456, "123,456"},
		{-123_892, "-123,892"},
		{1_467_300, "1,467,300"},
		{-267_408_917, "-267,408,917"},
		{1_234_567_890_123, "1,234,567,890,123"},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.input), func(t *testing.T) {
			got := util.FormatNumber(tt.input)
			if got != tt.expected {
				t.Errorf("FormatNumber() = %v, want %v", got, tt.expected)
			}
		})
	}
}

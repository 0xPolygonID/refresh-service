package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertID(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		expected string
	}{
		{
			name:     "Old id format with URL info",
			arg:      "https://example.com/e342def6-620e-4394-8ea1-7448ea81bb72",
			expected: "e342def6-620e-4394-8ea1-7448ea81bb72",
		},
		{
			name:     "New format with urn:uuid: prefix",
			arg:      "urn:uuid:e342def6-620e-4394-8ea1-7448ea81bb72",
			expected: "e342def6-620e-4394-8ea1-7448ea81bb72",
		},
		{
			name:     "Clear format",
			arg:      "e342def6-620e-4394-8ea1-7448ea81bb72",
			expected: "e342def6-620e-4394-8ea1-7448ea81bb72",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := convertID(tt.arg)
			require.Equal(t, tt.expected, actual)
		})
	}
}

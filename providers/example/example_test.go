package example

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBalanceByAddress(t *testing.T) {
	tests := []struct {
		name    string
		want    map[string]interface{}
		address string
	}{
		{
			name: "Get balance by address",
			want: map[string]interface{}{
				"address": "0x1234567890",
			},
			address: "0x1234567890",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBalanceByAddress(tt.address)
			require.NoError(t, err)
			require.Equal(t, tt.want["address"], got["address"])
			require.NotZero(t, got["balance"])
		})
	}
}

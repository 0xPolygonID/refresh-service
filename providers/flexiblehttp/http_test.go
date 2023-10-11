package flexiblehttp

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name              string
		credentialType    string
		pathToTestVector  string
		credentialSubject map[string]interface{}
		expectedURL       string
		expectedMethod    string
		expectedHeaders   http.Header
	}{
		{
			name:             "Balance testvector",
			credentialType:   "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/balance.json-ld#Balance",
			pathToTestVector: "./testvectors/balance.yaml",
			credentialSubject: map[string]interface{}{
				"address":  "0x6ae7E07c8763C284B7C91371f934E46c766D0ec6",
				"currency": "MATIC",
			},
			expectedURL:    "https://api-testnet.polygonscan.com/api/currency/MATIC?module=account&action=balance&address=0x6ae7E07c8763C284B7C91371f934E46c766D0ec6&apikey=RET2WHC1B3UDM9PQQ12ZUG2ZE289D1TCY9",
			expectedMethod: "GET",
			expectedHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewFactoryFlexibleHTTP(tt.pathToTestVector, nil)
			require.NoError(t, err)
			provider, err := factory.ProduceFlexibleHTTP(tt.credentialType)
			require.NoError(t, err)
			request, err := provider.BuildRequest(tt.credentialSubject)
			require.NoError(t, err)

			compareURLs(t, tt.expectedURL, request.URL.String())
			require.Equal(t, tt.expectedMethod, request.Method)
			require.Equal(t, tt.expectedHeaders, request.Header)
		})
	}
}

func compareURLs(t *testing.T, expected string, actual string) {
	urlx, err := url.Parse(expected)
	require.NoError(t, err)
	urly, err := url.Parse(actual)
	require.NoError(t, err)

	require.Equal(t, urlx.Scheme, urly.Scheme)
	require.Equal(t, urlx.Host, urly.Host)
	require.Equal(t, urlx.Path, urly.Path)
	require.Equal(t, urlx.Query(), urly.Query())
}

func TestDecodeResponse(t *testing.T) {
	tests := []struct {
		name                  string
		credentialType        string
		pathToTestVector      string
		responseBody          []byte
		expectedUpdatedFields map[string]interface{}
	}{
		{
			name:             "One level response",
			credentialType:   "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/balance.json-ld#Balance",
			pathToTestVector: "./testvectors/balance.yaml",
			responseBody: []byte(`{
				"status": "1",
				"message": "OK",
				"result": "1200145884000"
			}`),
			expectedUpdatedFields: map[string]interface{}{
				"balance": "1200145884000",
			},
		},
		{
			name:             "Embeded json in response",
			credentialType:   "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/balance.json-ld#DeepEmbeded",
			pathToTestVector: "./testvectors/balance.yaml",
			responseBody: []byte(`{
				"wallet": {
					"eth": {
						"balance": "1200145884000"
					}
				}
			}`),
			expectedUpdatedFields: map[string]interface{}{
				"balance": "1200145884000",
			},
		},
		{
			name:             "Embeded array of objects in response",
			credentialType:   "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/balance.json-ld#EmbededArray",
			pathToTestVector: "./testvectors/balance.yaml",
			responseBody: []byte(`{
				"wallet": {
					"eth": [{
						"balance": "1200145884000"
					}]
				}
			}`),
			expectedUpdatedFields: map[string]interface{}{
				"balance": "1200145884000",
			},
		},
		{
			name:             "Embeded array of values in response",
			credentialType:   "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/balance.json-ld#EmbededValuesArray",
			pathToTestVector: "./testvectors/balance.yaml",
			responseBody: []byte(`{
				"wallet": {
					"eth": ["1200145884000"]
				}
			}`),
			expectedUpdatedFields: map[string]interface{}{
				"balance": "1200145884000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory, err := NewFactoryFlexibleHTTP(tt.pathToTestVector, nil)
			require.NoError(t, err)
			provider, err := factory.ProduceFlexibleHTTP(tt.credentialType)
			require.NoError(t, err)
			body := bytes.NewReader(tt.responseBody)
			updatedFields, err := provider.DecodeResponse(body, "json")
			require.NoError(t, err)

			require.Equal(t, tt.expectedUpdatedFields, updatedFields)
		})
	}
}

package flexiblehttp

import (
	"bytes"
	"encoding/json"
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
			var response map[string]interface{}
			err = json.NewDecoder(bytes.NewReader(tt.responseBody)).Decode(&response)
			require.NoError(t, err)
			updatedFields, err := provider.DecodeResponse(response)
			require.NoError(t, err)

			require.Equal(t, tt.expectedUpdatedFields, updatedFields)
		})
	}
}

func TestCastToType(t *testing.T) {
	tests := []struct {
		name        string
		jsonBody    string
		convertType string
		expected    interface{}
	}{
		// string
		{
			name:        "string to string",
			jsonBody:    `{"data": "test"}`,
			convertType: "string",
			expected:    "test",
		},
		{
			name:        "string to int",
			jsonBody:    `{"data": "123"}`,
			convertType: "integer",
			expected:    123,
		},
		{
			name:        "string to float",
			jsonBody:    `{"data": "123.0000001"}`,
			convertType: "double",
			expected:    123.0000001,
		},
		{
			name:        "string to bool",
			jsonBody:    `{"data": "true"}`,
			convertType: "boolean",
			expected:    true,
		},
		{
			name:        "large string to float",
			jsonBody:    `{"data": "123456789012345678901234567890.12345"}`,
			convertType: "double",
			expected:    123456789012345678901234567890.12345,
		},
		// float
		{
			name:        "float to string",
			jsonBody:    `{"data": 123.0000001}`,
			convertType: "string",
			expected:    "123.0000001",
		},
		{
			name:        "float to integer",
			jsonBody:    `{"data": 123}`,
			convertType: "integer",
			expected:    123,
		},
		{
			name:        "float to float",
			jsonBody:    `{"data": 123.0000001}`,
			convertType: "float",
			expected:    123.0000001,
		},
		{
			name:        "large float to string",
			jsonBody:    `{"data": 123456789012345678901234567890.12345}`,
			convertType: "string",
			expected:    "1.23456789e+29",
		},
		// boolean
		{
			name:        "bool to string",
			jsonBody:    `{"data": true}`,
			convertType: "string",
			expected:    "true",
		},
		{
			name:        "bool to integer",
			jsonBody:    `{"data": true}`,
			convertType: "integer",
			expected:    1,
		},
		{
			name:        "bool to bool",
			jsonBody:    `{"data": true}`,
			convertType: "boolean",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonMap map[string]interface{}
			err := json.Unmarshal([]byte(tt.jsonBody), &jsonMap)
			require.NoError(t, err)
			actual, err := castToType(jsonMap["data"], tt.convertType)
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestCastToType_Error(t *testing.T) {
	tests := []struct {
		name        string
		jsonBody    string
		convertType string
	}{
		{
			name:        "large string to int",
			jsonBody:    `{"data": "123456789012345678901234567890"}`,
			convertType: "integer",
		},
		{
			name:        "large float to integer",
			jsonBody:    `{"data": 123456789012345678901234567890.12345}`,
			convertType: "integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonMap map[string]interface{}
			err := json.Unmarshal([]byte(tt.jsonBody), &jsonMap)
			require.NoError(t, err)
			_, err = castToType(jsonMap["data"], tt.convertType)
			require.Error(t, err)
		})
	}
}

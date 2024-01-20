package example

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
)

// requestToHttpbin shows "real" external call to data provider.
func requestToHttpbin(address string) (io.ReadCloser, error) {
	url := "http://httpbin.org/post"
	payload := map[string]interface{}{
		"ownerAddress": address,
		//nolint:gosec // this is just example
		"ownerBalance": rand.Intn(1000000),
	}
	bytesPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bytesPayload))
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// GetBalanceByAddress custom business logic that process data from external data provider
// and return new data for credential subject in the credential subject format.
func GetBalanceByAddress(address string) (map[string]interface{}, error) {
	resp, err := requestToHttpbin(address)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	var response map[string]interface{}
	err = json.NewDecoder(resp).Decode(&response)
	if err != nil {
		return nil, err
	}
	return convertResponse(response), nil

}

// convertResponse convert response from data provider to credential subject format.
func convertResponse(response map[string]any) map[string]interface{} {
	payload := response["json"]
	body := payload.(map[string]any)
	// these fields was defined in jsonld schema.
	return map[string]any{
		"address": body["ownerAddress"],
		"balance": body["ownerBalance"],
	}
}

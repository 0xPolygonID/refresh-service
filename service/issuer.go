package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iden3/go-schema-processor/v2/verifiable"
)

// IssuerService is service for communication with issuer node
type IssuerService struct {
	supportedIssuers map[string]string
	do               http.Client
}

func NewIssuerService(supportedIssuers map[string]string, client *http.Client) *IssuerService {
	if client == nil {
		client = http.DefaultClient
	}
	return &IssuerService{
		supportedIssuers: supportedIssuers,
		do:               *client,
	}
}

func (is *IssuerService) ListOfClaimsByID(issuerDID string,
	claimIDs []string) (credential []verifiable.W3CCredential, error error) {
	credential = make([]verifiable.W3CCredential, 0, len(claimIDs))

	for _, claimID := range claimIDs {
		c, err := is.GetClaimByID(issuerDID, claimID)
		if err != nil {
			return credential, err
		}
		credential = append(credential, c)
	}

	return credential, nil
}

func (is *IssuerService) GetClaimByID(issuerDID, claimID string) (
	credential verifiable.W3CCredential, error error) {
	issuerNode, err := is.getIssuerURL(issuerDID)
	if err != nil {
		return credential, err
	}
	fmt.Printf("use issuer node '%s' for issuer '%s'", issuerNode, issuerDID)

	resp, err := is.do.Get(
		fmt.Sprintf("%s/api/v1/identities/%s/claims/%s", issuerNode, issuerDID, claimID),
	)
	if err != nil {
		return credential, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return credential, fmt.Errorf("invalid status code")
	}
	err = json.NewDecoder(resp.Body).Decode(&credential)
	if err != nil {
		return credential, err
	}
	return credential, nil
}

func (is *IssuerService) CreateCredential(issuerDID string, credentialRequest credentialRequest) (
	id string,
	err error,
) {
	issuerNode, err := is.getIssuerURL(issuerDID)
	if err != nil {
		return id, err
	}
	fmt.Printf("use issuer node '%s' for issuer '%s'", issuerNode, issuerDID)

	body := bytes.NewBuffer([]byte{})
	json.NewEncoder(body).Encode(credentialRequest)
	resp, err := http.DefaultClient.Post(
		fmt.Sprintf("%s/api/v1/identities/%s/claims", issuerNode, issuerDID),
		"application/json",
		body,
	)
	if err != nil {
		return id, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return id, fmt.Errorf("invalid status code")
	}
	responseBody := struct {
		ID string `json:"id"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	if err != nil {
		return id, err
	}
	return responseBody.ID, nil
}

func (is *IssuerService) getIssuerURL(issuerDID string) (string, error) {
	url, ok := is.supportedIssuers[issuerDID]
	if !ok {
		url, ok = is.supportedIssuers["*"]
		if !ok {
			return "", fmt.Errorf("issuer %s is not supported", issuerDID)
		}
		fmt.Println("use default issuer URL")
	}
	return url, nil
}

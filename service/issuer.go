package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/0xPolygonID/refresh-service/logger"
	"github.com/iden3/go-schema-processor/v2/verifiable"
	"github.com/pkg/errors"
)

var (
	ErrIssuerNotSupported = errors.New("issuer is not supported")
	ErrGetClaim           = errors.New("failed to get claim")
	ErrCreateClaim        = errors.New("failed to create claim")
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

func (is *IssuerService) GetClaimByID(issuerDID, claimID string) (
	credential verifiable.W3CCredential, error error) {
	issuerNode, err := is.getIssuerURL(issuerDID)
	if err != nil {
		return credential, err
	}
	logger.DefaultLogger.Infof("use issuer node '%s' for issuer '%s'", issuerNode, issuerDID)

	resp, err := is.do.Get(
		fmt.Sprintf("%s/api/v1/identities/%s/claims/%s", issuerNode, issuerDID, claimID),
	)
	if err != nil {
		return credential, errors.Wrapf(ErrGetClaim,
			"failed http GET request: '%v'", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return credential, errors.Wrapf(ErrGetClaim,
			"invalid status code: '%d'", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&credential)
	if err != nil {
		return credential, errors.Wrapf(ErrGetClaim,
			"failed to decode response: '%v'", err)
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
	logger.DefaultLogger.Infof("use issuer node '%s' for issuer '%s'", issuerNode, issuerDID)

	body := bytes.NewBuffer([]byte{})
	err = json.NewEncoder(body).Encode(credentialRequest)
	if err != nil {
		return id, errors.Wrapf(ErrCreateClaim,
			"credential request serialization error")
	}
	resp, err := http.DefaultClient.Post(
		fmt.Sprintf("%s/api/v1/identities/%s/claims", issuerNode, issuerDID),
		"application/json",
		body,
	)
	if err != nil {
		return id, errors.Wrapf(ErrCreateClaim,
			"failed http POST request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return id, errors.Wrap(ErrCreateClaim,
			"invalid status code")
	}
	responseBody := struct {
		ID string `json:"id"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	if err != nil {
		return id, errors.Wrapf(ErrCreateClaim,
			"failed to decode response: %v", err)
	}
	return responseBody.ID, nil
}

func (is *IssuerService) getIssuerURL(issuerDID string) (string, error) {
	url, ok := is.supportedIssuers[issuerDID]
	if !ok {
		url, ok = is.supportedIssuers["*"]
		if !ok {
			return "", errors.Wrapf(ErrIssuerNotSupported, "id '%s'", issuerDID)
		}
	}
	return url, nil
}

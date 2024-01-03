package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	issuerBasicAuth  map[string]string
	do               http.Client
}

func NewIssuerService(
	supportedIssuers map[string]string,
	issuerBasicAuth map[string]string,
	client *http.Client,
) *IssuerService {
	if client == nil {
		client = http.DefaultClient
	}
	return &IssuerService{
		supportedIssuers: supportedIssuers,
		issuerBasicAuth:  issuerBasicAuth,
		do:               *client,
	}
}

func (is *IssuerService) GetClaimByID(issuerDID, claimID string) (*verifiable.W3CCredential, error) {
	issuerNode, err := is.getIssuerURL(issuerDID)
	if err != nil {
		return nil, err
	}
	logger.DefaultLogger.Infof("use issuer node '%s' for issuer '%s'", issuerNode, issuerDID)

	getRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/v1/%s/claims/%s", issuerNode, issuerDID, claimID),
		http.NoBody,
	)
	if err != nil {
		return nil, errors.Wrapf(ErrGetClaim,
			"failed to create http request: '%v'", err)
	}
	if err := is.setBasicAuth(issuerDID, getRequest); err != nil {
		return nil, err
	}

	resp, err := is.do.Do(getRequest)
	if err != nil {
		return nil, errors.Wrapf(ErrGetClaim,
			"failed http GET request: '%v'", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(ErrGetClaim,
			"invalid status code: '%d'", resp.StatusCode)
	}
	credential := &verifiable.W3CCredential{}
	err = json.NewDecoder(resp.Body).Decode(credential)
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

	postRequest, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/v1/%s/claims", issuerNode, issuerDID),
		body,
	)
	if err != nil {
		return id, errors.Wrapf(ErrCreateClaim,
			"failed to create http request: '%v'", err)
	}
	if err := is.setBasicAuth(issuerDID, postRequest); err != nil {
		return id, err
	}

	resp, err := is.do.Do(postRequest)
	if err != nil {
		return id, errors.Wrapf(ErrCreateClaim,
			"failed http POST request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
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

func (is *IssuerService) setBasicAuth(issuerDID string, request *http.Request) error {
	if is.issuerBasicAuth == nil {
		return nil
	}
	namepass, ok := is.issuerBasicAuth[issuerDID]
	if !ok {
		globalNamepass, ok := is.issuerBasicAuth["*"]
		if !ok {
			logger.DefaultLogger.Warnf("issuer '%s' not found in basic auth map", issuerDID)
			return nil
		}
		namepass = globalNamepass
	}

	namepassPair := strings.Split(namepass, ":")
	if len(namepassPair) != 2 {
		return errors.Errorf("invalid basic auth: %q", namepass)
	}

	request.SetBasicAuth(namepassPair[0], namepassPair[1])
	return nil
}

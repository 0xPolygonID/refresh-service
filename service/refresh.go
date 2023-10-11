package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/0xPolygonID/refresh-service/providers/flexiblehttp"
	jsonproc "github.com/iden3/go-schema-processor/v2/json"
	"github.com/iden3/go-schema-processor/v2/merklize"
	"github.com/iden3/go-schema-processor/v2/verifiable"
	"github.com/piprate/json-gold/ld"
)

var (
	errIndexSlotsNotUpdated = fmt.Errorf("no index fields were updated")
)

type RefreshService struct {
	issuerService  *IssuerService
	documentLoader ld.DocumentLoader
	providers      flexiblehttp.FactoryFlexibleHTTP
}

func NewRefreshService(
	issuerService *IssuerService,
	decumentLoader ld.DocumentLoader,
	providers flexiblehttp.FactoryFlexibleHTTP,
) *RefreshService {
	return &RefreshService{
		issuerService:  issuerService,
		documentLoader: decumentLoader,
		providers:      providers,
	}
}

type credentialRequest struct {
	CredentialSchema  string                 `json:"credentialSchema"`
	Type              string                 `json:"type"`
	CredentialSubject map[string]interface{} `json:"credentialSubject"`
	Expiration        int64                  `json:"expiration"`
}

func (rs *RefreshService) Process(issuer string, owner string, ids []string) ([]*verifiable.W3CCredential, error) {
	potentialCredentialUpdates, err := rs.issuerService.ListOfClaimsByID(issuer, ids)
	if err != nil {
		return nil, err
	}
	refreshedCredentials := make([]*verifiable.W3CCredential, 0, len(potentialCredentialUpdates))
	for _, credential := range potentialCredentialUpdates {
		err = checkOwnerShip(credential, owner)
		if err != nil {
			return nil, err
		}
		err = isUpdatable(credential)
		if err != nil {
			return nil, err
		}

		credentialBytes, err := json.Marshal(credential)
		if err != nil {
			return nil, err
		}
		credentialType, err := merklize.Options{
			DocumentLoader: rs.documentLoader,
		}.TypeIDFromContext(credentialBytes, credential.CredentialSubject["type"].(string))
		if err != nil {
			return nil, err
		}

		flexibleHTTP, err := rs.providers.ProduceFlexibleHTTP(credentialType)
		if err != nil {
			return nil, err
		}
		updatedFields, err := flexibleHTTP.Provide(credential.CredentialSubject)
		if err != nil {
			return nil, err
		}

		if err := isUpdatedIndexSlots(credentialBytes,
			credential.CredentialSubject, updatedFields); err != nil {
			return nil, err
		}

		for k, v := range updatedFields {
			credential.CredentialSubject[k] = v
		}

		credentialRequest := credentialRequest{
			CredentialSchema:  credential.CredentialSchema.ID,
			Type:              credential.CredentialSubject["type"].(string),
			CredentialSubject: credential.CredentialSubject,
			Expiration:        time.Now().Add(flexibleHTTP.Settings.TimeExpiration).Unix(),
		}

		refreshedID, err := rs.issuerService.CreateCredential(issuer, credentialRequest)
		if err != nil {
			return nil, err
		}
		rc, err := rs.issuerService.GetClaimByID(issuer, refreshedID)
		if err != nil {
			return nil, err
		}
		refreshedCredentials = append(refreshedCredentials, &rc)
	}

	return refreshedCredentials, nil
}

func isUpdatable(credential verifiable.W3CCredential) error {
	if credential.Expiration.After(time.Now()) {
		return fmt.Errorf("credential '%s' is not expired", credential.ID)
	}
	if credential.CredentialSubject["id"] == "" {
		return fmt.Errorf("the credential '%s' does not have an id", credential.ID)
	}
	return nil
}

func checkOwnerShip(credential verifiable.W3CCredential, owner string) error {
	if credential.CredentialSubject["id"] != owner {
		return fmt.Errorf("the credential was issued for another identity. expected %s actual %s",
			owner, credential.CredentialSubject["id"])
	}
	return nil
}

func isUpdatedIndexSlots(credentialBytes []byte,
	oldValues, newValues map[string]interface{}) error {

	for k, v := range oldValues {
		if k == "type" || k == "id" {
			continue
		}
		slotIndex, err := jsonproc.Parser{}.GetFieldSlotIndex(
			k, oldValues["type"].(string), credentialBytes)
		if err != nil && strings.Contains(err.Error(), "not specified in serialization info") {
			// invalid schema or merklized credential
			return nil
		} else if err != nil {
			return err
		}
		if (slotIndex == 2 || slotIndex == 3) && v != newValues[k] {
			return nil
		}
	}

	return errIndexSlotsNotUpdated
}

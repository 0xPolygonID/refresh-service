package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/0xPolygonID/refresh-service/providers/flexiblehttp"
	jsonproc "github.com/iden3/go-schema-processor/v2/json"
	"github.com/iden3/go-schema-processor/v2/merklize"
	"github.com/iden3/go-schema-processor/v2/verifiable"
	"github.com/piprate/json-gold/ld"
	"github.com/pkg/errors"
)

var (
	ErrCredentialNotUpdatable = errors.New("not updatable")
	errIndexSlotsNotUpdated   = errors.New("no index fields were updated")
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
	CredentialSchema  string                     `json:"credentialSchema"`
	Type              string                     `json:"type"`
	CredentialSubject map[string]interface{}     `json:"credentialSubject"`
	Expiration        int64                      `json:"expiration"`
	Updatable         bool                       `json:"updatable,omitempty"`
	RefreshService    *verifiable.RefreshService `json:"refreshService,omitempty"`
}

func (rs *RefreshService) Process(issuer string,
	owner string, id string) (
	verifiable.W3CCredential, error) {
	credential, err := rs.issuerService.GetClaimByID(issuer, id)
	if err != nil {
		return verifiable.W3CCredential{}, err
	}

	err = isUpdatable(credential)
	if err != nil {
		return verifiable.W3CCredential{},
			errors.Wrapf(ErrCredentialNotUpdatable,
				"credential '%s': %v", credential.ID, err)
	}
	err = checkOwnerShip(credential, owner)
	if err != nil {
		return verifiable.W3CCredential{},
			errors.Wrapf(ErrCredentialNotUpdatable, "credential '%s': %v", credential.ID, err)
	}

	credentialBytes, err := json.Marshal(credential)
	if err != nil {
		return verifiable.W3CCredential{}, err
	}
	credentialType, err := merklize.Options{
		DocumentLoader: rs.documentLoader,
	}.TypeIDFromContext(credentialBytes, credential.CredentialSubject["type"].(string))
	if err != nil {
		return verifiable.W3CCredential{}, err
	}

	flexibleHTTP, err := rs.providers.ProduceFlexibleHTTP(credentialType)
	if err != nil {
		return verifiable.W3CCredential{},
			errors.Wrapf(ErrCredentialNotUpdatable,
				"for credential '%s' not possible to find a data provider", credential.ID, err)

	}
	updatedFields, err := flexibleHTTP.Provide(credential.CredentialSubject)
	if err != nil {
		return verifiable.W3CCredential{}, err
	}

	if err := isUpdatedIndexSlots(credentialBytes,
		credential.CredentialSubject, updatedFields); err != nil {
		return verifiable.W3CCredential{},
			errors.Wrapf(ErrCredentialNotUpdatable,
				"for credential '%s' index slots: %v", credential.ID, err)
	}

	for k, v := range updatedFields {
		credential.CredentialSubject[k] = v
	}

	credentialRequest := credentialRequest{
		CredentialSchema:  credential.CredentialSchema.ID,
		Type:              credential.CredentialSubject["type"].(string),
		CredentialSubject: credential.CredentialSubject,
		Expiration:        time.Now().Add(flexibleHTTP.Settings.TimeExpiration).Unix(),
		Updatable:         true,
		RefreshService:    credential.RefreshService,
	}

	refreshedID, err := rs.issuerService.CreateCredential(issuer, credentialRequest)
	if err != nil {
		return verifiable.W3CCredential{}, err
	}
	rc, err := rs.issuerService.GetClaimByID(issuer, refreshedID)
	if err != nil {
		return verifiable.W3CCredential{}, err
	}

	return rc, nil
}

func isUpdatable(credential verifiable.W3CCredential) error {
	if credential.Expiration.After(time.Now()) {
		return errors.New("expired")
	}
	if credential.CredentialSubject["id"] == "" {
		return errors.New("subject does not have an id")
	}
	coreClaim, err := credential.GetCoreClaimFromProof(verifiable.BJJSignatureProofType)
	if err != nil {
		return errors.Errorf("unable to get core claim from BJJSignatureProofType: %v", err)
	}
	if !coreClaim.GetFlagUpdatable() {
		return errors.New("updatable flag is not set")
	}
	return nil
}

func checkOwnerShip(credential verifiable.W3CCredential, owner string) error {
	if credential.CredentialSubject["id"] != owner {
		return errors.New("not owner of the credential")
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

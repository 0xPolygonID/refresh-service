package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/0xPolygonID/refresh-service/providers/flexiblehttp"
	core "github.com/iden3/go-iden3-core/v2"
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
	RefreshService    *verifiable.RefreshService `json:"refreshService,omitempty"`
	RevNonce          *uint64                    `json:"revNonce,omitempty"`
}

func (rs *RefreshService) Process(
	ctx context.Context,
	issuer, owner, id string) (
	*verifiable.W3CCredential, error) {
	credential, err := rs.issuerService.GetClaimByID(issuer, id)
	if err != nil {
		return nil, err
	}

	err = isUpdatable(credential)
	if err != nil {
		return nil,
			errors.Wrapf(ErrCredentialNotUpdatable,
				"credential '%s': %v", credential.ID, err)
	}
	err = checkOwnerShip(credential, owner)
	if err != nil {
		return nil,
			errors.Wrapf(ErrCredentialNotUpdatable, "credential '%s': %v", credential.ID, err)
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
		return nil,
			errors.Wrapf(ErrCredentialNotUpdatable,
				"for credential '%s' not possible to find a data provider: %v", credential.ID, err)

	}
	updatedFields, err := flexibleHTTP.Provide(credential.CredentialSubject)
	if err != nil {
		return nil, err
	}

	if err := rs.isUpdatedIndexSlots(ctx, credential,
		credential.CredentialSubject, updatedFields); err != nil {
		return nil,
			errors.Wrapf(ErrCredentialNotUpdatable,
				"for credential '%s' index slots parsing process error: %v", credential.ID, err)
	}

	for k, v := range updatedFields {
		credential.CredentialSubject[k] = v
	}

	revNonce, err := extractRevocationNonce(credential)
	if err != nil {
		return nil, err
	}

	credentialRequest := credentialRequest{
		CredentialSchema:  credential.CredentialSchema.ID,
		Type:              credential.CredentialSubject["type"].(string),
		CredentialSubject: credential.CredentialSubject,
		Expiration:        time.Now().Add(flexibleHTTP.Settings.TimeExpiration).Unix(),
		RefreshService:    credential.RefreshService,
		RevNonce:          &revNonce,
	}

	refreshedID, err := rs.issuerService.CreateCredential(issuer, credentialRequest)
	if err != nil {
		return nil, err
	}
	rc, err := rs.issuerService.GetClaimByID(issuer, refreshedID)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (rs *RefreshService) loadContexts(contexts []string) ([]byte, error) {
	type uploadedContexts struct {
		Contexts []interface{} `json:"@context"`
	}
	var res uploadedContexts
	for _, context := range contexts {
		remoteDocument, err := rs.documentLoader.LoadDocument(context)
		if err != nil {
			return nil, err
		}
		document, ok := remoteDocument.Document.(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid context")
		}
		ldContext, ok := document["@context"]
		if !ok {
			return nil, errors.New("@context key word didn't find")
		}
		if v, ok := ldContext.([]interface{}); ok {
			res.Contexts = append(res.Contexts, v...)
		} else {
			res.Contexts = append(res.Contexts, ldContext)
		}
	}
	return json.Marshal(res)
}

func isUpdatable(credential *verifiable.W3CCredential) error {
	if credential.Expiration.After(time.Now()) {
		return errors.New("not expired")
	}
	if credential.CredentialSubject["id"] == "" {
		return errors.New("credential subject does not have an id")
	}
	return nil
}

func checkOwnerShip(credential *verifiable.W3CCredential, owner string) error {
	if credential.CredentialSubject["id"] != owner {
		return errors.New("not owner of the credential")
	}
	return nil
}

func (rs *RefreshService) isUpdatedIndexSlots(
	ctx context.Context,
	credential *verifiable.W3CCredential,
	oldValues, newValues map[string]interface{},
) error {
	claim, err := jsonproc.Parser{}.ParseClaim(ctx, *credential, nil)
	if err != nil {
		return errors.Errorf("invalid w3c credential: %v", err)
	}

	merklizedRootPosition, err := claim.GetMerklizedPosition()
	if err != nil {
		return errors.Errorf("failed to get merklized position: %v", err)
	}

	switch merklizedRootPosition {
	case core.MerklizedRootPositionIndex:
		return nil
	case core.MerklizedRootPositionValue:
		return errIndexSlotsNotUpdated
	case core.MerklizedRootPositionNone:
		credentialBytes, err := rs.loadContexts(credential.Context)
		if err != nil {
			return errors.Errorf("failed to load contexts: %v", err)
		}
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
	}
	return errIndexSlotsNotUpdated
}

func extractRevocationNonce(credential *verifiable.W3CCredential) (uint64, error) {
	credentialStatusInfo, ok := credential.CredentialStatus.(map[string]interface{})
	if !ok {
		return 0,
			errors.New("invalid credential status")
	}
	nonce, ok := credentialStatusInfo["revocationNonce"]
	if !ok {
		return 0,
			errors.New("revocationNonce not found in credential status")
	}
	n, ok := nonce.(float64)
	if !ok {
		return 0,
			errors.New("revocationNonce is not a number")
	}
	return uint64(n), nil
}

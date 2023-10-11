package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var nonMerklizedCredential = []byte(`{
    "id": "https://dd25-62-87-103-47.ngrok-free.app/api/v1/identities/did:polygonid:polygon:mumbai:2qMPnHfStSRPTEEuoYKApnh8j8ppVYUAJDNRJwXUzf/claims/e6d0e822-686c-11ee-8afb-3ec1cb517438",
    "@context": [
        "https://www.w3.org/2018/credentials/v1",
        "https://schema.iden3.io/core/jsonld/iden3proofs.jsonld",
        "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/kyc-nonmerklized.jsonld"
    ],
    "type": [
        "VerifiableCredential",
        "KYCAgeCredential"
    ],
    "expirationDate": "2361-03-21T21:14:48+02:00",
    "issuanceDate": "2023-10-11T22:32:23.327095+03:00",
    "credentialSubject": {
        "birthday": 19960424,
        "documentType": 99,
        "id": "did:polygonid:polygon:mumbai:2qHYafoww8yJcMhXk5jvgL33QuDGaasaqwjjVUXDP1",
        "type": "KYCAgeCredential"
    },
    "credentialStatus": {
        "id": "https://dd25-62-87-103-47.ngrok-free.app/api/v1/identities/did%3Apolygonid%3Apolygon%3Amumbai%3A2qMPnHfStSRPTEEuoYKApnh8j8ppVYUAJDNRJwXUzf/claims/revocation/status/2876560823",
        "revocationNonce": 2876560823,
        "type": "SparseMerkleTreeProof"
    },
    "issuer": "did:polygonid:polygon:mumbai:2qMPnHfStSRPTEEuoYKApnh8j8ppVYUAJDNRJwXUzf",
    "credentialSchema": {
        "id": "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json/kyc-nonmerklized.json",
        "type": "JsonSchema2023"
    },
    "proof": [
        {
            "type": "BJJSignature2021",
            "issuerData": {
                "id": "did:polygonid:polygon:mumbai:2qMPnHfStSRPTEEuoYKApnh8j8ppVYUAJDNRJwXUzf",
                "state": {
                    "claimsTreeRoot": "9491c992271b6b1733d3f48f44f7fd800a51d8df686657107f2a6edc06417804",
                    "value": "6314b1ee1ac49f1b5a127b24718a74e15137edba3a8b4bff0c9c4c7f64b2fb07"
                },
                "authCoreClaim": "cca3371a6cb1b715004407e325bd993c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000b5653a26b0fd26fb9673f9c310874491d5a2b339f833102f955eac449d76920a1526225a79cf9ba9c1f7ec0ee982be5441f8d5a18337cad148065327f67f901e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
                "mtp": {
                    "existence": true,
                    "siblings": []
                },
                "credentialStatus": {
                    "id": "https://dd25-62-87-103-47.ngrok-free.app/api/v1/identities/did%3Apolygonid%3Apolygon%3Amumbai%3A2qMPnHfStSRPTEEuoYKApnh8j8ppVYUAJDNRJwXUzf/claims/revocation/status/0",
                    "revocationNonce": 0,
                    "type": "SparseMerkleTreeProof"
                }
            },
            "coreClaim": "cb373906ed88fff9332f71521b712c950a00000000000000000000000000000002126fda1e9e75859b1bababe0b52850185869b9adb87a34d17127222f8b0c0068923001000000000000000000000000000000000000000000000000000000006300000000000000000000000000000000000000000000000000000000000000b7d574ab00000000281cdcdf0200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
            "signature": "074f6133d906f9469ef385abe449d425c4a39248958ee5961c22eae7436c82109eb1ad76eb5948d55a2d02fd9a71f971d50fba81b0d954a36e9416c448d0b902"
        }
    ]
}`)

func TestIsUpdatedIndexSlot(t *testing.T) {
	tests := []struct {
		name      string
		oldValues map[string]interface{}
		newValues map[string]interface{}
	}{
		{
			name: "Index slots were changed",
			oldValues: map[string]interface{}{
				"type":         "KYCAgeCredential",
				"birthday":     19960424,
				"documentType": 99,
			},
			newValues: map[string]interface{}{
				"birthday":     19960424,
				"documentType": 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isUpdatedIndexSlots(nonMerklizedCredential,
				tt.oldValues, tt.newValues)
			require.NoError(t, err)
		})
	}
}

func TestIsUpdatedIndexSlot_Error(t *testing.T) {
	tests := []struct {
		name      string
		oldValues map[string]interface{}
		newValues map[string]interface{}
	}{
		{
			name: "Index slots were not changed",
			oldValues: map[string]interface{}{
				"type":         "KYCAgeCredential",
				"birthday":     19960424,
				"documentType": 99,
			},
			newValues: map[string]interface{}{
				"birthday":     19960424,
				"documentType": 99,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isUpdatedIndexSlots(nonMerklizedCredential,
				tt.oldValues, tt.newValues)
			require.ErrorIs(t, err, errIndexSlotsNotUpdated)
		})
	}
}

# Refresh service

## Description

Verifiable credentials are subject to expiration. In certain scenarios, having mechanisms to refresh these credentials proves beneficial. Refreshing can be carried out either manually or automatically. Through the utilization of a refresh service, we gain the ability to establish short-term credentials. This approach ensures that users are consistently using the most current information pertaining to themselves, such as their user balance, game score or other very dynamic data.

## Example of short-term credentials

As an illustration, consider our intention to issue a claim regarding a user's account balance. However, it's important to note that a user's account balance is highly dynamic in nature. If we were to issue the claim with an extended validity period, the claim holder could potentially engage in double spending activities. To counteract this issue, one possible solution is to distribute short-term credentials. However, this approach necessitates frequent interactions between the user and the issuer node. The user would be required to consistently furnish the most up-to-date information about their balance. This process can be demanding for the user and consume a substantial amount of time. Indeed, the refresh service can streamline and automate this process, making it significantly easier and more user-friendly.

## Support on the Issuer side

1. **Optional support for embedded structure**: Issuer servers may optionally support a new embedded structure within the credential request. This structure, named `refreshService`, includes an identifier and a type. For example:
    
    ```json
    "refreshService": { 
       "id": "http://refreshservice.example:8002", 
       "type": "Iden3RefreshService2023"
    }
    ```
    
2. **Validation of refresh service**: Issuers should validate whether they support the specified `refreshService.type`. Additionally, they need to check if the credential has an expiration date only if a refresh service was provided. Implementing a refresh service for long-term credentials is generally unnecessary. However, issuers can develop custom logic to suit specific requirements for their issuer and refresh services.
3. **Endpoints for Iden3RefreshService2023**: To support the `Iden3RefreshService2023` refresh service type, issuers should provide public endpoints. These endpoints may be secured using methods like API keys or basic authentication. The specific endpoints include:
    - `GET /api/v1/identities/{issuer_did}/claims/{claim_id}`: To retrieve a claim by its issuerDID and claim ID.
    - `POST /api/v1/identities/{issuer_did}/claims`: To issue a credential under an issuer.
    
    > **NOTE:** These URL are customizable for different implementations of the refresh service.
    > 
4. **Return of refreshService structure**: If a `refreshService` is added for a verifiable credential, the issuer should include the `refreshService` structure in the returned verifiable credential. This ensures the VC carries information about its refresh capabilities.

## Algorithm of client  interaction with refresh service

To implement credential refreshing, we need to follow the following algorithm to look up the credentials when a proof request is received:

- **Auto refresh**
    
    
    ![auto-refresh.drawio.svg](assets/auto-refresh.drawio.svg)
    
    **Select all credentials that satisfy context + type. If not found, return an error.**
    
    - In this step, you are searching for credentials based on a specified context and type. If credentials are found, they proceed to the next step. Otherwise, an error is returned.
    
    **If credentials are found, check the skipRevocation flag.**
    
    - If credentials are retrieved in the previous step, the algorithm checks the value of the `skipRevocation` flag from the proof request. If it is set to **true**, the algorithm skips the revocation check and moves on to the next step. If **false**, it proceeds to the revocation check.
    
    **Check revocation for selected credentials if needed.**
    
    - If the `skipRevocation` flag is false, this step involves verifying whether the selected credentials have been revoked. If all credentials are revoked - return an error.
    
    **Select all credentials that are non-expired and matched to the proof request.**
    
    - This step involves filtering out credentials that are expired and(or) don't match the proof request. If non-expired and matched to proof request credential(s) was/were found. Try to generate proof.
    
    **If valid credentials are not found, filter all credentials that have a refresh service.**
    
    - If there are no suitable credentials in the previous step, the algorithm filters out credentials that have a refresh service. These are credentials that can be refreshed.
    
    **Select one credential with a refresh service and refresh it.**
    
    - From the credentials with refresh services, the algorithm selects one and initiates a refresh. After refreshing, the algorithm saves the updated credential.
    
    **Check if the refreshed credential satisfies the proof request. If yes, try to generate proof. If no, repeat previous step.**
    
    - After refreshing the credential, the algorithm checks if the updated credential satisfies the proof request. If it does, the algorithm attempts to generate a proof. If the credential still doesn't meet the proof request, the process repeats, selecting another credential with a refresh service and refreshing it.
- **Manual refresh**
    
    ![manual-refresh.drawio.svg](assets/manual-refresh.drawio.svg)
    
    **Select all credentials that satisfy context + type. If not found, return an error.**
    
    - In this step, you are searching for credentials based on a specified context and type. If any credentials are found, they proceed to the next step. Otherwise, an error is returned.
    
    **If credentials are found, check the skipRevocation flag.**
    
    - If credentials are retrieved in the previous step, the algorithm checks the value of the `skipRevocation` flag from the proof request. If it is set to **true**, the algorithm skips the revocation check and moves on to the next step. If **false**, it proceeds to the revocation check.
    
    **Check revocation for selected credentials if needed.**
    
    - If the `skipRevocation` flag is false, this step involves verifying whether the selected credentials have been revoked. If all credentials are revoked - return an error.
    
    **Select all credentials that are non-expired and matched to the proof request.**
    
    - This step involves filtering out credentials that are expired and(or) don't match the proof request. If non-expired and matched to proof request credential(s) was/were found. Try to generate proof.
    
    I**f valid credentials are not found, filter all credentials that have a refresh service.**
    
    - If there are no suitable credentials in the previous step, the algorithm filters out credentials that have a refresh service. These are credentials that can be refreshed.
    
    **Initialize User Interface (UI):**
    
    - Display filtered credentials with refreshed services.
    - Each credential has two special markers:
        - An "expired" marker if the credential is out of date.
        - A "non-matched" marker if the credential does not match the proof request.
    
    **User Interaction:**
    
    - The user has the option to initiate a refresh process by pressing a "refresh" button near each credential.
    - The user has the option to mark the credential as an `auto refresh` which means that for the type of credentials, we should use automation refresh flow.
    
    **Credential Refresh Process:**
    
    - When the "refresh" button is pressed, the system triggers a call to a "refresh service."
    - The refresh service attempts to update the selected credential.
    
    **Post-Refresh Validation:**
    
    - After a credential is refreshed, the system saves the refreshed credential.
    - It then checks whether the refreshed credential matches the proof request criteria.
        - If it matches, the system proceeds to generate proof.
        - If it does not match, the system returns the user to the UI with the list of credentials, indicating that a different credential may need to be refreshed.


>💡 **NOTE:** If expired credentials are revoked, the current recommended algorithm will not process or update such credentials.
>


## Client communication with refresh service:

If the refreshServer section within a verifiable credential is of type **Iden3RefreshService2023**, the user is required to construct a  [refresh Iden3Comm ZKP message](https://iden3-communication.io/credentials/1.0/refresh/). This message should then be sent to the agent endpoint specified by `refreshServer.id`.

## Changes in the verifiable credential

In our scenario, the revocation service is integrated into the verifiable credential itself:

```json
{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    ...
  ],
  ...
  "refreshService": {
		"id": "https://refreshService.example", // agent
    "type": "Iden3RefreshService2023"
  }
}
```

The data model of `refershService`:

| Field name | Data type | Description | Required |
| --- | --- | --- | --- |
| type | string | Type of refresh service. | ✅ |
| id | string | Refresh service identification | ✅ |

Supported types:

1. **Iden3RefreshService2023:** for this type, the ID is equal to the refresh service agent URL.

## Proposal

Schema of workflow:

![General flow.svg](assets/work-flow.svg)

1. The verifier initiates a proof request to the holder.
2. The holder attempts to locate credentials based on the request.
3. The holder verifies the expiration date.
4. The holder sends a `Refresh request` to the `refreshService` within the verifiable credential.
5. The `refreshService` examines whether a refreshed claim exists in the chance layer. If one exists, the service retrieves a cached VC. If the record has a `pending` status, the `refreshService` responds with a `pending` status code.
6. The `refreshService` sends a request to the issuer node to retrieve claim information.
7. The issuer node provides the claim details to the `refreshService`.
8. The `refreshService` validates the VC's expiration date.
9. Using the VC's `credential type + context`, the `refreshService` selects a data provider and makes a request.
10. The `refreshService` contacts the `issuer node` to create a new claim using the data received from the data provider.
11. The `issuer node` generates the refreshed claim and sends it to the `refreshService`.
12. The `refreshService` delivers an [issuance response](https://iden3-communication.io/credentials/1.0/issuance-response/) to the holder's mobile app or extension.
    
    **Signature (SIG):**
    
    1. The `refreshService` provides the refreshed credential to the holder.
    2. The holder can generate a proof for the signature request.
    
    Merkle tree proof **(MTP):**

    The workflow for MTP is in development. However, you still receive notifications about MTP proof on the mobile application after refreshing credentials that have MTP proof.

    1. The `refreshService` informs the holder about the `pending` status.
    2. The holder monitors the credential status for a certain duration.
    
    **In cases where the holder intends to generate proof for an MTP proof request but their credential has been refreshed, the holder should decline the proof request. This is because generating an MTP proof might require a significant amount of time to become ready.**

## Integration examples:

1. Golang integration:
    1. Issue a credential with RefreshService and use the RefreshService to generate proof: https://github.com/iden3/identity-server/pull/309
    2. Issuer a credential with RefresService only: https://github.com/0xPolygonID/issuer-node/pull/581
2. JS integration:
    1. https://github.com/0xPolygonID/js-sdk/pull/165.
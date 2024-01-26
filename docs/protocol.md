# Refresh service

## Description

Credentials can expire. In some cases, having mechanisms to refresh issued credentials can be helpful. A refresh service allows issuing a short term credentials that will be updated. This approach ensures that users are consistently using the most current information provided by the issuer, such as for example: an user balance, an user game score or other data that is dynamic by its nature.

## Example of short-term credentials

Consider an example of balance credentials, where a user proves his balance to get some benefits. The balance can be changed a lot of times over a short period. If an issuer issues a balance credential with a long-term expiration, the credential holder could potentially benefit of using credential with outdated information in it. To minimize this risk, one solution is to use short-term credentials. However, this would require a user frequently requests a new credential from an issuer (e.g., every 5 minutes), which is quite inefficient. This is where the refresh service comes in handy. The refresh service handles all the necessary updates instead of the user.

## Changes in the verifiable credential

In protocol level, the revocation service is integrated into the verifiable credential itself:

```json
{
  "@context": [
    "https://www.w3.org/2018/credentials/v1",
    ...
  ],
  ...
  "refreshService": {
	"id": "https://refreshService.example", // iden3comm agent endpoint
    "type": "Iden3RefreshService2023"
  }
}
```

The data model of `refershService`:

| Field name | Data type | Description | Required |
| --- | --- | --- | --- |
| type | string | Type of refresh service. | ✅ |
| id | string | URL to [iden3comm](https://iden3-communication.io/) agent endpoint | ✅ |

Supported types:
- Iden3RefreshService2023

## Algorithm of client interaction with refresh service

To implement credential refreshing, we need to follow next algorithm to look up for the credentials when a proof request is received:

![auto-refresh.drawio.svg](assets/auto-refresh.drawio.svg)

**Select all credentials that satisfy context + type.**

- Looking for credentials based on a specified context and type. If found, go to the next step. Otherwise, return an error.

**Credentials are found, check the skipRevocation flag.**

- Check the value of the `skipRevocation` flag from the proof request. **true**, you should skip the revocation check and move to the next step. If **false**, process the revocation check.

**Check revocation for selected credentials.**

- `skipRevocation` flag is false, verify whether the selected credentials have been revoked. If all credentials are revoked - return an error.

**Select all credentials that are non-expired and matched to the proof request.**

- This step involves filtering out credentials that are expired and(or) don't match the proof request. If non-expired and matched to proof request credentials were found. Try to generate a proof.

**Valid credentials are not found, filter all credentials that have a refresh service.**

- The algorithm filters out credentials that have a refresh service.

**Select one credential with a refresh service and refresh it.**

- From the credentials with refresh services, the algorithm selects one and initiates a refresh. After refreshing, the algorithm saves the updated credential.

**Check if the refreshed credential satisfies the proof request. If yes, try to generate proof. If no, repeat previous step.**

- After refreshing the credential, the algorithm checks if the updated credential satisfies the proof request. If it does, the algorithm attempts to generate a proof. If the credential still doesn't meet the proof request, the process repeats, selecting another credential with a refresh service and refreshing it.

>💡 **NOTE:** If expired credentials are revoked, the current recommended algorithm will not process or update such credentials.
>


## Client communication with refresh service:

If the refreshServer section within a verifiable credential is of type **Iden3RefreshService2023**, the user is required to construct a  [refresh Iden3Comm ZKP message](https://iden3-communication.io/credentials/1.0/refresh/). This message should then be sent to the agent endpoint specified by `refreshServer.id`.

## Proposal

Workflow algorithm:

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
    
    **Merkle tree proof (MTP):**

    > **NOTE:** The workflow for MTP is in development. However, you still can receive notifications about MTP proof on the mobile application after refreshing credential(-s) that have MTP proof are performed.
    >

    1. The `refreshService` informs the holder about the `pending` status.
    2. The holder monitors the credential status for a certain duration.
    
    **In cases where the holder intends to generate proof for an MTP proof request but their credential has been refreshed, the holder should decline the proof request. This is because generating an MTP proof might require a significant amount of time to become ready.**

## Integration examples:

1. Golang integration:
    1. Issue a credential with RefreshService and use the RefreshService to generate proof: https://github.com/iden3/identity-server/pull/309
    2. Issuer a credential with RefresService only: https://github.com/0xPolygonID/issuer-node/pull/581
2. JS integration:
    1. https://github.com/0xPolygonID/js-sdk/pull/165.
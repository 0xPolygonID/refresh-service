# Refresh service
Users can utilize the refresh service to renew expired credentials. This server operates as a proxy intermediary between data providers and user credentials. The server's behavior is contingent on the credential type and subject, allowing it to discern and engage with the appropriate data provider to retrieve relevant data. Then, it builds a new credential request, configuring it accordingly, and forwards this request to the issuer node for the issuance of a fresh credential.

It is **important to note** that the refresh service imposes a constraint on non-merklized credentials. In cases where values are stored within index slots and remain unaltered by the data provider, the service will return an error. This occurs because merkle trees do not accommodate credentials with equal index slots.

To run this service, users should manage two configurations: one in a `.env` file and another in `config.yaml`. `.env` configuration is used for configure the server, `config.yaml` configuration is used for configure HTTP data provider.
1. `.env` file:
 
| Config Name                | Description                                                                                   | Required | Default Value       | Format   | Example                                                           |
|----------------------------|-----------------------------------------------------------------------------------------------|----------|---------------------|----------|-------------------------------------------------------------------|
| SUPPORTED_ISSUERS          | A list of supported issuers with their corresponding node URLs.                               | Yes      | -                   | `issuerDID=issuerNodeURL,...` | `did:example:issuer1=https://issuer1.com,did:example:issuer2=https://issuer2.com`<br/>or<br/>`*=https://common.issuer.com>` |
| IPFS_GATEWAY_URL           | The URL of the IPFS gateway.                                                                 | No       | https://ipfs.io                   | URL      | `https://ipfs.example.com`                                       |
| SERVER_HOST                | The server host.                                                                              | No       | localhost:8002      | Host:Port | `localhost:8002`                                                  |
| HTTP_CONFIG_PATH           | The path to the HTTP provider configuration.                                                           | No       | config.yaml                   | Path     | `/path/to/http/config`                                           |
| SUPPORTED_RPC              | Supported RPC endpoints for different blockchain chains.                                      | Yes      | -                   | `chainID=RPC_URL,...` | `80002=https://amoy.infura,137=https://main.infura` |
| SUPPORTED_STATE_CONTRACTS  | Supported state contracts for different blockchain chains.                                    | Yes      | -                   | `chainID=contractAddress,...` | `80002=0x123abc...,137=0x456def...`                        |
| CIRCUITS_FOLDER_PATH       | The path to the circuits folder.                                                             | No       | keys                   | Path     | `/path/to/circuits`                                               |
| ISSUERS_BASIC_AUTH         | Basic authentication credentials for issuer nodes.                                            | No       | -                   | `issuerDID=user:password,...` | `did:example:issuer1=admin:pass123,did:example:issuer2=guest:pass321`<br/>or<br/>`*=common:pass987` |
| SUPPORTED_CUSTOM_DID_METHODS | Register custom networks for DID methods.                                                     | No       | -                   | JSON Array | `[{"blockchain":"linea","network":"testnet","networkFlag":"0b01000001","chainID":59140}]` |

2. `config.yaml` for configure HTTP request to data providers:
Example:
    ```yml
    urn:uuid:069dccf5-0d79-49fd-aed5-e7301956d0f4:
      settings:
        timeExpiration: 5m
      provider:
        url: https://api.polygonscan.com/api
        method: GET
      requestSchema:
        params:
          module: account
          action: balance
          address: "{{ credentialSubject.address }}"
          apikey: API_KEY
        headers:
          Content-Type: application/json
      responseSchema:
        type: json
        properties:
          result:
            type: string
            match: credentialSubject.balance
    ```
    First, we create a data provider for the credential type (in our case, urn:uuid:069dccf5-0d79-49fd-aed5-e7301956d0f4).

    `settings` section:
    ```
    timeExpiration: This defines how long a credential must remain valid after a refresh.
    ```

    `provider` section:
    ```
    url: The provider URL.
    method: The type of HTTP request to the URL.
    ```

    `requestSchema` describes the format of a request to the data provider:
    ```
    params: A key-value list that will be substituted into provider.url. You can use the template value {{ credential.field }} to substitute a value from the user's credentials.
    headers: A list of headers that will be added to the request.
    ```

    `responseSchema` describes how to convert the data provider's response to a credential request:
    ```
    type: The response type (currently, only JSON is supported).
    properties: A list of response_field: { type, match } pairs. These match fields from the data provider response to the credential request.
    ```

## How to run:
1. Run docker-compose file:
    ```bash
    docker-compose up -d
    ```

## License

refresh-service is part of the 0xPolygonID project copyright 2024 ZKID Labs AG

This project is licensed under either of

- [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0) ([`LICENSE-APACHE`](LICENSE-APACHE))
- [MIT license](https://opensource.org/licenses/MIT) ([`LICENSE-MIT`](LICENSE-MIT))

at your option.

# Refresh service
Users can utilize the refresh service to renew expired credentials. This server operates as a proxy intermediary between data providers and user credentials. The server's behavior is contingent on the credential type and subject, allowing it to discern and engage with the appropriate data provider to retrieve relevant data. Then, it builds a new credential request, configuring it accordingly, and forwards this request to the issuer node for the issuance of a fresh credential.

It is **important to note** that the refresh service imposes a constraint on non-merklized credentials. In cases where values are stored within index slots and remain unaltered by the data provider, the service will return an error. This occurs because merkle trees do not accommodate credentials with equal index slots.

To run this service, users should manage two configurations: one in a `.env` file and another in `config.yaml`. `.env` configuration is used for configure the server, `config.yaml` configuration is used for configure HTTP data provider.
1. `.env` file:
    ```
    SUPPORTED_ISSUERS - A list of supported issuers in the format `issuerDID:issuerNodeURL`. You can also use `*` to set a default node.
    IPFS_URL - The URL of the IPFS node.
    SERVER_PORT - The server port. The default is 8002.
    HTTP_CONFIG_PATH - The path to the HTTP configuration.
    SUPPORTED_RPC - Supported RPC in the format `chainID:URL`.
    SUPPORTED_STATE_CONTRACTS - Supported state contracts in the format `chainID:contractAddress`.
    CIRCUITS_FOLDER_PATH - The path to the circuits folder.
    ```
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
1. Build the docker container:
    ```bash
    docker build -t refresh-service:local .
    ```
2. Run the docker container:
    ```bash
    docker run --env-file .env -v ./config.yaml:/app/config.yaml .
    ```

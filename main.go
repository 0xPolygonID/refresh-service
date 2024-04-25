package main

import (
	_ "embed"
	"log"
	"strings"

	_ "github.com/0xPolygonID/refresh-service/logger"
	"github.com/0xPolygonID/refresh-service/packagemanager"
	"github.com/0xPolygonID/refresh-service/pkg/ipfs"
	"github.com/0xPolygonID/refresh-service/providers/flexiblehttp"
	"github.com/0xPolygonID/refresh-service/server"
	"github.com/0xPolygonID/refresh-service/service"
	"github.com/iden3/go-schema-processor/v2/loaders"
	"github.com/kelseyhightower/envconfig"
	"github.com/piprate/json-gold/ld"
	"github.com/pkg/errors"
)

var (
	w3cSchemaURL = "https://www.w3.org/2018/credentials/v1"
	//go:embed w3cSchema.json
	w3cSchemaBody []byte
)

type KVstring map[string]string

func (c *KVstring) Decode(value string) error {
	value = strings.Trim(value, "\"")
	if value == "" {
		*c = make(map[string]string)
		return nil
	}
	contracts := make(map[string]string)
	pairs := strings.Split(value, ",")
	for _, pair := range pairs {
		kvpair := strings.Split(pair, "=")
		if len(kvpair) != 2 {
			return errors.Errorf("invalid map item: %q", pair)
		}
		contracts[kvpair[0]] = kvpair[1]

	}
	*c = KVstring(contracts)
	return nil
}

type Config struct {
	SupportedIssuers          KVstring `envconfig:"SUPPORTED_ISSUERS" required:"true"`
	IPFSURLGateway            string   `envconfig:"IPFS_URL_GATEWAY" default:"https://ipfs.io"`
	ServerHost                string   `envconfig:"SERVER_HOST" default:"localhost:8002"`
	HTTPConfigPath            string   `envconfig:"HTTP_CONFIG_PATH" default:"config.yaml"`
	SupportedRPC              KVstring `envconfig:"SUPPORTED_RPC" required:"true"`
	SupportedStateContracts   KVstring `envconfig:"SUPPORTED_STATE_CONTRACTS" required:"true"`
	CircuitsFolderPath        string   `envconfig:"CIRCUITS_FOLDER_PATH" default:"keys"`
	SupportedIssuersBasicAuth KVstring `envconfig:"ISSUERS_BASIC_AUTH" required:"true"`
}

func (c *Config) getServerHost() string {
	return strings.TrimSuffix(c.ServerHost, "/")
}

func (c *Config) getSupportedIssuers() map[string]string {
	var supportedIssuers = make(map[string]string, len(c.SupportedIssuers))
	for k, v := range c.SupportedIssuers {
		supportedIssuers[k] = strings.TrimSuffix(v, "/")
	}
	return supportedIssuers
}

func main() {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("failed init config: %v", err)
	}

	packageManager, err := packagemanager.NewPackageManager(
		cfg.SupportedRPC,
		cfg.SupportedStateContracts,
		packagemanager.WithVerificationKeyPath(cfg.CircuitsFolderPath),
	)
	if err != nil {
		log.Fatalf("failed init package manager: %v", err)
	}

	issuerService := service.NewIssuerService(
		cfg.getSupportedIssuers(),
		cfg.SupportedIssuersBasicAuth,
		nil,
	)

	ipfsCli := ipfs.NewIPFSClient(cfg.IPFSURLGateway)
	documentLoader, err := initDocumentLoaderWithCache(ipfsCli)
	if err != nil {
		log.Fatalf("failed init ipfs loader: %v", err)
	}

	flexhttp, err := flexiblehttp.NewFactoryFlexibleHTTP(
		cfg.HTTPConfigPath,
		nil,
	)
	if err != nil {
		log.Fatalf("failed init flexiblehttp: %v", err)
	}

	refreshService := service.NewRefreshService(
		issuerService,
		documentLoader,
		flexhttp,
	)

	agentService := service.NewAgentService(
		refreshService,
		packageManager,
	)

	h := server.NewHandlers(
		agentService,
	)

	log.Fatal(h.Run(cfg.getServerHost()))
}

func initDocumentLoaderWithCache(ipfsCli loaders.IPFSClient) (ld.DocumentLoader, error) {
	opts := loaders.WithEmbeddedDocumentBytes(
		w3cSchemaURL, w3cSchemaBody,
	)
	memoryCacheEngine, err := loaders.NewMemoryCacheEngine(opts)
	if err != nil {
		return nil, err
	}
	l := loaders.NewDocumentLoader(ipfsCli, "", loaders.WithCacheEngine(memoryCacheEngine))
	return l, nil
}

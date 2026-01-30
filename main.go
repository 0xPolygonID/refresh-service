package main

import (
	_ "embed"
	"log"
	"strings"

	_ "github.com/0xPolygonID/refresh-service/logger"
	"github.com/0xPolygonID/refresh-service/packagemanager"
	"github.com/0xPolygonID/refresh-service/providers/flexiblehttp"
	"github.com/0xPolygonID/refresh-service/server"
	"github.com/0xPolygonID/refresh-service/service"
	"github.com/iden3/go-schema-processor/v2/loaders"
	"github.com/kelseyhightower/envconfig"
	"github.com/piprate/json-gold/ld"
	"github.com/pkg/errors"
	"github.com/joho/godotenv"
)

var (
	w3cSchemaURL = "https://www.w3.org/2018/credentials/v1"
	//go:embed w3cSchema.json
	w3cSchemaBody []byte
)

type KVstring map[string]string

func (c *KVstring) Decode(value string) error {
	const delimiter = ";"

	value = strings.Trim(value, "\"")
	if value == "" {
		*c = make(map[string]string)
		return nil
	}
	contracts := make(map[string]string)
	pairs := strings.Split(value, delimiter)
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
	IPFSGWURL                 string   `envconfig:"IPFS_GATEWAY_URL" default:"https://ipfs.io"`
	ServerHost                string   `envconfig:"SERVER_HOST" default:":8002"`
	HTTPConfigPath            string   `envconfig:"HTTP_CONFIG_PATH" default:"config.yaml"`
	SupportedRPC              KVstring `envconfig:"SUPPORTED_RPC" required:"true"`
	SupportedStateContracts   KVstring `envconfig:"SUPPORTED_STATE_CONTRACTS" required:"true"`
	CircuitsFolderPath        string   `envconfig:"CIRCUITS_FOLDER_PATH" default:"keys"`
	SupportedIssuersBasicAuth KVstring `envconfig:"ISSUERS_BASIC_AUTH"`
	SupportedCustomDIDMethods string   `envconfig:"SUPPORTED_CUSTOM_DID_METHODS"`
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
	
	if err := godotenv.Load(); err != nil {
		log.Info(ctx, "Error loading .env file")
	}
	
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("failed init config: %v", err)
	}

	packageManager, err := packagemanager.NewPackageManager(
		cfg.SupportedRPC,
		cfg.SupportedStateContracts,
		packagemanager.WithVerificationKeyPath(cfg.CircuitsFolderPath),
		packagemanager.WithCustomDIDMethods(cfg.SupportedCustomDIDMethods),
	)
	if err != nil {
		log.Fatalf("failed init package manager: %v", err)
	}

	issuerService := service.NewIssuerService(
		cfg.getSupportedIssuers(),
		cfg.SupportedIssuersBasicAuth,
		nil,
	)

	documentLoader, err := initDocumentLoaderWithCache(cfg.IPFSGWURL)
	if err != nil {
		log.Fatalf("failed init document loader: %v", err)
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

func initDocumentLoaderWithCache(ipfsGW string) (ld.DocumentLoader, error) {
	opts := loaders.WithEmbeddedDocumentBytes(
		w3cSchemaURL, w3cSchemaBody,
	)
	memoryCacheEngine, err := loaders.NewMemoryCacheEngine(opts)
	if err != nil {
		return nil, err
	}
	l := loaders.NewDocumentLoader(nil, ipfsGW, loaders.WithCacheEngine(memoryCacheEngine))
	return l, nil
}

package main

import (
	"log"
	"strings"

	"github.com/0xPolygonID/refresh-service/packagemanager"
	"github.com/0xPolygonID/refresh-service/providers/flexiblehttp"
	"github.com/0xPolygonID/refresh-service/server"
	"github.com/0xPolygonID/refresh-service/service"
	"github.com/iden3/go-schema-processor/v2/loaders"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
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
	SupportedIssuers        KVstring `envconfig:"SUPPORTED_ISSUERS" required:"true"`
	IPFSURL                 string   `envconfig:"IPFS_URL" required:"true"`
	ServerHost              string   `envconfig:"SERVER_HOST" default:"localhost:8002"`
	HTTPConfigPath          string   `envconfig:"HTTP_CONFIG_PATH" default:"config.yaml"`
	SupportedRPC            KVstring `envconfig:"SUPPORTED_RPC" required:"true"`
	SupportedStateContracts KVstring `envconfig:"SUPPORTED_STATE_CONTRACTS" required:"true"`
	CircuitsFolderPath      string   `envconfig:"CIRCUITS_FOLDER_PATH" default:"circuits"`
}

func main() {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("failed init config: %v", err)
	}

	packageManager, err := packagemanager.NewPackageManager(
		cfg.SupportedRPC,
		cfg.SupportedStateContracts,
		cfg.CircuitsFolderPath,
	)
	if err != nil {
		log.Fatalf("failed init package manager: %v", err)
	}

	issuerService := service.NewIssuerService(
		cfg.SupportedIssuers,
		nil,
	)

	ipfsCli := shell.NewShell(cfg.IPFSURL)
	documentLoader := loaders.NewDocumentLoader(ipfsCli, "")

	flexiblehttp, err := flexiblehttp.NewFactoryFlexibleHTTP(
		cfg.HTTPConfigPath,
		nil,
	)
	if err != nil {
		log.Fatalf("failed init flexiblehttp: %v", err)
	}

	refreshService := service.NewRefreshService(
		issuerService,
		documentLoader,
		flexiblehttp,
	)

	h := server.NewHandlers(
		packageManager,
		refreshService,
	)

	log.Fatal(h.Run(cfg.ServerHost))
}

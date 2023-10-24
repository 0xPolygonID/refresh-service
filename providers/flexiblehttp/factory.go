package flexiblehttp

import (
	"net/http"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type FactoryFlexibleHTTP struct {
	configuration map[string]FlexibleHTTP
	httpcli       *http.Client
}

func NewFactoryFlexibleHTTP(configPath string, httpcli *http.Client) (FactoryFlexibleHTTP, error) {
	f, err := os.ReadFile(configPath)
	if err != nil {
		return FactoryFlexibleHTTP{}, err
	}
	if httpcli == nil {
		httpcli = http.DefaultClient
	}
	cfgs := make(map[string]FlexibleHTTP)
	if err := yaml.Unmarshal(f, &cfgs); err != nil {
		return FactoryFlexibleHTTP{}, err
	}
	return FactoryFlexibleHTTP{
		configuration: cfgs,
		httpcli:       httpcli,
	}, nil
}

func (factory *FactoryFlexibleHTTP) ProduceFlexibleHTTP(credentialType string) (FlexibleHTTP, error) {
	fh, ok := factory.configuration[credentialType]
	if !ok {
		return FlexibleHTTP{}, errors.Errorf("not found configuration for '%s'", credentialType)
	}
	fh.httpcli = factory.httpcli
	return fh, nil
}

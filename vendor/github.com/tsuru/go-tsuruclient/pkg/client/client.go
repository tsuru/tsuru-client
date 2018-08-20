package client

import (
	"github.com/tsuru/go-tsuruclient/pkg/tsuru"
	"github.com/tsuru/tsuru/cmd"
)

func ClientFromEnvironment(cfg *tsuru.Configuration) (*tsuru.APIClient, error) {
	if cfg == nil {
		cfg = &tsuru.Configuration{}
	}
	var err error
	if cfg.BasePath == "" {
		cfg.BasePath, err = cmd.GetTarget()
		if err != nil {
			return nil, err
		}
	}
	if cfg.DefaultHeader == nil {
		cfg.DefaultHeader = map[string]string{}
	}
	if _, authSet := cfg.DefaultHeader["Authorization"]; !authSet {
		if token, tokenErr := cmd.ReadToken(); tokenErr == nil && token != "" {
			cfg.DefaultHeader["Authorization"] = "bearer " + token
		}
	}
	cli := tsuru.NewAPIClient(cfg)
	return cli, nil
}

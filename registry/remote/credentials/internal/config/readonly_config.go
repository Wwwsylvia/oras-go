package config

import (
	"encoding/json"
	"io"

	"oras.land/oras-go/v2/registry/remote/auth"
)

type ReadOnlyConfig struct {
	auths map[string]AuthConfig
}

func LoadFromReader(reader io.Reader) (*ReadOnlyConfig, error) {
	cfg := &ReadOnlyConfig{}
	if err := json.NewDecoder(reader).Decode(&cfg.auths); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *ReadOnlyConfig) GetCredential(serverAddress string) (auth.Credential, error) {
	authCfg, ok := matchAuth(c.auths, serverAddress)
	if !ok {
		return auth.EmptyCredential, nil
	}
	return authCfg.Credential()
}

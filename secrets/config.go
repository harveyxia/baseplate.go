package secrets

import (
	"context"
	"fmt"
	"time"

	"github.com/reddit/baseplate.go/log"
)

type Provider int

const (
	// Default Vault provider, uses a sidecar to fetch secrets from Vault.
	VaultProvider Provider = iota

	// Uses Vault CSI to fetch secrets from Vault.
	VaultCsiProvider
)

// Config is the confuration struct for the secrets package.
//
// Can be deserialized from YAML.
type Config struct {
	// Path is the path to the secrets.json file file to load your service's
	// secrets from.
	Path string `yaml:"path"`

	// The secrets provider, acceptable values are 'vault' and 'vault_csi'. Defaults to 'vault'
	Provider string `yaml:"provider"`
}

func (c Config) getProvider() (Provider, error) {
	switch c.Provider {
	case "vault":
		return VaultProvider, nil
	case "vault_csi":
		return VaultCsiProvider, nil
	default:
		return VaultProvider, fmt.Errorf("unknown secret provider %s, must be one of ['vault', 'vault_csi']", c.Provider)
	}
}

// InitFromConfig returns a new *secrets.Store using the given context and config.
func InitFromConfig(ctx context.Context, cfg Config) (*Store, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	store, err := NewStore(ctx, cfg.Path, log.ErrorWithSentryWrapper())
	if err != nil {
		return nil, err
	}
	return store, nil
}

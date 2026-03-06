package config

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont/config"
)

// InitVaultProvider is used to initialize and register the VaultProvider
type InitVaultProvider struct {
	Server     string `config:"VAULT_ADDR"`
	Token      string `config:"VAULT_TOKEN"`
	MountPath  string `config:"VAULT_MOUNT_PATH"`
	SecretPath string `config:"VAULT_SECRET_PATH"`
}

// Initialize creates a VaultProvider with the provided configuration and registers it as the global provider.
func (ivp InitVaultProvider) Initialize(ctx context.Context) (context.Context, error) {
	vaultProvider, err := NewVaultProvider(ivp.Server, ivp.Token, ivp.MountPath, ivp.SecretPath)
	if err != nil {
		return ctx, fmt.Errorf("failed to initialize Vault provider: %w", err)
	}

	config.SetGlobalProvider(
		config.NewCompositeProvider(
			config.EnvVarProvider{},
			vaultProvider,
		),
	)

	return ctx, nil
}

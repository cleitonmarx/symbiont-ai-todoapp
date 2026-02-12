package config

import (
	"context"
	"fmt"

	"github.com/cleitonmarx/symbiont/config"
	"github.com/hashicorp/vault/api"
)

// VaultProvider provides configuration values from HashiCorp Vault.
type VaultProvider struct {
	client     *api.Client
	mountPath  string
	secretPath string
}

// NewVaultProvider creates a new VaultProvider.
//
// The server is the Vault server address (e.g., "http://localhost:8200").
// The token is the Vault authentication token.
// The mountPath is the mount point for the KV secrets engine (e.g., "secret").
// The secretPath is the path to the secret within the mount (e.g., "appname").
func NewVaultProvider(server, token, mountPath, secretPath string) (VaultProvider, error) {
	if server == "" {
		return VaultProvider{}, fmt.Errorf("server is required")
	}
	if token == "" {
		return VaultProvider{}, fmt.Errorf("token is required")
	}
	if mountPath == "" {
		return VaultProvider{}, fmt.Errorf("mountPath is required")
	}
	if secretPath == "" {
		return VaultProvider{}, fmt.Errorf("secretPath is required")
	}

	cfg := api.DefaultConfig()
	cfg.Address = server

	client, err := api.NewClient(cfg)
	if err != nil {
		return VaultProvider{}, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(token)

	vp := VaultProvider{
		client:     client,
		mountPath:  mountPath,
		secretPath: secretPath,
	}

	return vp, nil
}

// Get retrieves a configuration value from Vault.
//
// It looks up the key in the configured secret path.
// Returns an error if the secret or key is not found.
func (vp VaultProvider) Get(ctx context.Context, key string) (string, error) {
	secret, err := vp.client.KVv2(vp.mountPath).Get(ctx, vp.secretPath)
	if err != nil {
		return "", err
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("vault secret %s not found", vp.secretPath)
	}

	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("vault secret %s does not contain key %s", vp.secretPath, key)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("vault secret %s is not a string", key)
	}

	return strValue, nil
}

// Ensure VaultProvider implements config.Provider interface.
var _ config.Provider = (*VaultProvider)(nil)

// InitVaultProvider is used to initialize and register the VaultProvider
type InitVaultProvider struct {
	Server     string `config:"VAULT_ADDR"`
	Token      string `config:"VAULT_TOKEN"`
	MountPath  string `config:"VAULT_MOUNT_PATH"`
	SecretPath string `config:"VAULT_SECRET_PATH"`
}

// Initialize sets up the VaultProvider and registers it in a composite provider as a global config provider.
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

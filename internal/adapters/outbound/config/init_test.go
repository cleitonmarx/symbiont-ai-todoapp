package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitVaultProvider_Initialize(t *testing.T) {
	i := InitVaultProvider{}

	ctx, err := i.Initialize(t.Context())
	require.Error(t, err)
	assert.NotNil(t, ctx)
	assert.Contains(t, err.Error(), "failed to initialize Vault provider")
}

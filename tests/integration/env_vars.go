package integration

import (
	"context"
	"os"
)

// InitEnvVars is responsible for setting environment variables needed for integration tests.
type InitEnvVars struct {
	envVars map[string]string
}

// Initialize sets the specified environment variables for the duration of the tests.
func (i *InitEnvVars) Initialize(ctx context.Context) (context.Context, error) {
	for key, value := range i.envVars {
		_ = os.Setenv(key, value)
	}
	return ctx, nil
}

// Close unsets the environment variables that were set for the tests.
func (i *InitEnvVars) Close() {
	for key := range i.envVars {
		_ = os.Unsetenv(key)
	}
}

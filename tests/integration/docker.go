package integration

import (
	"context"
	"log"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// InitDockerCompose is responsible for starting and stopping the Docker Compose environment for integration tests.
type InitDockerCompose struct {
	compose *compose.DockerCompose
}

// Initialize starts the Docker Compose environment and waits for the specified services to be healthy.
func (i *InitDockerCompose) Initialize(ctx context.Context) (context.Context, error) {
	dc, err := compose.NewDockerCompose("../../docker-compose.deps.yml")
	if err != nil {
		return ctx, err
	}
	i.compose = dc

	err = i.compose.
		WaitForService("postgres", wait.ForHealthCheck()).
		WaitForService("vault", wait.ForHealthCheck()).
		WaitForService("mcp-gateway", wait.ForHealthCheck()).
		Up(ctx, compose.Wait(true))
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

// Close stops the Docker Compose environment and cleans up resources.
func (i InitDockerCompose) Close() {
	if i.compose != nil {
		cancelCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		err := i.compose.Down(
			cancelCtx,
			compose.RemoveOrphans(true),
			compose.RemoveVolumes(true),
			compose.RemoveImages(compose.RemoveImagesLocal),
		)
		if err != nil {
			log.Printf("failed to stop docker compose: %v", err)
		}
	}
}

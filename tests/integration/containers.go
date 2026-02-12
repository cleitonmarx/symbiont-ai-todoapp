package integration

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

type InitDockerCompose struct {
	compose *compose.DockerCompose
}

func (i *InitDockerCompose) Initialize(ctx context.Context) (context.Context, error) {
	dc, err := compose.NewDockerCompose("../../docker-compose.deps.yml")
	if err != nil {
		return ctx, err
	}
	i.compose = dc

	err = i.compose.
		WaitForService("postgres", wait.ForHealthCheck()).
		WaitForService("vault", wait.ForHealthCheck()).
		Up(ctx, compose.Wait(true))
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

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

type initEnvVars struct {
	envVars map[string]string
}

func (i *initEnvVars) Initialize(ctx context.Context) (context.Context, error) {
	for key, value := range i.envVars {
		_ = os.Setenv(key, value)
	}
	return ctx, nil
}

func (i *initEnvVars) Close() {
	for key := range i.envVars {
		_ = os.Unsetenv(key)
	}
}

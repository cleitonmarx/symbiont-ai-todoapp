package integration

import (
	"context"
	"log"
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
		WaitForService("postgres", wait.NewLogStrategy(
			"database system is ready to accept connections",
		)).
		WaitForService("vault", wait.NewLogStrategy(
			"Vault server started!",
		)).
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

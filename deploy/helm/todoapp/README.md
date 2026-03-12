# todoapp Helm chart

This chart deploys:

- App split workloads: `http-api`, `graphql-api`, `message-relay`, `board-summary-generator`, `chat-summary-generator`, `conversation-title-generator`
- In-cluster dependencies: PostgreSQL (pgvector), Vault (dev mode), Pub/Sub emulator, MCP gateway (docker-compose parity mode)

## Key values

- `image.repository`, `image.tag`, `image.pullPolicy`
- `services.http.type`, `services.graphql.type`
- `services.http.nodePort`, `services.graphql.nodePort` (if using `NodePort`)
- `ingress.enabled`, `ingress.className`, `ingress.hosts.http`, `ingress.hosts.graphql`, `ingress.annotations`
- `replicas.*`
- `env.common`
- `env.secrets.*`
- `postgres.persistence.size`, `postgres.persistence.storageClass`, `postgres.persistence.mountPath`
- `vault.devToken`, `vault.mountPath`, `vault.secretPath`
- `vault.initJob.enabled`
- `pubsub.projectId`, `pubsub.topicIds.*`, `pubsub.subscriptionIds.*`, `pubsub.subscriptionPrefixes.*`
- `mcp.transport`, `mcp.servers`, `mcp.tools`, `mcp.dockerSocket.*`

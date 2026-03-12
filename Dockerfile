# syntax=docker/dockerfile:1.7

## Node build stage for webapp
FROM node:22-alpine AS webapp-builder

WORKDIR /webapp

COPY webapp/package.json webapp/package-lock.json ./

ARG VITE_API_BASE_URL=
ARG VITE_GRAPHQL_ENDPOINT=

ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_GRAPHQL_ENDPOINT=$VITE_GRAPHQL_ENDPOINT

RUN --mount=type=cache,target=/root/.npm npm ci --legacy-peer-deps

COPY webapp/ ./

RUN npm run build

## Go build stage
FROM golang:1.25.3 AS go-builder

WORKDIR /build

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download -x

COPY . .

# Copy webapp static files
COPY --from=webapp-builder /webapp/dist ./internal/adapters/inbound/http/webappdist

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    set -eux; \ 
    CGO_ENABLED=0 GOOS=linux go build -trimpath -v -o /out/healthchecker ./cmd/health-checker;\
    for cmd in monolithic http-api graphql-api message-relay board-summary-generator chat-summary-generator conversation-title-generator; do \
      CGO_ENABLED=0 GOOS=linux go build -trimpath -v -o /out/${cmd} ./cmd/${cmd}; \
    done

## Minimal runtime image
FROM scratch

COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=go-builder /out/ /

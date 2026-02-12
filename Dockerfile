## Node build stage for webapp
FROM node:22-alpine AS webapp-builder

WORKDIR /webapp

COPY webapp/ ./

ARG VITE_API_BASE_URL=http://localhost:8080
ARG VITE_GRAPHQL_ENDPOINT=http://localhost:8085/v1/query

ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_GRAPHQL_ENDPOINT=$VITE_GRAPHQL_ENDPOINT

RUN npm install --legacy-peer-deps

RUN npm run build

## Go build stage
FROM golang:1.25.3 AS go-builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download -x

COPY . .

##WORKDIR /build/todoapp

# Copy webapp static files
COPY --from=webapp-builder /webapp/dist ./internal/adapters/inbound/http/webappdist

RUN CGO_ENABLED=0 GOOS=linux go build -v -o todoapp ./cmd/todoapp

## Minimal runtime image
FROM scratch

COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=go-builder /build/todoapp .

CMD ["/todoapp"]
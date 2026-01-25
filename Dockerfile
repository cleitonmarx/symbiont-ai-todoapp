## Node build stage for webapp
FROM node:18-alpine AS webapp-builder

WORKDIR /webapp

COPY examples/todoapp/webapp/ ./

RUN npm install --legacy-peer-deps

RUN npm run build

## Go build stage
FROM golang:1.25.3 AS go-builder

WORKDIR /build

# Copy the parent symbiont module to /build
COPY . .

WORKDIR /build/examples/todoapp

# Copy webapp static files
COPY --from=webapp-builder /webapp/dist ./internal/adapters/inbound/http/webappdist

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o todoapp ./cmd/todoapp

## Minimal runtime image
FROM scratch

COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=go-builder /build/examples/todoapp/todoapp .

CMD ["/todoapp"]
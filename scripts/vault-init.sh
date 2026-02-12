#!/bin/sh

export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root-token'

echo "Waiting for Vault to be ready..."

# Wait for Vault to be ready with better retry logic
max_retries=30
count=0
until vault status > /dev/null 2>&1; do
  count=$((count + 1))
  if [ $count -ge $max_retries ]; then
    echo "Vault failed to start after $max_retries attempts"
    exit 1
  fi
  echo "Waiting for Vault... attempt $count/$max_retries"
  sleep 2
done

echo "Vault is ready. Initializing secrets..."

# Enable KV v2 secrets engine (ignore error if already enabled)
vault secrets enable -path=secret kv-v2 2>/dev/null || echo "KV secrets engine already enabled"

# Add application secrets
echo "Creating todoapp secrets..."
vault kv put secret/todoapp \
  DB_USER=todoapp \
  DB_PASS=todoapppass

if [ $? -eq 0 ]; then
  echo "✓ Created secret/todoapp"
else
  echo "✗ Failed to create secret/todoapp"
  exit 1
fi

echo ""
echo "=== Vault initialization complete! ==="
echo ""
echo "Available secrets:"
vault kv list secret/
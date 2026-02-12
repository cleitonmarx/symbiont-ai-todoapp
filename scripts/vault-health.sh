#!/bin/sh

export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='root-token'

# Check if Vault is unsealed and ready
if ! vault status > /dev/null 2>&1; then
  exit 1
fi

# Check if secrets exist
if ! vault kv get secret/todoapp > /dev/null 2>&1; then
  exit 1
fi

exit 0
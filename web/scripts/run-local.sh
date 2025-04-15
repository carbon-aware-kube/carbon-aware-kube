#!/bin/bash

# Export environment variables from .env file
set -a
[ -f .env ] && . ./.env
set +a

# Run the Go webserver
go run ./cmd/api/main.go
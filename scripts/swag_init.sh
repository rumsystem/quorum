#!/bin/bash
cd "$(dirname "$0")"
cd ..
swag init -g ./cmd/main.go --parseDependency --parseInternal --parseDepth 2

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
PROTO_DIR="$ROOT_DIR/third_party/proto/procurator.core"
OUT_DIR="$ROOT_DIR/internal/pb/procuratorcore"

mkdir -p "$OUT_DIR"

cat <<MSG
Для полноценной генерации нужны:
  protoc
  protoc-gen-go
  protoc-gen-go-grpc

Ожидаемая команда будет примерно такой:

protoc \
  -I "$PROTO_DIR" \
  -I /usr/local/include \
  --go_out="$OUT_DIR" \
  --go_opt=paths=source_relative \
  --go-grpc_out="$OUT_DIR" \
  --go-grpc_opt=paths=source_relative \
  "$PROTO_DIR"/*.proto
MSG

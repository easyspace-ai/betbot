#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FLAGS=()

if [[ "${PUBLIC_ONLY:-}" == "1" ]]; then
  FLAGS+=("--public-only")
fi
if [[ "${SKIP_WS:-}" == "1" ]]; then
  FLAGS+=("--skip-ws")
fi
if [[ "${SKIP_RTDS:-}" == "1" ]]; then
  FLAGS+=("--skip-rtds")
fi
if [[ "${STRICT:-}" == "1" ]]; then
  FLAGS+=("--strict")
fi
if [[ -n "${TIMEOUT:-}" ]]; then
  FLAGS+=("--timeout" "$TIMEOUT")
fi
if [[ -n "${POLYMARKET_TOKEN_ID:-}" ]]; then
  FLAGS+=("--token" "$POLYMARKET_TOKEN_ID")
fi
if [[ -n "${POLYMARKET_MARKET_ID:-}" ]]; then
  FLAGS+=("--market" "$POLYMARKET_MARKET_ID")
fi

if (( ${#FLAGS[@]} )); then
  exec go run ./cmd/acceptance "${FLAGS[@]}"
fi
exec go run ./cmd/acceptance

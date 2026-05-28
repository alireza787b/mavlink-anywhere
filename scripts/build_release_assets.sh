#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="$(. "$ROOT_DIR/lib/common.sh"; printf '%s' "$MAVLINK_ANYWHERE_VERSION")"
OUT_DIR="${1:-$ROOT_DIR/dist}"
BUILD_TIME="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
GO_BIN="${GO_BIN:-go}"

mkdir -p "$OUT_DIR"
cd "$ROOT_DIR/dashboard"

build_one() {
    local goarch="$1"
    local goarm="$2"
    local suffix="$3"
    local output="$OUT_DIR/mavlink-anywhere-${suffix}"

    echo "building ${output}"
    if [[ -n "$goarm" ]]; then
        env CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" GOARM="$goarm" \
            "$GO_BIN" build -trimpath -ldflags "-s -w -X main.Version=v${VERSION} -X main.BuildTime=${BUILD_TIME}" \
            -o "$output" ./cmd
    else
        env CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" \
            "$GO_BIN" build -trimpath -ldflags "-s -w -X main.Version=v${VERSION} -X main.BuildTime=${BUILD_TIME}" \
            -o "$output" ./cmd
    fi
}

build_one amd64 "" linux-amd64
build_one arm64 "" linux-arm64
build_one arm 6 linux-arm6

"$OUT_DIR/mavlink-anywhere-linux-amd64" --version
printf 'release-smoke-password' | "$OUT_DIR/mavlink-anywhere-linux-amd64" --hash-password >/dev/null
sha256sum "$OUT_DIR"/mavlink-anywhere-linux-*

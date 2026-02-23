#!/usr/bin/env bash
set -euo pipefail

BINARY="ultimate-proxy-profile-switcher"
LDFLAGS="-s -w"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

rm -rf dist
mkdir -p dist

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS="${PLATFORM%/*}"
    GOARCH="${PLATFORM#*/}"

    EXT=""
    if [ "$GOOS" = "windows" ]; then
        EXT=".exe"
    fi

    STAGE="${BINARY}_${VERSION}_${GOOS}_${GOARCH}"
    STAGE_DIR="dist/${STAGE}"
    mkdir -p "${STAGE_DIR}"

    echo "→ Building ${GOOS}/${GOARCH}..."
    CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
        -ldflags="${LDFLAGS}" \
        -trimpath \
        -o "${STAGE_DIR}/${BINARY}${EXT}" .

    cp config.example.yaml "${STAGE_DIR}/"

    if [ "$GOOS" = "windows" ]; then
        (cd dist && zip -r "${STAGE}.zip" "${STAGE}")
    else
        (cd dist && tar -czf "${STAGE}.tar.gz" "${STAGE}")
    fi

    rm -rf "${STAGE_DIR}"
    echo "   dist/${STAGE}.$([ "$GOOS" = "windows" ] && echo zip || echo tar.gz)"
done

echo ""
echo "Done. Artifacts in dist/:"
ls dist/

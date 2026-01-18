#!/usr/bin/env bash
#
# Build MCPB bundles for Claude Desktop
# Usage: ./mcpb/build.sh [version]
#
# Examples:
#   ./mcpb/build.sh dev      # Build development bundles
#   ./mcpb/build.sh v0.1.0   # Build release bundles
#
set -euo pipefail

VERSION="${1:-dev}"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MCPB_DIR="${PROJECT_ROOT}/mcpb"
BUILD_DIR="${PROJECT_ROOT}/dist/mcpb"

# Target platforms for MCPB (Claude Desktop supported platforms)
PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

echo "Building MCPB bundles for mcp-s3 ${VERSION}..."
echo "Project root: ${PROJECT_ROOT}"

# Clean and create build directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

for platform in "${PLATFORMS[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"

    BUNDLE_NAME="mcp-s3_${VERSION}_${GOOS}_${GOARCH}.mcpb"
    BUNDLE_DIR="${BUILD_DIR}/${GOOS}_${GOARCH}"

    echo "Building for ${GOOS}/${GOARCH}..."

    # Create bundle directory
    mkdir -p "${BUNDLE_DIR}"

    # Build binary
    BINARY_NAME="mcp-s3"
    if [ "${GOOS}" = "windows" ]; then
        BINARY_NAME="mcp-s3.exe"
    fi

    CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
        -trimpath \
        -ldflags="-s -w -X github.com/txn2/mcp-s3/internal/server.Version=${VERSION}" \
        -o "${BUNDLE_DIR}/${BINARY_NAME}" \
        "${PROJECT_ROOT}/cmd/mcp-s3"

    # Copy manifest
    cp "${MCPB_DIR}/manifest.json" "${BUNDLE_DIR}/"

    # Create the .mcpb bundle (ZIP format)
    echo "Creating ${BUNDLE_NAME}..."
    (cd "${BUNDLE_DIR}" && zip -r "${BUILD_DIR}/${BUNDLE_NAME}" .)

    # Generate SHA256 checksum
    (cd "${BUILD_DIR}" && sha256sum "${BUNDLE_NAME}" > "${BUNDLE_NAME}.sha256")

    # Cleanup temporary directory
    rm -rf "${BUNDLE_DIR}"

    echo "Created: ${BUILD_DIR}/${BUNDLE_NAME}"
done

echo ""
echo "MCPB bundles created in ${BUILD_DIR}:"
ls -la "${BUILD_DIR}"/*.mcpb

echo ""
echo "Checksums:"
cat "${BUILD_DIR}"/*.sha256

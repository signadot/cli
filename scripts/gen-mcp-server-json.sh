#!/usr/bin/env bash
# gen-mcp-server-json.sh - Generate server.json for MCP registry publishing.
#
# Usage: ./scripts/gen-mcp-server-json.sh <version>
# Example: ./scripts/gen-mcp-server-json.sh 1.5.0
#
# Downloads the release archives for the given version, computes their SHA256
# hashes, and writes server.json to the current directory. Then prints the
# commands needed to publish to the MCP registry.

set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  echo "Usage: $0 <version>" >&2
  echo "Example: $0 1.5.0" >&2
  exit 1
fi

# Strip leading 'v' if present so callers can pass either form.
VERSION="${VERSION#v}"
TAG="v${VERSION}"

BASE_URL="https://github.com/signadot/cli/releases/download/${TAG}"
PLATFORMS=("linux_amd64" "linux_arm64" "darwin_amd64" "darwin_arm64")
OUTPUT="server.json"

# Portable SHA256
sha256() {
  if command -v sha256sum &>/dev/null; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading release archives for ${TAG}..." >&2

PACKAGES=""
for PLATFORM in "${PLATFORMS[@]}"; do
  ARCHIVE="signadot-cli_mcp_${PLATFORM}.tar.gz"
  URL="${BASE_URL}/${ARCHIVE}"
  DEST="${TMPDIR}/${ARCHIVE}"

  echo "  ${ARCHIVE}" >&2
  curl -fsSL -o "$DEST" "$URL"

  HASH=$(sha256 "$DEST")
  echo "    sha256: ${HASH}" >&2

  [ -n "$PACKAGES" ] && PACKAGES="${PACKAGES},"
  PACKAGES="${PACKAGES}
    {
      \"registryType\": \"mcpb\",
      \"identifier\": \"${URL}\",
      \"fileSha256\": \"${HASH}\",
      \"transport\": {
        \"type\": \"stdio\"
      },
      \"environmentVariables\": [
        {
          \"description\": \"Signadot API Key\",
          \"isRequired\": true,
          \"isSecret\": true,
          \"name\": \"SIGNADOT_API_KEY\"
        }
      ]
    }"
done

cat > "$OUTPUT" <<EOF
{
  "\$schema": "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json",
  "name": "io.github.signadot/cli",
  "description": "Connect to Signadot to manage ephemeral environments and route traffic to local services.",
  "title": "Signadot MCP Server",
  "repository": {
    "url": "https://github.com/signadot/cli",
    "source": "github"
  },
  "version": "${VERSION}",
  "packages": [${PACKAGES}
  ]
}
EOF

echo "" >&2
echo "Generated ${OUTPUT}" >&2
echo "" >&2

echo "Run the following commands to publish to the MCP registry:"
echo ""
echo "  mcp-publisher login github"
echo "  mcp-publisher publish ${OUTPUT}"

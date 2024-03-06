#!/bin/sh
#
# Signadot CLI installer
#
# Usage:
#   curl -sSLf https://raw.githubusercontent.com/signadot/cli/main/scripts/install.sh | sh

echo "Installing Signadot CLI"

if [ ! -z "${DEBUG}" ];
then set -x
fi

_detect_arch() {
    case $(uname -m) in
    amd64|x86_64) echo "amd64"
    ;;
    arm64|aarch64) echo "arm64"
    ;;
    *) echo "Unsupported processor architecture";
    return 1
    ;;
     esac
}

_detect_os(){
    case $(uname) in
    Linux) echo "linux"
    ;;
    Darwin) echo "darwin"
    ;;
    *) echo "Unsupported os";
    return 1
    ;;
    esac
}

_download_url() {
  local arch
  local os
  local tag
  local version

  arch="$(_detect_arch)"
  os="$(_detect_os)"
  if [ -z "$SIGNADOT_CLI_VERSION" ]; then
      tag="$(
        curl -s "https://api.github.com/repos/signadot/cli/releases/latest" \
        2>/dev/null \
        | jq -r '.tag_name' \
      )"
  else
    tag="$SIGNADOT_CLI_VERSION"
  fi

  echo "https://github.com/signadot/cli/releases/download/${tag}/signadot-cli_${os}_${arch}.tar.gz"
}

echo "Downloading signadot binary from URL: $(_download_url)"
curl -sSLf "$(_download_url)" > signadot-cli.tar.gz

tar -xzf signadot-cli.tar.gz signadot
rm signadot-cli.tar.gz

if [ -z "$SIGNADOT_CLI_PATH" ]; then
    cli_path="/usr/local/bin/signadot"
    sudo mv signadot $cli_path
else
    cli_path="${SIGNADOT_CLI_PATH%/}" # remove traling / if present
    cli_path="$cli_path/signadot"
    mv signadot $cli_path
fi
echo "signadot installed in:"
echo "- $cli_path"
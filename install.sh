#!/usr/bin/env bash
set -euo pipefail

REPO="QMZCLL/cronix"
BINARY="cronix"
INSTALL_DIR="/usr/local/bin"

OS="$(uname -s)"
if [ "${OS}" != "Linux" ]; then
  echo "error: cronix installer only supports Linux (got ${OS})" >&2
  exit 1
fi

ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)
    ARCH_SUFFIX="amd64"
    ;;
  aarch64 | arm64)
    ARCH_SUFFIX="arm64"
    ;;
  *)
    echo "error: unsupported architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

ARTIFACT="cronix-linux-${ARCH_SUFFIX}"

echo "Fetching latest release info for ${REPO}..."
LATEST_TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | head -1 \
  | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"

if [ -z "${LATEST_TAG}" ]; then
  echo "error: could not determine latest release tag" >&2
  exit 1
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ARTIFACT}"

echo "Installing ${BINARY} ${LATEST_TAG} (linux/${ARCH_SUFFIX})"
echo "Downloading ${DOWNLOAD_URL}..."

TMP_FILE="$(mktemp)"
trap 'rm -f "${TMP_FILE}"' EXIT

curl -fsSL -o "${TMP_FILE}" "${DOWNLOAD_URL}"
chmod +x "${TMP_FILE}"

if [ -w "${INSTALL_DIR}" ]; then
  mv "${TMP_FILE}" "${INSTALL_DIR}/${BINARY}"
else
  echo "${INSTALL_DIR} is not writable, trying sudo..."
  sudo mv "${TMP_FILE}" "${INSTALL_DIR}/${BINARY}"
fi

if ! command -v "${BINARY}" >/dev/null 2>&1; then
  echo "error: ${BINARY} not found in PATH after installation" >&2
  exit 1
fi

INSTALLED_VERSION="$("${BINARY}" --version 2>&1 || true)"
echo "Successfully installed: ${INSTALLED_VERSION}"
echo "Run 'cronix init' to get started."

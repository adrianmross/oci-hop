#!/usr/bin/env bash
set -euo pipefail

repo="adrianmross/oci-hop"
prefix="${PREFIX:-/usr/local}"
bin_dir="${prefix}/bin"
version="${VERSION:-latest}"
tool="oci-hop"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required dependency: $1" >&2
    exit 1
  fi
}

for cmd in curl tar install grep awk uname mktemp; do
  require_cmd "$cmd"
done

if [[ "${version}" == "latest" ]]; then
  redirected=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${repo}/releases/latest" || true)
  version=$(basename "${redirected}")
fi
if [[ -z "${version}" || "${version}" == "null" ]]; then
  echo "Unable to determine release version." >&2
  exit 1
fi

uname_s=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

version_no_v="${version#v}"
asset="${tool}_${version_no_v}_${uname_s}_${arch}.tar.gz"
tmp_dir=$(mktemp -d)
trap 'rm -rf "${tmp_dir}"' EXIT

curl -fsSL --retry 3 --retry-all-errors -o "${tmp_dir}/${asset}" "https://github.com/${repo}/releases/download/${version}/${asset}"
curl -fsSL --retry 3 --retry-all-errors -o "${tmp_dir}/checksums.txt" "https://github.com/${repo}/releases/download/${version}/checksums.txt"
expected_checksum=$(grep "  ${asset}$" "${tmp_dir}/checksums.txt" | awk '{print $1}')
if command -v shasum >/dev/null 2>&1; then
  actual_checksum=$(shasum -a 256 "${tmp_dir}/${asset}" | awk '{print $1}')
else
  actual_checksum=$(sha256sum "${tmp_dir}/${asset}" | awk '{print $1}')
fi
if [[ "${expected_checksum}" != "${actual_checksum}" ]]; then
  echo "Checksum mismatch for ${asset}" >&2
  exit 1
fi
tar -xzf "${tmp_dir}/${asset}" -C "${tmp_dir}"
install -d "${bin_dir}"
for binary in oci-hop hop; do
  if [[ -x "${tmp_dir}/${binary}" ]]; then
    install "${tmp_dir}/${binary}" "${bin_dir}/${binary}"
    echo "Installed ${binary} to ${bin_dir}/${binary}"
  fi
done

#!/usr/bin/env sh
# Installs the xo CLI from the latest GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/littlejo/xo-gocli/main/install.sh | sh
#
# Override the default values with environment variables:
#   XOA_INSTALL_REPO   GitHub repository (owner/name), default littlejo/xo-gocli
#   XOA_INSTALL_VERSION  Specific version to install, default: latest release
#   XOA_INSTALL_DIR    Installation directory, default /usr/local/bin
#                     (falls back to ~/.local/bin when not writable)

set -eu

REPO="${XOA_INSTALL_REPO:-littlejo/xo-gocli}"
INSTALL_DIR="${XOA_INSTALL_DIR:-}"

err() {
    printf 'error: %s\n' "$*" >&2
    exit 1
}

say() {
    printf '==> %s\n' "$*"
}

# --- OS / architecture -------------------------------------------------------

detect_os() {
    case "$(uname -s)" in
    Linux)  echo linux ;;
    Darwin) echo darwin ;;
    MINGW* | MSYS* | CYGWIN*) err "Windows is not supported by this script; download the .zip from the releases page" ;;
    *)      err "unsupported operating system: $(uname -s)" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
    x86_64 | amd64) echo amd64 ;;
    arm64 | aarch64) echo arm64 ;;
    *)              err "unsupported architecture: $(uname -m) (only amd64 and arm64 are available)" ;;
    esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
VERSION="${XOA_INSTALL_VERSION:-}"

if [ -z "$VERSION" ]; then
    # -f makes curl fail on HTTP errors (e.g. no release yet); the resulting
    # empty tag_name is caught by the test below.
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' \
        | head -n1)"
    [ -n "$VERSION" ] || err "no release found for ${REPO} (is one published?)"
fi
# goreleaser tags may or may not carry a leading "v"; asset names do not.
VERSION="${VERSION#v}"

# --- Download ----------------------------------------------------------------

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

archive="xo_${VERSION}_${OS}_${ARCH}.tar.gz"
checksum_file="xo_${VERSION}_checksums.txt"

say "installing xo ${VERSION} (${OS}/${ARCH})"

download() {
    url="https://github.com/${REPO}/releases/download/v${VERSION}/$1"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$TMPDIR/$1" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$TMPDIR/$1" "$url"
    else
        err "curl or wget is required to download ${url}"
    fi
}

download "$archive"

# Checksum verification: done when a sha256 tool is available, skipped (with a
# warning) otherwise.
HAVE_VERIFY=0
if command -v sha256sum >/dev/null 2>&1; then
    do_verify() { sha256sum "$1" | awk '{print $1}'; }
    HAVE_VERIFY=1
elif command -v shasum >/dev/null 2>&1; then
    do_verify() { shasum -a 256 "$1" | awk '{print $1}'; }
    HAVE_VERIFY=1
fi

if [ "$HAVE_VERIFY" -eq 1 ]; then
    download "$checksum_file" || err "failed to download ${checksum_file}"
    expected="$(awk -v f="$archive" '$2 == f {print tolower($1)}' "$TMPDIR/$checksum_file" | head -n1)"
    [ -n "$expected" ] || err "${archive} not found in ${checksum_file}"
    actual="$(do_verify "$TMPDIR/$archive")"
    [ "$expected" = "$actual" ] || err "checksum mismatch for $archive (expected $expected, got $actual)"
    say "checksum verified"
else
    say "warning: no sha256sum/shasum found, checksum not verified"
fi

# --- Install -----------------------------------------------------------------

if command -v tar >/dev/null 2>&1; then
    tar -xzf "$TMPDIR/$archive" -C "$TMPDIR"
else
    err "tar is required to extract $archive"
fi

[ -f "$TMPDIR/xo" ] || err "the archive does not contain the xo binary"

if [ -z "$INSTALL_DIR" ]; then
    if [ -w /usr/local/bin ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="${HOME}/.local/bin"
        mkdir -p "$INSTALL_DIR"
        case "$PATH" in
        *"$INSTALL_DIR"*) : ;;
        *) say "NOTE: add $INSTALL_DIR to your PATH" ;;
        esac
    fi
fi
mkdir -p "$INSTALL_DIR" || err "cannot create $INSTALL_DIR"
cp "$TMPDIR/xo" "$INSTALL_DIR/xo"
chmod 0755 "$INSTALL_DIR/xo"

say "xo installed to $INSTALL_DIR/xo"

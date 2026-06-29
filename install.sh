#!/usr/bin/env sh
set -eu

BINARY_NAME="ucloud-sandbox-cli"
DEFAULT_BIN_DIR="/usr/local/bin"
DEFAULT_BASE_URL="https://github.com/ucloud/ucloud-sandbox-cli/releases"
DOCS_URL="https://astraflow.ucloud.cn/docs/agent-sandbox/product/cli"

BOLD="$(tput bold 2>/dev/null || printf '')"
GREEN="$(tput setaf 2 2>/dev/null || printf '')"
YELLOW="$(tput setaf 3 2>/dev/null || printf '')"
BLUE="$(tput setaf 4 2>/dev/null || printf '')"
RED="$(tput setaf 1 2>/dev/null || printf '')"
NO_COLOR="$(tput sgr0 2>/dev/null || printf '')"

BIN_DIR="${BIN_DIR:-$DEFAULT_BIN_DIR}"
VERSION="${VERSION:-latest}"
BASE_URL="${BASE_URL:-$DEFAULT_BASE_URL}"
YES=0
TMP_DIR=""

info() {
	printf '%s\n' "${BOLD}>${NO_COLOR} $*"
}

warn() {
	printf '%s\n' "${YELLOW}! $*${NO_COLOR}"
}

error() {
	printf '%s\n' "${RED}x $*${NO_COLOR}" >&2
}

success() {
	printf '%s\n' "${GREEN}+${NO_COLOR} $*"
}

has() {
	command -v "$1" >/dev/null 2>&1
}

usage() {
	cat <<EOF
install.sh [options]

Download and install ${BINARY_NAME}.

Options:
  -y, --yes              Skip confirmation prompts
  -p, --path <dir>       Installation directory [default: ${DEFAULT_BIN_DIR}]
  -v, --version <ver>    Version to install [default: latest]
  -u, --url <url>        Release download base URL [default: ${DEFAULT_BASE_URL}]
  -h, --help             Show this help message

Examples:
  sh install.sh
  sh install.sh -y -p "\$HOME/.local/bin"
  sh install.sh -v v1.2.3

When running through curl:
  curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y
EOF
}

need_value() {
	if [ "$#" -lt 2 ] || [ -z "$2" ]; then
		error "Option $1 requires a value."
		exit 1
	fi
}

read_from_tty() {
	if [ -r /dev/tty ]; then
		IFS= read -r REPLY </dev/tty
	else
		IFS= read -r REPLY
	fi
}

choose_install_dir() {
	if [ "$YES" -eq 1 ]; then
		return 0
	fi

	printf 'Installation directory [%s]: ' "$BIN_DIR" >/dev/tty 2>/dev/null || printf 'Installation directory [%s]: ' "$BIN_DIR"
	if ! read_from_tty; then
		error "Unable to read installation directory. Re-run with -y or pass -p <dir>."
		exit 1
	fi

	if [ -n "$REPLY" ]; then
		BIN_DIR="$REPLY"
	fi

	if [ -z "$BIN_DIR" ]; then
		error "Installation directory cannot be empty."
		exit 1
	fi
}

detect_os() {
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	case "$os" in
		linux)
			printf 'linux'
			;;
		darwin)
			printf 'darwin'
			;;
		*)
			error "Unsupported operating system: $os"
			info "Supported operating systems are: linux, darwin."
			exit 1
			;;
	esac
}

detect_arch() {
	arch="$(uname -m | tr '[:upper:]' '[:lower:]')"
	case "$arch" in
		x86_64 | amd64)
			printf 'amd64'
			;;
		arm64 | aarch64)
			printf 'arm64'
			;;
		*)
			error "Unsupported architecture: $arch"
			info "Supported architectures are: amd64, arm64."
			exit 1
			;;
	esac
}

build_url() {
	asset="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
	base="${BASE_URL%/}"

	if [ "$VERSION" = "latest" ]; then
		printf '%s/latest/download/%s' "$base" "$asset"
	else
		printf '%s/download/%s/%s' "$base" "$VERSION" "$asset"
	fi
}

download() {
	file="$1"
	url="$2"

	if has curl; then
		curl --fail --location --progress-bar --output "$file" "$url"
	elif has wget; then
		wget --output-document="$file" "$url"
	else
		error "Neither curl nor wget was found."
		info "Please install curl or wget first, then run this installer again."
		exit 1
	fi
}

make_tmp_dir() {
	if has mktemp; then
		TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t ucloud-sandbox-cli)"
	else
		TMP_DIR="/tmp/ucloud-sandbox-cli-install.$$"
		mkdir -p "$TMP_DIR"
	fi
}

cleanup() {
	if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
		rm -rf "$TMP_DIR"
	fi
}

test_writable() {
	dir="$1"
	test_file="${dir}/.ucloud-sandbox-cli-install-test.$$"

	if touch "$test_file" 2>/dev/null; then
		rm -f "$test_file"
		return 0
	fi

	return 1
}

elevate_privileges() {
	if ! has sudo; then
		error "The installation directory is not writable and sudo is not available."
		info "Install sudo, run this installer as root, or pass -p with a writable directory."
		exit 1
	fi

	if ! sudo -v; then
		error "Could not obtain sudo permissions."
		exit 1
	fi
}

prepare_install_dir() {
	SUDO=""

	if [ -d "$BIN_DIR" ]; then
		if test_writable "$BIN_DIR"; then
			return 0
		fi

		warn "Escalated permissions are required to write to ${BIN_DIR}."
		elevate_privileges
		SUDO="sudo"
		return 0
	fi

	if mkdir -p "$BIN_DIR" 2>/dev/null && test_writable "$BIN_DIR"; then
		return 0
	fi

	warn "Escalated permissions are required to create ${BIN_DIR}."
	elevate_privileges
	SUDO="sudo"
	$SUDO mkdir -p "$BIN_DIR"
}

install_binary() {
	archive="$1"
	extract_dir="${TMP_DIR}/extract"
	mkdir -p "$extract_dir"

	tar -xzf "$archive" -C "$extract_dir"

	if [ ! -f "${extract_dir}/${BINARY_NAME}" ]; then
		error "Archive did not contain ${BINARY_NAME}."
		exit 1
	fi

	chmod +x "${extract_dir}/${BINARY_NAME}"
	$SUDO cp "${extract_dir}/${BINARY_NAME}" "${BIN_DIR}/${BINARY_NAME}"
	$SUDO chmod 755 "${BIN_DIR}/${BINARY_NAME}"
}

verify_installation() {
	printf '\n'
	if [ -x "${BIN_DIR}/${BINARY_NAME}" ]; then
		info "Installed binary version:"
		"${BIN_DIR}/${BINARY_NAME}" version
	fi

	if has "$BINARY_NAME"; then
		found_path="$(command -v "$BINARY_NAME")"
		if [ "$found_path" != "${BIN_DIR}/${BINARY_NAME}" ]; then
			warn "${BINARY_NAME} is available in PATH as ${found_path}, which is different from ${BIN_DIR}/${BINARY_NAME}."
		fi
		return 0
	fi

	warn "${BINARY_NAME} was installed to ${BIN_DIR}, but the command was not found in PATH."
	info "Add the installation directory to PATH, for example:"
	printf '  export PATH="%s:$PATH"\n' "$BIN_DIR"
}

parse_args() {
	while [ "$#" -gt 0 ]; do
		case "$1" in
			-y | --yes)
				YES=1
				shift
				;;
			-p | --path)
				need_value "$1" "${2-}"
				BIN_DIR="$2"
				shift 2
				;;
			-p=* | --path=*)
				BIN_DIR="${1#*=}"
				shift
				;;
			-v | --version)
				need_value "$1" "${2-}"
				VERSION="$2"
				shift 2
				;;
			-v=* | --version=*)
				VERSION="${1#*=}"
				shift
				;;
			-u | --url)
				need_value "$1" "${2-}"
				BASE_URL="$2"
				shift 2
				;;
			-u=* | --url=*)
				BASE_URL="${1#*=}"
				shift
				;;
			-h | --help)
				usage
				exit 0
				;;
			*)
				error "Unknown option: $1"
				usage
				exit 1
				;;
		esac
	done

	if [ -z "$BIN_DIR" ]; then
		error "Installation directory cannot be empty."
		exit 1
	fi
	if [ -z "$VERSION" ]; then
		error "Version cannot be empty."
		exit 1
	fi
	if [ -z "$BASE_URL" ]; then
		error "Download URL cannot be empty."
		exit 1
	fi
}

main() {
	parse_args "$@"

	printf '\n'
	info "Welcome to the ${BINARY_NAME} installer."

	if ! has tar; then
		error "tar was not found."
		info "Please install tar first, then run this installer again."
		exit 1
	fi

	OS="$(detect_os)"
	ARCH="$(detect_arch)"
	URL="$(build_url)"

	info "Installer configuration:"
	info "  Version: ${GREEN}${VERSION}${NO_COLOR}"
	info "  OS: ${GREEN}${OS}${NO_COLOR}"
	info "  Arch: ${GREEN}${ARCH}${NO_COLOR}"
	info "  Install dir: ${GREEN}${BIN_DIR}${NO_COLOR}"
	info "  Download URL: ${BLUE}${URL}${NO_COLOR}"
	printf '\n'

	choose_install_dir
	prepare_install_dir

	make_tmp_dir
	trap cleanup EXIT INT TERM

	archive="${TMP_DIR}/${BINARY_NAME}.tar.gz"
	info "Downloading ${BINARY_NAME}..."
	if ! download "$archive" "$URL"; then
		error "Download failed: $URL"
		if [ "$VERSION" != "latest" ]; then
			info "Make sure the release tag matches exactly, for example v1.2.3 if tags use the v prefix."
		fi
		exit 1
	fi

	info "Installing ${BINARY_NAME} to ${BIN_DIR}..."
	install_binary "$archive"
	success "${BINARY_NAME} installed successfully."

	verify_installation

	printf '\n'
	info "Documentation: ${DOCS_URL}"
	info "Run '${BINARY_NAME} login' first, then start using the CLI."
}

main "$@"

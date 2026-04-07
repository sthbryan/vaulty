#!/usr/bin/env bash
#
# Vaulty Auto-Installer
# Usage: curl -sSL https://raw.githubusercontent.com/sthbryan/vaulty/main/install.sh | bash
#
set -e

REPO_OWNER="sthbryan"
REPO_NAME="vaulty"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="vty"
PATH_LINE='export PATH="$HOME/.local/bin:$PATH"'

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*)    echo "darwin" ;;
        Linux*)     echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)          echo "unsupported" ;;
    esac
}

# Detect Architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64)           echo "amd64" ;;
        arm64|aarch64)    echo "arm64" ;;
        *)                echo "amd64" ;;
    esac
}

# Get latest release info
get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    curl -sSL "$api_url" 2>/dev/null | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4
}

# Get download URL
get_download_url() {
    local os="$1"
    local arch="$2"
    local version="$3"
    
    local asset_name="${BINARY_NAME}-${os}-${arch}"
    [ "$os" = "windows" ] && asset_name="${asset_name}.exe"
    
    echo "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${asset_name}"
}

# Remove quarantine attribute on macOS
remove_quarantine() {
    local file="$1"
    if [ "$(detect_os)" = "darwin" ] && [ -f "$file" ]; then
        xattr -d com.apple.quarantine "$file" 2>/dev/null || true
    fi
}

# Add PATH to shell config
add_to_shell_config() {
    local shell_rc=""
    
    # Detect shell config file
    if [ -n "$ZSH_VERSION" ]; then
        shell_rc="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        shell_rc="$HOME/.bashrc"
    elif [ -f "$HOME/.zshrc" ]; then
        shell_rc="$HOME/.zshrc"
    elif [ -f "$HOME/.bashrc" ]; then
        shell_rc="$HOME/.bashrc"
    elif [ -f "$HOME/.bash_profile" ]; then
        shell_rc="$HOME/.bash_profile"
    fi
    
    # Check if already in PATH
    if [ -f "$shell_rc" ] && grep -q "\.local/bin" "$shell_rc" 2>/dev/null; then
        log_info "PATH already configured in $shell_rc"
        return 0
    fi
    
    # Add to shell config
    if [ -n "$shell_rc" ]; then
        if [ -f "$shell_rc" ]; then
            echo "" >> "$shell_rc"
            echo "# Vaulty" >> "$shell_rc"
        fi
        echo "$PATH_LINE" >> "$shell_rc"
        log_info "Added PATH to $shell_rc"
        log_warn "Run 'source $shell_rc' or restart your terminal"
    else
        log_warn "Could not detect shell config. Add manually:"
        echo "  $PATH_LINE"
    fi
}

# Main installation
install_vaulty() {
    echo ""
    echo "🔐 Vaulty Installer"
    echo "==================="
    echo ""
    
    # Check for curl
    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed."
        exit 1
    fi
    
    # Detect system
    OS=$(detect_os)
    ARCH=$(detect_arch)
    
    if [ "$OS" = "unsupported" ]; then
        log_error "Unsupported operating system."
        exit 1
    fi
    
    log_info "Detected: ${OS}/${ARCH}"
    
    # Get latest version
    log_info "Fetching latest version..."
    VERSION=$(get_latest_version)
    
    if [ -z "$VERSION" ]; then
        log_error "Could not fetch latest version. Is the repository public?"
        exit 1
    fi
    
    log_info "Latest version: ${VERSION}"
    
    # Create install directory
    mkdir -p "$INSTALL_DIR"
    
    # Download binary
    DOWNLOAD_URL=$(get_download_url "$OS" "$ARCH" "$VERSION")
    INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}"
    TEMP_FILE=$(mktemp)
    
    log_info "Downloading from GitHub..."
    
    if ! curl -sSL "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
        log_error "Download failed. Check if the release exists for ${OS}/${ARCH}."
        rm -f "$TEMP_FILE"
        exit 1
    fi
    
    # Verify download
    if [ ! -s "$TEMP_FILE" ]; then
        log_error "Downloaded file is empty."
        rm -f "$TEMP_FILE"
        exit 1
    fi
    
    # Move to install location
    mv "$TEMP_FILE" "$INSTALL_PATH"
    
    # Set permissions
    chmod +x "$INSTALL_PATH"
    
    # Remove macOS quarantine
    remove_quarantine "$INSTALL_PATH"
    
    log_info "Installed to: ${INSTALL_PATH}"
    
    # Add to PATH if needed
    if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
        add_to_shell_config
    fi
    
    # Verify installation
    echo ""
    log_info "Installation complete!"
    
    "$INSTALL_PATH" --version 2>/dev/null || true
    
    echo ""
}

# Check for update flag
if [ "$1" = "--check" ]; then
    CURRENT_VERSION=$("$INSTALL_DIR/$BINARY_NAME" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || echo "not installed")
    
    echo ""
    echo "Current:  ${CURRENT_VERSION}"
    echo "Latest:   $(get_latest_version | sed 's/v//')"
    echo ""
    
    if [ "$CURRENT_VERSION" = "not installed" ]; then
        log_info "Vaulty is not installed. Run the installer to install."
    elif [ "$CURRENT_VERSION" = "$(get_latest_version | sed 's/v//')" ]; then
        log_info "You're up to date!"
    else
        log_warn "A new version is available."
    fi
    exit 0
fi

# Run installation
install_vaulty

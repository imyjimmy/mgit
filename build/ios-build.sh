#!/bin/zsh

# iOS Cross-Compilation Script for mgit
# This script builds mgit binary for iOS devices and simulator

set -e  # Exit on any error

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DIST_DIR="$PROJECT_ROOT/dist"
GO_MOD_DIR="$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Setup macOS toolchain for iOS simulator and development builds
setup_macos_toolchain() {
    # iOS simulator uses macOS runtime, so use standard macOS toolchain
    export CC="$(xcrun --find clang)"
    export CXX="$(xcrun --find clang++)"
    
    # Minimal flags for macOS/simulator builds
    export CGO_CFLAGS="-O2"
    export CGO_LDFLAGS=""
}

# Setup iOS toolchain for CGO
setup_ios_toolchain() {
    log_info "Setting up iOS toolchain for CGO..."
    
    # Find Xcode and iOS SDK
    XCODE_PATH=$(xcode-select -p 2>/dev/null)
    if [ -z "$XCODE_PATH" ]; then
        log_error "Xcode command line tools not found. Please install with: xcode-select --install"
        exit 1
    fi
    
    IOS_SDK_PATH="$XCODE_PATH/Platforms/iPhoneOS.platform/Developer/SDKs"
    IOS_SDK=$(find "$IOS_SDK_PATH" -name "iPhoneOS*.sdk" | sort -V | tail -1)
    
    if [ -z "$IOS_SDK" ]; then
        log_error "iOS SDK not found in $IOS_SDK_PATH"
        exit 1
    fi
    
    log_info "Using iOS SDK: $IOS_SDK"
    
    # Set CGO environment for iOS
    export CC="$(xcrun --find clang)"
    export CXX="$(xcrun --find clang++)"
    export CGO_CFLAGS="-arch arm64 -isysroot $IOS_SDK -mios-version-min=11.0"
    export CGO_CXXFLAGS="-arch arm64 -isysroot $IOS_SDK -mios-version-min=11.0"
    export CGO_LDFLAGS="-arch arm64 -isysroot $IOS_SDK -mios-version-min=11.0"
    
    log_success "iOS toolchain configured"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Using Go version: $GO_VERSION"
    
    # Check if Go version supports iOS
    MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    MINOR=$(echo $GO_VERSION | cut -d. -f2)
    
    if [ "$MAJOR" -lt 1 ] || ([ "$MAJOR" -eq 1 ] && [ "$MINOR" -lt 16 ]); then
        log_warning "Go version $GO_VERSION might not fully support iOS. Recommended: Go 1.16+"
    fi
}

# Create dist directories
setup_dirs() {
    log_info "Setting up build directories..."
    mkdir -p "$DIST_DIR/ios-arm64"
    mkdir -p "$DIST_DIR/ios-simulator"
    mkdir -p "$DIST_DIR/darwin-amd64"
    log_success "Build directories created"
}

# Build for iOS device (ARM64)
build_ios_device() {
    log_info "Building mgit for iOS device (ARM64)..."
    
    cd "$GO_MOD_DIR"
    
    export GOOS=ios
    export GOARCH=arm64
    export CGO_ENABLED=1
    
    # Set iOS SDK paths
    setup_ios_toolchain
    
    # Build with specific flags for iOS
    go build \
        -buildmode=pie \
        -buildvcs=false \
        -trimpath \
        -ldflags="-s -w" \
        -o "$DIST_DIR/ios-arm64/mgit" \
        .
    
    if [ $? -eq 0 ]; then
        log_success "iOS device binary built successfully"
        log_info "Binary location: $DIST_DIR/ios-arm64/mgit"
        
        # Check binary info
        file "$DIST_DIR/ios-arm64/mgit" || true
        ls -lh "$DIST_DIR/ios-arm64/mgit"
    else
        log_error "Failed to build iOS device binary"
        exit 1
    fi
}

# Build for iOS simulator (x86_64 for Intel Macs, arm64 for Apple Silicon)
build_ios_simulator() {
    log_info "Building mgit for iOS simulator..."
    
    cd "$GO_MOD_DIR"
    
    # Determine host architecture for simulator build
    HOST_ARCH=$(uname -m)
    if [ "$HOST_ARCH" = "arm64" ]; then
        SIM_ARCH="arm64"
        log_info "Building simulator binary for Apple Silicon (arm64)"
    else
        SIM_ARCH="amd64"
        log_info "Building simulator binary for Intel (amd64)"
    fi
    
    export GOOS=darwin  # iOS simulator uses darwin
    export GOARCH="$SIM_ARCH"
    export CGO_ENABLED=0
    
    go build \
        -buildvcs=false \
        -trimpath \
        -ldflags="-s -w" \
        -o "$DIST_DIR/ios-simulator/mgit" \
        .
    
    if [ $? -eq 0 ]; then
        log_success "iOS simulator binary built successfully"
        log_info "Binary location: $DIST_DIR/ios-simulator/mgit"
        
        # Check binary info
        file "$DIST_DIR/ios-simulator/mgit" || true
        ls -lh "$DIST_DIR/ios-simulator/mgit"
    else
        log_error "Failed to build iOS simulator binary"
        exit 1
    fi
}

# Build for macOS (for development/testing)
build_macos() {
    log_info "Building mgit for macOS (development/testing)..."
    
    cd "$GO_MOD_DIR"
    
    export GOOS=darwin
    export GOARCH=amd64
    export CGO_ENABLED=1
    
    setup_macos_toolchain
    
    go build \
        -buildvcs=false \
        -trimpath \
        -ldflags="-s -w" \
        -o "$DIST_DIR/darwin-amd64/mgit" \
        .
    
    if [ $? -eq 0 ]; then
        log_success "macOS binary built successfully"
        log_info "Binary location: $DIST_DIR/darwin-amd64/mgit"
        
        # Make executable and test
        chmod +x "$DIST_DIR/darwin-amd64/mgit"
        
        # Quick test if we're on macOS
        if [[ "$OSTYPE" == "darwin"* ]]; then
            log_info "Testing macOS binary..."
            if "$DIST_DIR/darwin-amd64/mgit" --help &>/dev/null || echo "Exit code: $?"; then
                log_success "macOS binary test passed"
            else
                log_warning "macOS binary test showed warnings (might be normal)"
            fi
        fi
        
        ls -lh "$DIST_DIR/darwin-amd64/mgit"
    else
        log_error "Failed to build macOS binary"
        exit 1
    fi
}

# Test iOS binary (basic validation)
test_ios_binary() {
    log_info "Validating iOS binaries..."
    
    # Check iOS device binary
    if [ -f "$DIST_DIR/ios-arm64/mgit" ]; then
        SIZE_IOS=$(stat -f%z "$DIST_DIR/ios-arm64/mgit" 2>/dev/null || stat -c%s "$DIST_DIR/ios-arm64/mgit" 2>/dev/null)
        log_info "iOS device binary size: $SIZE_IOS bytes"
        
        # Check if it's a valid binary (not empty, has expected headers)
        if [ "$SIZE_IOS" -gt 1000000 ]; then  # Should be at least 1MB for a Go binary
            log_success "iOS device binary size looks reasonable"
        else
            log_warning "iOS device binary seems small ($SIZE_IOS bytes)"
        fi
    else
        log_error "iOS device binary not found"
        return 1
    fi
    
    # Check iOS simulator binary  
    if [ -f "$DIST_DIR/ios-simulator/mgit" ]; then
        SIZE_SIM=$(stat -f%z "$DIST_DIR/ios-simulator/mgit" 2>/dev/null || stat -c%s "$DIST_DIR/ios-simulator/mgit" 2>/dev/null)
        log_info "iOS simulator binary size: $SIZE_SIM bytes"
        
        if [ "$SIZE_SIM" -gt 1000000 ]; then
            log_success "iOS simulator binary size looks reasonable"
        else
            log_warning "iOS simulator binary seems small ($SIZE_SIM bytes)"
        fi
    else
        log_error "iOS simulator binary not found"
        return 1
    fi
}

# Print build summary
print_summary() {
    echo
    echo "=============================================="
    log_success "iOS Build Summary"
    echo "=============================================="
    
    echo "Built binaries:"
    
    if [ -f "$DIST_DIR/ios-arm64/mgit" ]; then
        echo "  ✓ iOS Device (ARM64):    $DIST_DIR/ios-arm64/mgit"
    else
        echo "  ✗ iOS Device (ARM64):    FAILED"
    fi
    
    if [ -f "$DIST_DIR/ios-simulator/mgit" ]; then
        echo "  ✓ iOS Simulator:         $DIST_DIR/ios-simulator/mgit"
    else
        echo "  ✗ iOS Simulator:         FAILED"
    fi
    
    if [ -f "$DIST_DIR/darwin-amd64/mgit" ]; then
        echo "  ✓ macOS (testing):       $DIST_DIR/darwin-amd64/mgit"
    else
        echo "  ✗ macOS (testing):       FAILED"
    fi
    
    echo
    echo "Next steps:"
    echo "  1. Test the iOS simulator binary on iOS Simulator"
    echo "  2. Integrate iOS device binary into your React Native iOS build"
    echo "  3. Update your podspec to bundle these binaries"
    echo
}

# Main execution
main() {
    echo "=============================================="
    log_info "mgit iOS Cross-Compilation Build"
    echo "=============================================="
    
    check_go
    setup_dirs
    
    # Build all targets
    build_ios_device
    build_ios_simulator
    build_macos
    
    # Validate
    test_ios_binary
    
    # Summary
    print_summary
}

# Handle command line arguments
case "${1:-all}" in
    "device")
        check_go && setup_dirs && build_ios_device
        ;;
    "simulator")
        check_go && setup_dirs && build_ios_simulator
        ;;
    "macos")
        check_go && setup_dirs && build_macos
        ;;
    "test")
        test_ios_binary
        ;;
    "clean")
        log_info "Cleaning build directory..."
        rm -rf "$DIST_DIR"
        log_success "Build directory cleaned"
        ;;
    "all")
        main
        ;;
    *)
        echo "Usage: $0 [device|simulator|macos|test|clean|all]"
        echo "  device     - Build only iOS device binary (ARM64)"
        echo "  simulator  - Build only iOS simulator binary"
        echo "  macos      - Build only macOS binary (for testing)"
        echo "  test       - Test/validate existing binaries"
        echo "  clean      - Clean build directory"
        echo "  all        - Build all binaries (default)"
        exit 1
        ;;
esac
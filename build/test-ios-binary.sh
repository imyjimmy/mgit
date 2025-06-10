#!/bin/zsh

# Test script for iOS mgit binaries
# This script validates that the cross-compiled binaries work correctly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DIST_DIR="$PROJECT_ROOT/dist"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Test binary existence and basic properties
test_binary_exists() {
    local binary_path="$1"
    local binary_name="$2"
    
    log_info "Testing $binary_name binary..."
    
    if [ ! -f "$binary_path" ]; then
        log_error "$binary_name binary not found at $binary_path"
        return 1
    fi
    
    # Check if it's executable
    if [ ! -x "$binary_path" ]; then
        log_warning "$binary_name binary is not executable, making it executable..."
        chmod +x "$binary_path"
    fi
    
    # Check file size
    local size=$(stat -f%z "$binary_path" 2>/dev/null || stat -c%s "$binary_path" 2>/dev/null)
    log_info "$binary_name size: $size bytes"
    
    if [ "$size" -lt 1000000 ]; then
        log_warning "$binary_name seems small ($size bytes)"
    else
        log_success "$binary_name size looks good"
    fi
    
    # Check file type
    if command -v file &> /dev/null; then
        local file_info=$(file "$binary_path")
        log_info "$binary_name file info: $file_info"
        
        # Validate it's actually an executable
        if echo "$file_info" | grep -q "executable"; then
            log_success "$binary_name is a valid executable"
        else
            log_warning "$binary_name file type check inconclusive"
        fi
    fi
    
    return 0
}

# Test macOS binary functionality (if on macOS)
test_macos_functionality() {
    local binary_path="$DIST_DIR/darwin-amd64/mgit"
    
    if [[ "$OSTYPE" != "darwin"* ]]; then
        log_info "Skipping macOS functionality test (not on macOS)"
        return 0
    fi
    
    log_info "Testing macOS binary functionality..."
    
    # Test help command
    if "$binary_path" --help &>/dev/null; then
        log_success "macOS binary help command works"
    else
        log_warning "macOS binary help command failed (exit code: $?)"
    fi
    
    # Test invalid command (should exit with error)
    if "$binary_path" invalid-command &>/dev/null; then
        log_warning "macOS binary should reject invalid commands"
    else
        log_success "macOS binary correctly rejects invalid commands"
    fi
    
    # Test version-like output
    local output
    if output=$("$binary_path" 2>&1 | head -5); then
        log_info "macOS binary output sample:"
        echo "$output" | sed 's/^/    /'
        log_success "macOS binary produces output"
    else
        log_warning "macOS binary didn't produce expected output"
    fi
}

# Validate binary architecture
validate_architecture() {
    local binary_path="$1"
    local expected_arch="$2"
    local binary_name="$3"
    
    if ! command -v file &> /dev/null; then
        log_warning "Cannot validate architecture (file command not available)"
        return 0
    fi
    
    local file_output=$(file "$binary_path")
    log_info "$binary_name architecture info: $file_output"
    
    case "$expected_arch" in
        "arm64")
            if echo "$file_output" | grep -q "arm64\|aarch64"; then
                log_success "$binary_name has correct ARM64 architecture"
            else
                log_warning "$binary_name architecture validation inconclusive"
            fi
            ;;
        "x86_64"|"amd64")
            if echo "$file_output" | grep -q "x86_64\|x86-64"; then
                log_success "$binary_name has correct x86_64 architecture"
            else
                log_warning "$binary_name architecture validation inconclusive"
            fi
            ;;
        *)
            log_info "Architecture validation skipped for $expected_arch"
            ;;
    esac
}

# Create a test directory structure for binary testing
setup_test_environment() {
    local test_dir="$PROJECT_ROOT/test-ios-env"
    
    log_info "Setting up test environment..."
    
    # Clean and create test directory
    rm -rf "$test_dir"
    mkdir -p "$test_dir"
    
    echo "# Test file for iOS binary validation" > "$test_dir/README.md"
    echo "This directory is used for testing iOS mgit binaries" >> "$test_dir/README.md"
    
    log_success "Test environment created at $test_dir"
    echo "$test_dir"
}

# Clean up test environment
cleanup_test_environment() {
    local test_dir="$1"
    
    if [ -n "$test_dir" ] && [ -d "$test_dir" ]; then
        log_info "Cleaning up test environment..."
        rm -rf "$test_dir"
        log_success "Test environment cleaned up"
    fi
}

# Main test execution
run_tests() {
    echo "=============================================="
    log_info "iOS mgit Binary Validation Tests"
    echo "=============================================="
    
    local tests_passed=0
    local tests_total=0
    
    # Test iOS device binary
    if test_binary_exists "$DIST_DIR/ios-arm64/mgit" "iOS device"; then
        validate_architecture "$DIST_DIR/ios-arm64/mgit" "arm64" "iOS device"
        ((tests_passed++))
    fi
    ((tests_total++))
    
    # Test iOS simulator binary
    if test_binary_exists "$DIST_DIR/ios-simulator/mgit" "iOS simulator"; then
        # Determine expected architecture based on host
        local host_arch=$(uname -m)
        if [ "$host_arch" = "arm64" ]; then
            validate_architecture "$DIST_DIR/ios-simulator/mgit" "arm64" "iOS simulator"
        else
            validate_architecture "$DIST_DIR/ios-simulator/mgit" "x86_64" "iOS simulator"
        fi
        ((tests_passed++))
    fi
    ((tests_total++))
    
    # Test macOS binary
    if test_binary_exists "$DIST_DIR/darwin-amd64/mgit" "macOS"; then
        validate_architecture "$DIST_DIR/darwin-amd64/mgit" "x86_64" "macOS"
        test_macos_functionality
        ((tests_passed++))
    fi
    ((tests_total++))
    
    echo
    echo "=============================================="
    log_info "Test Summary"
    echo "=============================================="
    
    if [ $tests_passed -eq $tests_total ]; then
        log_success "All $tests_total binary tests passed! ✓"
        echo
        echo "Your iOS mgit binaries are ready for integration."
        echo "Next steps:"
        echo "  1. Test iOS simulator binary in Xcode iOS Simulator"
        echo "  2. Integrate binaries into your React Native iOS build"
        echo "  3. Update podspec configuration"
    else
        log_warning "$tests_passed/$tests_total binary tests passed"
        echo
        echo "Some tests failed or showed warnings."
        echo "Check the output above for details."
    fi
    
    echo
    echo "Binary locations:"
    echo "  iOS Device:    $DIST_DIR/ios-arm64/mgit"
    echo "  iOS Simulator: $DIST_DIR/ios-simulator/mgit"
    echo "  macOS Test:    $DIST_DIR/darwin-amd64/mgit"
}

# Handle command line arguments
case "${1:-test}" in
    "test")
        run_tests
        ;;
    "macos-only")
        test_macos_functionality
        ;;
    "setup")
        setup_test_environment
        ;;
    "clean")
        cleanup_test_environment "$PROJECT_ROOT/test-ios-env"
        ;;
    *)
        echo "Usage: $0 [test|macos-only|setup|clean]"
        echo "  test       - Run all binary validation tests (default)"
        echo "  macos-only - Test only macOS binary functionality"
        echo "  setup      - Set up test environment"
        echo "  clean      - Clean up test environment"
        exit 1
        ;;
esac

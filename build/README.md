# mgit iOS Cross-Compilation Build System

This directory contains the build system for cross-compiling the mgit Go binary for iOS devices and simulator.

## Overview

The build system creates three binaries:
- **iOS Device (ARM64)**: For real iOS devices
- **iOS Simulator**: For Xcode iOS Simulator (architecture matches host Mac)
- **macOS (x86_64)**: For development and testing

## Quick Start

### 1. Build All iOS Binaries
```bash
# From the mgit project root
cd build
chmod +x ios-build.sh
./ios-build.sh

# Or using Make
make all
```

### 2. Build Individual Targets
```bash
# iOS device only
./ios-build.sh device
# or
make ios-device

# iOS simulator only  
./ios-build.sh simulator
# or
make ios-simulator

# macOS for testing
./ios-build.sh macos
# or
make macos
```

### 3. Test the Binaries
```bash
./test-ios-binary.sh
# or
make test
```

## File Structure

```
build/
├── README.md              # This file
├── ios-build.sh          # Main build script
├── test-ios-binary.sh    # Binary validation script
├── build-config.env      # Environment configuration
└── Makefile              # Make targets

../dist/                  # Output directory (created during build)
├── ios-arm64/
│   └── mgit              # iOS device binary
├── ios-simulator/
│   └── mgit              # iOS simulator binary
└── darwin-amd64/
    └── mgit              # macOS test binary
```

## Requirements

- **Go 1.16+** (recommended 1.20+)
- **macOS** (for iOS builds)
- **Xcode Command Line Tools** (for file validation)

## Build Configuration

### Environment Variables
The build uses these key environment variables:
- `GOOS=ios` / `GOOS=darwin` - Target operating system
- `GOARCH=arm64` / `GOARCH=amd64` - Target architecture  
- `CGO_ENABLED=0` - Disable CGO for static binaries

### Build Flags
- `-buildmode=pie` - Position Independent Executable (iOS requirement)
- `-buildvcs=false` - Disable VCS info embedding
- `-trimpath` - Remove local path information
- `-ldflags="-s -w"` - Strip debug symbols and symbol tables

## iOS-Specific Considerations

### iOS Device Binary
- **Architecture**: ARM64 only
- **OS Target**: `GOOS=ios`
- **Requirements**: Must be position-independent executable (`-buildmode=pie`)
- **Limitations**: Cannot be executed directly on macOS for testing

### iOS Simulator Binary
- **Architecture**: Matches host Mac (ARM64 for Apple Silicon, x86_64 for Intel)
- **OS Target**: `GOOS=darwin` (simulator uses macOS runtime)
- **Testing**: Can be tested on macOS since it's a Darwin binary

## Validation and Testing

### Automated Tests
```bash
# Run all validation tests
./test-ios-binary.sh

# Test only macOS binary functionality
./test-ios-binary.sh macos-only
```

### Manual Testing
```bash
# Test macOS binary (should work on macOS)
../dist/darwin-amd64/mgit --help

# Check binary architecture
file ../dist/ios-arm64/mgit
file ../dist/ios-simulator/mgit
```

### Expected Output
- **iOS Device**: `Mach-O 64-bit executable arm64`
- **iOS Simulator**: `Mach-

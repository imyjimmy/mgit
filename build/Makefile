# Makefile for mgit iOS Cross-Compilation
# Provides convenient targets for building iOS binaries

SHELL := /bin/bash
PROJECT_ROOT := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BUILD_SCRIPT := $(PROJECT_ROOT)/ios-build.sh

# Colors
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m

.PHONY: all ios-device ios-simulator macos test clean help

# Default target
all: ios-device ios-simulator macos
	@echo -e "$(GREEN)✓ All iOS builds completed$(NC)"

# Build iOS device binary (ARM64)
ios-device:
	@echo -e "$(BLUE)Building iOS device binary...$(NC)"
	@$(BUILD_SCRIPT) device

# Build iOS simulator binary
ios-simulator:
	@echo -e "$(BLUE)Building iOS simulator binary...$(NC)"
	@$(BUILD_SCRIPT) simulator

# Build macOS binary (for testing)
macos:
	@echo -e "$(BLUE)Building macOS binary...$(NC)"
	@$(BUILD_SCRIPT) macos

# Test existing binaries
test:
	@echo -e "$(BLUE)Testing iOS binaries...$(NC)"
	@$(BUILD_SCRIPT) test

# Clean build artifacts
clean:
	@echo -e "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(BUILD_SCRIPT) clean

# Make build script executable (in case it gets reset)
setup:
	@echo -e "$(BLUE)Setting up build environment...$(NC)"
	@chmod +x $(BUILD_SCRIPT)
	@echo -e "$(GREEN)✓ Build script is now executable$(NC)"

# Quick build for development (just macOS for testing)
dev:
	@echo -e "$(BLUE)Quick development build (macOS only)...$(NC)"
	@$(BUILD_SCRIPT) macos

# Help target
help:
	@echo "mgit iOS Cross-Compilation Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Build all iOS binaries (device + simulator + macOS)"
	@echo "  ios-device   - Build iOS device binary (ARM64)"
	@echo "  ios-simulator- Build iOS simulator binary"
	@echo "  macos        - Build macOS binary (for testing)"
	@echo "  dev          - Quick macOS build for development"
	@echo "  test         - Test/validate existing binaries"
	@echo "  clean        - Clean build artifacts"
	@echo "  setup        - Make build script executable"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make              # Build all binaries"
	@echo "  make ios-device   # Build only iOS device binary"
	@echo "  make clean all    # Clean and rebuild everything"
	@echo "  make dev          # Quick macOS build for testing"

# Alias targets for convenience
build: all
ios: ios-device ios-simulator

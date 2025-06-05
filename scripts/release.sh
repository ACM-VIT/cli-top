#!/bin/bash

# CLI-TOP Release Helper Script
# This script helps manage versions and create releases locally

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

validate_version() {
    local version=$1
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        print_error "Invalid version format: $version"
        echo "Expected format: X.Y.Z or X.Y.Z-suffix (e.g., 2.9.1 or 2.9.1-beta)"
        exit 1
    fi
}

get_current_version() {
    grep 'var Version string =' debug/debug.go | sed 's/.*= "\(.*\)".*/\1/'
}

update_version() {
    local new_version=$1
    sed -i.bak "s/var Version string = \".*\"/var Version string = \"$new_version\"/" debug/debug.go
    rm debug/debug.go.bak 2>/dev/null || true
}

tag_exists() {
    local tag=$1
    git rev-parse "$tag" >/dev/null 2>&1
}

show_help() {
    echo "CLI-TOP Release Helper"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  version                    Show current version"
    echo "  update <version>          Update version in debug.go"
    echo "  release <version>         Update version and create tag"
    echo "  build                     Build binaries locally"
    echo "  build-installer           Build Windows installer (WiX)"
    echo "  test-goreleaser           Test GoReleaser configuration"
    echo "  help                      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 version                # Show current version"
    echo "  $0 update 2.9.1          # Update to version 2.9.1"
    echo "  $0 release 2.9.1         # Update to 2.9.1 and create tag"
    echo "  $0 build                  # Build for current platform"
    echo "  $0 build-installer        # Build Windows MSI installer"
    echo "  $0 test-goreleaser        # Test GoReleaser config without releasing"
}

case "${1:-help}" in
    "version")
        current_version=$(get_current_version)
        print_info "Current version: $current_version"
        ;;
    
    "update")
        if [ -z "$2" ]; then
            print_error "Version required"
            echo "Usage: $0 update <version>"
            exit 1
        fi
        
        new_version=$2
        validate_version "$new_version"
        
        current_version=$(get_current_version)
        print_info "Current version: $current_version"
        print_info "New version: $new_version"
        
        update_version "$new_version"
        print_status "Updated debug.go to version $new_version"
        ;;
    
    "release")
        if [ -z "$2" ]; then
            print_error "Version required"
            echo "Usage: $0 release <version>"
            exit 1
        fi
        
        new_version=$2
        validate_version "$new_version"
        
        tag_name="v$new_version"
        
        if tag_exists "$tag_name"; then
            print_error "Tag $tag_name already exists"
            exit 1
        fi
        
        current_version=$(get_current_version)
        print_info "Current version: $current_version"
        print_info "New version: $new_version"
        
        update_version "$new_version"
        print_status "Updated debug.go to version $new_version"
        
        git add debug/debug.go
        git commit -m "chore: bump version to $new_version"
        print_status "Committed version update"
        
        git tag -a "$tag_name" -m "Release version $new_version"
        print_status "Created tag $tag_name"
        
        print_warning "Ready to push! Run the following commands:"
        echo "  git push"
        echo "  git push origin $tag_name"
        print_info "This will trigger the automated release workflow"
        ;;
    
    "build")
        print_info "Building CLI-TOP for current platform..."
        
        cd "$(dirname "$0")/.."
        
        go build -trimpath -ldflags "-s -w -X cli-top/debug.Version=$(get_current_version)" -o cli-top .
        print_status "Built cli-top binary"
        ;;
    
    "test-goreleaser")
        print_info "Testing GoReleaser configuration..."
        if ! command -v goreleaser &> /dev/null; then
            print_error "GoReleaser not installed"
            print_info "Install with: go install github.com/goreleaser/goreleaser@latest"
            exit 1
        fi
        
        cd "$(dirname "$0")/.."
        
        goreleaser build --snapshot --clean
        print_status "GoReleaser test completed successfully"
        print_info "Check the 'dist' directory for built binaries"
        ;;
    
    "help"|*)
        show_help
        ;;
esac

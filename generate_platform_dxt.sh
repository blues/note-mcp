#!/bin/bash

# Script to generate platform-specific .dxt files from goreleaser binaries
set -e

# Define the MCP servers (compatible with bash 3.2)
servers="notecard notehub dev"

# Function to update manifest.json with platform
update_manifest() {
    local server_dir=$1
    local platform=$2
    local manifest_file="${server_dir}/manifest.json"

    # Create a temporary file with the updated manifest
    cat "$manifest_file" | jq --arg platform "$platform" '.compatibility.platforms = [$platform]' > "${manifest_file}.tmp"
    mv "${manifest_file}.tmp" "$manifest_file"
}

# Function to restore original manifest.json
restore_manifest() {
    local server_dir=$1
    local manifest_file="${server_dir}/manifest.json"

    # Reset platforms array to empty
    cat "$manifest_file" | jq '.compatibility.platforms = []' > "${manifest_file}.tmp"
    mv "${manifest_file}.tmp" "$manifest_file"
}

# Function to process a single platform/server combination
process_platform() {
    local server=$1
    local platform=$2
    local architecture=$3
    local binary_name="${server}_${platform}-${architecture}"

    echo "Processing ${server} for platform ${platform}-${architecture}..."

    # Step 1: Find and copy binary from goreleaser dist directory
    local source_binary=""
    local dest_binary="${server}/${server}"

    # Search for the binary in dist directories
    for dir in dist/${server}*; do
        if [ -d "$dir" ]; then
            # Check for binary with .exe extension (Windows)
            if [ -f "$dir/${binary_name}.exe" ]; then
                source_binary="$dir/${binary_name}.exe"
                break
            # Check for binary without extension
            elif [ -f "$dir/${binary_name}" ]; then
                source_binary="$dir/${binary_name}"
                break
            fi
        fi
    done

    if [ -z "$source_binary" ]; then
        echo "Warning: Binary not found for ${binary_name}"
        return 1
    fi

    # Backup original binary if it exists
    if [ -f "$dest_binary" ]; then
        cp "$dest_binary" "${dest_binary}.backup"
    fi

    # Copy the platform-specific binary
    cp "$source_binary" "$dest_binary"

    # Step 2: Update manifest.json with platform
    update_manifest "$server" "$platform"

    # Step 3: Run make dxt for this server
    echo "Running make dxt-${server}..."
    make "dxt-${server}"

    # Step 4: Find the generated .dxt file and rename it
    local generated_dxt="${server}/${server}.dxt"
    if [ -f "$generated_dxt" ]; then
        local platform_dxt="dxt/${server}_${platform}-${architecture}.dxt"
        mv "$generated_dxt" "$platform_dxt"
        echo "Generated: $platform_dxt"

        # Step 4.1: Copy the corresponding binary to the dxt directory
        local binary_dest="dxt/${binary_name}"
        if [ -f "$source_binary" ]; then
            cp "$source_binary" "$binary_dest"
            echo "Copied binary: $binary_dest"
        fi
    else
        echo "Warning: Expected .dxt file not found: $generated_dxt"
    fi

    # Step 5: Restore original manifest.json
    restore_manifest "$server"

    # Step 6: Restore original binary if it existed
    if [ -f "${dest_binary}.backup" ]; then
        mv "${dest_binary}.backup" "$dest_binary"
    else
        rm -f "$dest_binary"
    fi

    echo "Completed ${server} for platform ${platform}-${architecture}"
}

# Main execution
echo "Starting platform-specific .dxt generation..."

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed."
    exit 1
fi

# Check if make is available
if ! command -v make &> /dev/null; then
    echo "Error: make is required but not installed."
    exit 1
fi

# Define the platforms and architectures we expect
platforms_archs="darwin-amd64 darwin-arm64 linux-amd64 linux-arm linux-arm64 win32-amd64"

# Process each server and platform combination
for server in $servers; do
    echo "Processing server: $server"

    for platform_arch in $platforms_archs; do
        # Split platform and architecture
        platform=$(echo "$platform_arch" | cut -d'-' -f1)
        architecture=$(echo "$platform_arch" | cut -d'-' -f2)

        process_platform "$server" "$platform" "$architecture"
    done
done

echo "Platform-specific .dxt generation completed!"
echo "Generated .dxt files are in the dxt/ directory:"
ls -la dxt/

#!/bin/bash

# This script builds the Go binary and copies it along with the configs to a custom directory within the GOPATH.

# Set the custom binary directory within GOPATH
custom_bin_dir="$GOPATH/bin/commerce-tools"

# Create the custom binary directory if it doesn't exist
mkdir -p "$custom_bin_dir"

# Copy the configs and build the Go binary, placing it in the custom binary directory
cp -r configs "$custom_bin_dir"
go build -o "$custom_bin_dir"

echo "Binary and configs have been installed to $custom_bin_dir"
echo "executing the binary with the command: $custom_bin_dir/utils"

# Execute the binary with the command
"$custom_bin_dir/utils"

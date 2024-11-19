#!/bin/sh

# Determine the OS and architecture
OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

# Function to download appropriate binary
download_binary() {
    local download_url=""
    case $OS in
        darwin|linux)
            if [ "$ARCH" = "amd64" ]; then
                download_url="https://github.com/lemmego/cli/releases/download/v0.1.6/lemmego-v0.1.6-$OS-amd64"
            elif [ "$ARCH" = "arm64" ]; then
                download_url="https://github.com/lemmego/cli/releases/download/v0.1.6/lemmego-v0.1.6-$OS-arm64"
            fi
            ;;
        *)
            echo "Unsupported OS: $OS"
            exit 1
            ;;
    esac

    if [ -z "$download_url" ]; then
        echo "Failed to determine the download URL for this platform."
        exit 1
    fi

    echo "Downloading: $download_url"
    curl -L "$download_url" -o "lemmego-v0.1.6-$OS-$ARCH"
    if [ $? -ne 0 ]; then
        echo "Download failed. Please check if you have enough disk space or permissions."
        exit 1
    fi

    echo "Moving file to /usr/local/bin"
    sudo mv "lemmego-v0.1.6-$OS-$ARCH" /usr/local/bin/lemmego
    if [ $? -ne 0 ]; then
        echo "Failed to move file. You might need to run this script with sudo or check permissions."
        exit 1
    fi

    sudo chmod +x /usr/local/bin/lemmego
    echo "Installation completed."
}

# Run the installation
download_binary

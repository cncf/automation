#!/bin/bash
set -e
set -x

# Install actions runner
mkdir -p /opt/actions-runner
cd /opt/actions-runner
case $(uname -m) in
  x86_64)
    echo "Downloading x86_64 actions-runner"
    curl -o actions-runner.tar.gz -L https://github.com/actions/runner/releases/download/v2.323.0/actions-runner-linux-x64-2.323.0.tar.gz
    ;;
  arm64|aarch64)
    echo "Downloading arm64 actions-runner"
    curl -o actions-runner.tar.gz  -L https://github.com/actions/runner/releases/download/v2.323.0/actions-runner-linux-arm64-2.323.0.tar.gz
    ;;
  *)
    echo "Unsupported architecture: $(uname -m)"
    exit 1
    ;;
esac
tar xzf ./actions-runner.tar.gz
rm ./actions-runner.tar.gz

# Prep for disk imaging
sync
sync
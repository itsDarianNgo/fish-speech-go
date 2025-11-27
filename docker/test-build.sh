#!/bin/bash
# Test script to validate Docker inference build

set -e

echo "=== Testing Fish-Speech Inference Docker Build ==="

# Check if Dockerfile exists
if [ ! -f "Dockerfile.inference" ]; then
    echo "ERROR: Dockerfile.inference not found"
    exit 1
fi

echo "✓ Dockerfile.inference exists"

# Validate Dockerfile has required components
CHECKS=(
    "FROM nvidia/cuda"
    "torch==2.4.0"
    "torchaudio==2.4.0"
    "pip install -e"
    "huggingface_hub"
    "ln -sfn /app/checkpoints"
    "ENTRYPOINT"
)

for check in "${CHECKS[@]}"; do
    if grep -q "$check" Dockerfile.inference; then
        echo "✓ Found: $check"
    else
        echo "✗ Missing: $check"
        exit 1
    fi
done

echo ""
echo "=== All Dockerfile checks passed ==="
echo ""
echo "To build and test manually, run:"
echo "  cd docker"
echo "  docker compose build --no-cache inference"
echo "  docker compose up -d"
echo "  docker compose logs -f inference"

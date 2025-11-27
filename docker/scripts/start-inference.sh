#!/bin/bash
set -e

# Check if models exist, if not download them
CHECKPOINT_PATH="${CHECKPOINT_PATH:-/app/checkpoints/openaudio-s1-mini}"

if [ ! -d "$CHECKPOINT_PATH" ]; then
    echo "Downloading Fish-Speech models..."

    # Use huggingface-cli to download models
    pip install huggingface_hub

    python3 -c "
from huggingface_hub import snapshot_download
snapshot_download(
    repo_id='fishaudio/openaudio-s1-mini',
    local_dir='$CHECKPOINT_PATH',
    local_dir_use_symlinks=False
)
"
    echo "Models downloaded successfully"
fi

# Start the inference server
echo "Starting Fish-Speech inference server..."
exec python3 -m tools.api_server \
    --listen "${FISH_SPEECH_LISTEN:-0.0.0.0:8081}" \
    --llama-checkpoint-path "$CHECKPOINT_PATH" \
    --decoder-checkpoint-path "$CHECKPOINT_PATH/codec.pth" \
    --decoder-config-name modded_dac_vq \
    --device "${FISH_SPEECH_DEVICE:-cuda}" \
    ${FISH_SPEECH_COMPILE:+--compile} \
    "$@"

#!/bin/bash
set -e

find stacks -name "*.env.enc" | while read enc_file; do
    env_file="${enc_file%.enc}"
    echo "Decrypting $enc_file → $env_file"
    sops decrypt --input-type dotenv --output-type dotenv "$enc_file" > "$env_file"
done

echo "✓ All secrets decrypted"
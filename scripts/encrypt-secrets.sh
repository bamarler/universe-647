#!/bin/bash
set -e

find stacks -name ".env" -not -name "*.example" -not -name "*.enc" | while read env_file; do
    enc_file="${env_file}.enc"
    echo "Encrypting $env_file → $enc_file"
    sops encrypt --input-type dotenv --output-type dotenv "$env_file" > "$enc_file"
done

echo "✓ All secrets encrypted"
echo "Remember: git add *.env.enc && git commit"
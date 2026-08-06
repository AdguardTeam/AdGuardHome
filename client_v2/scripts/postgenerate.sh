#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
npx prettier --write "src/api/generated.ts" "src/api/model/**/*.ts"
npx eslint --quiet --fix "src/api/generated.ts" "src/api/model/**/*.ts" || true

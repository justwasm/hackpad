#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."

GO="${GO:-$(pwd)/cache/go/bin/go}"
WASM_EXEC_SRC="${WASM_EXEC_SRC:-$(pwd)/server/public/wasm/wasm_exec.js}"
OUT="$(pwd)/examples/node"

GO_DIR="$(dirname "$GO")"

echo "Building cmd/init → main.wasm ..."
PATH="$GO_DIR:$GO_DIR/../misc/wasm:$PATH" \
  GOOS=js GOARCH=wasm \
  "$GO" build -o "$OUT/main.wasm" ./cmd/init/

echo "Building examples/node/info → info.wasm ..."
PATH="$GO_DIR:$GO_DIR/../misc/wasm:$PATH" \
  GOOS=js GOARCH=wasm \
  "$GO" build -o "$OUT/info.wasm" ./examples/node/info/

echo "Copying wasm_exec.js ..."
cp "$WASM_EXEC_SRC" "$OUT/wasm_exec.js"

echo "Done. Run: node examples/node/run.mjs  or  deno run -A examples/node/run.ts"

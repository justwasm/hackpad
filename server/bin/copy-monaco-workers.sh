#!/bin/bash
# Copy Monaco Editor worker files from node_modules to public/
# These are self-contained minified workers, loadable as classic workers.
set -euo pipefail

MONACO_DIR=node_modules/monaco-editor/min/vs/assets
PUBLIC_DIR=public/vs/assets

if [ ! -d "$MONACO_DIR" ]; then
  echo "Monaco assets not found at $MONACO_DIR. Run npm install first."
  exit 1
fi

mkdir -p "$PUBLIC_DIR"

cp "$MONACO_DIR"/editor.worker-*.js  "$PUBLIC_DIR/editor.worker.js"
cp "$MONACO_DIR"/json.worker-*.js    "$PUBLIC_DIR/json.worker.js"
cp "$MONACO_DIR"/css.worker-*.js     "$PUBLIC_DIR/css.worker.js"
cp "$MONACO_DIR"/html.worker-*.js    "$PUBLIC_DIR/html.worker.js"
cp "$MONACO_DIR"/ts.worker-*.js      "$PUBLIC_DIR/ts.worker.js"

echo "Copied Monaco workers to $PUBLIC_DIR"

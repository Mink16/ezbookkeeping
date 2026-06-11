#!/bin/bash
# PostToolUse hook (async): 編集された .ts/.vue/.js を eslint --fix で整形する。
# 単一ファイルでも数秒かかるため async 実行 (編集をブロックしない)。
f=$(jq -r '.tool_response.filePath // .tool_input.file_path // ""')
case "$f" in
  */src/*.ts|*/src/*.vue|*/src/*.js)
    [ -f "$f" ] && cd "$CLAUDE_PROJECT_DIR" && npx eslint --fix "$f" 2>/dev/null
    ;;
esac
exit 0

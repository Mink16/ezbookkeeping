#!/bin/bash
# PostToolUse hook: 編集された .go ファイルを gofmt で整形する (本家のコードスタイル維持)
f=$(jq -r '.tool_response.filePath // .tool_input.file_path // ""')
case "$f" in
  *.go) [ -f "$f" ] && gofmt -w "$f" 2>/dev/null ;;
esac
exit 0

#!/bin/bash
# PreToolUse hook: main は upstream (mayswind) のミラーのため、
# main 上でコミットを作成する git コマンドを deny する。
cmd=$(jq -r '.tool_input.command // ""')

# コミットを作成する git サブコマンドのみ対象 (パイプや ; をまたぐ誤検知は除外)
printf '%s' "$cmd" | grep -qE '\bgit\b[^|;&]*\b(commit|merge|cherry-pick|revert|rebase)\b' || exit 0

branch=$(git branch --show-current 2>/dev/null)
[ "$branch" = "main" ] || exit 0

cat <<'EOF'
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"main は upstream (mayswind) のミラーのため直接コミット禁止です。custom/main または feature/* ブランチに切り替えて作業してください (.claude/rules/feature-development.md 参照)。"}}
EOF

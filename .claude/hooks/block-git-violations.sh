#!/usr/bin/env bash
# PreToolUse hook: fork 運用の git 規約 (.claude/rules/feature-development.md) を機械的に強制する。
#   1. upstream (本家 mayswind) への push 禁止 / push URL (DISABLED 固定) の復元禁止
#   2. origin の main への push 禁止 (main は本家の完全ミラー)。--mirror / --all も禁止
#   3. main 上でのコミット作成系コマンド・push 禁止
#   4. 同一コマンド内で main へ切り替えてコミットする複合コマンド禁止
# 注: 判定前にクォート文字列を除去し、コミットメッセージ内の文字列での誤検知 (over-deny) を防ぐ。
set -u
cmd=$(jq -r '.tool_input.command // ""')

deny() { jq -n --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'; exit 0; }
ask()  { jq -n --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"ask",permissionDecisionReason:$r}}'; exit 0; }

# git を含まなければ即終了
printf '%s' "$cmd" | grep -qE '\bgit\b' || exit 0

# PCRE (grep -P) 不在環境では精密判定が fail-open するため、安全側で git コマンドを ask に倒す
if ! printf 'x' | grep -qP 'x' 2>/dev/null; then
  ask "PCRE 非対応の grep 環境のため git 規約の精密判定ができません。安全のため確認をお願いします。"
fi

# クォート ("..." / '...') を除去してから判定 (コミットメッセージ内の文字列が規約パターンに誤一致するのを防ぐ)。
# 複数行メッセージにも対応するため、先に改行をスペースへ畳んでからクォートを除去する。
scrub=$(printf '%s' "$cmd" | tr '\n' ' ' | sed -E 's/"[^"]*"//g; s/'\''[^'\'']*'\''//g')

# 1. upstream への push (リモート名 upstream / URL mayswind)。set-url の --push は別ルールで扱う
if printf '%s' "$scrub" | grep -qP '\bgit\b[^|;&]*[[:space:]]push\b[^|;&]*(\bupstream\b|mayswind)'; then
  deny "upstream (mayswind/ezbookkeeping = 本家) への push は禁止です。push 先は origin (Mink16 fork) のみです。"
fi

# 1'. upstream の push URL (DISABLED 固定) の復元禁止
if printf '%s' "$scrub" | grep -qE '\bgit\b[^|;&]*remote[^|;&]*set-url[^|;&]*\bupstream\b' \
   && ! printf '%s' "$scrub" | grep -q 'DISABLED'; then
  deny "upstream の push URL は事故防止のため DISABLED に固定しています。変更しないでください。"
fi

# 2. origin の main への push (main / +main / HEAD:main / :main / refs/heads/main。custom/main は除外)
if printf '%s' "$scrub" | grep -qP '\bgit\b[^|;&]*[[:space:]]push\b[^|;&]*\borigin\b[^|;&]*(^|[[:space:]:+])(refs/heads/)?\+?main\b(?![/-])'; then
  deny "origin の main は本家の完全ミラーのため push 禁止です。push は custom/main または feature/* ブランチのみです。"
fi

# 2'. 全 ref を push する --mirror / --all (main を含みうる) → deny
if printf '%s' "$scrub" | grep -qP '\bgit\b[^|;&]*[[:space:]]push\b[^|;&]*(--mirror|--all)\b'; then
  deny "git push --mirror / --all は main を含む全 ref を push しうるため禁止です。ブランチを明示してください。"
fi

# 3. main へ切り替え + コミット作成系を同一コマンドで (hook はコマンド開始時のブランチしか見られない)
if printf '%s' "$scrub" | grep -qP '\bgit\b[^|;&]*\b(checkout|switch)\b[^|;&]*(^|[[:space:]])main\b(?![/-])' \
   && printf '%s' "$scrub" | grep -qE '\bgit\b[^|;&]*\b(commit|merge|cherry-pick|revert|rebase)\b'; then
  deny "main への切り替えとコミット系操作を同一コマンドで実行しないでください。コミット系は custom/main / feature/* 上で別コマンドとして実行してください。"
fi

# 4. 現在ブランチが main の場合のコミット作成系・push
branch=$(git branch --show-current 2>/dev/null)
if [ "$branch" = "main" ]; then
  if printf '%s' "$scrub" | grep -qE '\bgit\b[^|;&]*\b(commit|merge|cherry-pick|revert|rebase)\b'; then
    deny "main は upstream (mayswind) のミラーのため直接コミット禁止です。custom/main または feature/* ブランチに切り替えて作業してください (.claude/rules/feature-development.md 参照)。"
  fi
  if printf '%s' "$scrub" | grep -qE '\bgit\b[^|;&]*[[:space:]]push\b([^|;&]*\bHEAD\b|[[:space:]]*($|[|;&]))'; then
    deny "main 上での git push は main ミラーへの push になるため禁止です。"
  fi
fi

exit 0

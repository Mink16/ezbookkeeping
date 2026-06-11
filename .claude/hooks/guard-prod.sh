#!/usr/bin/env bash
# PreToolUse hook: 本番 (8081 = 家族の実データ) と本番データ (docker-data/, .env) を保護する。
#
# 【脅威モデルと限界】コマンド文字列を正規表現で検閲する方式のため、任意インタプリタ・変数展開・
# cd 後の相対パス等を使った "意図的な" 回避までは防げない(原理的限界)。この hook は「Claude が
# 良かれと思って打つ操作による事故」への一次防御。明白な破壊は deny、少しでも曖昧なものは ask に
# 倒して人間の確認を挟む。本番データの最終防壁は OS レベル(.env=600、docker-data の定期バックアップ)。
set -u
input=$(cat)
tool=$(printf '%s' "$input" | jq -r '.tool_name // ""')

deny() { jq -n --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'; exit 0; }
ask()  { jq -n --arg r "$1" '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"ask",permissionDecisionReason:$r}}'; exit 0; }

PROD_URL='(localhost|127\.0\.0\.1|\[::1\]|0\.0\.0\.0):8081'

# --- Playwright: 本番 URL があらゆる文字列フィールド (url/code/function/...) に出たら deny ---
#     navigate だけでなく run_code_unsafe / evaluate (任意 JS で本番に fetch 可能) も塞ぐ。
case "$tool" in
  mcp__playwright__*)
    if printf '%s' "$input" | jq -r '[.tool_input // {} | .. | strings] | .[]' 2>/dev/null | grep -qE "$PROD_URL"; then
      deny "本番 (8081) は家族の実データのためブラウザ操作禁止です。検証は dev (http://localhost:8082) か vite (http://localhost:8083) で行ってください。"
    fi
    exit 0 ;;
  Bash) ;;
  *) exit 0 ;;
esac

cmd=$(printf '%s' "$input" | jq -r '.tool_input.command // ""')
[ -n "$cmd" ] || exit 0

# PCRE (grep -P) 不在環境では -dev 区別等の精密判定が fail-open するため、安全側で全 Bash を ask に倒す
if ! printf 'x' | grep -qP 'x' 2>/dev/null; then
  ask "PCRE 非対応の grep 環境のため本番保護の精密判定ができません。安全のため確認をお願いします。"
fi

# ND = docker-data だが docker-data-dev でない (本番データのパス)
ND='docker-data(?!-dev)(?![A-Za-z])'

# --- 本番データ docker-data/ への明白な破壊操作 → deny ---
if printf '%s' "$cmd" | grep -qP "\b(rm|rmdir|shred|mv|dd|truncate)\b[^|;&]*${ND}" \
   || printf '%s' "$cmd" | grep -qP "\bfind\b[^|;&]*${ND}[^|;&]*-(delete|exec)\b" \
   || printf '%s' "$cmd" | grep -qP "\brsync\b[^|;&]*--delete[^|;&]*${ND}" \
   || printf '%s' "$cmd" | grep -qP ">{1,2}[[:space:]]*[^|;&>]*${ND}"; then
  deny "docker-data/ は本番 (8081) の実データです。削除・上書きは禁止です。dev のリセットは /dev-reset スキル (対象は docker-data-dev/ のみ) に従ってください。"
fi

# --- docker-data/ への曖昧な書き込み・コピー・cd・インタプリタ → ask (読み取りの ls/cat/du は通す) ---
#     インタプリタ (python 等) は -c のクォート内に docker-data を置けるので、セグメント非依存の AND で判定。
if printf '%s' "$cmd" | grep -qP "\b(cp|install|tee|ln|chmod|chown|chgrp|chattr)\b[^|;&]*${ND}" \
   || printf '%s' "$cmd" | grep -qP "\bcd\b[^|;&]*${ND}" \
   || { printf '%s' "$cmd" | grep -qP '\b(python3?|perl|ruby|node|php)\b' \
        && printf '%s' "$cmd" | grep -qP "${ND}"; }; then
  ask "本番データディレクトリ docker-data/ を操作しようとしています。本番 (8081) の実データに影響しうるため確認します (dev は docker-data-dev/)。"
fi

# --- git clean -x/-X は gitignore 済みの docker-data/ と .env を巻き込む → deny ---
if printf '%s' "$cmd" | grep -qE '\bgit\b[^|;&]*\bclean\b[^|;&]*-[a-zA-Z]*[xX]'; then
  deny "git clean の -x/-X は gitignore 済みの docker-data/ (本番実データ) や .env (秘密鍵) を削除します。対象を明示した操作にしてください。"
fi

# --- .env (秘密鍵) の閲覧・複製・読み込み → deny (.env.example / .env.sample のみ除外) ---
ENV_TOKEN='\.env(?!\.example\b|\.sample\b)(?![A-Za-z])'
ENV_READERS='\b(cat|bat|tac|nl|less|more|head|tail|strings|grep|egrep|fgrep|rg|ag|awk|gawk|mawk|sed|od|xxd|hexdump|base64|sort|uniq|cut|tr|rev|cp|install|tee|dd|vi|vim|view|nano|emacs|wc|md5sum|sha1sum|sha256sum|cksum)\b'
if printf '%s' "$cmd" | grep -qP "${ENV_READERS}[^|;&]*[ /]${ENV_TOKEN}" \
   || printf '%s' "$cmd" | grep -qP "<[[:space:]]*[^|;&]*[ /]?${ENV_TOKEN}" \
   || printf '%s' "$cmd" | grep -qP "\bsource\b[^|;&]*${ENV_TOKEN}" \
   || printf '%s' "$cmd" | grep -qP "(^|[|;&][[:space:]]*|&&[[:space:]]*)\.[[:space:]]+[^|;&]*${ENV_TOKEN}"; then
  deny ".env には EBK_SECURITY_SECRET_KEY 等の秘密情報が含まれるため閲覧・複製・読み込みを禁止します。設定キー名は docker-compose.yml か conf/ezbookkeeping.ini を参照してください。"
fi

# --- 本番 (8081) への書き込み系 HTTP リクエスト (curl/wget) → deny (GET 読み取りは通す) ---
if printf '%s' "$cmd" | grep -qE "$PROD_URL"; then
  if printf '%s' "$cmd" | grep -qP '\bcurl\b' \
     && printf '%s' "$cmd" | grep -qP '(-X[[:space:]]*(POST|PUT|DELETE|PATCH)|--request[[:space:]=]*(POST|PUT|DELETE|PATCH)|--data\b|--data-[a-z]+\b|[[:space:]]-d[[:space:]@=]|--json\b|--form\b|[[:space:]]-F[[:space:]]|--upload-file\b|[[:space:]]-T[[:space:]]|--method[[:space:]=]*(?!GET\b)[A-Z])'; then
    deny "本番 (8081) への書き込み HTTP リクエストは禁止です。本番への記帳はユーザー明示依頼時に ezbookkeeping MCP の add_transaction (ask) を使ってください。"
  fi
  if printf '%s' "$cmd" | grep -qP '\bwget\b' \
     && printf '%s' "$cmd" | grep -qP '(--post-data|--post-file|--body-data|--body-file|--method[[:space:]=]*(?!GET\b)[A-Z])'; then
    deny "本番 (8081) への wget 書き込みリクエストは禁止です。"
  fi
fi

# --- docker compose / docker-compose, docker exec/run: 本番を巻き込む or サービス未指定なら ask ---
#     複合コマンドはセグメント単位 (&&, ;, |) で評価する。dev (ezbookkeeping-dev) 限定なら許可。
while IFS= read -r seg; do
  # 本番コンテナへの docker exec/run (CLI は user-delete 等の破壊操作を含む)
  if printf '%s' "$seg" | grep -qP '\bdocker\b[^|;&]*\b(exec|run)\b[^|;&]*(?<![/\w.-])ezbookkeeping(?![-\w])'; then
    ask "本番コンテナ (ezbookkeeping = 8081) への docker exec/run です。実データに影響しうるため確認します。dev は ezbookkeeping-dev を使ってください。"
  fi
  # docker compose (スペース表記) / docker-compose (ハイフン表記) 両対応
  printf '%s' "$seg" | grep -qP '\bdocker(-compose\b|[[:space:]]+([^|;&]*[[:space:]])?compose\b)' || continue
  printf '%s' "$seg" | grep -qP '\b(up|down|build|restart|stop|start|rm|kill|create|pull|run|exec|cp)\b' || continue
  if printf '%s' "$seg" | grep -qP '(?<![/\w.-])ezbookkeeping(?![-\w])'; then
    ask "このコマンドは本番サービス (ezbookkeeping = 8081, 家族の実データ) に影響します。本番反映はユーザー承認が必要です。"
  fi
  if ! printf '%s' "$seg" | grep -q 'ezbookkeeping-dev'; then
    ask "サービス未指定の docker compose 操作は本番 (8081) も対象になります。dev だけなら ezbookkeeping-dev を明示してください。"
  fi
done < <(printf '%s\n' "$cmd" | tr ';|' '\n\n' | sed 's/&&/\n/g')

exit 0

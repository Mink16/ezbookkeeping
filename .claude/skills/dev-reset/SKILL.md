---
name: dev-reset
description: dev インスタンス (http://localhost:8082) のデータを初期化し、dev ユーザーと MCP トークンを再作成する手順。ユーザーが「dev をリセット」「dev のデータを初期化」「dev 環境を作り直して」と言ったときに使用する。本番 (8081 / docker-data/) には一切触れない。
---

# dev インスタンスのデータ初期化

dev (8082) は捨てデータ前提の検証環境。初期化は以下の手順を**コマンドを変更せずそのまま**実行する。

> ⚠️ 削除対象は `docker-data-dev/` のみ。`docker-data/`(本番の実データ)は絶対に対象にしない。
> 本番側への rm は `.claude/hooks/guard-prod.sh` が deny するが、hook を頼りにせずパスを必ず目視確認すること。

## 手順

1. **dev コンテナ停止**

   ```bash
   docker compose stop ezbookkeeping-dev
   ```

2. **dev データ削除(パス固定。変更禁止)**

   ```bash
   rm -rf ./docker-data-dev/data/* ./docker-data-dev/storage/* ./docker-data-dev/log/*
   ```

3. **起動と疎通確認**

   ```bash
   docker compose start ezbookkeeping-dev
   curl -s -o /dev/null -w '%{http_code}' http://localhost:8082/
   ```

   200 が返るまで数秒待つ。

4. **dev ユーザー再作成**(`dev` / `ezbk-dev-2026` — ローカル専用の捨てデータなので平文でよい)

   ```bash
   docker exec ezbookkeeping-dev ./ezbookkeeping userdata user-add \
     --username dev --password ezbk-dev-2026 --email dev@example.com --nickname dev --default-currency JPY
   ```

5. **dev MCP トークン再発行と再登録**(データ初期化で旧トークンは失効している)

   トークンは手で転記せずシェル変数経由で扱う(過去に転記ミスで invalid token になった実績がある):

   ```bash
   TOKEN=$(docker exec ezbookkeeping-dev ./ezbookkeeping userdata user-session-new \
     --username dev --type mcp --expiresInSeconds 0 --no-boot-log | grep -oP 'eyJ[A-Za-z0-9._-]+')
   claude mcp remove ezbookkeeping-dev -s local
   claude mcp add --transport http ezbookkeeping-dev http://localhost:8082/mcp \
     --header "Authorization: Bearer $TOKEN" -s local
   ```

   登録は **local スコープ**(project スコープはリポジトリにコミットされるため不可)。

6. **確認**: `claude mcp list` で `ezbookkeeping-dev` が Connected であること。MCP ツールの再認識にはセッション再起動が必要な場合があるので、必要ならユーザーに伝える。

> 補足: 手順 2 の `rm`(破壊操作)とトークン再発行・`claude mcp` 登録は `permissions.allow` に載せていないため確認プロンプトが出ることがある。これは安全側の意図的な挙動なので、表示されたら承認して進めてよい。

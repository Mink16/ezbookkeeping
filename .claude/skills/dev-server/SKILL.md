---
name: dev-server
description: ホットリロード開発サーバーの起動手順。フロントエンド (Vue) や バックエンド (Go) の変更を docker build なしで即座に確認したいとき、「開発サーバーを起動して」「ホットリロードで確認したい」と言われたときに使用する。
---

# ホットリロード開発サーバー

`docker compose build`(数分)を待たずに変更を確認するためのフロー。フロントエンドは保存した瞬間に反映される。

## ポート構成 (衝突回避)

| ポート | 用途 |
| --- | --- |
| 8080 | photo-clock (別プロジェクト、使用不可) |
| 8081 | 本番コンテナ (家族の実データ) |
| 8082 | dev コンテナ |
| 8083 | vite dev サーバー (ここにブラウザでアクセスする) |
| 8180 | ホスト Go バックエンド |

## 起動手順

1. **バックエンド** (ホストで Go を直接起動。コード変更時は再起動が必要):

   ```bash
   mkdir -p storage data log   # 初回のみ (gitignore 済みのローカル DB/ストレージ)
   EBK_SERVER_HTTP_PORT=8180 go run ezbookkeeping.go server run
   ```

   起動確認: `curl -s -o /dev/null -w "%{http_code}" http://localhost:8180/desktop/server_settings.js` が 200。
   ルート `/` は dist 未ビルドのため 404 で正常。

2. **フロントエンド** (vite ホットリロード):

   ```bash
   EZBOOKKEEPING_DEV_SERVER_PORT=8083 \
   EZBOOKKEEPING_DEV_API_PROXY_TARGET=http://127.0.0.1:8180/ \
   npm run serve
   ```

   ブラウザ/Playwright は http://localhost:8083/ にアクセスする。

3. **バックエンドの変更を試さない場合**は、手順 1 を省略してプロキシ先を dev コンテナにしてもよい:
   `EZBOOKKEEPING_DEV_API_PROXY_TARGET=http://127.0.0.1:8082/`

## 補足

- 環境変数 `EZBOOKKEEPING_DEV_SERVER_PORT` / `EZBOOKKEEPING_DEV_API_PROXY_TARGET` は custom/main で vite.config.ts に追加した独自拡張(未指定時は本家デフォルトの 8081 / 127.0.0.1:8080)
- ホスト Go バックエンドのデータはリポジトリ直下の `data/` / `storage/`(gitignore 済み、使い捨て)。ユーザーが必要なら `go run ezbookkeeping.go userdata user-add ...` で作成する
- vite は vue-tsc の型チェッカー (vite-plugin-checker) 同梱なので、型エラーは vite のログにも出る

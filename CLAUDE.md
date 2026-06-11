# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

ezBookkeeping — Go バックエンド + Vue 3 フロントエンドのセルフホスト家計簿アプリ。このリポジトリは mayswind/ezbookkeeping の fork で、独自機能を `custom/main` ブランチに載せて運用している。**ブランチ運用・コミット規約・検証環境の使い分けは `.claude/rules/feature-development.md` が正**(自動ロードされる)。

## コマンド

```bash
npm run lint                  # vue-tsc 型チェック + ESLint (--fix 付き)
npm test                      # Vitest 全テスト
npx vitest run src/lib/__tests__/math.test.ts   # 単一テストファイル
go test ./...                 # Go 全テスト
go test ./pkg/services/ -run TestAccountXxx     # 単一 Go テスト
./build.sh package -o ezbookkeeping.tar.gz      # 配布物ビルド
docker compose build && docker compose up -d    # デプロイ反映 (本番 8081 / dev 8082)
```

ホットリロード開発(vite:8083 + ホスト Go:8180)は `/dev-server` スキルの手順に従う。ポート 8080 は別プロジェクトが使用中なので使わない。

## バックエンド構成 (Go)

リクエストは「ルーター → API ハンドラ → サービス → データストア」と流れる:

- `ezbookkeeping.go` → `cmd/webserver.go` — エントリポイント。**全 HTTP ルーティングは `cmd/webserver.go` の `startWebServer()` 内**で `apiV1Route.GET("/xxx.json", bindApi(api.Xxx.Handler))` 形式で定義(`/api/v1/` には JWT 認可ミドルウェア適用済み)
- `pkg/api/` — ハンドラ層。シグネチャは `func (a *XxxApi) Handler(c *core.WebContext) (any, *errs.Error)`、各 API はシングルトン変数(`var Accounts = &AccountsApi{...}`)
- `pkg/services/` — ビジネスロジック。`func (s *XxxService) Method(c core.Context, uid int64, ...)`
- `pkg/models/` — XORM モデル(タグで DB カラム定義)+ リクエスト/レスポンス DTO
- `pkg/datastore/` — DB 抽象化(ORM は xorm)。スキーマ変更は **マイグレーションファイル不要**: モデルを変更し `cmd/database.go` の `updateAllDatabaseTablesStructure()` に `SyncStructs(new(models.Xxx))` があれば自動同期
- `pkg/settings/` — 設定。`conf/ezbookkeeping.ini` を `EBK_<セクション>_<キー>` 環境変数で上書きできる(例: `EBK_SERVER_HTTP_PORT`)

新 API エンドポイント追加で触るファイル: `pkg/models/xxx.go` → `pkg/services/xxx.go` → `pkg/api/xxx.go` → `cmd/webserver.go`(ルート登録)。テストは実装と同じディレクトリの `*_test.go`(testify/assert、テーブル駆動)。

## フロントエンド構成 (Vue 3)

**mobile と desktop の二重 UI** が最大の特徴。1 機能 = 2 画面実装が基本:

- マルチページ構成: `index.html`(UA 判定でリダイレクト)/ `mobile.html`(Framework7)/ `desktop.html`(Vuetify)。エントリは `src/index-main.ts` / `src/mobile-main.ts` / `src/desktop-main.ts`
- **Base パターン**: 共用ロジックは TypeScript(`src/views/base/XxxPageBase.ts`、`src/components/base/XxxBase.ts`)に置き、`src/views/{mobile,desktop}/` と `src/components/{mobile,desktop}/` の Vue コンポーネントがそれを利用して各 UI フレームワークでレンダリングする
- ルーティング: `src/router/mobile.ts` と `src/router/desktop.ts` の両方に登録が必要
- 状態管理: `src/stores/`(Pinia、Composition API スタイル)。ストアが `src/lib/services.ts`(axios ラッパー)経由で API を呼び、結果を保持する
- i18n: `src/locales/*.json`。コンポーネントからは `useI18n()`(`src/locales/helpers.ts`)の `tt('キー')` を使う。**独自機能では en.json と ja.json のみに追加**(本家は全ロケール同時追加だが、コンフリクト面を最小化するため)
- Vitest の対象は `src/lib/__tests__/` などロジック層のみ。UI の検証は dev インスタンス(8082)+ Playwright で行う

## 独自機能の実装方針

新規ファイル追加を優先し、既存ファイルの変更は最小限にする(本家マージのコンフリクト面を減らすため)。既存ファイルへの変更が必須なのは通常「ルート登録(`cmd/webserver.go`, `src/router/*.ts`)」「i18n(en/ja)」「`cmd/database.go`(新テーブル時)」の 3 箇所。

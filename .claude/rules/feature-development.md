# 独自機能開発の規約 (fork 運用)

このリポジトリは mayswind/ezbookkeeping (本家) の fork で、本家のアップデートを追従しながら独自機能を載せて運用する。以下の規約に必ず従うこと。

## ブランチ運用

- `main` は本家 (upstream) の完全ミラー。**独自コミットは絶対に入れない**。更新は `git pull upstream main` のみ
  - この禁止は PreToolUse hook (`.claude/hooks/block-git-violations.sh`) でも強制されており、main 上でのコミット作成系コマンド・`git push origin main`・upstream への push は自動的に deny される
- 運用ブランチは `custom/main`(本家リリース + 独自機能の統合ブランチ)
- 機能追加は `custom/main` 起点で `feature/xxx` ブランチを切り、完成後 `custom/main` にマージする
- 本家の取り込みは `/sync-upstream` スキルの手順に従う(**デフォルトはリリースタグ単位**。タグ未収載の機能が必要なときのみ、ユーザー合意の上で `upstream/main` を例外的に取り込む。日次の main 追従はしない)

## リモート構成

- `origin` = git@github.com:Mink16/ezbookkeeping.git(自分の fork。push 先)
- `upstream` = git@github.com:mayswind/ezbookkeeping.git(本家。fetch 専用、push 禁止 — push URL は `DISABLED` に固定済みで物理的に push 不可)
- `gh` のデフォルトリポジトリは本家(`gh issue` / `gh pr` は本家を対象に動く)

## コミット規約

- 本家流に合わせる: 英語・小文字始まり・簡潔な現在形(例: `add transaction picture upload resolution setting`)

## 実装方針

- できるだけ**新規ファイル追加**で実装し、既存ファイルへの変更は最小限にする(将来の本家マージのコンフリクト面を最小化するため)
- i18n キーを追加する場合は `en` と `ja` のみに追加する(本家は全ロケール同時追加だが、独自機能では追加しない)
- 一般に有用な機能は `main` 起点のブランチに切り出して本家へ PR することを検討する(本家ルールは README の Contributing 節: fork + PR のみ)

## 品質チェック (custom/main へのマージ前に必須)

```bash
npm run lint    # vue-tsc + ESLint
npm test        # Vitest
go test ./...
node .claude/scripts/check-i18n-parity.mjs   # en/ja の i18n キー集合一致
```

## 検証環境の使い分け

- **機能検証・テストデータ投入・Playwright E2E は必ず dev インスタンス (http://localhost:8082) で行う**
  - dev ユーザー: `dev` / `ezbk-dev-2026`(ローカル専用の捨てデータなので平文記載でよい)
  - MCP は `ezbookkeeping-dev`(dev 向け)を使う
- 本番インスタンス (http://localhost:8081) は家族の実データ。**機能検証に使わない**。`ezbookkeeping` MCP は実データの照会・記帳専用
  - この保護は PreToolUse hook (`.claude/hooks/guard-prod.sh`) でも強制: 8081 へのブラウザ操作 (navigate/evaluate)・`docker-data/` への破壊操作・`.env` 閲覧・本番への書き込み HTTP は deny、`docker-data/` への曖昧な書き込みと本番を巻き込む `docker compose`/`docker exec` は ask になる
  - **この hook はあくまで「事故」への一次防御**。正規表現でシェルを完全検閲することは原理的に不可能で、任意インタプリタや変数展開を使った回避までは防げない。本番データの最終防壁は OS レベルに置く: `.env` は `chmod 600`、`docker-data/` は定期バックアップ(DB マイグレーションが走る独自機能の追加前は特に)で守る
  - 本番 MCP (`ezbookkeeping`) は `permissions.allow` で `query_*` のみ無確認許可、`add_transaction` は `ask`。本家追従で新しい照会ツールが増えたら allow への追加が必要(漏れると毎回プロンプトになる)。dev MCP (`ezbookkeeping-dev`) はサーバー全体を allow(捨てデータのため書き込みも無確認)
- dev のデータ初期化は `/dev-reset` スキルの手順に従う(削除対象は `docker-data-dev/` 固定)

## デプロイ反映

独自機能は自前イメージの再ビルドで反映する(本家イメージ `mayswind/ezbookkeeping` への差し替えでは独自機能が消える):

```bash
docker compose build && docker compose up -d
```

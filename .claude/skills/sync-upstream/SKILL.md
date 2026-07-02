---
name: sync-upstream
description: 本家 (mayswind/ezbookkeeping) の更新を custom/main に取り込む手順。デフォルトはリリースタグ単位、タグ未収載の機能が必要な場合のみ upstream/main を例外的に取り込む。ユーザーが「本家に追従したい」「アップデートを取り込みたい」「新バージョンが出た」「本家の main を取り込みたい」と言ったときに使用する。
---

# 本家の取り込み手順 (タグ同期 / main 同期)

本家の更新を `custom/main` にマージし、独自機能を維持したままバージョンアップする。
**フローは共通で、取り込み対象 `<TARGET>` だけが異なる**(タグ = `vX.Y.Z` / main = `upstream/main`)。

## 0. モード判定

```bash
git fetch upstream --tags
git tag --sort=-creatordate | head -5                        # 最新タグ
git rev-list --count <最新タグ>..upstream/main               # タグ以降の main コミット数
git log --oneline <最新タグ>..upstream/main | head -20       # 未リリース分の内容
```

- **タグ同期(デフォルト)**: 新しいリリースタグがあればこちら。`<TARGET>` = タグ名
- **main 同期(例外)**: 欲しい機能・修正がまだタグに入っていない場合のみ。`<TARGET>` = `upstream/main`
  - **必ずユーザーの合意を得てから実行する**。確認事項: (1) 必要なコミットが main に入っているか (2) 未リリースコード(リファクタ途中・未確定の DB スキーマ)を実運用に載せるリスクを許容するか
  - main 同期後は次のタグが出たら通常のタグ同期に戻る(先行取り込み分だけ差分が小さくなる)

## 1. 事前確認と main ミラー更新

- `git status` がクリーンであること(未コミットは commit または stash)
- main ミラーの更新は **1 ステップずつ別コマンドで**実行する(hook が「main への切り替え + コミット系操作」の同一コマンド実行を deny するため):

```bash
git checkout main
git merge --ff-only upstream/main
git checkout custom/main
```

## 2. マージ開始とコンフリクトの切り分け

```bash
git merge --no-ff --no-commit <TARGET>
git diff --name-only --diff-filter=U      # コンフリクト一覧
```

`--no-commit` で止めるのは、解消内容と意味的修復(手順4)をまとめて 1 つのマージコミットにするため。

- 各コンフリクトの中身を確認し、「機械的に統合できるもの(インポート・定数・i18n)」と「設計判断が要るもの」を切り分ける
- 独自機能の中核(AI レシート認識・LLM プロファイル・admin 基盤など)と本家の変更が**同一領域を別設計で実装**している場合は、解消前に統合方針をユーザーに確認する
- 中断したくなったら `git merge --abort` でいつでもクリーンに戻せる

## 3. コンフリクト解消の指針

- 原則「**本家の変更を受け入れた上で、独自機能のフックを再適用する**」
- **ja.json / en.json**: コンフリクトマーカーを手で拾わず、プログラムで再構築するのが安全・確実:
  1. マージ済みの `en.json` をキー構造の基準にする
  2. 各キーの値を「upstream の ja 値 → custom の ja 値 → en 値」の順でフォールバックして合成(独自機能のキーは custom 値が残る)
  3. `JSON.stringify(obj, null, 4) + "\n"` で書き出す(両ファイルはこの形式と完全一致する。値行だけの差分になる)
  4. 独自の翻訳改善(override)がある場合は最後に再適用する
- Go ファイルを解消したら `gofmt -w <対象ファイル>` を通す
- 解消したら `git add` し、`git diff --name-only --diff-filter=U` が空になるまで繰り返す

## 4. 意味的破損の修復(最重要 — テキストコンフリクトがなくても壊れる)

本家のリファクタリングにより、**コンフリクトにならないのに独自コード(`custom_*` ファイル等)がビルド不能になる**ことがある。頻出パターン:

- 型・関数・エラー変数の**改名**(実例: `RecognizedReceiptImage*` → `RecognizedTransaction*`、`ErrNoTransactionInformationInImage` → `ErrNoTransactionInformation`)
- 共通関数の**シグネチャ変更**(実例: `bindApi` に `config` 引数が追加され、独自ルート登録 20 箇所すべてに追従が必要だった)
- auto-merge による**インポートの欠落**(fork 側で削除していた import を本家の新コードが必要とするケース)

手順:

```bash
go build ./...      # ← まずこれ。エラーが意味的破損の一覧になる
```

エラーが出た独自コードは**本家の改名・新シグネチャに追従させる**(独自機能の設計自体は維持する)。フロントエンドの同種の破損は手順 6 の lint (vue-tsc) が検出する。

## 5. 依存の更新

```bash
git diff HEAD --stat -- package.json package-lock.json go.mod
```

- `package-lock.json` が変わっていたら **`npm ci` を必ず実行**(怠ると lint/test が古い依存で誤動作する。実例: vite のメジャーアップデート)
- `go.mod` の変更は go コマンドが自動処理するので対応不要

## 6. 検証(すべて通ること)

```bash
go build ./... && go test ./...
npm run lint && npm test
node .claude/scripts/check-i18n-parity.mjs
```

あわせてマージ解消ミスで独自ファイルを失っていないかを確認する:

```bash
ls .claude/rules .claude/skills .claude/hooks docker-compose.yml .github/workflows/custom-branch-ci.yml
```

## 7. マージコミット

取り込み内容の要約と、**手順 4 で行った意味的追従(改名・シグネチャ変更への対応)**をコミットメッセージに記録する(次回マージ時の参照資料になる)。

## 8. dev での動作確認

dev インスタンス (http://localhost:8082) で独自機能を中心にスポット確認する(レシート認識・LLM プロファイル・admin)。**main 同期のときは必須**、タグ同期でも独自機能と衝突があった場合は行う。dev への反映は `docker compose build && docker compose up -d`(dev サービスも同一 compose)または `/dev-server` スキルのホットリロードで。

## 9. 本番反映

1. **`docker-data/` をバックアップ**(起動時の `SyncStructs` で DB スキーマが前進し、後戻りできないため。main 同期のときは特に必須)
2. `docker compose build && docker compose up -d`
3. http://localhost:8081/ で動作確認する

## 10. push

```bash
git push origin custom/main
```

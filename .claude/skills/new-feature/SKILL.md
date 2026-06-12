---
name: new-feature
description: ezBookkeeping への独自機能追加の定型フロー(ブランチ作成 → 実装 → 品質チェック → dev 検証 → custom/main マージ)。ユーザーが「〜機能を追加して」「〜を作って」「〜できるようにして」と新しい画面・設定項目・API・動作変更を依頼したら、明示的に「機能」と言われなくても必ずこのスキルを使うこと。バグ修正のみ・ドキュメント変更のみの場合は不要。
---

# 独自機能追加の定型フロー

機能追加を毎回同じ品質ゲートに通すためのフロー。規約の詳細は `.claude/rules/feature-development.md`、コードの触り方は `CLAUDE.md` が正。このスキルは「順序」と「抜けやすいチェック」を定義する。

## 1. 準備

1. `git status` を確認する。tracked ファイルに未コミットの変更があれば先に処理する(機能の diff に混入するため)。機能と無関係な untracked ファイルは残っていてよいが、feature のコミットに含めないこと
2. `custom/main` から feature ブランチを切る:

   ```bash
   git checkout custom/main && git pull origin custom/main
   git checkout -b feature/<英語ケバブケース>
   ```

## 2. 設計(実装前に必ず)

触るファイル一式を CLAUDE.md のアーキテクチャに沿って列挙し、ユーザーに一言で方針を伝えてから書き始める。特に:

- **フロントエンドは mobile と desktop の両方**が必要(views/base に共用ロジック → 各 UI)。片方だけ実装して完了と思い込むのが最頻出の抜け
- base の戻り値に項目を追加したら、**mobile と desktop 両方の Vue の分割代入(`const { ... } = useXxxPageBase()`)にも追加する**。テンプレートだけ書くと TS2339 になる(LSP 診断が即座に教えてくれる)
- 既存ファイルの変更は「ルート登録・i18n・cmd/database.go」の 3 箇所に収まるのが理想。それ以外の既存ファイルを大きく書き換えたくなったら、コンフリクト面が増える設計なので一度立ち止まって代替を検討する
- i18n キーは `en.json` と `ja.json` のみに追加する
- **XORM モデルで頭字語を含むフィールド(URL/API/ID 等)には明示カラム名タグ必須**: SnakeMapper は `BaseURL`→`base_u_r_l` のように分解するため、`Cols("base_url")` が実カラム名と一致せず**更新が黙って捨てられる**(過去の実バグ: 86a617cc)。`xorm:"'base_url' ..."` 形式で宣言し、新モデルは `pkg/models/custom_model_column_names_test.go` のチェック対象リストに追加する。SyncStructs で追加される NOT NULL カラムには `DEFAULT ''` を付ける(SQLite の ALTER が失敗するため)

## 3. 実装

- 開発中の動作確認はホットリロード(`/dev-server` スキル)を使うと速い
- 編集後の型エラーは LSP 診断と vite-plugin-checker が自動で出す。出たらその場で潰す(後回しにすると lint で まとめて返ってくる)

## 4. 品質チェックとコミット(マージ前に全部必須)

```bash
npm run lint && npm test && go test ./... && node .claude/scripts/check-i18n-parity.mjs
```

バックエンドを触っていなくても `go test ./...` は実行する(モデルの DTO 変更などが波及していることがある)。最後のパリティ検査は en.json / ja.json のキー集合一致の確認(片側だけの i18n 追加を検出)。

チェックが通ったら feature ブランチに**本家流メッセージ(英語・小文字始まり・簡潔な現在形)でコミットする**。dev 検証の前にコミットしておくと、検証で出た修正が差分として見える。

## 5. dev インスタンスでの動作検証

本番(8081)は家族の実データなので絶対に使わない。dev(8082)で:

```bash
docker compose build ezbookkeeping-dev && docker compose up -d ezbookkeeping-dev
```

- Playwright で http://localhost:8082 にログイン(`dev` / `ezbk-dev-2026`)し、**追加した機能を実際に操作して**スクリーンショットまたはスナップショットで確認する
- **更新系 API(modify/update)はラウンドトリップで検証する**: 全編集可能フィールドを**保存済みと異なる値**に変更 → 保存 → **再取得(get/list)または DB 直読み(`sqlite3 docker-data-dev/data/ezbookkeeping.db`)で永続化を確認**する。レスポンスはリクエストのエコーであることが多く保存失敗を検出できない。同値のまま送る・一部フィールドだけ変える検証は、フィールド単位の保存バグ(Cols 名不一致など)を素通しする(過去の実バグ: 86a617cc)
- 新テーブルを追加したら dev で `.schema <テーブル名>` を実行し、**実カラム名が想定どおりか目視確認**する
- UI の編集フォームは「開いて表示確認」で終わらせず、**値を変更 → 保存 → 再度開いて反映確認**まで行う
- 設定値に幅がある機能(モデル名・外部 URL 等)は、env の既定値だけでなく**ユーザーが実際に使いそうな別の値**(例: 大型モデル)でも 1 回は通す(タイムアウト等の限界はそこで露見する)
- テストデータが必要なら `ezbookkeeping-dev` MCP(`add_transaction` など)で投入する
- mobile UI も対象の機能なら、ブラウザを縮小するのではなく http://localhost:8082/mobile を直接開いて確認する
- **mobile はディープリンクが効かない**(`/mobile#/about` を直接開いてもホームに着地する)。Framework7 の画面遷移は UI のタブ・リストをクリックして辿ること
- ログインセッションはコンテナ再ビルド後も残っていることがある。ログイン画面が出なければそのまま進めてよい

## 6. マージと反映

1. `custom/main` にマージして push、feature ブランチを削除:

   ```bash
   git checkout custom/main && git merge feature/<name> && git push origin custom/main
   git branch -d feature/<name>
   ```

2. **本番への反映(`docker compose build && docker compose up -d`)は実データに影響するため、必ずユーザーに確認してから**実行する

## 7. 完了報告と後処理

- 何を実装し、どう検証したか(dev での操作内容)を報告する
- 機能が一般に有用なら「本家へ PR する価値がありそう」と一言添える(main 起点ブランチへの切り出しは別作業)

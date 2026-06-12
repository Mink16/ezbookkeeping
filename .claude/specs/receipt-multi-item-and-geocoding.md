# 指示書: レシートAI認識の複数明細分割登録 + 店舗所在地ジオコーディング

作成日: 2026-06-12 / 対象ブランチ: `custom/main` 起点の `feature/receipt-multi-item-geocoding`
ステータス: **要件確定済み(2026-06-12 ヒアリング完了、§7 参照)、実装着手可**

---

## 0. この指示書について

ユーザー要望は次の 2 点:

1. **機能A**: レシート画像の AI 認識結果が現在は 1 取引に合算されるが、レシート内に複数の商品明細がある場合、**明細ごとに複数の取引として登録できる**ようにしたい
2. **機能B**: レシートに住所や店舗支店名が記載されている場合、検索によって**座標を取得し、取引の地理座標に自動設定**できるようにしたい

本書はコードベース調査(実装箇所の特定)と外部 API 調査(ジオコーディング)に基づく要件定義・設計・実装手順書である。実装時は `/new-feature` スキルのフロー(ブランチ作成 → 実装 → 品質チェック → dev 検証 → `custom/main` マージ)に従うこと。

---

## 1. 現状アーキテクチャ(調査結果)

### 1.1 レシート認識のバックエンド

- エンドポイント: `POST /api/v1/llm/transactions/recognize_receipt_image.json`(`cmd/webserver.go:469`)
- ハンドラ: `RecognizeReceiptImageHandler`(`pkg/api/large_language_models.go:44`)。流れ:
  1. 画像のバリデーション → アカウント/カテゴリ(第2階層のみ)/タグの名前一覧を収集
  2. テンプレート `templates/prompt/receipt_image_recognition.tmpl` をレンダリング(compose で `templates-custom/prompt/receipt_image_recognition.tmpl` がマウント上書きされている)
  3. fork 独自フック `applyCustomReceiptSystemPrompt`(`pkg/api/llm_prompts.go:301`)でユーザー定義アクティブプロンプトに差し替え
  4. fork 独自フック `getCustomReceiptRecognitionJsonResponse`(`pkg/api/custom_llm_profiles.go:449`)で LLM 接続プロファイル経由のリクエスト
  5. レスポンス JSON を `models.RecognizedReceiptImageResult` に unmarshal → `parseRecognizedReceiptImageResponse`(同ファイル:250)で名前→ID 解決し `RecognizedReceiptImageResponse` を返す
- LLM 出力スキーマ(`pkg/models/large_language_model.go:17`): `type / time / amount / account / category / tags / description / destination_amount / destination_account` — **すべて単一取引前提。明細配列は存在しない**
- プロンプト自体に「**If the image contains multiple items, please combine them into a single transaction.**」(templates-custom 版ルール3)と合算を明示指示している
- `pkg/llm/` のプロバイダ抽象は `GetJsonResponse` の単発呼び出しのみ。**tool use / function calling は未実装**。Ollama アダプタは `"format":"json"`(JSON モード)を送信(`pkg/llm/provider/ollama/ollama_large_language_model_adapter.go:40`)。OpenAI 系アダプタのみ `ResponseJsonObjectType` による JSON Schema 構造化出力に対応しているが、レシート認識では未使用

### 1.2 レシート認識のフロントエンド

- mobile: `src/components/mobile/AIImageRecognitionSheet.vue` → 認識結果を `HomePage.vue:329 onReceiptRecognitionChanged` が**URL クエリパラメータに展開**して `/transaction/add?type=...&amount=...` へ遷移
- desktop: `src/views/desktop/transactions/list/dialogs/AIImageRecognitionDialog.vue`(Promise を返す `open()`)→ `ListPage.vue:1642 addByRecognizingImage` が `EditDialog.open({...})` にプリフィル
- どちらも**単一取引の編集画面へのプリフィル**であり、複数取引を確認・一括登録する UI は存在しない
- レシート画像の添付: `autoUploadTransactionPictureForAIRecognition` 設定が ON のとき、編集画面が画像を `POST /transaction/pictures/upload.json` でアップロードし、その取引 1 件に紐付ける(画像 1 枚 = 取引 1 件への紐付け)

### 1.3 取引作成 API と地理座標

- 作成: `POST /api/v1/transactions/add.json`(`pkg/api/transactions.go`)。**一括作成 API は存在しない**
- `TransactionCreateRequest.GeoLocation *TransactionGeoLocationRequest`(`pkg/models/transaction.go:155-158, 174`)で `latitude/longitude` を受け、`Transaction.GeoLongitude/GeoLatitude` に保存(`pkg/api/transactions.go:2952`)
- フロントは `Transaction.setGeoLocation()`(`src/models/transaction.ts:207`)+ 地図 UI(`src/components/common/MapView.vue`、mobile は `MapSheet.vue`)で手動設定のみ。`autoGetCurrentGeoLocation` 設定はブラウザの現在地を入れるだけ
- **住所→座標のジオコーディング機能はフロント・バックともに存在しない**
- サーバー側から外部 HTTP を叩く既存パターン: `pkg/httpclient/http_client.go NewHttpClient()` + `pkg/exchangerates/` のプロバイダ・ファクトリ構成、地図タイルプロキシ `pkg/api/map_image_proxies.go`

### 1.4 fork 占有領域(現状)

- errs カスタムサブカテゴリ: 90(プロンプト=290xxx)/ 91(admin=291xxx)/ 92(LLM プロファイル=292xxx)使用済み → **93(293xxx)以降が空き**
- `UuidType`: 本家 0-10、fork 14(LLM プロファイル)/ 15(プロンプト)使用済み → **11-13 が空き**
- XORM の罠: 頭字語フィールドは明示カラム名タグ必須(`pkg/models/custom_model_column_names_test.go` が検出)。新規 NOT NULL カラムは `DEFAULT` 必須(SQLite)

---

## 2. 要件定義

### 機能A: 複数明細の分割登録

| ID | 要件 | 優先度 |
|----|------|--------|
| A-1 | LLM はレシートから**取引全体の情報(従来通り)に加えて、商品明細の配列**(品名・金額・カテゴリ)を抽出する | MUST |
| A-2 | 認識後、ユーザーは**確認画面**で「1件にまとめて登録(従来動作)」と「明細ごとに分割して登録」を選択できる。既定は従来動作(まとめて 1 件) | MUST |
| A-3 | 確認画面では明細ごとに「登録対象 on/off・金額・カテゴリ・摘要」を修正できる | MUST |
| A-4 | 分割登録時、各明細は**個別の取引**として作成される(日時・アカウント・地理座標はレシート共通の値、カテゴリ/金額/摘要は明細ごと) | MUST |
| A-5 | 明細合計とレシート総額の差額は**仕組みで自動解消**し、登録される取引群の合計が常に支払総額と一致することを構築的に保証する(§4.4 差額ゼロ保証)。ユーザーが警告を読んで手作業で直す方式は採らない | MUST |
| A-6 | 明細抽出に失敗した(items が空/不正)場合は、**従来の単一取引フローに自動フォールバック**し、認識自体は成功扱いにする | MUST |
| A-7 | 分割登録時のレシート画像は**先頭の取引 1 件にのみ添付**する(`autoUploadTransactionPictureForAIRecognition` ON のとき)【決定済み §7-1】 | MUST |
| A-8 | mobile(Framework7)/ desktop(Vuetify)の両 UI で利用できる | MUST |
| A-9 | 一括登録は途中失敗時に**作成済み分を保持**し、失敗行を再試行できる(アトミック性は要求しない) | SHOULD |
| A-10 | LLM には印字行の**転記のみ**をさせ、計算・合算・按分は一切させない(LLM は算術が弱い)。値引き・クーポン行は**負の金額の明細**、外税の消費税行は**独立した明細**としてそのまま抽出する。判定と計算はすべて Go/TS の決定的コードで行う | MUST |
| A-11 | 各明細のカテゴリは LLM が品名から**明細ごとに推定**した値を初期値とし、確認画面で修正できる【決定済み §7-5】 | MUST |

### 機能B: 店舗所在地のジオコーディング

| ID | 要件 | 優先度 |
|----|------|--------|
| B-1 | LLM はレシートから**店舗名・支店名・住所の文字列を抽出するだけ**とし、座標の生成は行わない(幻覚座標の構造的排除) | MUST |
| B-2 | サーバーが抽出文字列を**決定的にジオコーディング API へ問い合わせ**、座標を解決する。LLM の tool use は使わない(§3.2 参照) | MUST |
| B-3 | プロバイダは**無料・キー不要の 2 段フォールバック**: ①国土地理院 AddressSearch(住所)→ ②OSM Nominatim(店舗名 POI・海外住所)。**有料プロバイダ(Google Places 等)は導入しない**【決定済み §7-2】 | MUST |
| B-4 | 解決結果(クエリ→座標)を**DB にキャッシュ**し、同一店舗の再問い合わせを回避する(Nominatim 利用規約のキャッシュ義務への適合) | MUST |
| B-5 | Nominatim 利用時は **1 req/s 上限・識別可能な User-Agent** を必ず守る | MUST |
| B-6 | ジオコーディング失敗は認識結果に影響させない(geo は null、警告ログのみ) | MUST |
| B-7 | 確認画面で解決済み座標を**地図プレビュー + 解決住所ラベル**で表示し、ユーザーが「位置情報を付与する」を on/off できる。座標が低信頼(複数ヒット・名称のみ一致)の場合は既定 off | MUST |
| B-8 | 分割登録時は全明細取引に同一座標を設定する | MUST |
| B-9 | 機能全体を設定(env)で無効化できる。外部送信されるのは店舗名・住所文字列のみであることを設定説明に明記 | MUST |

### 非機能要件

- 本家コンフリクト面の最小化: 既存ファイル変更は「ルート登録 (`cmd/webserver.go`)」「`cmd/database.go`(新テーブル)」「i18n (en/ja)」に限定し、ロジックは新規ファイルに置く
- 認識レイテンシ: LLM 認識が既に 10〜67 秒かかるため、ジオコーディング(数百 ms)は同期実行で許容
- gemma4 8B の出力安定性を考慮し、スキーマ拡張後も**トップレベルの従来フィールドは必須のまま維持**(items はあくまで追加情報)

---

## 3. 実現可能性調査の結論

### 3.1 機能A — 実現可能(中難度)

- 出力スキーマへの配列追加は Ollama JSON モードでも動くが、**gemma4 8B では items 配列の整形ミスのリスクが上がる**。緩和策: (1) few-shot 出力例をプロンプトに追加、(2) サーバー側で items を厳格バリデーションし不正なら捨てて A-6 フォールバック、(3) 精度不足なら `gemma4:31b` へ切替(pull 済み、env 差し替えのみ)、(4) 将来オプションとして Ollama の structured outputs(`format` に JSON Schema を渡す、v0.5+)をアダプタ拡張で利用可能
- 一括作成 API は本家に無いが、**フロントから既存 `/transactions/add.json` を逐次呼び出す方式で十分**。本家の取引作成は残高更新・タグインデックス等を含む複雑なトランザクションであり、バックエンドに一括 API を複製するとコンフリクト面とバグ面が増える。家計簿用途で明細数は高々数十件・逐次でも数秒
- 確認 UI は新規コンポーネントで実装可能。ただし mobile の現行ハンドオフ(URL クエリパラメータ)は配列を運べないため、**Pinia ストア経由のハンドオフに変更**が必要(新規ストアで吸収)

### 3.2 機能B — 実現可能(中難度)。「LLM 抽出 + サーバー決定的解決」一択

- `pkg/llm/` に tool use 基盤が無く、gemma4 系の tool calling は不安定。tool use 方式は実装コスト大 + Nominatim 規約(レート・UA・キャッシュ)の遵守保証が困難 + 幻覚座標リスクが残る。**B-1/B-2 の「抽出と解決の分離」が信頼性・実装コスト両面で明確に優位**
- ジオコーディング API 比較(2026-06 時点、詳細出典は調査ログ参照):

| プロバイダ | キー | 日本住所精度 | 店舗名+支店名 POI | 備考 |
|---|---|---|---|---|
| 国土地理院 AddressSearch (`https://msearch.gsi.go.jp/address-search/AddressSearch?q=`) | 不要 | ◎ 番地レベル | ✕(商用店舗は対象外) | レート明示なし・SLA なし(「現状有姿」)。レスポンスは GeoJSON 風で **coordinates は [経度, 緯度] の順** |
| OSM Nominatim (`https://nominatim.openstreetmap.org/search`) | 不要 | △ 町名止まりが多い | △ OSM 登録済み店舗のみ | **1 req/s・独自 UA 必須・結果キャッシュ義務**。`countrycodes=jp&accept-language=ja&format=jsonv2` |
| Google Places Text Search (New) | 必要(課金登録) | ◎ | ◎ 唯一実用レベル | 座標取得は Pro SKU = **月 5,000 回無料**。FieldMask `places.location` 必須 |
| Photon / OpenCage / Mapbox / HERE | — | △ | △ | 不採用(精度・規約・キー要件で上記に劣後) |

- **結論**: 「住所が読めたら GSI、店舗名しか無ければ Nominatim」の無料 2 段フォールバック。Google Places は精度面で唯一の店舗名検索実用レベルだが課金登録が必要なため**不採用が確定**(§7-2)。個人利用(日数回〜数十回)+ キャッシュで両プロバイダの規約に収まる
- レシートには通常、店舗住所が印字されている(日本のレシートはほぼ確実)ため、キー不要の GSI だけでも実用上の解決率は高い見込み

---

## 4. 設計

### 4.1 バックエンド: 新カスタムエンドポイント(本家ハンドラは不変更)

本家の `RecognizeReceiptImageHandler` と並存する **fork 専用エンドポイント**を新設する。本家ハンドラへのフック追加は行わない(コンフリクト面ゼロ)。

```
POST /api/v1/custom/llm/transactions/recognize_receipt_image_details.json
```

- 新規ファイル `pkg/api/custom_receipt_recognition.go`(package api)。`large_language_models.go` と同パッケージなので `applyCustomReceiptSystemPrompt` / `getCustomReceiptRecognitionJsonResponse` / `getLongDateTime` / `parseRecognizedReceiptImageResponse` をそのまま再利用できる
- ルート登録は `cmd/webserver.go` に 1 行(既存の `/api/v1/custom/` 群の隣)。ガード条件は本家と同じ `TransactionFromAIImageRecognition` + feature restriction
- **本家追従時の同期確認事項**として「本家 `RecognizeReceiptImageHandler` の処理変更(バリデーション・プロンプトパラメータ追加等)をカスタムハンドラへ反映する」を ollama-receipt-recognition メモリに追記すること

#### LLM 出力スキーマ(拡張、新規 struct)

```go
// pkg/models/custom_receipt_recognition.go (新規)
type RecognizedReceiptDetailsResult struct {
    models.RecognizedReceiptImageResult            // 従来フィールドを埋め込み(合算値、フォールバック用)
    Items    []*RecognizedReceiptItemResult `json:"items,omitempty"`
    Merchant *RecognizedReceiptMerchantResult `json:"merchant,omitempty"`
}
type RecognizedReceiptItemResult struct {
    Name     string `json:"name,omitempty"`     // 品名
    Amount   string `json:"amount,omitempty"`   // 税込金額 (文字列、本家 amount と同形式)
    Category string `json:"category,omitempty"` // Options から選択
}
type RecognizedReceiptMerchantResult struct {
    Name    string `json:"name,omitempty"`    // 店舗名 (例: セブン-イレブン)
    Branch  string `json:"branch,omitempty"`  // 支店名 (例: 渋谷駅前店)
    Address string `json:"address,omitempty"` // 印字住所そのまま
}
```

#### レスポンス DTO(拡張、新規 struct)

```go
type RecognizedReceiptDetailsResponse struct {
    models.RecognizedReceiptImageResponse                 // 合算値 (従来互換)
    Items       []*RecognizedReceiptItemResponse `json:"items,omitempty"`
    GeoLocation *models.TransactionGeoLocationResponse `json:"geoLocation,omitempty"`
    Location    *RecognizedReceiptLocationResponse `json:"location,omitempty"` // 解決住所ラベル・信頼度・プロバイダ名
}
type RecognizedReceiptItemResponse struct {
    CategoryId     int64  `json:"categoryId,string,omitempty"`
    Amount         int64  `json:"amount,omitempty"`  // ×100 (本家 sourceAmount と同じ)。値引き行は負値
    Comment        string `json:"comment,omitempty"` // 品名 (調整行はフロントが i18n で命名するため空)
    IsAdjustment   bool   `json:"isAdjustment,omitempty"`   // §4.4 のリコンサイルで自動生成された行
    AdjustmentKind string `json:"adjustmentKind,omitempty"` // "tax" (外税と推定) / "unknown" (読取り差異)
}
```

- パース時のバリデーション: 各 item の amount が `utils.ParseAmount` で解釈不能、または items 全体が空 → `Items=nil` で返す(A-6 フォールバック)
- バリデーション通過後、**サーバー側で §4.4 のリコンサイルを必ず実行**してから返す。したがって API レスポンスの時点で常に `Σ items[].Amount == SourceAmount` が成立している

#### プロンプトテンプレート

- 新規 `templates-custom/prompt/receipt_image_recognition_details.tmpl`。本家テンプレート機構(`pkg/templates`)に fork 専用定数を追加するか、`templates-custom` を直接読むかは実装時に `pkg/templates` の拡張性を見て判断(新規定数 1 行追加が最小)
- 内容: 現行 templates-custom 版をベースに、(1) ルール3「combine into single transaction」を「**合算値に加えて items 配列にも分解せよ**」へ変更、(2) items / merchant のスキーマ定義、(3) **複数明細レシートの few-shot 出力例**(gemma4 8B 対策、過去に `transaction_type` 誤りを出力例で矯正した実績の踏襲)、(4) 税・値引きの扱い(A-10)
- **プロンプト管理機能との関係(v1 の割り切り)**: ユーザー定義アクティブプロンプトは従来エンドポイント用スキーマで書かれているため、**details エンドポイントには適用しない**(`applyCustomReceiptSystemPrompt` を呼ばず、専用テンプレート固定)。プロンプト管理へ「details 用プロンプト種別」を追加するのは Phase 3(任意)

### 4.2 バックエンド: ジオコーディング基盤(新規パッケージ)

```
pkg/geocoding/
├── geocoding.go                 // インターフェース + フォールバック連鎖 + キャッシュ参照
├── gsi_provider.go              // 国土地理院 AddressSearch
├── nominatim_provider.go        // OSM Nominatim (1req/s スロットリング + UA)
└── *_test.go
```

(Google Places プロバイダは実装しない — §7-2 で不採用確定。インターフェースは将来追加可能な形だけ保つ)

```go
type GeocodingProvider interface {
    Name() string
    Search(c core.Context, query string) (*GeocodingResult, error) // 1位ヒットのみ返す
}
type GeocodingResult struct {
    Latitude, Longitude float64
    DisplayName         string  // 解決住所ラベル
    Confidence          GeocodingConfidence // high (住所一致) / low (名称のみ一致)
}
```

- HTTP は `pkg/httpclient.NewHttpClient()` を利用(exchangerates と同パターン)。タイムアウト・プロキシは設定値
- **クエリ構築(決定的、Go 側)**: ①住所があれば GSI に住所のみ(`address` を正規化: 全角数字→半角、「TEL」以降除去等)→ ②ヒットなし or 住所なしなら Nominatim に `"{name} {branch} {市区町村(住所から抽出できれば)}"`。番地まで含めて 0 件なら**末尾の番地を落として再検索**(GSI 1 回だけリトライ)
- **Nominatim 遵守**: provider 内部に `sync.Mutex` + 前回呼び出し時刻で最小間隔 1.1 秒を強制。User-Agent は `ezbookkeeping-custom/1.0 (self-hosted; +https://github.com/Mink16/ezbookkeeping)`
- **キャッシュテーブル**(新規モデル `pkg/models/custom_geocoding_cache.go`):

```go
type CustomGeocodingCache struct {
    QueryHash   string `xorm:"'query_hash' VARCHAR(64) PK"` // SHA-256(正規化クエリ)
    Query       string `xorm:"'query' VARCHAR(255) NOT NULL"`
    Provider    string `xorm:"'provider' VARCHAR(32) NOT NULL DEFAULT ''"`
    Latitude    float64 `xorm:"'latitude' NOT NULL DEFAULT 0"`
    Longitude   float64 `xorm:"'longitude' NOT NULL DEFAULT 0"`
    DisplayName string `xorm:"'display_name' VARCHAR(255) NOT NULL DEFAULT ''"`
    Confidence  byte   `xorm:"'confidence' NOT NULL DEFAULT 0"`
    Hit         bool   `xorm:"'hit' NOT NULL DEFAULT false"` // 0件結果もキャッシュ (negative cache)
    CreatedUnixTime int64 `xorm:"'created_unix_time' NOT NULL DEFAULT 0"`
}
```

  - PK はクエリハッシュなので **UuidType は消費しない**(11-13 温存)
  - 全フィールドに明示カラム名タグ(XORM 頭字語の罠の予防 + `custom_model_column_names_test.go` への登録)
  - `cmd/database.go` の `updateAllDatabaseTablesStructure()` に `SyncStructs` を 1 行追加(許容された変更点)
  - negative cache の TTL(例: 30 日)を超えたら再問い合わせ。positive は無期限
- **サービス層**: `pkg/services/custom_geocoding.go` — キャッシュ参照 → 連鎖実行 → キャッシュ保存。エラーはすべて吸収して `nil` を返す(B-6)
- **設定**: 既存 fork パターン(`EBK_CUSTOM_ADMIN_USERNAMES`)に倣い env で注入:
  - `EBK_CUSTOM_GEOCODING_ENABLE`(default false。**本番有効化は dev で精度確認後** — §7-3)
  - `EBK_CUSTOM_GEOCODING_PROVIDERS`(default `gsi,nominatim`)
  - `EBK_CUSTOM_GEOCODING_REQUEST_TIMEOUT` / `EBK_CUSTOM_GEOCODING_PROXY`
  - 有効フラグはフロントへ server settings(`window.EZBOOKKEEPING_SERVER_SETTINGS`)経由で渡す必要がある → `pkg/api/server_settings.go` への追記が必要になるため、**代替案**: details レスポンスの `location` の有無だけで UI を出し分ければ server settings 変更は不要。**代替案を採用**(変更ファイルを増やさない)
- エラーコード: 原則エラーは飲み込むため新設不要。バリデーション系で必要になったら **errs サブカテゴリ 93(293xxx)** を `pkg/errs/custom_geocoding.go` として予約

### 4.3 フロントエンド: 認識結果確認 UI

新規コンポーネント 3 点 + ストアハンドオフ:

```
src/views/base/transactions/ReceiptRecognitionConfirmBase.ts   // 共用ロジック
src/components/desktop/ReceiptRecognitionConfirmDialog.vue     // Vuetify
src/views/mobile/transactions/ReceiptRecognitionConfirmPage.vue // Framework7 (新ルート)
src/models/custom_receipt_recognition.ts                       // details レスポンス型
src/stores/customReceiptRecognition.ts                         // 認識結果の一時保持 + 一括登録ロジック
```

#### フロー変更

- `services.ts` に `recognizeReceiptImageDetails()` を追加(既存 `recognizeReceiptImage` と並存)。AIImageRecognitionSheet/Dialog は details 版を呼ぶように差し替え(この 2 ファイルは fork で改変済み領域ではないが、呼び出し関数名の変更のみで差分最小。**代替**: props で切り替え可能にして既存呼び出しを温存)
- レスポンスに `items` が 2 件以上ある場合のみ確認画面へ。0〜1 件なら**従来フロー(単一取引プリフィル)へ直行**(A-6)
- mobile のハンドオフは URL クエリではなく `customReceiptRecognition` ストアに結果を置き、確認ページ遷移後に読む

#### 確認画面の仕様

- ヘッダ: レシート画像サムネイル、店舗名、日時、アカウント、合計金額
- 位置情報セクション(`location` があるとき): 解決住所ラベル + `MapView.vue` プレビュー + 「位置情報を付与」トグル(confidence=high で既定 on、low で既定 off)(B-7)
- 登録モード切替: 「1件にまとめる(既定)」/「明細ごとに登録」(A-2)
- 明細リスト(分割モード時): 行ごとに [対象チェック / 品名(=comment、編集可) / 金額(編集可) / カテゴリセレクタ(LLM の明細別推定値が初期値、A-11)]。調整行(`isAdjustment`)は他の行と区別できる外観(アイコン + i18n 名「消費税」/「調整」)で表示
- フッタ: 「明細合計 = 支払総額」の成立状態を常時表示。差額の手動解消作業は発生しない(§4.4 の不変条件エンジンが調整行で自動吸収)。「差額を明細に按分」アクションをメニューに用意(最大剰余法、実行後も合計一致は厳密維持)
- 登録実行: 分割時は `transactionsStore.saveTransaction` 相当を**逐次 await**(時刻は全行同一で良い — サーバー側が同時刻の取引を順序付けする)。行ごとに成功/失敗ステータスを表示し、失敗行のみ再実行ボタン(A-9)。geo トグル on なら全行の `transaction.setGeoLocation()` に同一座標(B-8)
- 画像添付: 分割時は先頭行の取引にのみ `uploadTransactionPicture` で添付(A-7)。まとめモードは従来動作

#### i18n

`src/locales/en.json` / `ja.json` のみに追加(規約)。追加後 `node .claude/scripts/check-i18n-parity.mjs` を必ず実行。主なキー: 確認画面タイトル、登録モードラベル、差額警告、位置情報トグル、解決住所、登録進捗/失敗リトライ等(実装時に列挙)

### 4.4 差額ゼロ保証(Sum Invariant)の設計

ユーザー決定(§7-4)「差額が発生しないように仕組みで解決する」を実現する 3 層構成。**不変条件は `Σ(登録対象明細の金額) == 支払総額 T`** で、これを警告ではなく構築的に(by construction)成立させる。

#### 設計原則: LLM は転記、計算は決定的コード

差額の発生源は (1) 外税方式(明細が税抜)、(2) 値引き・クーポン行、(3) 軽減税率 8%/10% の混在、(4) LLM の行欠落・桁誤り、(5) 円未満の端数処理。このうち LLM に「税込に直せ」「合計を合わせろ」と算術をさせるのは 8B モデルでは最も壊れやすい。よって **LLM は印字された行の転記だけ**(A-10)を行い、以降はすべて整数演算(金額は ×100 の int64、浮動小数不使用)の決定的コードで処理する。

#### 層1: LLM 抽出の規約(プロンプトで指示)

- `items[].amount` はレシートに**印字された行の金額をそのまま転記**(税抜レシートなら税抜のまま)
- 値引き・クーポン・ポイント値引きの行は**負の金額の明細**として出力
- 外税の消費税行は「消費税」という独立明細として出力(専用 kind フィールドは設けない — スキーマを増やすほど 8B の整形ミスが増える)
- 小計・合計行は items に**含めない**。トップレベル `amount` には支払総額を転記

#### 層2: サーバー側リコンサイル(`reconcileReceiptItems`、Go・要ユニットテスト)

バリデーション通過後のレスポンス構築時に必ず実行する純関数:

```
入力: T = SourceAmount (支払総額 ×100), items[] (符号付き ×100)
S = Σ items[].Amount
delta = T − S
if delta == 0 → そのまま返す
if delta != 0 →
    調整行を items 末尾に追加 { Amount: delta, IsAdjustment: true, AdjustmentKind: kind }
    kind 判定 (表示ラベル用、計算には影響しない):
        delta > 0 かつ delta が「正の明細合計の 8% または 10%、もしくはその混合」の
        端数処理 (切捨て/四捨五入) ±2円 以内に一致 → "tax" (外税と推定)
        それ以外 → "unknown" (読取り差異)
出力: 常に Σ items[].Amount == T を満たす items
```

- 調整行方式を既定とする理由: 税の按分は各明細の金額をレシート印字値から変えてしまい、ユーザーが**レシートと突き合わせて検証できなくなる**。調整行は印字値を保存したまま合計一致を保証し、外税という実態とも一致する。さらに LLM の読取りミスが「不自然に大きい調整行」として**可視化**されるため、誤読の検出器としても機能する
- `T` 自体が欠落(LLM が総額を読めなかった)場合は `T := S` とし調整行なし(明細側を信頼)

#### 層3: フロント確認画面の不変条件エンジン(リアクティブ、要 Vitest)

確認画面では編集操作のたびに調整行が自動追従し、不変条件を維持し続ける:

| 操作 | 挙動 |
|---|---|
| 明細金額を修正 | 調整行が `T − Σ(調整行以外の対象行)` に即時再計算される。0 円になったら行を自動非表示 |
| 支払総額 T を修正(ヘッダで編集可、総額の誤読対応) | 同上の再計算 |
| 行のチェックを外す(登録除外) | 調整行は**追従しない**(追従すると除外の意図が無効になる)。「登録合計が支払総額と異なります(除外 n 件)」の情報バッジを表示 — これは意図的な操作なのでエラーにしない |
| 「差額を明細に按分」アクション | 調整行の金額を各明細へ最大剰余法で配分して調整行を消す(下記) |
| 登録実行 | 除外行が無いのに不変条件が破れている状態は設計上発生しない(発生したらフロントのバグなので登録をブロックし console error) |

#### 按分アルゴリズム(最大剰余法、整数演算で厳密)

```
配分額 delta (×100 整数), 対象 = 正の金額の明細 (重み w_i = Amount_i), W = Σ w_i
share_i = floor(delta × w_i / W)            // int64 演算
余り r = delta − Σ share_i                   // 0 ≤ r < 対象行数
小数部 (delta × w_i mod W) の大きい順に r 行へ最小通貨単位を 1 ずつ加算
⇒ Σ(Amount_i + share_i) == S + delta == T が厳密に成立 (丸め誤差なし)
```

実装は `src/lib/` 配下の純関数(`distributeAmountProportionally(delta, weights): number[]`)とし、Vitest のテーブル駆動テスト(端数・1円・全行同額・負額混在ケース)を必須とする。

### 4.5 変更ファイル一覧(見込み)

| 区分 | ファイル | 変更内容 |
|---|---|---|
| 新規 | `pkg/api/custom_receipt_recognition.go` | details ハンドラ |
| 新規 | `pkg/models/custom_receipt_recognition.go` | 拡張 Result/Response |
| 新規 | `pkg/models/custom_geocoding_cache.go` | キャッシュモデル |
| 新規 | `pkg/geocoding/*.go` | プロバイダ + 連鎖 |
| 新規 | `pkg/services/custom_geocoding.go` | 解決サービス |
| 新規 | `templates-custom/prompt/receipt_image_recognition_details.tmpl` | 拡張プロンプト |
| 新規 | フロント 5 ファイル(§4.3) | 確認 UI + ストア + 型 |
| 既存 | `cmd/webserver.go` | ルート 1 行 |
| 既存 | `cmd/database.go` | SyncStructs 1 行 |
| 既存 | `pkg/templates/`(該当ファイル) | テンプレート定数 1 行 |
| 既存 | `src/lib/services.ts` | API 関数 1 個追加 |
| 既存 | `AIImageRecognitionSheet.vue` / `AIImageRecognitionDialog.vue` / `HomePage.vue` / `ListPage.vue` | details 呼び出しへの分岐(最小差分) |
| 既存 | `src/router/mobile.ts` | 確認ページのルート登録 |
| 既存 | `src/locales/{en,ja}.json` | i18n キー |
| 既存 | `pkg/models/custom_model_column_names_test.go` | 新モデルの登録 |
| 既存 | `docker-compose.yml`(または env) | `EBK_CUSTOM_GEOCODING_*` |

---

## 5. 実装フェーズ

### Phase 1: バックエンド(機能A+B の API)
1. `feature/receipt-multi-item-geocoding` ブランチ作成(custom/main 起点)
2. 拡張モデル + details ハンドラ(まず items のみ、geo は nil 固定)+ プロンプトテンプレート + ルート登録
3. `pkg/geocoding/` 実装(GSI → Nominatim 連鎖、キャッシュ、設定)+ details ハンドラへ組み込み
4. Go テスト: items パース(正常/不正/空でのフォールバック)、**リコンサイル `reconcileReceiptItems`(差額ゼロ/外税8%/10%/混合/値引き負額/総額欠落/読取り差異)**、ジオコーディングのクエリ正規化・フォールバック・キャッシュ(HTTP はモック)、カラム名テスト登録
5. `curl` + dev インスタンスで実レシート画像 3 種(単品/複数明細/手書き風)の認識精度を確認。gemma4 8B で items が安定しなければプロンプト調整 → だめなら 31b で再評価

### Phase 2: フロントエンド(確認 UI)
6. 型・ストア・base ロジック → desktop ダイアログ → mobile ページの順で実装
7. 品質チェック一式: `npm run lint` / `npm test` / `go test ./...` / `check-i18n-parity.mjs`
8. dev インスタンス(8082、dev / ezbk-dev-2026)+ Playwright で E2E: 分割登録・調整行の自動追従(金額編集/総額編集/按分)・geo トグル・失敗リトライ・従来フロー非劣化(items 1 件のレシート)

### Phase 3(任意・別ブランチ可)
- プロンプト管理機能への「details 用プロンプト種別」追加(プレースホルダヒント拡張含む)
- Ollama アダプタの structured outputs(JSON Schema)対応 — 本家ファイル改変になるためコンフリクト面と相談

### 完了条件(DoD)
- 複数明細レシートで「明細ごとに登録」した結果、dev の取引一覧に明細数分の取引(各カテゴリ/金額/摘要、共通の日時・座標)が登録される
- 店舗住所入りレシートで座標が自動解決され、登録取引の地図表示が店舗位置を指す
- 単品レシート・items 抽出失敗時に従来とまったく同じ UX で登録できる
- 品質チェック 4 点グリーン、`custom/main` へマージ、`docker compose build && up -d` で本番反映
- ollama-receipt-recognition メモリに本家追従時の同期確認事項(details ハンドラの追随)を追記

---

## 6. リスクと緩和策

| リスク | 影響 | 緩和策 |
|---|---|---|
| gemma4 8B が items 配列を安定生成できない | 分割機能が実質使えない | few-shot 例 + 厳格バリデーション + A-6 フォールバック + 31b 切替。Phase 1 step 5 で早期検証し、だめなら本機能を「対応プロバイダでのみ有効」と割り切る |
| GSI API の仕様変更・停止(SLA なし) | geo 解決率低下 | Nominatim フォールバック + B-6 により機能停止しても認識は無傷 |
| Nominatim 規約違反による BAN | OSM 系全機能に影響 | 1.1s スロットル・UA・キャッシュをコードで強制。利用は認識 1 回につき最大 1〜2 クエリ |
| 逐次一括登録の途中失敗 | 取引の部分登録 | 行ステータス + 失敗行リトライ(A-9)。重複防止のため成功行は再実行対象から除外 |
| 本家がレシート認識を改修(明細対応を本家が実装する可能性も) | 同期コスト | details ハンドラは独立ファイルなのでビルドは壊れない。`/sync-upstream` 時の確認事項としてメモリに記録 |
| 確認画面追加による UX 後退(タップ数増) | 日常利用の摩擦 | items ≤1 のとき確認画面をスキップして従来フロー直行 |

---

## 7. 決定事項(2026-06-12 ユーザーヒアリング結果)

1. **画像添付先**(A-7): **先頭の取引のみ**に添付(ストレージ節約)
2. **Google Places**: **導入しない**。「無料で利用できる地図検索が無ければ座標検索は利用しない」がユーザーの判断基準であり、国土地理院 + Nominatim はキー不要・完全無料のため、機能Bは**無料プロバイダのみで実装**する(2 回目のヒアリングでスコープ確定)。Google 前提の設計要素(API キー設定・プロバイダ実装)はすべて削除済み
3. **本番有効化**: `EBK_CUSTOM_GEOCODING_ENABLE` は **dev(8082)で精度確認後に本番(8081)へ反映**。外部送信されるのは店舗名・住所文字列のみ
4. **差額処理**(A-5): 警告表示ではなく**「差額が発生しないように仕組みで解決」**。§4.4 の 3 層設計(LLM 転記主義 → サーバー側リコンサイルで調整行自動生成 → フロント不変条件エンジン + 最大剰余法の按分)を採用
5. **明細カテゴリ**(A-11): **LLM が明細ごとに推定**した値を初期値とし、確認画面で修正可能にする

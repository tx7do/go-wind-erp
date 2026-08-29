<div align="center">

# GoWind ERP

### 風行 · 中小チームのための軽量・オールインワン ERP 基盤

**導入コンサルタント不要 · 半日で使い始められる · コア業務データを完全に連携**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

**[English](./README.en-US.md)** · **[中文](./README.md)** · **日本語**

</div>

---

調達・在庫・財務——中小チームのビジネスの核心を、ひとつのシステムで閉じます。承認済みの発注から買掛金が自動生成され、入庫は在庫へ自動反映、低在庫は補充案を自動提案、買掛金エイジングもいつでも確認できます。データは一度入力すれば全体に行き渡ります。導入コンサルタントも、長い導入期間も不要です。

**GoWind ERP を 3 文で：**

- **導入コンサルタント不要** — 登録すればすぐ開始：テナント・部門・ロール・管理者は自動初期化、権限・承認・サイト内メッセージ・監査などの基盤機能をすべて内蔵
- **半日で使い始められる** — 管理コンソールとモバイルアプリを同梱。調達 → 承認 → 入庫 → 在庫 → 支払のコアフローがそのまま動き、倉庫の現場はスキャンで、経営層はダッシュボードで
- **コア業務データを完全に連携** — ドキュメントがドキュメントを駆動：承認済み発注は買掛金を自動生成、入庫は在庫を自動更新、低在庫は補充発注のドラフトを自動作成。データは一度の入力で全リンクに反映

開発者にも扱いやすい設計：API ファースト（RESTful / gRPC デュアルプロトコル、OpenAPI ドキュメント自動生成）、マイクロサービス、ネイティブ多言語対応——詳しくは下記の[テクノロジースタック](#テクノロジースタック)を参照してください。

## プロジェクトの状況

GoWind ERP は、組織と権限、認証、監査、ファイルストレージ、辞書、サイト内メッセージ、タスクスケジューリング、国際化といった汎用基盤機能に加え、3 つの ERP 業務モジュール（在庫 / WMS、調達 / SRM、財務 / AP）も実装済みであり、コアバックエンドサービス・管理コンソール・モバイルアプリの三層をすべてカバーしています。主要な業務フロー（発注承認から入庫まで、買掛金の支払・決済、在庫転送と取消、低在庫の自動補充）はエンドツーエンドで閉じており、重要な一貫性ロジック（入庫承認ゲート、転送の単一トランザクション原子性、エイジングレポートの未払残高集計）には回帰テストが付いています。

## コア機能

### 組織 & 権限

| 機能 | 説明 |
|:---|:---|
| マルチテナント管理 | テナントの追加、有効化/無効化、データ分離。新規テナントの部門、ロール、管理者を自動初期化 |
| ユーザー管理 | ライフサイクル全体を管理、複数ロール/部署/役職のバインディング |
| ロール管理 | ロール & ロールグループ管理、メニュー/API/データ権限の設定 |
| 権限管理 | 権限グループ、メニューノード、権限ポイント、ポリシーエンジンによる認可、ボタンレベルの制御 |
| メニュー管理 | ディレクトリ/ページ/ボタンの 3 レベル設定、権限に基づく動的レンダリング |
| 部門 & 役職管理 | 複数レベルの部門ツリーと役職階層、ユーザーとの連携バインディング |
| 認証 & ログインポリシー | JWT の発行と検証、ログインポリシー設定、資格情報 & トークンキャッシュ管理 |

### システム & 運用

| 機能 | 説明 |
|:---|:---|
| 監査ログ | ログイン、操作、API 呼び出し、データアクセス、権限変更、ポリシー評価の全リンク監査 — 操作者、IP、パラメータ & 結果を記録 |
| ファイル管理 | ローカルまたはオブジェクトストレージへの統一アップロード/ダウンロード、プレビュー & グループ管理 |
| 辞書管理 | データ辞書カテゴリ & アイテム、連携する多言語翻訳 |
| サイト内メッセージ | 複数レベルのメッセージカテゴリ、対象配送 & 既読ステータス追跡、個人受信トレイ |
| タスクスケジューリング | Cron ジョブ管理、開始/一時停止/即時実行、実行履歴 & ログ |
| 多言語管理 | 言語管理、コンテンツ/メニュー/UI テキストの統一翻訳 |

### ERP 業務モジュール

| モジュール | 説明 |
|:---|:---|
| 在庫 / WMS | 倉庫・在庫管理（状態機械）、入出庫移動（サーバー検証の書き戻し）、倉庫間転送（単一トランザクションで原子的実行）、移動の取消（冪等）、低在庫の自動補充提案（承認レール経由でドラフト発注を生成）、経営ダッシュボード集計 & 30 日間の移動トレンド |
| 調達 / SRM | サプライヤー管理、発注のライフサイクル全体（ドラフト → 提出 → 承認 → 差戻し再提出 → 入庫 → 全数入庫で自動完了 / キャンセル）、明細単位の入庫書き戻し（承認ゲート + 過剰入庫ガード） |
| 財務 / AP | 買掛金（発注承認時に自動生成）、支払申請（承認レール経由で計上）、部分支払と決済の状態機械、買掛金エイジングレポート |
| 承認レール（横断） | 発注 / 支払 / 補充の統一承認。申請者と承認者はサーバー側で導出し自己承認を禁止。承認結果は申請者にサイト内メッセージで通知し、補充ドラフト生成・発注自動完了の下流通知も発火 |
| モバイル業務機能 | WMS スキャン（入出庫 / 転送 / 取消）、承認センター、経営ダッシュボード |

## テクノロジースタック

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Flutter](https://img.shields.io/badge/Flutter-3.x-02569B?logo=flutter&logoColor=white)](https://flutter.dev/)
[![Kratos](https://img.shields.io/badge/Kratos-2.9-00ADD8?logo=go&logoColor=white)](https://go-kratos.dev/)

### バックエンド

| レイヤー | 技術 | 説明 |
|:---|:---|:---|
| 言語 | [Go 1.25+](https://go.dev/) | 高性能コンパイル言語 |
| フレームワーク | [go-kratos](https://go-kratos.dev/) | マイクロサービスフレームワーク |
| DI | [Wire](https://github.com/google/wire) | コンパイル時依存性注入 |
| ORM | [Ent](https://entgo.io/) | Go エンティティフレームワーク |
| データベース | [PostgreSQL](https://www.postgresql.org/) | リレーショナルデータベース |
| キャッシュ | [Redis](https://redis.io/) | インメモリデータストア |
| オブジェクトストレージ | [MinIO](https://min.io/) | S3 互換オブジェクトストレージ |
| サービスレジストリ | [Etcd](https://etcd.io/) | サービスディスカバリ & 設定管理 |
| トレーシング | [Jaeger](https://www.jaegertracing.io/) + [OpenTelemetry](https://opentelemetry.io/) | 分散オブザーバビリティ |
| API 定義 | [Protobuf](https://protobuf.dev/) + [buf.build](https://buf.build/) | コントラクトファースト API 設計 |

### 管理コンソールフロントエンド

| 技術 | 説明 |
|:---|:---|
| [Vue 3](https://vuejs.org/) | プログレッシブフロントエンドフレームワーク |
| [TypeScript](https://www.typescriptlang.org/) | 型安全開発 |
| [Ant Design Vue](https://antdv.com/) | エンタープライズ UI コンポーネントライブラリ |
| [Vben Admin](https://doc.vben.pro/) | 管理画面フレームワーク |

### モバイルフロントエンド

| バージョン | 技術スタック | 用途 |
|:---|:---|:---|
| Flutter | [Flutter](https://flutter.dev/) + [BLoC](https://bloclibrary.dev/) | クロスプラットフォームネイティブアプリ |

## プロジェクト構造

```
go-wind-erp/
├── backend/                        # バックエンドサービス
│   ├── api/                        # Protobuf API 定義 & 生成コード
│   │   ├── protos/                 # .proto ソースファイル
│   │   └── gen/                    # 生成コード (Go / Dart / OpenAPI)
│   ├── app/
│   │   ├── admin/service/          # 管理用 BFF (HTTP/gRPC)
│   │   ├── app/service/            # モバイル用 BFF (HTTP/gRPC)
│   │   └── core/service/           # コアサービス (ビジネスロジック + データ層)
│   ├── pkg/                        # 共通パッケージ (認可/暗号化/イベントバス/JWT/ミドルウェア/OSS...)
│   └── scripts/                    # デプロイスクリプト (Docker/環境設定)
├── frontend/
│   ├── admin/                      # 管理コンソールフロントエンド (Vue3 + Ant Design Vue + Vben Admin)
│   └── app/
│       └── flutter_app/            # モバイルアプリ (Flutter クロスプラットフォームネイティブ)
└── LICENSE
```

## クイックスタート

### 前提条件

- Go 1.25+
- Docker & Docker Compose
- Node.js 18+ & pnpm
- Flutter SDK
- buf (Protobuf ツールチェーン)

### 1. 依存サービスの起動

```bash
cd backend

# Windows
.\scripts\docker\libs_only.ps1

# Linux / macOS
./scripts/docker/libs_only.sh
```

### 2. バックエンドの起動

```bash
# 推奨：gow CLI を使用
gow run admin
gow run app
```

### 3. フロントエンドの起動

```bash
# 管理コンソール
cd frontend/admin
pnpm install
pnpm dev

# モバイルアプリ (Flutter)
cd frontend/app/flutter_app
flutter pub get
flutter run
```

### よく使うコマンド

```bash
cd backend

# Protobuf API コードの生成
make api

# OpenAPI ドキュメントの生成
make openapi

# 全コードの生成 (ent + wire + api + openapi)
make gen

# 全サービスのビルド
make build

# テストの実行
make test
```

> 開発ワークフローの詳細は [バックエンドドキュメント](./backend/README.md) と [スクリプトガイド](./backend/scripts/WORKFLOWS_AND_BEST_PRACTICES.md) を参照してください。

## お問い合わせ

- WeChat 個人アカウント：`yang_lin_bo`（備考：`go-wind-erp`）

## コントリビューション

以下のような貢献を歓迎します：

- [Issue](../../issues) の提出：バグ報告や機能提案
- [Pull Request](../../pulls) の提出：修正や新機能の追加
- ドキュメントや翻訳の改善
- 利用経験の共有

## ライセンス

このプロジェクトは [MIT License](./LICENSE) の下で公開されています。

## 謝辞

[![JetBrains](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)

[JetBrains](https://jb.gg/OpenSource) から無料の GoLand & WebStorm オープンソースライセンスを提供していただき、感謝いたします。

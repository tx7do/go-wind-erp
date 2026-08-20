<div align="center">

# GoWind ERP

### 風行 · Golang マイクロサービスに基づくエンタープライズ向け ERP 基盤プラットフォーム

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Flutter](https://img.shields.io/badge/Flutter-3.x-02569B?logo=flutter&logoColor=white)](https://flutter.dev/)
[![Kratos](https://img.shields.io/badge/Kratos-2.9-00ADD8?logo=go&logoColor=white)](https://go-kratos.dev/)

**[English](./README.en-US.md)** · **[中文](./README.md)** · **日本語**

</div>

---

GoWind ERP は、Golang マイクロサービスアーキテクチャに基づくエンタープライズ向け ERP 基盤プラットフォームであり、API ファースト、フロントエンド・バックエンド分離の設計を採用しています。本プラットフォームは、組織と権限、認証、監査、ファイルストレージ、辞書、サイト内メッセージ、タスクスケジューリング、国際化といった汎用基盤機能を提供し、在庫 / WMS、調達 / SRM、財務 / AP などの ERP 業務モジュールを段階的に発展させるための統一基盤となります。

**主な特徴：**

- **API ファースト** — 完全な RESTful / gRPC デュアルプロトコルインターフェース、OpenAPI ドキュメント自動生成
- **マルチプラットフォーム** — 管理コンソール（Vue3）とモバイルアプリ（Flutter）の 2 つのフロントエンド
- **マルチテナント** — テナントデータの分離、部門・ロール・管理者の自動初期化
- **きめ細かい権限管理** — メニュー権限、API 権限、データ権限の 3 レベル制御、ポリシーエンジンによる認可、機微な操作の全リンク監査
- **ネイティブ i18n** — コンテンツ、メニュー、UI テキストの統一翻訳管理
- **マイクロサービス** — go-kratos ベース、サービスディスカバリと分散トレーシング対応

## プロジェクトの状況

GoWind ERP は現在 **基盤構築フェーズ** にあります。組織と権限、認証、監査、ファイルストレージ、辞書、サイト内メッセージ、タスクスケジューリング、国際化といった汎用基盤機能は実装済みであり、管理コンソールとモバイルのエントリポイントも提供されています。ERP 業務モジュール（在庫 / WMS、調達 / SRM、財務 / AP）およびそれに対応するモバイル機能は、現在 planning・開発中であり、**現時点のコードベースには含まれていません**。以下の機能表で「計画中」と記載された項目がこれに該当します。

## テクノロジースタック

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

### ERP 業務モジュール（計画中 — 現リリースでは未実装）

| モジュール | 説明 |
|:---|:---|
| 在庫 / WMS | 倉庫、ロケーション、在庫の原子的操作 & 状態機械（モジュール 1、開発中） |
| 調達 / SRM | サプライヤー & 調達ワークフロー（モジュール 2、計画中） |
| 財務 / AP | 買掛金 & 財務会計（モジュール 3、計画中） |
| モバイル業務機能 | WMS スキャン、承認センター、経営ダッシュボード（計画中、バックエンド整備待ち） |

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

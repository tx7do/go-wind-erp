<div align="center">

# GoWind ERP

### FengXing · A Lightweight, All-in-One ERP Foundation for Small & Medium Teams

**No implementation consultants · Up and running in half a day · Core business data fully connected**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)

**English** · **[中文](./README.md)** · **[日本語](./README.ja-JP.md)**

</div>

---

Procurement, inventory, and finance — the core of a small business — run as one system: approved purchase orders automatically create payables, goods receipts write stock back, low stock triggers replenishment proposals, and AP aging is always one click away. Data is entered once and flows everywhere. No implementation consultants, no months-long rollout.

**GoWind ERP in three sentences:**

- **No implementation consultants** — Sign up and go: tenant, departments, roles, and admin are initialized automatically; permissions, approvals, internal messaging, and audit are all built in
- **Up and running in half a day** — Admin console plus mobile app out of the box; the core flow — procure → approve → receive → stock → pay — works from day one, with warehouse scanning on the floor and an executive dashboard for management
- **Core business data fully connected** — Documents drive documents: an approved PO auto-creates its payable, receipts write stock back, low stock drafts a replenishment PO; data is entered once and takes effect across the whole chain

Developers are well served too: API-first (RESTful / gRPC dual protocol with auto-generated OpenAPI docs), microservices, and native i18n — see the [Tech Stack](#tech-stack) below.

## Project Status

Both the common foundation capabilities of GoWind ERP — organization and permissions, authentication, audit, file storage, dictionaries, internal messaging, task scheduling, internationalization — and the three ERP business modules (Inventory/WMS, Procurement/SRM, Finance/AP) are implemented across the core backend service, the admin console, and the mobile app. The core business loops — PO approval through goods receipt into stock, payable-to-payment settlement, stock transfer and reversal, and low-stock auto-replenishment — are closed end to end, with regression tests covering the key consistency logic (receiving approval gate, single-transaction stock transfer, aging outstanding-balance reporting).

## Core Features

### Organization & Permissions

| Feature | Description |
|:---|:---|
| Multi-tenant Management | Tenant creation, enable/disable & data isolation; auto-init departments, roles & admin |
| User Management | Full lifecycle management, multi-role/dept/position binding |
| Role Management | Role & role group management with menu, API & data permissions |
| Permission Management | Permission groups, menu nodes & permission points, policy-engine-driven authorization, button-level control |
| Menu Management | Visual menu config with directory/page/button levels, dynamic rendering by permission |
| Department & Position Management | Multi-level department trees and position hierarchies with user binding |
| Authentication & Login Policy | JWT issuance & validation, login policy config, credential & token cache management |

### System & Operations

| Feature | Description |
|:---|:---|
| Audit Logs | Full-link audit of login, operation, API call, data access, permission change & policy evaluation — operator, IP, params & results |
| File Management | Unified upload/download to local or object storage, preview & grouping |
| Dictionary Management | Data dictionary categories & items with linked multilingual translation |
| Internal Messaging | Multi-level message categories, targeted delivery & read-status tracking, personal inbox |
| Task Scheduling | Cron job management, start/pause/execute, execution history & logs |
| Multi-language Management | Language management, unified translation for content, menus & UI text |

### ERP Business Modules

| Module | Description |
|:---|:---|
| Inventory / WMS | Warehouse & stock management (state machine), inbound/outbound movements (server-verified write-back), cross-warehouse transfer (single-transaction atomic), movement reversal (idempotent), low-stock auto-replenishment proposals (draft PO via the approval rail), executive dashboard aggregation & 30-day movement trend |
| Procurement / SRM | Supplier management, full PO lifecycle (draft → submit → approve → revise-and-resubmit → receive → auto-complete on full receipt / cancel), per-item goods receipt with approval gate & over-receipt guard |
| Finance / AP | Payables (auto-created on PO approval), payment requests (applied via the approval rail), partial-payment & settlement state machine, AP aging report |
| Approval Rail (cross-module) | Unified approval for POs / payments / replenishments; applicant & approver derived server-side, self-approval blocked; resolution notifies the applicant, with downstream notifications on replenishment-draft creation and PO auto-completion |
| Mobile business features | WMS scanning (in/out / transfer / reversal), approval center, executive dashboard |

## Tech Stack

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Flutter](https://img.shields.io/badge/Flutter-3.x-02569B?logo=flutter&logoColor=white)](https://flutter.dev/)
[![Kratos](https://img.shields.io/badge/Kratos-2.9-00ADD8?logo=go&logoColor=white)](https://go-kratos.dev/)

### Backend

| Layer | Technology | Description |
|:---|:---|:---|
| Language | [Go 1.25+](https://go.dev/) | High-performance compiled language |
| Framework | [go-kratos](https://go-kratos.dev/) | Microservice framework |
| DI | [Wire](https://github.com/google/wire) | Compile-time dependency injection |
| ORM | [Ent](https://entgo.io/) | Go entity framework |
| Database | [PostgreSQL](https://www.postgresql.org/) | Relational database |
| Cache | [Redis](https://redis.io/) | In-memory data store |
| Object Storage | [MinIO](https://min.io/) | S3-compatible object storage |
| Service Registry | [Etcd](https://etcd.io/) | Service discovery & configuration |
| Tracing | [Jaeger](https://www.jaegertracing.io/) + [OpenTelemetry](https://opentelemetry.io/) | Distributed observability |
| API Definition | [Protobuf](https://protobuf.dev/) + [buf.build](https://buf.build/) | Contract-first API design |

### Admin Frontend

| Technology | Description |
|:---|:---|
| [Vue 3](https://vuejs.org/) | Progressive frontend framework |
| [TypeScript](https://www.typescriptlang.org/) | Type-safe development |
| [Ant Design Vue](https://antdv.com/) | Enterprise UI component library |
| [Vben Admin](https://doc.vben.pro/) | Admin management framework |

### Mobile Frontend

| Version | Tech Stack | Use Case |
|:---|:---|:---|
| Flutter | [Flutter](https://flutter.dev/) + [BLoC](https://bloclibrary.dev/) | Cross-platform native app |

## Project Structure

```
go-wind-erp/
├── backend/                        # Backend services
│   ├── api/                        # Protobuf API definitions & generated code
│   │   ├── protos/                 # .proto source files
│   │   └── gen/                    # Generated code (Go / Dart / OpenAPI)
│   ├── app/
│   │   ├── admin/service/          # Admin BFF (HTTP/gRPC)
│   │   ├── app/service/            # Mobile BFF (HTTP/gRPC)
│   │   └── core/service/           # Core service (business logic + data layer)
│   ├── pkg/                        # Shared packages (auth/crypto/eventbus/JWT/middleware/OSS...)
│   └── scripts/                    # Deploy scripts (Docker/env setup)
├── frontend/
│   ├── admin/                      # Admin frontend (Vue3 + Ant Design Vue + Vben Admin)
│   └── app/
│       └── flutter_app/            # Mobile app (Flutter cross-platform native)
└── LICENSE
```

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Node.js 18+ & pnpm
- Flutter SDK
- buf (Protobuf toolchain)

### 1. Start Dependencies

```bash
cd backend

# Windows
.\scripts\docker\libs_only.ps1

# Linux / macOS
./scripts/docker/libs_only.sh
```

### 2. Start Backend

```bash
# Recommended: use gow CLI
gow run admin
gow run app
```

### 3. Start Frontend

```bash
# Admin panel
cd frontend/admin
pnpm install
pnpm dev

# Mobile app (Flutter)
cd frontend/app/flutter_app
flutter pub get
flutter run
```

### Common Commands

```bash
cd backend

# Generate Protobuf API code
make api

# Generate OpenAPI docs
make openapi

# Generate all code (ent + wire + api + openapi)
make gen

# Build all services
make build

# Run tests
make test
```

> For more development workflows, see [Backend Docs](./backend/README.md) and [Scripts Guide](./backend/scripts/WORKFLOWS_AND_BEST_PRACTICES.md).

## Contact

- WeChat: `yang_lin_bo` (Note: `go-wind-erp`)

## Contributing

We welcome all forms of contribution, including but not limited to:

- Submit [Issues](../../issues) to report bugs or suggest features
- Submit [Pull Requests](../../pulls) to fix issues or add features
- Improve documentation and translations
- Share your experience

## License

This project is licensed under the [MIT License](./LICENSE).

## Acknowledgements

[![JetBrains](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)

Thanks to [JetBrains](https://jb.gg/OpenSource) for providing free GoLand & WebStorm open source licenses.

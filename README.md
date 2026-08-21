<div align="center">

# GoWind ERP

### 风行 · 基于 Golang 微服务的企业级 ERP 基座平台

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org/)
[![Flutter](https://img.shields.io/badge/Flutter-3.x-02569B?logo=flutter&logoColor=white)](https://flutter.dev/)
[![Kratos](https://img.shields.io/badge/Kratos-2.9-00ADD8?logo=go&logoColor=white)](https://go-kratos.dev/)

**[English](./README.en-US.md)** · **中文** · **[日本語](./README.ja-JP.md)**

</div>

---

GoWind ERP 是一款基于 Golang 微服务架构的企业级 ERP 基座平台，采用 API 优先、前后端分离架构。平台提供组织权限、认证、审计、文件存储、字典、站内消息、任务调度、多语言等通用基座能力，为库存 / WMS、采购 / SRM、财务 / AP 等 ERP 业务模块的迭代演进提供统一底座。

**核心亮点：**

- **API 优先** — 完整的 RESTful / gRPC 双协议接口，OpenAPI 文档自动生成
- **多端适配** — 管理后台（Vue3）与移动端（Flutter）两套前端
- **多租户架构** — 租户数据隔离，新增租户自动初始化部门、角色与管理员
- **精细化权限** — 菜单权限、接口权限、数据权限三级管控，策略引擎驱动鉴权，关键操作全程审计
- **原生多语言** — 内容、菜单、提示信息统一翻译管理
- **微服务架构** — 基于 go-kratos，支持服务发现与链路追踪

## 项目状态

GoWind ERP 的通用基座能力（组织权限、认证、审计、文件存储、字典、站内消息、任务调度、多语言）与三大 ERP 业务模块（库存 / WMS、采购 / SRM、财务 / AP）均已实现，覆盖后端核心服务、管理后台与移动端。核心业务链路——采购单审批收货入库、应付付款结算、库存调拨与冲正、低库存自动补货——已端到端闭环，关键一致性逻辑（收货审批闸门、调拨单事务原子性、账龄未清余额口径）有回归测试覆盖。

## 技术栈

### 后端

| 层级     | 技术                                                                     | 说明            |
|:-------|:-------------------------------------------------------------------------|:--------------|
| 语言     | [Go 1.25+](https://go.dev/)                                             | 高性能编译型语言    |
| 框架     | [go-kratos](https://go-kratos.dev/)                                     | 微服务框架       |
| 依赖注入   | [Wire](https://github.com/google/wire)                                  | 编译时依赖注入      |
| ORM    | [Ent](https://entgo.io/)                                                | Go 实体框架      |
| 数据库    | [PostgreSQL](https://www.postgresql.org/)                               | 关系型数据库       |
| 缓存     | [Redis](https://redis.io/)                                              | 内存数据库       |
| 对象存储   | [MinIO](https://min.io/)                                                | 兼容 S3 的对象存储 |
| 服务注册   | [Etcd](https://etcd.io/)                                                | 服务发现与配置      |
| 链路追踪   | [Jaeger](https://www.jaegertracing.io/) + [OpenTelemetry](https://opentelemetry.io/) | 分布式可观测       |
| API 定义 | [Protobuf](https://protobuf.dev/) + [buf.build](https://buf.build/)     | 接口契约优先       |

### 管理后台前端

| 技术                                            | 说明         |
|:----------------------------------------------|:-----------|
| [Vue 3](https://vuejs.org/)                   | 渐进式前端框架    |
| [TypeScript](https://www.typescriptlang.org/) | 类型安全       |
| [Ant Design Vue](https://antdv.com/)          | 企业级 UI 组件库 |
| [Vben Admin](https://doc.vben.pro/)           | 后台管理框架     |

### 移动端前端

| 版本      | 技术栈                                 | 适用场景       |
|:--------|:-------------------------------------|:-----------|
| Flutter | [Flutter](https://flutter.dev/) + [BLoC](https://bloclibrary.dev/) | 跨平台原生应用 |

## 核心功能

### 组织与权限

| 功能        | 说明                                                |
|:----------|:--------------------------------------------------|
| 多租户管理     | 租户新增、启用/禁用与数据隔离；新租户自动初始化部门、角色与管理员 |
| 用户管理      | 用户全生命周期管理，支持多角色、多部门、多岗位绑定         |
| 角色管理      | 角色与角色分组管理，配置菜单、接口与数据权限           |
| 权限管理      | 权限分组、菜单节点与权限点，策略引擎驱动鉴权，按钮级控制 |
| 菜单管理      | 可视化菜单配置，目录/页面/按钮三级，按权限动态渲染      |
| 部门与岗位管理   | 多级部门树与岗位体系，联动绑定用户                |
| 认证与登录策略   | JWT 签发与校验、登录策略配置、凭据与令牌缓存管理      |

### 系统与运维

| 功能      | 说明                                          |
|:--------|:--------------------------------------------|
| 审计日志    | 登录、操作、接口调用、数据访问、权限变更、策略评估全链路审计，记录操作人、IP、参数与结果 |
| 文件资源管理  | 本地或对象存储统一上传/下载，预览与分组管理            |
| 字典管理    | 数据字典大类与子项管理，多语言翻译联动               |
| 站内消息    | 多级消息分类，定向投递与已读状态追踪，个人消息中心      |
| 任务调度    | 定时任务管理，启动/暂停/立即执行，查看执行记录与日志    |
| 多语言管理   | 语种管理，内容、菜单、提示信息统一翻译            |

### ERP 业务模块

| 模块          | 说明                              |
|:------------|:----------------------------------|
| 库存 / WMS    | 仓库与库存管理（状态机）、出入库流水（服务端校验回写）、跨仓调拨（单事务原子执行）、流水冲正（幂等防重复）、低库存自动补货建议（经审批轨生成草稿采购单）、经营看板聚合与 30 天流水趋势 |
| 采购 / SRM    | 供应商管理、采购单全生命周期（草稿→提交→审批→驳回重提→收货→全额自动完结/取消）、按明细收货回写（审批闸门 + 防超收） |
| 财务 / AP     | 应付单（采购单获批自动生成）、付款申请（经审批轨入账）、部分付款与结清状态机、应付账龄报告 |
| 审批轨（通用）    | 采购单 / 付款 / 补货三类单据统一审批；申请人与审批人由服务端推导，禁止自审自批，审结站内信通知申请人，补货草稿创建与采购单自动完结触发下游通知 |
| 移动端业务功能    | WMS 扫码（出入库 / 调拨 / 冲正）、审批中心、经营看板 |

## 项目结构

```
go-wind-erp/
├── backend/                        # 后端服务
│   ├── api/                        # Protobuf API 定义 & 生成代码
│   │   ├── protos/                 # .proto 源文件
│   │   └── gen/                    # 生成代码 (Go / Dart / OpenAPI)
│   ├── app/
│   │   ├── admin/service/          # 管理后台 BFF (HTTP/gRPC)
│   │   ├── app/service/            # 移动端 BFF (HTTP/gRPC)
│   │   └── core/service/           # 核心服务 (业务逻辑 + 数据层)
│   ├── pkg/                        # 公共包 (鉴权/加密/事件总线/JWT/中间件/OSS...)
│   └── scripts/                    # 部署脚本 (Docker/环境安装)
├── frontend/
│   ├── admin/                      # 管理后台前端 (Vue3 + Ant Design Vue + Vben Admin)
│   └── app/
│       └── flutter_app/            # 移动端应用 (Flutter 跨平台原生)
└── LICENSE
```

## 快速开始

### 环境要求

- Go 1.25+
- Docker & Docker Compose
- Node.js 18+ & pnpm
- Flutter SDK
- buf (Protobuf 工具链)

### 1. 启动依赖服务

```bash
cd backend

# Windows
.\scripts\docker\libs_only.ps1

# Linux / macOS
./scripts/docker/libs_only.sh
```

### 2. 启动后端服务

```bash
# 推荐方式：使用 gow CLI
gow run admin
gow run app
```

### 3. 启动前端

```bash
# 管理后台
cd frontend/admin
pnpm install
pnpm dev

# 移动端 (Flutter)
cd frontend/app/flutter_app
flutter pub get
flutter run
```

### 常用命令

```bash
cd backend

# 生成 Protobuf API 代码
make api

# 生成 OpenAPI 文档
make openapi

# 一键生成全部代码 (ent + wire + api + openapi)
make gen

# 构建所有服务
make build

# 运行测试
make test
```

> 更多开发工作流请参考 [后端文档](./backend/README.md) 和 [脚本指南](./backend/scripts/WORKFLOWS_AND_BEST_PRACTICES.md)。

## 联系我们

- 微信个人号：`yang_lin_bo`（备注：`go-wind-erp`）

## 参与贡献

我们欢迎各种形式的贡献，包括但不限于：

- 提交 [Issue](../../issues) 报告 Bug 或提出功能建议
- 提交 [Pull Request](../../pulls) 修复问题或添加新功能
- 完善文档和翻译
- 分享使用经验

## 开源许可

本项目基于 [MIT License](./LICENSE) 开源。

## 致谢

[![JetBrains](https://resources.jetbrains.com/storage/products/company/brand/logos/jb_beam.svg)](https://jb.gg/OpenSource)

感谢 [JetBrains](https://jb.gg/OpenSource) 提供的免费 GoLand & WebStorm 开源许可。

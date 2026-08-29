# GoWind ERP 部署运维指南

> 本指南面向负责把 GoWind ERP 部署到生产环境的运维/后端人员，覆盖：依赖服务、后端微服务、管理后台前端、移动端发布的完整部署路径，以及配置改造、安全清单、备份恢复、升级与已知问题。
>
> 开发环境搭建见根 [README](../README.md)「快速开始」；脚本细节见 [backend/scripts/README.md](../backend/scripts/README.md)。基于 2026-08-29 的 main 分支编写。

---

## 目录

- [1. 架构与端口总览](#1-架构与端口总览)
- [2. 两种部署形态怎么选](#2-两种部署形态怎么选)
- [3. 环境准备](#3-环境准备)
- [4. 推荐路径：Docker 跑依赖 + PM2 跑服务](#4-推荐路径docker-跑依赖--pm2-跑服务)
- [5. 生产配置改造清单（必读）](#5-生产配置改造清单必读)
- [6. 数据库初始化与种子数据](#6-数据库初始化与种子数据)
- [7. 管理后台前端构建与发布](#7-管理后台前端构建与发布)
- [8. 移动端 App 发布](#8-移动端-app-发布)
- [9. 反向代理与 HTTPS](#9-反向代理与-https)
- [10. 备份与恢复](#10-备份与恢复)
- [11. 升级](#11-升级)
- [12. 日常运维与可观测性](#12-日常运维与可观测性)
- [13. 安全清单](#13-安全清单)
- [14. 已知问题与坑](#14-已知问题与坑)

---

## 1. 架构与端口总览

三个后端服务（Go / Kratos 微服务）+ 五个依赖：

| 组件 | 说明 | 监听 |
|:---|:---|:---|
| admin-service | 管理后台 BFF（Web 前端的唯一后端入口） | REST `6600`，SSE `6601`（`/events`），gRPC 随机端口 |
| app-service | 移动端 BFF（Flutter App 的唯一后端入口） | REST `6700`，SSE `6701`（`/events`），gRPC 随机端口 |
| core-service | 核心业务服务，**唯一连数据库的服务**，内嵌 Asynq 异步任务 | 仅 gRPC（随机端口，经 etcd 注册被发现） |
| PostgreSQL | 主数据库（库名 `go_wind_erp`） | `5432` |
| Redis | 缓存 + Asynq 任务队列（core 使用 DB 1） | `6379` |
| MinIO | S3 兼容对象存储（文件上传），默认桶 `images` | API `9000`，控制台 `9001` |
| etcd | 服务注册与发现 | `2379`（客户端）/ `2380`（peer） |
| Jaeger | 链路追踪（OTLP 收集 + UI） | UI `16686`，OTLP `4317`(gRPC) / `4318`(HTTP) |

关键调用关系：admin/app 两个 BFF **不直接连数据库**，通过 etcd 发现 core-service 转发业务请求；所有业务规则（状态机、防超收、审批闸门）都在 core 强制执行。

服务间发现名：`gowind-erp/<admin-service|app-service|core-service>`，gRPC 目标写作 `discovery:///<service>`。

## 2. 两种部署形态怎么选

| 形态 | 说明 | 适用 |
|:---|:---|:---|
| **A. 依赖进 Docker + 服务跑宿主机（PM2）** | Postgres/Redis/MinIO/etcd/Jaeger 用 docker compose 起；三个 Go 服务编译成二进制，用 PM2 守护 | **推荐**。与仓库现有脚本（`full_deploy.sh` + `pm2_service.sh`）匹配，配置只需按[第 5 章](#5-生产配置改造清单必读)微调 |
| B. 全容器化（应用 + 依赖都进 Docker） | 仓库有 `backend/docker-compose.yaml` 和统一 Dockerfile，但**当前存在若干缺口**（应用端口错配、缺 app-service、数据卷未挂载、部分配置地址写的 localhost），直接 `up` 不能用 | 暂不推荐；要用需先按[第 14 章](#14-已知问题与坑)清单改造 |

下文主推形态 A；形态 B 的改造要点在[第 14 章](#14-已知问题与坑)。

## 3. 环境准备

生产服务器（Linux x86_64）一键安装依赖工具：

```bash
cd backend
./scripts/env/install_unix_prod.sh
```

自动安装：git / wget / jq、Node 22 LTS、PM2、Docker + Compose 插件、Go。支持 Ubuntu/Debian/CentOS/RHEL/Rocky/Alma/Fedora/macOS。

版本要求速查：

| 组件 | 要求 |
|:---|:---|
| Go | 以 `backend/go.mod` 声明为准（当前 1.26+；`go` 会按 GOTOOLCHAIN 自动下载所需版本） |
| Node.js | 22 LTS（前端构建） |
| pnpm | 前端 monorepo 使用（`corepack enable` 或 npm i -g pnpm） |
| Docker / Compose | 任意近期版本 |
| Flutter SDK | 仅移动端发布时需要 |

## 4. 推荐路径：Docker 跑依赖 + PM2 跑服务

### 4.1 启动依赖服务

```bash
cd backend

# Linux / macOS
./scripts/docker/full_deploy.sh

# Windows PowerShell
.\scripts\docker\full_deploy.ps1
```

脚本默认使用 `backend/docker-compose.yaml`（含被注释掉的应用段，实际只起 5 个依赖），并创建数据目录。可用环境变量定制：

| 变量 | 默认 | 说明 |
|:---|:---|:---|
| `APP_ROOT`（sh）/ `-AppRoot`（ps1） | `/root/app` / `C:\app` | 依赖数据根目录 |
| `COMPOSE_FILE` / `-ComposeFile` | 自动探测 | 指定 compose 文件 |
| `POSTGRES_PASSWORD` | `dev_only_change_me` | **生产必须设置** |
| `REDIS_PASSWORD` | `dev_only_change_me` | **生产必须设置** |
| `MINIO_ROOT_PASSWORD` | `dev_only_change_me` | **生产必须设置** |

> ⚠️ 不要用 `make compose-up-libs`——Makefile 里该目标有反引号 bug（见[第 14 章](#14-已知问题与坑)），起依赖请用上面的脚本。

验证依赖就绪：

```bash
docker ps   # 应看到 postgres / redis / minio / etcd / jaeger 五个容器
```

### 4.2 构建后端并安装为 PM2 服务

```bash
cd backend
make pm2-deploy
```

该目标等价于执行 `scripts/deploy/pm2_service.sh`，它会：

1. 加载 `backend/.env`（`PROJECT_NAME=go_wind_erp`、`SERVICE_APP_VERSION=1.0.0`）；
2. 执行 `make build_only` 编译三个服务，产物在各服务 `bin/server`；
3. 安装到 `~/app/go_wind_erp/<admin|app|core>/service/`（二进制 + configs 目录一并复制）；
4. 用 PM2 启动并守护：`pm2 start ... --name go_wind_erp-<服务名> -- -c <configs 目录>`，随后 `pm2 save`。

验证服务：

```bash
pm2 status                 # 三个服务 online
curl http://127.0.0.1:6600/   # admin 端口有响应即可（无专用健康端点）
curl http://127.0.0.1:6700/
```

> 首次启动 core-service 时会**自动建表**（ent 自动迁移，受 `configs/data.yaml` 的 `migrate: true` 控制）并写入内置角色/权限模板；users 表为空时自动创建默认管理员 `admin / admin`——**登录后立刻改密**（见[第 13 章](#13-安全清单)）。

### 4.3 开机自启

```bash
pm2 startup   # 按输出提示执行生成的命令
pm2 save
```

## 5. 生产配置改造清单（必读）

三个服务的配置在各服务 `backend/app/<服务名>/service/configs/` 目录下的 yaml 文件中（`-c` 参数指向该目录）。PM2 安装后位于 `~/app/go_wind_erp/<服务名>/service/configs/`，**可直接改服务器上的副本**（改完 `pm2 restart` 生效）。

**先理解一条机制**：配置文件中的 `${key:默认值}` 占位符由 Kratos 从**配置文件树内部**解析——在配置树里任何 yaml 顶层写 `key: 值` 即可覆盖占位符；**设置同名操作系统环境变量无效**（配置文件里的注释有误导性，以此为准）。

### 5.1 数据库与缓存（core-service）

`core/service/configs/data.yaml`：

```yaml
database:
  driver: "postgres"
  # 宿主机运行时依赖端口已由 compose 发布到 127.0.0.1，host 不要写容器名
  source: "host=127.0.0.1 port=5432 user=postgres password=<你的PG密码> dbname=go_wind_erp sslmode=disable"
  migrate: true        # 首次启动建表；稳定运行后可改为 false
  max_open_conn: 25
  redis:
    addr: "127.0.0.1:6379"
    password: "<你的Redis密码>"
```

`core/service/configs/server.yaml` 中的 `asynq.uri` 同步改：

```yaml
asynq:
  uri: "redis://:<你的Redis密码>@127.0.0.1:6379/1"
```

admin/app 两个服务的 `data.yaml` 里各有一段 redis 配置，同样改地址与密码。

### 5.2 对象存储（三个服务都有 oss.yaml）

```yaml
oss:
  endpoint: "<MinIO地址>:9000"        # 服务端上传用
  upload_host: "<公网可达的MinIO地址>:9000"   # 前端/App 直传地址
  download_host: "https://<公网可达的MinIO地址>"  # 浏览器下载文件用，必须公网可达
  access_key: "root"
  secret_key: "<与 MINIO_ROOT_PASSWORD 一致>"
  use_ssl: false        # 走 HTTPS 反代时改 true
```

> `upload_host` / `download_host` 是给**浏览器和手机 App**用的，写 127.0.0.1 会导致文件上传下载在前端不可用；建议用反向代理把 MinIO 暴露为 `https://minio.你的域名`（见[第 9 章](#9-反向代理与-https)）。

### 5.3 服务注册与追踪

`registry.yaml`（三个服务）：宿主机运行保持 `endpoints: ["localhost:2379"]` 即可（etcd 端口已发布）。

`trace.yaml`（三个服务）：生产建议把采样率从全量调低，并把环境标识改为 prod：

```yaml
trace:
  endpoint: "localhost:4317"
  exporter: "otlp-grpc"
  sampler: 0.1        # 生产建议 0.1，开发为 1.0 全量
  env: "prod"
```

### 5.4 JWT 签名密钥（三个服务必须一致）

JWT 密钥的占位符是 `${jwt_signing_key:dev_only_change_me_in_prod}`，出现在：

- `admin/service/configs/server.yaml`（rest 中间件）、`admin/service/configs/client.yaml`
- `app/service/configs/server.yaml`、`app/service/configs/client.yaml`
- `core/service/configs/authenticator.yaml`

**改法**：在上述每个 configs 目录里，任选一个 yaml 在顶层加一行（或直接替换占位符默认值）：

```yaml
jwt_signing_key: "<48位以上随机字符串>"
```

生成密钥：`openssl rand -base64 48`。三处不一致会导致 token 校验失败、登录后接口 401。

### 5.5 CORS（admin-service）

`admin/service/configs/server.yaml` 的 CORS 白名单默认只有 `http://localhost:5666/5667`。生产前端域名必须加进去，否则浏览器全被拦：

```yaml
server:
  rest:
    middleware:
      cors:
        origins:
          - "https://erp.你的域名"
```

app-service 的 CORS 默认是 `*`，公网部署建议同样收敛为移动端实际使用的地址。

### 5.6 密码类默认值对齐

仓库里 compose 默认密码（`dev_only_change_me`）与各配置文件写死的默认（`*Abcd123456`）**不一致**。要么改 compose 环境变量，要么改配置文件，最终保证四处一致：PostgreSQL（compose ↔ core `data.yaml`）、Redis（compose ↔ 各 `data.yaml` ↔ core `asynq.uri`）、MinIO（compose `MINIO_ROOT_PASSWORD` ↔ 各 `oss.yaml`）。

### 5.7 日志（可选）

`logger.yaml` 默认双写：stdout + 滚动文件 `./logs/info.log`（1MB×5 个，保留 30 天，相对进程工作目录）。PM2 模式另有 `~/app/go_wind_erp/<服务名>/service/bin/stdout.log|stderr.log`。

## 6. 数据库初始化与种子数据

**全新生产库不需要导入任何 SQL**：

1. 建空库：compose 已自动建 `go_wind_erp`；自建库时用 `CREATE DATABASE go_wind_erp;`
2. 启动 core-service：ent 自动迁移建全部表（`migrate: true`）
3. 启动逻辑自动写入：角色/权限模板、内置权限点，以及默认管理员 **admin / admin**

**演示数据（可选）**：`backend/sql/postgresql-demo-data.sql` 内含演示租户 `tenant_admin`（密码同 admin）、商品/库存/采购/财务/审批样例数据，适合给客户做演示环境。

> ⚠️ 该文件开头会对 20 张业务表执行 `TRUNCATE ... RESTART IDENTITY CASCADE`——**只能在空演示库导入，绝不能在已有真实数据的库上执行**。`mysql-demo-data.sql` 仅用于 MySQL 8.0+ 的特殊场景，且与 PG 版内容并非逐字段同步（以 PG 版为准）。

## 7. 管理后台前端构建与发布

前端 API 地址在**构建时**注入，改地址必须重新构建（运行时不可配）：

```bash
cd frontend/admin

# 1. 按你的域名修改 apps/admin/.env.production
#    VITE_GLOB_API_URL=https://api.你的域名        → admin-service :6600
#    VITE_GLOB_SSE_URL=https://api.你的域名/events → admin-service :6601
#    （也可拆成两个域名，分别指向 6600 与 6601）

# 2. 安装与构建
pnpm install --frozen-lockfile
pnpm build:antd        # 产物在 apps/admin/dist/（另生成 dist.zip）

# 3. 把 dist/ 发布到任意静态服务器（nginx 示例见第 9 章）
```

注意：

- 路由默认 hash 模式（`VITE_ROUTER_HISTORY=hash`），静态服务器**无需** SPA fallback 也能用；若改 history 模式则必须配置 fallback 到 `index.html`；
- `frontend/admin/scripts/deploy/Dockerfile` 是 Vben 上游遗留模板，其中 COPY 的是 `playground/dist` 而非本项目产物，直接使用会部署错内容——要么把路径改为 `apps/admin/dist`，要么用自己的静态服务器方案。

## 8. 移动端 App 发布

App 的服务器地址同样在构建前写死在环境文件里：

```bash
cd frontend/app/flutter_app

# 发布版：修改 .env
#   API_BASE_URL=https://app-api.你的域名     → app-service :6700
#   SSE_URL=https://app-api.你的域名/events   → app-service :6701
#   （SSE 与 API 同域不同路径即可，用反代区分）

flutter pub get
flutter build apk --release    # 或 appbundle / ipa
```

`AES_KEY` 等密钥字段保持默认即可（见[第 14 章](#14-已知问题与坑)第 3 条——该密钥后端侧硬编码于上游库，**不要单方面修改**，否则登录解密失败）。

## 9. 反向代理与 HTTPS

推荐统一收口到一个 nginx（或 Caddy），示意（按你的域名调整）：

```nginx
# 管理后台静态资源
server {
    listen 443 ssl;
    server_name erp.你的域名;
    root /var/www/gowind-admin;        # apps/admin/dist 产物
    index index.html;

    location / { try_files $uri $uri/ /index.html; }   # history 模式必需；hash 模式可省
}

# 管理后台 API（前端构建时的 VITE_GLOB_API_URL）
server {
    listen 443 ssl;
    server_name api.你的域名;

    location / {
        proxy_pass http://127.0.0.1:6600;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    location /events {                 # SSE（VITE_GLOB_SSE_URL）
        proxy_pass http://127.0.0.1:6601/events;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_read_timeout 3600s;      # SSE 长连接，放宽读超时
        proxy_buffering off;
    }
}

# 移动端 API（Flutter .env 的 API_BASE_URL）
server {
    listen 443 ssl;
    server_name app-api.你的域名;

    location / {
        proxy_pass http://127.0.0.1:6700;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    location /events {
        proxy_pass http://127.0.0.1:6701/events;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_read_timeout 3600s;
        proxy_buffering off;
    }
}

# MinIO（对应 oss.yaml 的 upload_host / download_host）
server {
    listen 443 ssl;
    server_name minio.你的域名;

    client_max_body_size 100m;         # 按最大上传文件调整
    location / {
        proxy_pass http://127.0.0.1:9000;
        proxy_set_header Host $host;   # MinIO S3 签名依赖原始 Host
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

开启 HTTPS 后记得同步：

1. 前端 `.env.production` 的两个 URL 改为 `https://` 并**重新构建**；
2. Flutter `.env` 改为 `https://` 并重新打包；
3. admin `server.yaml` 的 CORS origins 改为 `https://erp.你的域名`；
4. oss.yaml 的 `use_ssl: true`（若 MinIO 走 TLS 反代）。

## 10. 备份与恢复

需要备份的数据：PostgreSQL（全部业务数据）+ MinIO 数据目录（上传的文件）。Redis 是缓存/队列可不备，etcd 是服务注册状态可重建。

**PostgreSQL 备份**（每日 cron 建议）：

```bash
# 备份（容器名以 docker ps 实际为准）
docker exec <postgres容器> pg_dump -U postgres go_wind_erp | gzip > /backup/gowind_$(date +%F).sql.gz

# 恢复
gunzip -c /backup/gowind_2026-08-29.sql.gz | docker exec -i <postgres容器> psql -U postgres go_wind_erp
```

**MinIO 备份**：直接备份数据目录（compose 未挂载卷时在容器内 `/data`，生产应按[第 14 章](#14-已知问题与坑)第 2 条补挂载后备份宿主机目录），或用 `mc mirror` 同步到远端。

> ⚠️ 形态 A 下若未给依赖容器补挂载卷，**容器重建即丢数据**——部署完成后第一件事就是把[第 14 章](#14-已知问题与坑)第 2 条的卷挂载加上。

## 11. 升级

```bash
# 0. 备份数据库（见第 10 章）
# 1. 拉取新代码
cd /path/to/go-wind-erp && git pull

# 2. 重建前端（如前端有变更）并发布 dist/
cd frontend/admin && pnpm install --frozen-lockfile && pnpm build:antd

# 3. 重建并重装后端服务（自动 build + 拷贝二进制与配置 + 重启）
cd ../../backend && make pm2-deploy

# 4. 验证
pm2 status
curl -s http://127.0.0.1:6600/ >/dev/null && echo admin-ok
curl -s http://127.0.0.1:6700/ >/dev/null && echo app-ok
```

说明：

- core 启动时 ent 自动迁移会平滑加表加列（`migrate: true` 时）；**升级前务必备份**，跨大版本的破坏性变更以发布说明为准；
- `pm2_service.sh` 会重新复制 configs 目录——**在服务器 configs 里手工改过的配置会被仓库版本覆盖**，升级前先备份服务器上的 configs，重装后 diff 合并再重启；
- 回滚 = 检出上一个 tag/commit 重新执行第 3 步 + 恢复数据库备份。

## 12. 日常运维与可观测性

| 事项 | 现状与做法 |
|:---|:---|
| 进程守护 | PM2 自动重启；`pm2 status` / `pm2 logs go_wind_erp-admin` |
| 健康检查 | **目前没有专用健康端点**（无 /healthz），用端口探测（6600/6700）或 `pm2 status` 判断存活；接入监控建议用 TCP 探测 |
| 指标 | 未暴露 Prometheus 端点 |
| 链路追踪 | Jaeger UI `http://<服务器>:16686`（生产建议不对公网开放）；采样率见 5.3 |
| 业务日志 | 各服务 `./logs/info.log`（滚动 1MB×5，30 天）+ PM2 stdout/stderr.log |
| Swagger | 默认关闭；需要时把对应服务 `server.yaml` 的 `enable_swagger` 改 `true`，访问 `http://<host>:6600/docs/`，**用完关闭**，勿对公网开放 |
| 定时任务 | core 内嵌 Asynq worker（Redis DB 1），任务定义见 `sys_tasks` 表 |
| Redis 运维 | 依赖容器设置了 `REDIS_DISABLE_COMMANDS=FLUSHDB,FLUSHALL,CONFIG`，运维脚本中的 FLUSH 操作会被拒绝（防误删，属预期） |

## 13. 安全清单

上线前逐项确认：

- [ ] **默认管理员改密**：首次登录 `admin / admin` 后立即在「个人设置 → 修改密码」改掉（该账号只在 users 表为空时创建，但默认密码是公开知识）；
- [ ] **JWT 签名密钥**：按 5.4 替换三处服务的 `jwt_signing_key`，不用默认值；
- [ ] **三个依赖密码**：`POSTGRES_PASSWORD` / `REDIS_PASSWORD` / `MINIO_ROOT_PASSWORD` 全部改强密码，并与配置文件对齐（5.6）；
- [ ] **CORS 收敛**：admin 的 origins 改为实际前端域名（5.5）；app 的 `*` 同样收敛；
- [ ] **Jaeger / MinIO 控制台（9001）/ Swagger 不对公网开放**：用防火墙或反代 ACL 限制；
- [ ] **数据库不对公网开放**：compose 的 5432 等端口如无远程管理需求，建议改为 `127.0.0.1:5432:5432`；
- [ ] **租户自助注册开关**：注册接口是公开端点，面向企业内/受邀场景部署时应通过反代封禁注册路由；
- [ ] HTTPS 全站启用（第 9 章）。

> 关于登录密码传输 AES 密钥（前端/Flutter/后端三处 `f51d66a73d8a0927`）：该密钥在后端侧硬编码于上游工具库，**当前无法通过纯配置轮换**（详见第 14 章第 3 条）。它只覆盖传输层混淆，安全性依赖全站 HTTPS，请务必完成上一条。

## 14. 已知问题与坑

以下为当前代码里实际存在、部署时会踩到的问题（形态 B 全容器化前必须逐条处理）：

1. **`make compose-up-libs` 不可用**：`backend/Makefile` 中该目标把 compose 文件名写成了反引号命令替换（`` -f `docker-compose.libs.yaml` ``）会执行失败。起依赖用 `scripts/docker/libs_only.sh`（开发）或 `full_deploy.sh`（生产）。
2. **compose 数据卷未挂载**：`docker-compose.yaml` 里所有依赖的 `volumes:` 段都被注释，脚本创建的 `$APP_ROOT/*` 数据目录没有被使用——容器重建即丢数据。生产必须恢复挂载，例如：
   ```yaml
   postgres:
     volumes:
       - ${APP_ROOT:-/root/app}/postgres/data:/bitnami/postgresql
   redis:
     volumes:
       - ${APP_ROOT:-/root/app}/redis/data:/bitnami/redis/data
   minio:
     volumes:
       - ${APP_ROOT:-/root/app}/minio/data:/data
   etcd:
     volumes:
       - ${APP_ROOT:-/root/app}/etcd/data:/bitnami/etcd/data
   ```
   并顺手把不需要对外的端口改为 `127.0.0.1:` 前缀、把 `latest` 镜像固定到具体版本。
3. **登录 AES 密钥不可配置轮换**：前端 `.env`、Flutter `.env`、core `authenticator.yaml` 三处的 `f51d66a73d8a0927` 必须一致，且后端解密默认值硬编码在上游库 `go-utils/crypto` 中——只改前端或配置会导致全部登录失败。轮换需同步修改上游库或提交补丁。
4. **compose 应用段不可直接用**（仅形态 B 相关）：`docker-compose.yaml` 的 `admin-service` 端口映射是 `9700:9700/9701:9701`，与镜像内配置监听的 `6600/6601` 不匹配；`app-service` 整段缺失；`core-service` 无端口（正常，走注册发现）。且 admin/app 的 `registry.yaml`（`localhost:2379`）、`oss.yaml`（`127.0.0.1:9000`）、`trace.yaml`（`localhost:4317`）在容器网络内不可达，需改为 compose 服务名；etcd 的 `ETCD_ADVERTISE_CLIENT_URLS=http://127.0.0.1:2379` 也会导致跨容器服务发现失效，需改为容器网卡地址。
5. **`${jwt_signing_key:...}` 占位符不读系统环境变量**：Kratos 配置解析只查配置文件树（5.4 的改法），yaml 内注释如暗示可用环境变量覆盖，勿信。
6. **依赖镜像未锁版本**：除 etcd（v3.6.8）外均为 `latest`，升级 compose 依赖时可能引入不兼容变更，生产建议固定 tag。
7. **Dockerfile 的 Go 构建版本**：`backend/Dockerfile` 声明 `GO_VERSION=1.25.3` 而 `go.mod` 要求更高版本，镜像构建依赖 GOTOOLCHAIN 自动下载工具链（已配置 GOPROXY），离线环境构建会失败，需把 `GO_VERSION` 与 `go.mod` 对齐。

发现新问题请先查 [backend/scripts/WORKFLOWS_AND_BEST_PRACTICES.md](../backend/scripts/WORKFLOWS_AND_BEST_PRACTICES.md) 与 [backend/AGENTS.md](../backend/AGENTS.md)，仍未解决可提 Issue。

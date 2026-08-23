# API 层开发指南

本文档面向二开人员，介绍 GoWind ERP Flutter 端的 API 架构设计、代码生成流程、仓储层编写规范和分页查询机制。

---

## 1. 整体架构

API 层遵循 **生成层 → 仓储层 → 调用层** 的三层分离，配合 clean architecture 的 data / domain / presentation 划分：

```
┌─────────────────────────────────────────────────────────────┐
│  Presentation 层（Page + Cubit）                            │
│  Cubit 调用抽象 Repository 接口，持有 sealed State          │
├─────────────────────────────────────────────────────────────┤
│  Domain 层（features/<module>/domain/）                     │
│  抽象 Repository 接口 + sealed Failure + 值对象模型         │
├─────────────────────────────────────────────────────────────┤
│  Data 层（features/<module>/data/）                         │
│  RepositoryImpl（调 RemoteDataSource、DTO→domain 映射、     │
│  DioException→Failure 映射）+ RemoteDataSource              │
│  （持 *ServiceClient、调生成方法）                          │
├─────────────────────────────────────────────────────────────┤
│  生成层（lib/generated/api/app/service/v1/index.dart）      │
│  ApiClient（聚合 10 个 *ServiceClient 懒加载 getter）        │
├─────────────────────────────────────────────────────────────┤
│  传输层（lib/src/core/transport/）                          │
│  DioClientTransport 桥接 Dio（+ UnifiedAuthInterceptor）   │
└─────────────────────────────────────────────────────────────┘
```

**调用链路：** `Page → Cubit → 抽象 Repository → RepositoryImpl → RemoteDataSource → ApiClient.*ServiceClient → DioClientTransport → Dio → HTTP`

关键约束：Cubit **只依赖** domain 层的抽象 Repository 接口，不直接接触 `*ServiceClient` 或 `ApiClient`；`*ServiceClient` 仅由 data 层的 `RemoteDataSource` 持有。

---

## 2. 代码生成流程

后端 gRPC `.proto` 定义经 **`protoc-gen-dart-http`** 生成 Dart HTTP 客户端代码。生成在**后端侧**完成（`make api`），移动端通过重新拉取生成的 `lib/generated/api/` 同步。

### 2.1 生成产物

```
lib/generated/api/
├── transport.dart                          # ClientTransport 接口（生成）
└── app/service/v1/
    └── index.dart                          # ApiClient + 10 个 *ServiceClient + 全部请求/响应模型
```

`index.dart` 内含两类产物：

- **`ApiClient`**（`index.dart` 顶部）：聚合类，构造时接收一个 `ClientTransport`，以**懒加载 getter** 形式暴露 10 个 `*ServiceClient`（首次访问才实例化，缓存于私有字段）。
- **`*ServiceClient`** 与**模型类**：每个服务对应一个 client（如 `StockPickingServiceClient`），其方法接收强类型的请求模型、返回强类型的响应模型。模型类名带 proto 包前缀（如 `InventoryServiceV1StockPicking`、`InventoryServiceV1StockQuant`、`InventoryServiceV1CreateStockPickingRequest`）。

### 2.2 `ApiClient` 的 10 个 ServiceClient

| getter | ServiceClient 类 | feature 消费情况 |
|---|---|---|
| `authenticationService` | `AuthenticationServiceClient` | ✅ auth |
| `approvalRequestService` | `ApprovalRequestServiceClient` | ✅ approval |
| `stockQuantService` | `StockQuantServiceClient` | ✅ dashboard、wms |
| `stockPickingService` | `StockPickingServiceClient` | ✅ wms |
| `warehouseService` | `WarehouseServiceClient` | ✅ wms |
| `fileTransferService` | `FileTransferServiceClient` | ❌ 已生成，未消费 |
| `payableService` | `PayableServiceClient` | ❌ 已生成，未消费 |
| `paymentService` | `PaymentServiceClient` | ❌ 已生成，未消费 |
| `purchaseOrderService` | `PurchaseOrderServiceClient` | ❌ 已生成，未消费 |
| `userProfileService` | `UserProfileServiceClient` | ❌ 已生成，未消费 |

> 未消费的 5 个 client 是为未来模块预生成的脚手架；接入时无需重新生成，直接在 data 层适配即可。

### 2.3 重要原则

- **禁止手动编辑** `lib/generated/api/` 下的任何文件
- 修改后端 proto 后，后端侧 `make api` 重新生成，移动端拉取新的 `lib/generated/api/`
- 生成代码不经 `build_runner`、不经 `swagger_parser`、不经 `retrofit_generator`（pubspec 虽列这些依赖但实际未用）

---

## 3. 传输层

### 3.1 Dio 初始化

文件：`lib/src/core/transport/http/http_client.dart`

```dart
Dio createDio() {
  final dio = Dio();
  dio.options.baseUrl = Environments.apiBaseUrl;     // 从 .env 读取
  dio.options.connectTimeout = Environments.connectionTimeout;
  dio.options.receiveTimeout = Environments.receiveTimeout;
  dio.options.responseType = ResponseType.json;
  // ...
  return dio;
}
```

Dio 实例通过 GetIt 注册为全局单例：

```dart
// lib/src/core/transport/init.dart
getIt.registerLazySingleton<Dio>(() => createDio());
```

**单一拦截器**：Dio 仅安装 `UnifiedAuthInterceptor`（见 §6），无日志拦截器、无 locale 拦截器。

### 3.2 环境配置

文件：`lib/src/core/config/environments.dart`

通过 `flutter_dotenv` 从 `.env`（生产）或 `.dev.env`（开发）加载。开发 `.dev.env` 指向本地 app BFF（Android 模拟器以 `10.0.2.2` 访问宿主 `6700` 端口）：

```env
# .dev.env
API_BASE_URL="http://127.0.0.1:6700"
SSE_URL="https://127.0.0.1:6701/events"
CONNECTION_TIMEOUT=3000
RECEIVE_TIMEOUT=3000
```

### 3.3 `ApiClient` 聚合入口

文件：`lib/generated/api/app/service/v1/index.dart`

`ApiClient` 是 10 个 `*ServiceClient` 的聚合入口，在 `_initTransport()`（`lib/src/init.dart`）中经 `DioClientTransport` 桥接到上述 Dio 单例后注册为 lazy singleton：

```dart
// lib/src/init.dart :: _initTransport()
transport.init();   // 注册 Dio lazy singleton
getIt.registerLazySingleton<ApiClient>(
  () => ApiClient(DioClientTransport(dio: GetIt.instance<Dio>())),
);
```

data 层通过 `GetIt.instance<ApiClient>()` 取得 `ApiClient`，再经其 getter 取得需要的 `*ServiceClient`。详见 §4。

---

## 4. 仓储层（Repository）编写规范

### 4.1 三层文件结构

每个 feature 模块的 API 接入由 domain + data 两层、共四类文件协作完成：

| 层 | 文件 | 职责 |
|---|---|---|
| `domain/` | `*_repository.dart` | 抽象 `XxxRepository` 接口，只声明返回**领域值对象**的方法，不出现任何 DTO 或 ServiceClient 类型 |
| `domain/` | `*_failure.dart` | sealed failure 类型，承载该模块的业务错误 |
| `domain/` | `*_models.dart` | 领域值对象（不依赖生成代码） |
| `data/` | `*_repository_impl.dart` | `XxxRepositoryImpl implements XxxRepository`；调数据源、DTO→domain 映射、`DioException`→failure 映射；附 `createXxxRepositoryImpl()` 工厂 |
| `data/` | `*_remote_data_source.dart` | 持 `*ServiceClient` 引用，直接调用生成方法返回原始 DTO |

> 两种 `RemoteDataSource` 持有风格均存在于现网代码：
> - **经 `RemoteDataSource` 间接持有**（wms / auth）：`RepositoryImpl` 注入 `RemoteDataSource`，后者在构造时从 `ApiClient` 取所需 `*ServiceClient`。
> - **`RepositoryImpl` 直接持有 `ApiClient`**（dashboard / approval）：无独立 `RemoteDataSource`，`RepositoryImpl` 自身持 `ApiClient`，直接调 `*_api.xxxService.method()`。

### 4.2 标准结构示例（wms 模块，间接持有）

**domain 抽象**（`wms/domain/wms_repository.dart`）——只出现领域类型：

```dart
abstract class WmsRepository {
  Future<List<WarehouseInfo>> listWarehouses();
  Future<InventoryInfo?> findInventory(String warehouseCode, String productCode);
  Future<void> submitInternalTransfer(InternalTransferDraft draft);
  Future<List<PickingRecord>> listPickings({int limit = 20});
}
```

**data 实现**（`wms/data/wms_repository_impl.dart`）——调用数据源、映射响应、映射异常：

```dart
class WmsRepositoryImpl implements WmsRepository {
  final WmsRemoteDataSource _dataSource;
  WmsRepositoryImpl(this._dataSource);

  @override
  Future<List<WarehouseInfo>> listWarehouses() async {
    try {
      final resp = await _dataSource.listWarehouses();
      return WmsRemoteDataSource.toWarehouseInfos(resp);   // DTO → domain
    } on DioException catch (e) {
      throw _toFailure(e);                                 // 异常 → Failure
    }
  }
  // …其余方法同构…

  WmsFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);             // 传输层已封装
    switch (api.category) {
      case ApiExceptionCategory.auth: return const WmsUnauthorizedFailure();
      case ApiExceptionCategory.business: return const WmsInvalidInputFailure();
      case ApiExceptionCategory.server:
      case ApiExceptionCategory.network: return const WmsNetworkFailure();
      case ApiExceptionCategory.unknown: return const WmsUnknownFailure();
    }
  }
}

/// 供 init.dart 注册时构造。
WmsRepositoryImpl createWmsRepositoryImpl() {
  return WmsRepositoryImpl(
    WmsRemoteDataSource(GetIt.instance<ApiClient>()),
  );
}
```

**remote data source**（`wms/data/wms_remote_data_source.dart`）——从 `ApiClient` 取 client、直接调生成方法：

```dart
class WmsRemoteDataSource {
  final WarehouseServiceClient _warehouses;
  final StockQuantServiceClient _stockQuants;
  final StockPickingServiceClient _stockPickings;

  WmsRemoteDataSource(ApiClient api)
      : _warehouses = api.warehouseService,
        _stockQuants = api.stockQuantService,
        _stockPickings = api.stockPickingService;

  Future<InventoryServiceV1ListWarehouseResponse> listWarehouses() {
    return _warehouses.list(PaginationPagingRequest(noPaging: true));
  }
  // …
}
```

### 4.3 标准结构示例（dashboard 模块，直接持有）

```dart
class DashboardRepositoryImpl implements DashboardRepository {
  final ApiClient _api;
  DashboardRepositoryImpl(this._api);

  @override
  Future<DashboardOverview> getOverview() async {
    try {
      final resp = await _api.stockQuantService.getOverview(...);
      return _toDomain(resp);                              // DTO → domain
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }
  // …
}

DashboardRepositoryImpl createDashboardRepositoryImpl() {
  return DashboardRepositoryImpl(GetIt.instance<ApiClient>());
}
```

### 4.4 sealed Failure 与异常映射

每个模块的 `*_failure.dart` 定义一组 sealed 错误类型，与 `ApiExceptionCategory` 的四类（`auth` / `business` / `server` / `network` / `unknown`）一一映射。`ApiException.fromDioError`（`core/transport/http/api_exception.dart`）依据 HTTP 状态码与 Kratos 错误体分类：

| HTTP / 条件 | `ApiExceptionCategory` |
|---|---|
| 401，或 403 且 reason ∈ {UNAUTHORIZED, FORBIDDEN} | `auth` |
| 4xx（其余业务错误） | `business` |
| 5xx | `server` |
| 无响应（连接失败 / 超时 / 取消） | `network` |
| 无法判定 | `unknown` |

`RepositoryImpl._toFailure` 将 `ApiException.category` 映射为对应模块 failure 子类后抛出；Cubit 在 `try-catch` 中捕获 failure、映射为对应的 `*State`。

> `ApiException` 已由 `UnifiedAuthInterceptor` 在拦截阶段附加到 `DioException.error` 上；`fromDioError` 若发现 `error` 已是 `ApiException` 则直接返回，不重复解析。鉴权类错误（`auth`）由拦截器额外触发 `UserAuthCache.clearTokens()` → `loginStateNotifier` → 路由重定向 `/login`。

---

## 5. 分页请求

生成层提供统一的分页请求模型 `PaginationPagingRequest`（`index.dart`），字段：

| 字段 | 类型 | 含义 |
|---|---|---|
| `pageSize` | `int?` | 每页条数；不传则不分页 |
| `page` | `int?` | 页码（从 1 起）；不传则不分页 |
| `noPaging` | `bool` | 是否全量加载（`page`/`pageSize` 均不传时设为 `true`） |

调用示例（取自 `wms_remote_data_source.dart`）：

```dart
// 全量加载
_warehouses.list(PaginationPagingRequest(noPaging: true));

// 分页加载
_stockPickings.list(PaginationPagingRequest(pageSize: limit));
```

> 不存在手写的 `PaginationQuery`、不存在 locale 自动注入、不存在 fieldMask。后端返回的列表响应均带 `items` 数组，由 data 层遍历映射为 domain 模型。

---

## 6. 认证拦截器（UnifiedAuthInterceptor）

文件：`lib/src/core/transport/http/interceptors/unified_auth_interceptor.dart`

Dio 在 `createDio()` 中安装**唯一**拦截器 `UnifiedAuthInterceptor`，其同时承担请求侧与错误侧职责：

**请求侧**
- 对非登录路径（`/app/v1/login` 之外）注入 `Authorization: Bearer <token>`，token 取自 `UserAuthCache.accessToken`
- 登录路径不注入

**错误侧**
- 将非 2xx 响应中的 Kratos 错误体（`{code, reason, message, metadata}`）解析为 `ApiException`（附 `category`），附加到 `DioException.error` 透传给调用方
- 鉴权类错误（401、403-UNAUTHORIZED/FORBIDDEN）调用 `UserAuthCache.clearTokens()`，随后 `loginStateNotifier` 通知 `GoRouter` 守卫重定向到 `/login`
- 非鉴权类、非预期错误（server / network / unknown）经 `GlobalErrorNotifier`（`core/transport/http/global_error_notifier.dart`，在 `_initTransport()` 注册）统一上报

> **主动令牌刷新**由 `SessionManager`（`core/session/session_manager.dart`，`main()` 中 `SessionManager().start()` 启动）在访问令牌过期前发起，**不在** 拦截器内反应式处理 401。
>
> 不存在 `AuthenticationInterceptor` 类、不存在 `registerInterceptor`、不存在 `autoRefreshToken` 参数、不存在 locale 拦截器。

---

## 7. 文件传输

`ApiClient.fileTransferService` 暴露 `FileTransferServiceClient`，含 `downloadFile` / `putUploadFile` / `postUploadFile` 三个方法（对应 `GET/PUT/POST /app/v1/files/...`）。

> **现状：未被任何 feature 消费。** 文件传输客户端已由 proto 生成并通过 `ApiClient` 暴露，但当前四个 feature 模块（auth / dashboard / approval / wms）均未接入。需要文件上传/下载的新模块应按 §4 的仓储模式在 data 层适配此 client（无需重新生成代码）。

---

## 8. 新增 API 接入 Checklist

当后端新增或修改 API 后，按以下步骤操作：

1. **确认生成产物** — 确认 `lib/generated/api/app/service/v1/index.dart` 的 `ApiClient` 已有目标 `*ServiceClient` getter，且有对应请求/响应模型。若 proto 已更新，后端侧 `make api` 重新生成，移动端拉取新的 `lib/generated/api/`。

2. **实现 feature 三层**（`features/<module>/`）
   - `domain/`：抽象 `XxxRepository` 接口 + sealed `XxxFailure` + 领域值对象模型
   - `data/`：`XxxRepositoryImpl`（DTO→domain 映射 + `ApiException.category`→Failure 映射）+ `createXxxRepositoryImpl()` 工厂；如需分离数据源，另建 `*_remote_data_source.dart` 从 `ApiClient` 取 `*ServiceClient`
   - `presentation/`：Cubit + sealed State + Page widget

3. **注册仓储** — 在 `lib/src/init.dart` 的 `_initTransport()` 中经 `createXxxRepositoryImpl()` 工厂注册为 lazy singleton（按 `DashboardRepositoryImpl` / `WmsRepositoryImpl` 现有写法）。

4. **接入 UI** — Cubit 只依赖抽象 `XxxRepository` 接口，捕获 failure 映射为 State；Page 只渲染 State。

---

## 9. 相关文件索引

| 文件路径 | 说明 |
|---|---|
| `lib/generated/api/app/service/v1/index.dart` | `ApiClient` + 10 个 `*ServiceClient` + 全部请求/响应模型（protoc-gen-dart-http 生成） |
| `lib/generated/api/transport.dart` | `ClientTransport` / `TransportMeta` 接口（生成） |
| `lib/src/core/transport/http/http_client.dart` | `createDio()`：baseUrl / timeouts / responseType.json + 安装 `UnifiedAuthInterceptor` |
| `lib/src/core/transport/http/dio_client_transport.dart` | `DioClientTransport`：把生成 client 的调用桥接到 Dio |
| `lib/src/core/transport/http/api_exception.dart` | `ApiException` + `ApiExceptionCategory` + `fromDioError`（状态码/Kratos 体→分类） |
| `lib/src/core/transport/http/interceptors/unified_auth_interceptor.dart` | 唯一拦截器：Bearer 注入 + 错误体封装 + 鉴权失效登出 |
| `lib/src/core/transport/init.dart` | `transport.init()`：注册 Dio lazy singleton |
| `lib/src/init.dart` | `_initTransport()`：注册 `ApiClient`（经 `DioClientTransport`）+ `GlobalErrorNotifier` + 四个 feature 仓储 |
| `lib/src/core/config/environments.dart` | `.dev.env`/`.env` → `Environments` 导出 |
| `.dev.env` / `.env` | 环境配置文件（gitignored，不在版本控制） |

# AGENTS.md — Flutter ERP 移动端开发指南

> 本文件是 `frontend/app/flutter_app` 子项目的 AI 编码规范单一事实源，适用于所有支持 AGENTS.md 的 AI 编码工具（ZCode、GitHub Copilot、Cursor、Codex、Gemini CLI 等）。Claude Code 通过同级 `CLAUDE.md` 中的 `@AGENTS.md` 引用加载。

## 项目概览

基于 **Flutter** 的 ERP 移动端助手（`pubspec.yaml` 中 `name: go_wind_erp`、`description: "GoWind ERP mobile assistant."`）。当前实现的功能模块：登录鉴权、运营仪表盘、审批中心、WMS 仓库作业。一套 Dart 代码编译为 Android / iOS。

**核心技术栈**：Flutter (Dart 3.12+) + flutter_bloc/Cubit（状态管理）+ GoRouter（路由）+ GetIt（IoC）+ Dio（HTTP）+ ScreenUtil（响应式尺寸）+ ColorScheme.fromSeed（Material 3 主题）+ flutter_intl / Localizely 插件（i18n）

**代码生成工具链**：`protoc-gen-dart-http`（API 客户端 + transport）+ Flutter Intl IDE 插件（i18n `S` 类）。注意：`pubspec.yaml` 虽列 `swagger_parser` / `retrofit` 为依赖，但 `lib/generated/api/` 实际由 `protoc-gen-dart-http` 产出（见生成文件头部注释）。

## 关键架构认知

### Feature-First 模块化架构（data / domain / presentation 三层）

```
lib/
├── main.dart                        # 入口：init() + SessionManager.start() + MultiBlocProvider(AppThemeCubit) → ERPApp
├── src/
│   ├── app.dart                     # ERPApp：ScreenUtilInit + MaterialApp.router（routerConfig=createAppRouter）
│   ├── init.dart                    # 应用初始化（见下文“初始化序列”）
│   ├── init_thirdparty_plugins.dart # 第三方插件初始化（SharedPreferences 等）
│   ├── app/
│   │   └── home_shell.dart          # 3-tab 底部导航 shell（StatefulNavigationShell + NavigationBar）
│   ├── app_router/                  # GoRouter 路由配置 + 路由名称常量
│   │   ├── app_router.dart
│   │   └── route_names.dart
│   ├── core/                        # 核心基础设施（见下文）
│   └── features/                    # ★ Feature-First 业务模块（每个模块 data/domain/presentation 三层）
│       ├── approval/                #   审批中心
│       ├── auth/                    #   登录鉴权
│       ├── dashboard/              #   运营仪表盘
│       └── wms/                     #   WMS 仓库作业
├── generated/                       # [自动生成]
│   ├── api/                         #   protoc-gen-dart-http 产出
│   │   ├── transport.dart
│   │   └── app/service/v1/index.dart   # ApiClient + 各 *ServiceClient
│   ├── intl/
│   └── l10n.dart                    #   Flutter Intl 插件产出的 S 类
└── l10n/                            # i18n ARB 文件（intl_zh_CN.arb / intl_en_US.arb）
```

每个 feature 模块严格遵循 **data / domain / presentation** 三层（clean architecture）：

| 层 | 职责 | 典型文件 |
|---|---|---|
| `data/` | 仓储实现 + 远程数据源（适配生成的 `*ServiceClient`，调用 API，映射响应到 domain 值对象，`DioException`→failure） | `*_repository_impl.dart`、`*_remote_data_source.dart` |
| `domain/` | sealed failure 类型 + 值对象模型 + 仓储抽象接口 | `*_failure.dart`、`*_models.dart`、`*_repository.dart` |
| `presentation/` | Cubit（viewmodel）+ sealed State + Page widget | `*_cubit.dart`、`*_state.dart`、`*_page.dart` |

> 仓储实现与抽象接口分离：`domain/*_repository.dart` 定义抽象 `XxxRepository`，`data/*_repository_impl.dart` 提供实现并通过 `createXxxRepositoryImpl()` 工厂注入。Cubit 只依赖抽象接口，不直接接触 `ServiceClient`。Feature 仓储当前消费的 ServiceClient：`ApprovalRequestServiceClient`、`AuthenticationServiceClient`、`InventoryServiceClient`、`StockMovementServiceClient`、`WarehouseServiceClient`（其余如 `PayableServiceClient` / `PaymentServiceClient` / `PurchaseOrderServiceClient` / `FileTransferServiceClient` / `UserProfileServiceClient` 已生成但尚未被任何 feature 消费）。

### 路由（GoRouter + 鉴权守卫）

路由在 `app_router/app_router.dart` 的 `createAppRouter()` 中构造单个 `GoRouter`：

- `initialLocation: AppRoutePath.home`（`/`）
- `refreshListenable: cache.loginStateNotifier` —— 登录态变化时重跑守卫
- `redirect: _guard(cache, state)` —— 鉴权守卫：未登录且不在 login 路由 → 重定向 `/login`；已登录且在 login 路由 → 重定向 `/`；其余放行
- `errorBuilder` → `NotFoundPage`（`core/widgets/not_found_page.dart`）
- `StatefulShellRoute.indexedStack` —— 三个分支，builder 渲染 `HomeShell(navigationShell)`（`app/home_shell.dart`）
- 顶层 `GoRoute`：login 路由，builder 用 `BlocProvider` 注入 `LoginCubit`

路由名称与路径常量：

| `RouteNames` | `AppRoutePath` | 用途 |
|---|---|---|
| `RouteNames.login` | `/login` | 登录页（顶层 GoRoute） |
| `RouteNames.home` | `/` | 仪表盘（StatefulShellRoute branch 0，`DashboardCubit` + `DashboardPage`） |
| `RouteNames.approval` | `/approval` | 审批中心（branch 1，`ApprovalCubit` + `ApprovalPage`） |
| `RouteNames.wms` | `/wms` | WMS 仓库作业（branch 2，`WmsCubit` + `WmsPage`） |

> `RouteNames` 定义在 `app_router/route_names.dart`；`AppRoutePath` 定义在 `core/constants/router_paths.dart`，经 `core/constants/index.dart` 重导出。`HomeShell` 的 tab 切换通过 `navigationShell.goBranch(index, initialLocation: ...)`，**无** RouteName→tab-index 显式映射，索引纯位置性。

`HomeShell` 渲染 **固定 3-tab 底部导航栏**（Material `NavigationBar`），三个 destination 为 dashboard / approval / wms，标签文本来自 i18n（`context.loc.navDashboard` / `navApproval` / `navWms`）。**不存在** `web_shell_layout` 组件，shell 层也 **不使用** `ResponsiveLayout`。

### 响应式布局（三级断点）

| 设备 | 屏宽 | 判定 |
|------|------|------|
| 手机 Mobile | < 600 dp | `Breakpoints.isMobile` / `ResponsiveUtils.isMobile` |
| 平板 Tablet | 600 ~ 1024 dp | `Breakpoints.isTablet` / `ResponsiveUtils.isTablet` |
| 网页 Web | > 1024 dp | `Breakpoints.isWeb` / `ResponsiveUtils.isWideScreen` |

`ResponsiveLayout`（`core/widgets/responsive_layout.dart`）按断点选择 `mobileBody` / `tabletBody?` / `webBody` 渲染：

```dart
ResponsiveLayout(
  mobileBody: _buildMobileView(),
  webBody: _buildWebView(),
)
```

`ResponsiveUtils`（`core/utils/responsive_utils.dart` —— **注意是 `utils/` 不是 `utilities/`**）提供 `isMobile` / `isTablet` / `isWideScreen` / `gridColumns` / `postGridColumns` / `categoryGridColumns` / `contentMaxWidth` / `padding` / `horizontalPadding` / `fontSize` / `spacing` 等静态方法。断点常量集中在 `core/constants/breakpoints.dart`（`mobile=600`、`tablet=1024`、`webContentMaxWidth=1140`、`webSidebarWidth=260`、`webContentPadding=32`）。

`ScreenUtil` 在 `ERPApp.build` 中以 `designSize: Breakpoints.designSize`（`Size(375, 812)`）初始化；Web 端在 builder 内将 designSize 重设为当前视窗尺寸（1:1），因此 **Web 端禁止 `.w`/`.h`/`.sp`**（始终 1:1，等同固定值），手机端可用。

### 传输层（Dio + UnifiedAuthInterceptor）

Dio 在 `core/transport/init.dart` 注册为 lazy singleton，由 `core/transport/http/http_client.dart` 的 `createDio()` 构造：`baseUrl = Environments.apiBaseUrl`、`connectTimeout`/`receiveTimeout` 来自 dotenv、`responseType = json`。

**仅有一个拦截器** `UnifiedAuthInterceptor`（`core/transport/http/interceptors/unified_auth_interceptor.dart`），同时承担：
- **请求侧**：对非登录路径（`kLoginPath = '/app/v1/login'`）注入 `Authorization: Bearer <token>`（从 `UserAuthCache.accessToken`）
- **错误侧**：将 Kratos 错误体封装为 `ApiException`（`core/transport/http/api_exception.dart`），按状态码分类（401、403-UNAUTHORIZED/FORBIDDEN 归为 `auth` 类）；鉴权类错误调用 `UserAuthCache.clearTokens()` 触发本地登出，`loginStateNotifier` 随之驱动路由重定向到 `/login`
- **非鉴权类非预期错误**：经 `GlobalErrorNotifier`（`core/transport/http/global_error_notifier.dart`，在 `_initTransport()` 注册）统一通知

> 主动令牌刷新由 `SessionManager`（`core/session/session_manager.dart`，`main()` 中 `SessionManager().start()` 启动）在访问令牌过期前发起，**不在** 拦截器内反应式处理。**无** locale 拦截器（无 `Accept-Language` 注入）。

生成的 `ApiClient`（`lib/generated/api/app/service/v1/index.dart`）通过 `DioClientTransport`（`core/transport/http/dio_client_transport.dart`）桥接到上述 Dio 实例。`ApiClient` 与各 `*ServiceClient` 在 `_initTransport()`（`init.dart`）中注册为 lazy singleton。

### 鉴权缓存（UserAuthCache）

`core/repositories/user_auth_cache.dart`，在 `core/repositories/init.dart`（**非** 顶层 `init.dart`）注册为 lazy singleton。公开 API：`loginStateNotifier`（`ValueNotifier<bool>`，驱动路由守卫与登录态 UI）、`accessToken` / `refreshToken` / `accessTokenExpiresAt`（从 JWT 解析）、`hasLogin`、`saveAuthInfo` / `saveAccessToken` / `saveRefreshToken` / `clearTokens` / `clearAccessToken` / `clearRefreshToken`、`init()`（从 SharedPreferences 水合 `ACCESS_TOKEN`/`REFRESH_TOKEN` 键）。`clearTokens()` 后 `loginStateNotifier` 通知守卫重定向到 `/login`。

### 主题系统（Material 3 + ColorScheme.fromSeed）

`AppThemeCubit`（`core/themes/cubit/app_theme_cubit.dart`）管理 `themeMode`（light/dark/system）、`seedColor`、`locale` 三个状态，由 `MultiBlocProvider` 在 `main.dart` 顶层注入。light/dark 主题在 `core/themes/light_theme.dart` / `dark_theme.dart` 中用 `ColorScheme.fromSeed` 生成。`ERPApp.build` 通过 `context.watch<AppThemeCubit>()` 读取 `themeMode` / `currentLocale` / `seedColor` 传给 `MaterialApp.router`。

> 仅有 **一个** 预设种子色常量 `kDefaultSeedColor = Color(0xFF3A7CA5)`（`light_theme.dart`）。运行时种子色由用户经 `modifySeedColor` 选择，作为 `int` 持久化到 SharedPreferences 的 `SEED_COLOR` 键；主题模式持久化到 `USER_PREFERENCE` 键（经 `UserPreferenceCache`）。**不存在** “8 种预设主题色” 数组。

### 国际化（flutter_intl / Localizely 插件）

```
lib/l10n/intl_zh_CN.arb / intl_en_US.arb   # 翻译源（main_locale: zh_CN）
lib/generated/l10n.dart                     # [生成] S 类（由 Flutter Intl IDE 插件 / Localizely 生成）
```

配置在 `pubspec.yaml` 的 `flutter_intl:` 块（**非** `intl_utils` 依赖 —— `intl_utils` 未列入 pubspec）：

```yaml
flutter_intl:
  enabled: true
  arb_dir: lib/l10n
  main_locale: zh_CN
  template_arb_file: intl_en_US.arb
  output_dir: lib/generated
  output_file_name: l10n.dart
  class_name: S
```

访问翻译经 `core/extensions/app_localizations_context.dart` 提供的 `context.loc` 扩展（内部调用 `S.of(context)`）：

```dart
Text(context.loc.appName)              // 获取翻译
```

配置的两个 locale：`zh_CN`、`en_US`。`S` 类由 `AppLocalizationDelegate`（`S.delegate`）加载，在 `ERPApp._buildMaterialApp` 中注册到 `MaterialApp.router(localizationsDelegates: [..., S.delegate])`。

### 初始化序列（init.dart）

`init()` 按序执行：

1. `WidgetsFlutterBinding.ensureInitialized()` —— `main()` 中已调用并保留原生闪屏
2. `Environments.init()` —— 经 flutter_dotenv 加载 `.dev.env`/`.env`，暴露 `API_BASE_URL` / `SSE_URL` / `CONNECTION_TIMEOUT` / `RECEIVE_TIMEOUT` / `NTP_HOST` / `AES_KEY` / `SENTRY_DSN`
3. `initThirdPartyPlugins()` —— 注册 `SharedPreferences`（eager singleton），锁定竖屏、移动端启用 WakelockPlus、设置透明状态栏
4. `_initTransport()` —— `transport.init()`（注册 Dio lazy singleton），随后注册 `ApiClient` / `GlobalErrorNotifier` / `AuthRepository` / `DashboardRepository` / `WmsRepository` / `ApprovalRepository`（均 lazy singleton，仓储经各自 `createXxxRepositoryImpl` 工厂）
5. `repos.init()`（`core/repositories/init.dart`）—— 注册 `UserAuthCache` / `UserPreferenceCache` / `LanguageListRepository`（lazy singleton）并调用前两者的 `.init()`
6. `_initErrorWidget()` —— 安装自定义 `ErrorWidget.builder`

> `SessionManager().start()` 在 `main()` 中、`init()` 完成后启动，监听登录态并在访问令牌过期前主动刷新。

## 关键约定（必须遵守）

1. **feature 模块必须 data/domain/presentation 三层** —— 仓储实现与抽象接口分离，Cubit 只依赖抽象接口，不直接接触 `ServiceClient`
2. **禁止手改 `lib/generated/`** —— `generated/api/` 由 protoc-gen-dart-http 生成，`generated/l10n.dart` 由 Flutter Intl 插件生成
3. **HTTP 仅走 Dio + UnifiedAuthInterceptor** —— 不自建 client、不绕过拦截器、不自行注入 `Authorization` 头
4. **登录态由 `UserAuthCache.loginStateNotifier` 驱动** —— 不自行维护登录状态变量
5. **路由用 `context.go()`（顶级切换）/ `context.push()`（子页面）** —— 返回用 `AppBackButton`（`core/widgets/app_back_button.dart`，内置 canPop 检查）
6. **路由路径集中管理** —— `core/constants/router_paths.dart`（`AppRoutePath`）+ `app_router/route_names.dart`（`RouteNames`），不硬编码路径
7. **响应式用 `ResponsiveLayout`** —— 不在一个 build 方法混用 mobile/web 视图
8. **断点用 `Breakpoints` 常量** —— 不硬编码屏宽数值
9. **Web 端禁止 `.w`/`.h`/`.sp`** —— Web 端 ScreenUtil designSize 设为视窗尺寸（1:1），用固定值；手机端可用
10. **状态管理用 Cubit** —— 全局状态 `AppThemeCubit`，页面局部状态 `StatefulWidget` + `setState`；Cubit 通过 `BlocProvider` 在路由 builder 中注入
11. **i18n 文本不硬编码** —— 用 `context.loc.xxx`，翻译写到 ARB 文件后重新生成
12. **业务逻辑写在 Cubit，Page 只渲染 State** —— Page widget 不直接调用 `ServiceClient` 或 `Repository`

## 代码生成（改后必须重新生成）

| 修改内容 | 命令 |
|---|---|
| proto API 定义 | 后端 `make api` 生成，移动端重新拉取生成的 `lib/generated/api/` |
| ARB 翻译文件 | Flutter Intl IDE 插件生成（或 `intl_utils` CLI，需另行安装） |

## 开发命令

```bash
flutter pub get                              # 安装依赖
flutter run -d android / ios                 # 移动端开发
flutter build apk / ios                      # 构建生产产物
flutter analyze                              # 代码分析
flutter test                                 # 测试
```

**环境变量**（`.dev.env` Debug / `.env` Release，通过 flutter_dotenv 加载到 `Environments`）。`.dev.env` 示例：

```env
API_BASE_URL="http://10.0.2.2:6700"          # 指向本地 app BFF（Android 模拟器 10.0.2.2 → 宿主 6700 端口）
SSE_URL="https://sse.gowind.cloud/events"
CONNECTION_TIMEOUT=3000
RECEIVE_TIMEOUT=3000
AES_KEY="f51d66a73d8a0927"
NTP_HOST="time.google.com"
SENTRY_DSN="https://ingest.sentry.io/"
```

> 配置产物注记：生产 `.env` 的 `API_BASE_URL` / `SSE_URL` 仍指向历史 `*.erp.gowind.cloud` 子域。这是运行时配置遗留 —— feature 源码中无 `erp` 字样（`grep -rin erp lib/src/features/` 零命中），但部署前需确认这两个 URL 已切到 ERP 后端。

## 新增业务模块 Checklist

```
- [ ] Step 1: 确认 generated ApiClient 中已有目标 ServiceClient（后端 proto 已生成并同步到移动端）
- [ ] Step 2: 实现 feature 三层（features/<module>/data/ + domain/ + presentation/）
  - [ ] domain: sealed failure + 值对象模型 + 抽象 Repository 接口
  - [ ] data: RemoteDataSource（适配 ServiceClient）+ RepositoryImpl + createXxxRepositoryImpl 工厂
  - [ ] presentation: Cubit + sealed State + Page widget
- [ ] Step 3: 注册仓储到 init.dart 的 _initTransport()（GetIt.registerLazySingleton + createXxxRepositoryImpl）
- [ ] Step 4: 注册路由（app_router.dart 的 StatefulShellRoute 新增 branch + route_names.dart + router_paths.dart）
- [ ] Step 5: 添加导航入口（home_shell.dart 的 NavigationBar 新增 NavigationDestination）
- [ ] Step 6: 添加 i18n（l10n/intl_zh_CN.arb + intl_en_US.arb，Flutter Intl 插件重新生成）
```

## 常见错误与纠正

| 错误做法 | 正确做法 |
|---|---|
| 手改 `lib/generated/` | 改 proto / ARB 源后重新生成 |
| 绕过 `UnifiedAuthInterceptor` 自建 HTTP client | 走 Dio（GetIt 单例）+ 生成 ApiClient |
| 自行注入 `Authorization` 头 | 由 `UnifiedAuthInterceptor` 统一注入 |
| 自行维护登录状态变量 | 用 `UserAuthCache.loginStateNotifier` |
| 一个 build 方法混用 mobile/web | 用 `ResponsiveLayout` 双视图 |
| 硬编码屏宽数值 | 用 `Breakpoints` 常量（`core/constants/breakpoints.dart`） |
| `responsive_utils` 按 `core/utilities/` 导入 | 实际在 `core/utils/` |
| 硬编码路由路径 | 用 `AppRoutePath` + `RouteNames` |
| 返回按钮不检查 canPop | 用 `AppBackButton`（`core/widgets/app_back_button.dart`） |
| 业务逻辑写在 Page widget | 写在 Cubit，Page 只渲染 State |
| Cubit 直接依赖 `ServiceClient` | Cubit 只依赖抽象 Repository 接口 |
| 声称“8 种预设主题色” | 仅 1 个常量 `kDefaultSeedColor`，运行色由用户选择 |
| 声称生成器是 swagger_parser / intl_utils | API 是 protoc-gen-dart-http，i18n 是 flutter_intl 配置块（Localizely 插件） |
| 声称存在 `web_shell_layout` / `core/services/base_service.dart` / `PaginationQuery` | 均不存在；shell 是固定 3-tab `NavigationBar` |
| Web 端用 `.w`/`.h`/`.sp` | Web 端用固定值（手机端才用 ScreenUtil） |

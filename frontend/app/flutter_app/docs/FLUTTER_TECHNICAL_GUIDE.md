# 基于 Flutter 的 Headless ERP 全平台前端架构：技术解析与二次开发导引

> 本文面向希望基于此项目进行二次开发的 Flutter 工程师，从技术栈选型、核心架构设计、关键模块实现到二开实践路径，提供一份完整的技术地图。

---

## 一、技术栈总览

本项目是一个 **Flutter 跨端** ERP 移动助手，一套 Dart 代码编译为 Android / iOS：

| 层面       | 技术                       | 版本               | 用途                      |
|----------|--------------------------|------------------|-------------------------|
| 框架       | Flutter                  | 3.x (Dart 3.12+) | 跨端 UI 框架               |
| 语言       | Dart                     | 3.x              | 类型安全 + 空安全              |
| 状态管理     | flutter_bloc + Cubit     | 9.x              | 响应式状态（Cubit 模式）         |
| 服务定位     | GetIt                    | 9.x              | 轻量 IoC 容器（单例管理）         |
| 路由       | GoRouter                 | 17.x             | 声明式路由 + StatefulShellRoute |
| 国际化      | flutter_intl (intl)      | 0.21.x           | ARB 翻译文件 + 代码生成         |
| HTTP 客户端 | Dio                      | 5.x              | REST 通信                 |
| API 代码生成 | protoc-gen-dart-http     | —                | 后端 .proto → Dart HTTP 客户端 + 模型 |
| 响应式适配    | flutter_screenutil       | 5.x              | 手机端设计稿适配（Web 端禁用）       |
| 加密       | encrypt + crypto         | 5.x / 3.x        | AES 加密（Token 持久化）       |
| JWT      | jose                     | 0.3.x            | JWT Token 解析            |
| 图片缓存     | cached_network_image     | 3.x              | 网络图片缓存                  |
| 骨架屏      | shimmer                  | 3.x              | 加载占位动画                  |
| 环境变量     | flutter_dotenv           | 6.x              | .env 文件管理               |
| 日志       | logger                   | 2.x              | 结构化日志                   |

> **关于 pubspec 中的未用依赖：** `pubspec.yaml` 仍列 `swagger_parser` / `retrofit` / `retrofit_generator` / `freezed` / `json_serializable` / `intl_utils` / `cached_query` / `flutter_markdown` / `flutter_widget_from_html` 等包，但 `lib/src/` 与 `lib/generated/api/` 实际均不使用它们——API 客户端由 `protoc-gen-dart-http` 生成（见生成文件头部注释），状态管理用 `Equatable` + `@immutable` 而非 freezed，无内容渲染、无 Query/Mutation 缓存。这些依赖为历史 CMS 工程遗留，清理待后续。

**代码生成工具链：** `protoc-gen-dart-http`（API 客户端 + transport，后端侧 `make api`） + Flutter Intl IDE 插件（i18n `S` 类）

---

## 二、核心架构设计

### 2.1 全平台编译模式

项目基于 Flutter 3.x，通过 `flutter` CLI 将同一份 Dart 代码编译为多端产物：

```bash
# Web 开发
flutter run -d chrome

# iOS 开发
flutter run -d ios

# Android 开发
flutter run -d android

# 构建产物
flutter build web          # → build/web/ (SPA)
flutter build apk          # → build/app/outputs/ (APK)
flutter build ios          # → build/ios/ (IPA)
flutter build macos        # → build/macos/ (macOS App)
flutter build windows      # → build/windows/ (Windows App)
```

**环境配置**（`.dev.env` / `.env`）：

```env
API_BASE_URL="https://api.erp.gowind.cloud"    # 后端 API 地址
SSE_URL="https://api.erp.gowind.cloud/events"   # SSE 推送地址
CONNECTION_TIMEOUT=3000                          # 连接超时（毫秒）
RECEIVE_TIMEOUT=3000                             # 接收超时（毫秒）
AES_KEY="f51d66a73d8a0927"                       # AES 加密密钥
SENTRY_DSN="https://ingest.sentry.io/"           # Sentry 异常监控
```

> 环境变量通过 `flutter_dotenv` 加载，在 `Environments` 类中统一导出。Debug 模式加载 `.dev.env`，Release 模式加载 `.env`。

### 2.2 路由架构（GoRouter + 鉴权守卫 + StatefulShellRoute）

路由在 `lib/src/app_router/app_router.dart` 的 `createAppRouter()` 中构造单个 `GoRouter`，配置如下：

- `initialLocation: AppRoutePath.home`（`/`）
- `refreshListenable: cache.loginStateNotifier` —— 登录态变化时重跑守卫
- `redirect: _guard(cache, state)` —— 鉴权守卫：未登录访问任意非登录路由 → 重定向 `/login`；已登录访问登录页 → 重定向 `/`；其余放行
- `errorBuilder` → `NotFoundPage`（`core/widgets/not_found_page.dart`）
- `routes` 含两类：
  - 顶层 `GoRoute`：登录路由（`/login`），builder 用 `BlocProvider` 注入 `LoginCubit(authRepository)`
  - `StatefulShellRoute.indexedStack`：三个分支，builder 渲染 `HomeShell(navigationShell)`（`app/home_shell.dart`）

三个分支与 feature 对应：

| 分支 | `RouteNames` | `AppRoutePath` | 注入 Cubit + Page |
|---|---|---|---|
| 0 | `home` | `/` | `DashboardCubit` + `DashboardPage` |
| 1 | `approval` | `/approval` | `ApprovalCubit` + `ApprovalPage` |
| 2 | `wms` | `/wms` | `WmsCubit` + `WmsPage` |

`HomeShell` 渲染固定 3-tab 底部导航栏（Material `NavigationBar`），三个 destination 为 dashboard / approval / wms，标签文本来自 i18n（`context.loc.navDashboard` / `navApproval` / `navWms`）。tab 切换通过 `navigationShell.goBranch(index, initialLocation: ...)`，无 RouteName→tab-index 显式映射。**不存在** `WebShellLayout` 组件、不存在 `ShellRoute` 顶部导航壳。

路由常量集中管理（经 `core/constants/index.dart` 重导出）：

| 文件 | 内容 |
|---|---|
| `app_router/route_names.dart` | `RouteNames`（`login` / `home` / `approval` / `wms`） |
| `core/constants/router_paths.dart` | `AppRoutePath`（`/login` / `/` / `/approval` / `/wms`） |

```
lib/src/app_router/
├── app_router.dart          # createAppRouter()：GoRouter 配置
└── route_names.dart         # RouteNames 常量
```

### 2.3 三层 API 架构（生成层 → 仓储层 → 调用层）

API 层遵循 clean architecture 的 data / domain / presentation 划分，配合生成层：

```
lib/generated/api/app/service/v1/index.dart   # [生成] ApiClient + 10 个 *ServiceClient + 全部请求/响应模型
lib/generated/api/transport.dart              # [生成] ClientTransport 接口

lib/src/features/<module>/                    # [业务] 每个 feature 三层
├── domain/                                   #   抽象 Repository 接口 + sealed Failure + 值对象模型
├── data/                                     #   RepositoryImpl + RemoteDataSource（DTO→domain、DioException→Failure）
└── presentation/                             #   Cubit + sealed State + Page
```

**第一层 — 生成的客户端**（`generated/api/`）：由后端 `.proto` 经 `protoc-gen-dart-http` 生成，`ApiClient` 聚合 10 个 `*ServiceClient`（懒加载 getter），经 `DioClientTransport` 桥接到 Dio。不应手动编辑。

**第二层 — 仓储层**（`data/` + `domain/`）：domain 层定义抽象 `XxxRepository`（只出现领域值对象）与 sealed `XxxFailure`；data 层 `XxxRepositoryImpl` 实现，调用 `RemoteDataSource`（持 `*ServiceClient`）、把响应 DTO 映射为领域模型、把 `DioException`（已由统一拦截器封装为 `ApiException`）按 `category` 映射为对应 failure 子类抛出。

```dart
// wms/data/wms_repository_impl.dart
class WmsRepositoryImpl implements WmsRepository {
  final WmsRemoteDataSource _dataSource;
  WmsRepositoryImpl(this._dataSource);

  @override
  Future<List<WarehouseInfo>> listWarehouses() async {
    try {
      final resp = await _dataSource.listWarehouses();
      return WmsRemoteDataSource.toWarehouseInfos(resp);   // DTO → domain
    } on DioException catch (e) {
      throw _toFailure(e);                                 // → WmsFailure
    }
  }

  WmsFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);             // 传输层已封装
    switch (api.category) { /* auth/business/server/network/unknown → Failure 子类 */ }
  }
}
```

详见 `docs/api-guide.md`。

**第三层 — presentation**：Cubit 通过 `BlocProvider` 在路由 builder 中注入，只依赖抽象 `XxxRepository` 接口，捕获 failure 映射为 sealed `*State`；Page 只渲染 State。

### 2.4 Dio — HTTP 通信内核

基于 Dio 的全局单例（通过 GetIt 注册），仅安装一个拦截器 `UnifiedAuthInterceptor`：

```
请求侧：非登录路径注入 Authorization: Bearer <token>
错误侧：Kratos 错误体 → ApiException（附 category）附加到 DioException.error 透传；
        鉴权类错误（401 / 403-UNAUTHORIZED/FORBIDDEN）触发 clearTokens → 重定向 /login；
        非预期错误（server/network/unknown）经 GlobalErrorNotifier 上报
```

**初始化流程**：

```dart
// init.dart
Future<void> init() async {
  WidgetsFlutterBinding.ensureInitialized();
  await Environments.init();           // 加载 .env
  await initThirdPartyPlugins();       // 初始化 SharedPreferences 等
  _initTransport();                    // 注册 Dio + ApiClient + 各 Repository 到 GetIt
  await repos.init();                  // 初始化缓存仓库（UserAuthCache / UserPreferenceCache）
  _initErrorWidget();
}

void _initTransport() {
  transport.init();                    // 注册 Dio lazy singleton
  final getIt = GetIt.instance;
  getIt.registerLazySingleton<ApiClient>(
    () => ApiClient(DioClientTransport(dio: GetIt.instance<Dio>())),  // 生成 ApiClient
  );
  getIt.registerLazySingleton<GlobalErrorNotifier>(() => GlobalErrorNotifier());
  getIt.registerLazySingleton<AuthRepository>(() => createAuthRepositoryImpl());
  getIt.registerLazySingleton<DashboardRepository>(() => createDashboardRepositoryImpl());
  getIt.registerLazySingleton<WmsRepository>(() => createWmsRepositoryImpl());
  getIt.registerLazySingleton<ApprovalRepository>(() => createApprovalRepositoryImpl());
}
```

> 不存在 `BaseService`、不存在 `handleDioError`、不存在 locale 拦截器、不存在响应数据解构。错误处理经 `ApiException.fromDioError`（`core/transport/http/api_exception.dart`）分类后由各 feature 的 `_toFailure` 映射为领域 failure。详见 `docs/api-guide.md` §4.4、§6。

二开时如需对接不同后端，只需修改 `.env` 中的 `API_BASE_URL`；API 客户端代码由后端 proto 生成，移动端拉取即可。

### 2.5 状态管理 — BLoC / Cubit 模式

采用 `flutter_bloc` 的 Cubit 模式（Bloc 的简化版），用于主题、语言等全局状态：

```dart
// main.dart
void main() async {
  await init();
  runApp(
    MultiBlocProvider(
      providers: [BlocProvider(create: (_) => AppThemeCubit())],
      child: const ERPApp(),
    ),
  );
}
```

**AppThemeCubit** 同时管理主题模式（亮色/暗色/跟随系统）、主题色（seedColor）和语言切换：

```dart
// app_theme_cubit.dart
class AppThemeCubit extends Cubit<AppThemeState> {
  // 主题模式
  modify(ThemeMode newMode) async { ... }
  // 主题色
  void modifySeedColor(Color color) async { ... }
  // 语言切换
  void modifyLocale(Locale newLocale) async { ... }
}
```

**页面级状态**：页面内的局部状态（如列表数据、加载状态）使用 `StatefulWidget` 的 `setState` 管理，不引入额外的状态管理库。

### 2.6 偏好持久化 — UserPreferenceCache

`UserPreferenceCache` 基于 `SharedPreferences` 封装，管理用户偏好设置：

```dart
class UserPreferenceCache {
  // 主题模式
  ThemeMode get themeMode;
  Future<void> setMaterialThemeMode(ThemeMode mode);
  // 主题色
  Color? get seedColorValue;
  Future<void> setSeedColor(Color color);
  // 语言
  String get language;        // "zh_CN" / "en_US"
  Future<void> setLanguage(String lang, bool notify);
}
```

主题支持三种模式：`light`（亮色）、`dark`（暗色）、`system`（跟随系统），通过 Material 3 的 `ColorScheme.fromSeed` 动态生成色板。

### 2.7 国际化体系

使用 **flutter_intl** (intl + ARB 文件) 实现国际化：

```
lib/l10n/
├── intl_zh_CN.arb       # 中文翻译（main_locale）
└── intl_en_US.arb       # 英文翻译（template_arb_file）

lib/generated/           # [自动生成] Flutter Intl IDE 插件（Localizely）产出
├── l10n.dart            # S 类（统一导出所有翻译方法）
└── intl/
    ├── messages_zh_CN.dart
    └── messages_en_US.dart
```

配置在 `pubspec.yaml` 的 `flutter_intl:` 块（定义 `arb_dir` / `main_locale` / `template_arb_file` / `output_dir` / `output_file_name` / `class_name`）。由 Flutter Intl IDE 插件生成，**不存在** `intl_utils` CLI 命令。

**使用方式**：

```dart
// 在 Widget 中
Text(S.of(context).appName)           // 获取当前语言的翻译
Text(S.of(context).postsCount(5))     // 带参数的翻译
```

> 不存在 `translation_helpers.dart`、不存在 `getPostTitle` 等辅助函数、不存在后端实体的 `translations[]` 数组——本项目无内容渲染、无多语言实体字段。

---

## 三、关键模块深度解析

### 3.1 响应式布局系统

项目支持全平台响应式，采用断点 + 双视图模式：


| 设备类型      | 屏幕宽度        | 布局策略               |
|-----------|-------------|--------------------|
| 手机 Mobile | < 600 dp    | 单栏 + 底部导航栏    |
| 平板 Tablet | 600~1024 dp | 双栏布局               |
| 网页 Web    | > 1024 dp   | 三栏/居中布局 |

> 现状：本项目移动端定向（Android / iOS），实际仅手机断点生效。`ResponsiveLayout` / `ResponsiveUtils` / 三级断点作为共享基础设施存在，为未来 web 端扩展预留；当前仅登录页实际使用 `isMobile` 三元分流（见 `docs/ui-adaptation.md` §7）。


**核心组件**：

```dart
// ResponsiveLayout — 双视图切换（调用方无需手写 isMobile 判断）
ResponsiveLayout(
  mobileBody: _buildMobileView(),
  webBody: _buildWebView(),
)

// ResponsiveUtils — 响应式工具
ResponsiveUtils.isMobile(context)     // 是否手机
ResponsiveUtils.postGridColumns(ctx)  // 网格列数（1/2/3）
```

> 不存在 `WebShellLayout`、不存在 `ShellRoute` 顶部导航壳。导航仅由 `HomeShell` 的固定 3-tab 底部 `NavigationBar` 提供（见 §2.2）。

**ScreenUtil 策略**：
- 手机端：使用 `.w` / `.h` / `.sp` 适配不同手机分辨率
- 宽屏端：动态将 `designSize` 设为当前视窗尺寸，使 `.w/.h/.sp` 始终 1:1，字体不随窗口缩放

### 3.2 主题系统

采用 **Material 3 + ColorScheme.fromSeed** 方案，支持动态主题色切换：

```dart
// light_theme.dart
ThemeData getLightTheme({Color? seedColor}) {
  final colorScheme = ColorScheme.fromSeed(
    seedColor: seedColor ?? kDefaultSeedColor,
    brightness: Brightness.light,
  );
  return ThemeData(
    colorScheme: colorScheme,
    useMaterial3: true,
    // ...
  );
}
```

**种子色**：仅有一个预设常量 `kDefaultSeedColor = Color(0xFF3A7CA5)`（`light_theme.dart`）。运行时种子色由用户经 `AppThemeCubit.modifySeedColor` 选择，作为 `int` 持久化到 SharedPreferences 的 `SEED_COLOR` 键；主题模式持久化到 `USER_PREFERENCE` 键（经 `UserPreferenceCache`）。**不存在** “8 种预设主题色” 数组、不存在设置页 `_presetColors` 列表。

**设计要点：**
- **暗色模式**：通过 `themeMode`（light/dark/system）切换 `theme` / `darkTheme`
- **动态主题色**：`AppThemeCubit` 管理状态，单一用户选定种子色，SharedPreferences 持久化

### 3.3 内容渲染管线

本项目为 ERP 作业应用，无内容渲染管线。`flutter_markdown` / `flutter_widget_from_html` 虽列于 pubspec 但 `lib/src/` 未使用，不存在 `ContentViewer` 组件。

### 3.4 认证流程

```
用户登录 → 存储 accessToken + refreshToken（SharedPreferences + AES 加密）
    ↓
每次请求 → UnifiedAuthInterceptor 注入 Authorization 头（登录路径除外）
    ↓
401 / 403-UNAUTHORIZED/FORBIDDEN 响应 → clearTokens → loginStateNotifier → GoRouter 重定向 /login
```

> **主动令牌刷新**由 `SessionManager`（`core/session/session_manager.dart`，`main()` 中 `SessionManager().start()` 启动）在访问令牌过期前发起，**不在** 拦截器内反应式处理。不存在 “401 → 自动调用 refreshToken 接口” 的反应式刷新链。

**登录状态管理**：通过 `UserAuthCache`（基于 GetIt 单例）+ `loginStateNotifier`（`ValueNotifier<bool>`）实现响应式登录状态，驱动 `GoRouter` 守卫重定向：

```dart
// 路由守卫依据 hasLogin 决定重定向（app_router.dart :: _guard）
String? _guard(UserAuthCache cache, GoRouterState state) {
  final loggedIn = cache.hasLogin;
  final isLoginRoute = state.matchedLocation == AppRoutePath.login;
  if (!loggedIn) {
    return isLoginRoute ? null : AppRoutePath.login;   // 未登录 → /login
  }
  return isLoginRoute ? AppRoutePath.home : null;       // 已登录访问 /login → /
}
```

---

## 四、项目目录结构与职责

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
│   └── features/                    # Feature-First 业务模块（每个模块 data/domain/presentation 三层）
│       ├── approval/                #   审批中心
│       ├── auth/                    #   登录鉴权（另含 widgets/ 子目录）
│       ├── dashboard/               #   运营仪表盘
│       └── wms/                     #   WMS 仓库作业
├── generated/                       # [自动生成]
│   ├── api/                         #   protoc-gen-dart-http 产出
│   │   ├── transport.dart
│   │   └── app/service/v1/index.dart   # ApiClient + 各 *ServiceClient + 请求/响应模型
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

> 仓储实现与抽象接口分离：`domain/*_repository.dart` 定义抽象 `XxxRepository`，`data/*_repository_impl.dart` 提供实现并通过 `createXxxRepositoryImpl()` 工厂注入。Cubit 只依赖抽象接口，不直接接触 `ServiceClient`。Feature 仓储当前消费的 ServiceClient：`ApprovalRequestServiceClient`、`AuthenticationServiceClient`、`StockQuantServiceClient`、`StockPickingServiceClient`、`WarehouseServiceClient`（其余如 `PayableServiceClient` / `PaymentServiceClient` / `PurchaseOrderServiceClient` / `FileTransferServiceClient` / `UserProfileServiceClient` 已生成但尚未被任何 feature 消费）。

---

## 五、二次开发导引

### 5.1 环境搭建

```bash
# 安装依赖
flutter pub get

# 启动 iOS 开发
flutter run -d ios

# 启动 Android 开发
flutter run -d android

# 构建生产产物
flutter build apk
flutter build ios

# 代码分析
flutter analyze

# 运行测试
flutter test
```

> **代码生成不在移动端执行**：API 客户端由后端 `.proto` 经 `protoc-gen-dart-http` 生成（后端侧 `make api`），移动端通过拉取 `lib/generated/api/` 同步；i18n `S` 类由 Flutter Intl IDE 插件（Localizely）从 ARB 生成到 `lib/generated/l10n.dart`。不存在 `build_runner` / `intl_utils` CLI / `swagger_parser` 命令。

**环境变量配置**（`.dev.env`）：

```env
API_BASE_URL="http://127.0.0.1:6700"
SSE_URL="https://127.0.0.1:6701/events"
CONNECTION_TIMEOUT=3000
RECEIVE_TIMEOUT=3000
AES_KEY="f51d66a73d8a0927"
```

> 开发 `.dev.env` 的 `API_BASE_URL` 指向本地 app BFF（Android 模拟器以 `10.0.2.2` 访问宿主 `6700` 端口）；生产 `.env` 指向 `api.erp.gowind.cloud`（gitignored，不在版本控制）。Debug 模式加载 `.dev.env`，Release 模式加载 `.env`；修改后需重启应用。

### 5.2 新增一个业务模块

新增一个完整 feature 模块（以「某业务」为例）需打通 data/domain/presentation 三层、仓储注册、路由与导航入口、i18n。参考现有 `wms` / `approval` / `dashboard` 模块的实现。

```
- [ ] Step 1: 确认 generated ApiClient 中已有目标 ServiceClient（后端 proto 已生成并同步到移动端 lib/generated/api/）
- [ ] Step 2: 实现 feature 三层（features/<module>/data/ + domain/ + presentation/）
  - [ ] domain: sealed failure（参考 wms_failure.dart）+ 值对象模型 + 抽象 Repository 接口
  - [ ] data: RemoteDataSource（适配 ServiceClient，参考 wms_remote_data_source.dart）+ RepositoryImpl（DTO→domain 映射 + ApiException.category→Failure 映射，参考 wms_repository_impl.dart）+ createXxxRepositoryImpl 工厂
  - [ ] presentation: Cubit + sealed State + Page widget
- [ ] Step 3: 注册仓储到 init.dart 的 _initTransport()（GetIt.registerLazySingleton + createXxxRepositoryImpl）
- [ ] Step 4: 注册路由（app_router.dart 的 StatefulShellRoute 新增 branch + route_names.dart + router_paths.dart）
- [ ] Step 5: 添加导航入口（home_shell.dart 的 NavigationBar 新增 NavigationDestination）
- [ ] Step 6: 添加 i18n（l10n/intl_zh_CN.arb + intl_en_US.arb，Flutter Intl 插件重新生成）
```

关键约束：Cubit 只依赖抽象 Repository 接口；Page 只渲染 State，不直接调用 Repository 或 ServiceClient；错误经 `_toFailure` 映射为领域 failure 后由 Cubit 捕获映射为 State。详见 `docs/api-guide.md` §4 与 AGENTS.md。

### 5.3 新增一种语言

**Step 1 — 创建翻译文件**

在 `lib/l10n/` 下创建新的 ARB 文件（如 `intl_ja_JP.arb`），复制现有文件并翻译。

**Step 2 — 生成代码**

由 Flutter Intl IDE 插件（Localizely）读取 `pubspec.yaml` 的 `flutter_intl:` 配置块生成 `lib/generated/l10n.dart` 的 `S` 类。配置块定义 `arb_dir` / `main_locale` / `template_arb_file` / `output_dir` / `output_file_name` / `class_name`；新增语言需在 `supportedLocales`（由 `S.delegate.supportedLocales` 提供）覆盖范围内。

> 不存在 `intl_utils` CLI 命令、不存在 `settings_page.dart` 的 `localeLabels` 列表（本项目无设置页——路由仅 login/home/approval/wms）。

### 5.4 自定义主题配色

本项目仅有一个预设种子色常量 `kDefaultSeedColor = Color(0xFF3A7CA5)`（`light_theme.dart`）。运行时种子色由用户经 `AppThemeCubit.modifySeedColor` 选择，作为 `int` 持久化到 SharedPreferences 的 `SEED_COLOR` 键。

所有使用 `theme.colorScheme.*` 的组件会自动跟随变化，因为色板由 `ColorScheme.fromSeed` 动态生成。**不存在** “8 种预设主题色” 数组、不存在设置页 `_presetColors` 列表。

### 5.5 对接不同后端

本项目前端与后端通过 REST API 通信，对接不同后端的核心修改点：

1. **`.dev.env` / `.env`** — 修改 `API_BASE_URL`（指向新后端）
2. **API 客户端代码** — 由后端 `.proto` 经 `protoc-gen-dart-http` 生成（后端侧 `make api`），移动端拉取新的 `lib/generated/api/`；不存在 `swagger_parser.yaml`、不需 `build_runner`
3. **data 层适配** — 若新后端的请求/响应结构变化，调整 `*_remote_data_source.dart` 的调用与 `*_repository_impl.dart` 的 DTO→domain 映射；若状态码语义变化，调整 `_toFailure` 的 `ApiExceptionCategory`→Failure 映射

> 认证流程由 `UnifiedAuthInterceptor`（`core/transport/http/interceptors/unified_auth_interceptor.dart`）统一处理，一般不需修改；如新后端的鉴权头格式不同，在该拦截器内调整。

### 5.6 部署

**Web 部署：**

```bash
flutter build web   # 输出到 build/web/
```

产物为 SPA 静态文件，需配置服务器 fallback 到 `index.html`。

**Nginx 配置示例：**

```nginx
server {
    listen 80;
    root /var/www/erp;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

**Android 部署：**

```bash
flutter build apk --release     # 输出 APK
flutter build appbundle --release # 输出 AAB（推荐用于 Play Store）
```

**iOS 部署：**

```bash
flutter build ios --release   # 输出到 build/ios/
```

通过 Xcode 打开 `Runner.xcworkspace`，签名后上传至 App Store Connect。

---

## 六、开发规范与注意事项

### 6.1 响应式适配规范

- 使用 `ResponsiveLayout` 组件区分 mobile/web 视图，不要在一个 build 方法中混用
- 手机端可使用 `flutter_screenutil` 的 `.w` / `.h` / `.sp` 适配
- **Web 端禁止使用 `.w` / `.h` / `.sp`**（Web 端 ScreenUtil designSize 被设为视窗尺寸，1:1 映射，使用固定值）
- 使用 `ResponsiveUtils.isMobile(context)` 判断设备类型
- 使用 `Breakpoints` 常量定义断点值，不要硬编码

### 6.2 API 仓储规范

- 业务 API 接入遵循 feature 三层（data/domain/presentation）：domain 定义抽象 Repository + sealed Failure，data 提供 RepositoryImpl + RemoteDataSource，presentation 的 Cubit 只依赖抽象接口
- 错误经 `ApiException.fromDioError`（返回带 `category` 的 `ApiException`）在 `RepositoryImpl._toFailure` 中映射为对应模块的 Failure 子类抛出；Cubit 捕获 Failure 映射为 State，Page 只渲染 State
- 分页用生成的 `PaginationPagingRequest`（`noPaging` / `pageSize` / `page`），不存在手写 `PaginationQuery`、不存在 locale 自动注入

### 6.3 路由注意事项

- 使用 `context.go()` 进行顶级路由切换（如首页 → 设置页）
- 使用 `context.push()` 进行子页面导航（如首页 → 文章详情）
- 返回按钮使用 `AppBackButton`（内置 canPop 检查 + fallback）
- 路由路径常量集中管理在 `router_paths.dart` 和 `route_names.dart`

### 6.4 平台限制

本项目为移动端定向（Android / iOS），路由为 `StatefulShellRoute.indexedStack` 三分支 + `HomeShell` 固定 3-tab 底部导航，**不存在** Web 端、不存在 `WebShellLayout`、不存在 `ShellRoute` 顶部导航壳。`ResponsiveLayout` / `ResponsiveUtils` / 三级断点作为共享基础设施存在，当前仅登录页实际使用 `isMobile` 三元分流（见 `docs/ui-adaptation.md` §7）。

### 6.6 环境变量

环境变量通过 `flutter_dotenv` 加载，在 `Environments` 类中统一导出。`.env` 文件中键名自由命名（不要求前缀），通过 `dotenv.get('KEY')` 读取。

### 6.7 代码生成

| 修改内容 | 生成方式 |
|---|---|
| proto API 定义 | 后端 `make api`（protoc-gen-dart-http）生成，移动端拉取新的 `lib/generated/api/` |
| ARB 翻译文件 | Flutter Intl IDE 插件（Localizely）按 `pubspec.yaml` 的 `flutter_intl:` 配置块生成 `lib/generated/l10n.dart` |

> 不存在 `build_runner` / `swagger_parser` / `retrofit_generator` / `freezed` / `json_serializable` 的生成命令；这些包虽列于 pubspec 但 `lib/` 实际不使用。

---

## 七、技术亮点总结

1. **一套代码跨端编译**：Flutter 框架实现 Android / iOS 统一代码库
2. **类型安全的 API 层**：后端 `.proto` 经 `protoc-gen-dart-http` 生成强类型 `*ServiceClient` + 请求/响应模型，移动端仅消费
3. **Clean architecture 分层**：每个 feature 模块 data/domain/presentation 三层，Cubit 只依赖抽象 Repository 接口，实现与抽象分离
4. **统一错误分类与领域映射**：`ApiException.fromDioError` 按 HTTP 状态码/Kratos 错误体分类（auth/business/server/network/unknown），各 feature `_toFailure` 映射为 sealed Failure，Cubit 捕获映射为 State
5. **响应式断点基础设施**：三级断点（`Breakpoints`）+ `ResponsiveLayout` / `ResponsiveUtils`，手机端 ScreenUtil 等比适配、宽屏端固定值
6. **Material 3 动态主题**：`ColorScheme.fromSeed` 基于单一用户选定种子色动态生成色板，`AppThemeCubit` 管理主题模式/种子色/locale
7. **Token AES 加密持久化**：SharedPreferences + AES 对称加密，安全存储认证凭证
8. **flutter_bloc Cubit 模式**：轻量级状态管理，主题/语言状态经顶层 `MultiBlocProvider` 全局共享，feature Cubit 经路由 builder 注入

---

> **快速开始**：`flutter pub get && flutter run -d android`（或 `-d ios`）即可运行。

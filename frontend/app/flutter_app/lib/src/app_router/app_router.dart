import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/src/core/constants/index.dart' show AppRoutePath;
import 'package:go_wind_erp/src/core/repositories/user_auth_cache.dart'
    show UserAuthCache;
import 'package:go_wind_erp/src/core/widgets/not_found_page.dart';
import 'package:go_wind_erp/src/app/home_shell.dart';
import 'package:go_wind_erp/src/app_router/route_names.dart' show RouteNames;
import 'package:go_wind_erp/src/features/auth/domain/auth_repository.dart'
    show AuthRepository;
import 'package:go_wind_erp/src/features/auth/presentation/login_cubit.dart';
import 'package:go_wind_erp/src/features/auth/presentation/login_page.dart';
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_repository.dart'
    show DashboardRepository;
import 'package:go_wind_erp/src/features/dashboard/presentation/dashboard_cubit.dart';
import 'package:go_wind_erp/src/features/dashboard/presentation/dashboard_page.dart';
import 'package:go_wind_erp/src/features/approval/domain/approval_models.dart'
    show ApprovalFilter;
import 'package:go_wind_erp/src/features/approval/domain/approval_repository.dart'
    show ApprovalRepository;
import 'package:go_wind_erp/src/features/approval/presentation/approval_cubit.dart';
import 'package:go_wind_erp/src/features/approval/presentation/approval_page.dart';
import 'package:go_wind_erp/src/features/wms/domain/wms_repository.dart'
    show WmsRepository;
import 'package:go_wind_erp/src/features/wms/presentation/wms_cubit.dart';
import 'package:go_wind_erp/src/features/wms/presentation/wms_page.dart';
import 'package:go_wind_erp/src/features/sales/domain/sales_repository.dart'
    show SalesRepository;
import 'package:go_wind_erp/src/features/sales/presentation/sales_cubit.dart';
import 'package:go_wind_erp/src/features/sales/presentation/sales_page.dart';
import 'package:go_wind_erp/src/features/sales/presentation/sales_order_detail_page.dart';
import 'package:go_wind_erp/src/features/procurement/presentation/purchase_order_detail_page.dart';

/// 路由构造。
///
/// 必须在 [UserAuthCache] / [AuthRepository] 已于 GetIt 注册（[init] 完成）后
/// 调用：[refreshListenable] 依赖 `loginStateNotifier`，[redirect] 读取 `hasLogin`，
/// 登录路由的 [LoginCubit] 依赖 [AuthRepository]。
GoRouter createAppRouter() {
  final cache = GetIt.instance<UserAuthCache>();
  final authRepository = GetIt.instance<AuthRepository>();

  return GoRouter(
    initialLocation: AppRoutePath.home,
    refreshListenable: cache.loginStateNotifier,
    redirect: (context, state) => _guard(cache, state),
    errorBuilder: (context, state) => const NotFoundPage(),
    routes: [
      GoRoute(
        name: RouteNames.login,
        path: AppRoutePath.login,
        builder: (context, state) => BlocProvider(
          create: (_) => LoginCubit(authRepository),
          child: const LoginPage(),
        ),
      ),
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) =>
            HomeShell(navigationShell: navigationShell),
        branches: [
          StatefulShellBranch(
            routes: [
              GoRoute(
                name: RouteNames.home,
                path: AppRoutePath.home,
                builder: (context, state) => BlocProvider(
                  create: (_) => DashboardCubit(
                    GetIt.instance<DashboardRepository>(),
                  )..load(),
                  child: const DashboardPage(),
                ),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                name: RouteNames.approval,
                path: AppRoutePath.approval,
                builder: (context, state) => BlocProvider(
                  create: (_) => ApprovalCubit(
                    GetIt.instance<ApprovalRepository>(),
                  )..load(ApprovalFilter.all),
                  child: const ApprovalPage(),
                ),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                name: RouteNames.wms,
                path: AppRoutePath.wms,
                builder: (context, state) => BlocProvider(
                  create: (_) => WmsCubit(
                    GetIt.instance<WmsRepository>(),
                  )..load(),
                  child: const WmsPage(),
                ),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                name: RouteNames.sales,
                path: AppRoutePath.sales,
                builder: (context, state) => BlocProvider(
                  create: (_) => SalesCubit(
                    GetIt.instance<SalesRepository>(),
                  )..load(),
                  child: const SalesPage(),
                ),
                routes: [
                  GoRoute(
                    name: RouteNames.salesDetail,
                    path: 'detail/:id',
                    builder: (context, state) {
                      final id = int.tryParse(
                            state.pathParameters['id'] ?? '',
                          ) ??
                          0;
                      return SalesOrderDetailPage(orderId: id);
                    },
                  ),
                ],
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                name: RouteNames.poDetail,
                path: '${AppRoutePath.poDetailPrefix}:id',
                builder: (context, state) {
                  final id = int.tryParse(
                        state.pathParameters['id'] ?? '',
                      ) ??
                      0;
                  return PurchaseOrderDetailPage(orderId: id);
                },
              ),
            ],
          ),
        ],
      ),
    ],
  );
}

/// 鉴权路由守卫。
///
/// - 未登录访问任意非登录路由 → 重定向至登录页。
/// - 已登录访问登录页 → 重定向至 home。
/// - 其余放行。
///
/// `loginStateNotifier` 变化时 GoRouter 经 `refreshListenable` 重新求值本守卫，
/// 从而在登录/登出瞬间完成跳转。
String? _guard(UserAuthCache cache, GoRouterState state) {
  final loggedIn = cache.hasLogin;
  final isLoginRoute = state.matchedLocation == AppRoutePath.login;
  if (!loggedIn) {
    return isLoginRoute ? null : AppRoutePath.login;
  }
  return isLoginRoute ? AppRoutePath.home : null;
}

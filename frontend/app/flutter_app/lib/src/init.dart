import 'dart:async';

import 'package:dio/dio.dart' show Dio;
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show ApiClient;

import 'package:go_wind_erp/src/core/config/environments.dart';
import 'package:go_wind_erp/src/core/repositories/init.dart' as repos;
import 'package:go_wind_erp/src/core/transport/http/index.dart'
    show DioClientTransport;
import 'package:go_wind_erp/src/core/transport/init.dart' as transport;
import 'package:go_wind_erp/src/core/transport/http/global_error_notifier.dart'
    show GlobalErrorNotifier;
import 'package:go_wind_erp/src/core/widgets/error_page.dart';
import 'package:go_wind_erp/src/features/approval/data/approval_repository_impl.dart'
    show createApprovalRepositoryImpl;
import 'package:go_wind_erp/src/features/approval/domain/approval_repository.dart'
    show ApprovalRepository;
import 'package:go_wind_erp/src/features/auth/data/auth_repository_impl.dart'
    show createAuthRepositoryImpl;
import 'package:go_wind_erp/src/features/auth/domain/auth_repository.dart'
    show AuthRepository;
import 'package:go_wind_erp/src/features/dashboard/data/dashboard_repository_impl.dart'
    show createDashboardRepositoryImpl;
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_repository.dart'
    show DashboardRepository;
import 'package:go_wind_erp/src/features/wms/data/wms_repository_impl.dart'
    show createWmsRepositoryImpl;
import 'package:go_wind_erp/src/features/wms/domain/wms_repository.dart'
    show WmsRepository;
import 'package:get_it/get_it.dart' show GetIt;

import 'init_thirdparty_plugins.dart';

/// 应用初始化
Future<void> init() async {
  WidgetsFlutterBinding.ensureInitialized();

  await Environments.init();

  await initThirdPartyPlugins();

  _initTransport();

  await repos.init();

  _initErrorWidget();
}

/// 初始化传输层
void _initTransport() {
  transport.init();

  final getIt = GetIt.instance;
  getIt.registerLazySingleton<ApiClient>(
    () => ApiClient(DioClientTransport(dio: GetIt.instance<Dio>())),
  );
  getIt.registerLazySingleton<GlobalErrorNotifier>(
    () => GlobalErrorNotifier(),
  );
  getIt.registerLazySingleton<AuthRepository>(
    () => createAuthRepositoryImpl(),
  );
  getIt.registerLazySingleton<DashboardRepository>(
    () => createDashboardRepositoryImpl(),
  );
  getIt.registerLazySingleton<WmsRepository>(
    () => createWmsRepositoryImpl(),
  );
  getIt.registerLazySingleton<ApprovalRepository>(
    () => createApprovalRepositoryImpl(),
  );
}

/// 自定义报错页面
void _initErrorWidget() {
  ErrorWidget.builder = (FlutterErrorDetails details) {
    debugPrint(details.toString());

    if (kDebugMode) {
      return ErrorWidget(details.exception);
    }

    return CustomErrorWidget(errorMessage: details.exceptionAsString());
  };
}

/// 清理
void globalDispose() {}

import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_native_splash/flutter_native_splash.dart';

import 'package:go_wind_erp/src/app.dart' show CMSApp;
import 'package:go_wind_erp/src/init.dart' show init;
import 'package:go_wind_erp/src/core/session/session_manager.dart'
    show SessionManager;
import 'package:go_wind_erp/src/core/themes/index.dart' show AppThemeCubit;

/// 程序入口
Future<void> main() async {
  final widgetsBinding = WidgetsFlutterBinding.ensureInitialized();
  FlutterNativeSplash.preserve(widgetsBinding: widgetsBinding);

  await init();

  // 启动会话刷新管理器：监听登录状态，在访问令牌过期前主动刷新。
  // 必须在 init() 完成（UserAuthCache 已注册）后启动。
  SessionManager().start();

  // 初始化完成后移除原生闪屏
  FlutterNativeSplash.remove();

  run();
}

void run() {
  // ignore: missing_provider_scope
  runApp(
    MultiBlocProvider(
      providers: [BlocProvider(create: (_) => AppThemeCubit())],
      child: const CMSApp(),
    ),
  );
}

import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart' show GoRouter;
import 'package:provider/provider.dart';

import 'package:flutter_screenutil/flutter_screenutil.dart';
import 'package:flutter_easyloading/flutter_easyloading.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

import 'package:go_wind_erp/generated/l10n.dart' as l10n;

import 'package:go_wind_erp/src/core/extensions/app_localizations_context.dart';
import 'package:go_wind_erp/src/core/constants/breakpoints.dart';
import 'package:go_wind_erp/src/core/themes/index.dart' as theme;
import 'package:go_wind_erp/src/core/utilities/logger.dart' show debug;

import 'app_router/app_router.dart' as pages;

/// 应用程序
class ERPApp extends StatefulWidget {
  const ERPApp({super.key});

  @override
  State<ERPApp> createState() => _ERPAppState();
}

class _ERPAppState extends State<ERPApp> {
  // GoRouter 必须整 app 生命周期只建一次：build 里新建会换掉 MaterialApp 的
  // routerConfig（go_router 每次 new 一个 root navigatorKey GlobalKey），
  // 整棵导航树被迫重挂载。移动端 web 键盘弹起/浏览器缩放等任何 metrics
  // 变化都会经 ScreenUtilInit 重建到这里——焦点丢失、软键盘弹起即收、
  // 页面输入内容清空（go-wind-trade 2026-11 手机软键盘闪现即收的根因）。
  late final GoRouter _router = pages.createAppRouter();

  @override
  void initState() {
    super.initState();
    // 主题 cubit 初始化是副作用，只在挂载时执行一次；不放 build
    //（build 会在每次 metrics 变化时重跑）。
    context.read<theme.AppThemeCubit>().init();
  }

  @override
  Widget build(BuildContext context) {
    return ScreenUtilInit(
      designSize: Breakpoints.designSize,
      minTextAdapt: true,
      splitScreenMode: true,
      ensureScreenSize: true,
      builder: (ctx, child) {
        // Web 端：动态将 designSize 设为当前视窗尺寸，使 .w/.h/.sp 始终 1:1
        // 用户缩小浏览器窗口时字体不会跟着缩小，完美符合 Web 阅读习惯
        if (kIsWeb) {
          ScreenUtil.init(
            ctx,
            designSize: Size(
              MediaQuery.of(ctx).size.width,
              MediaQuery.of(ctx).size.height,
            ),
            minTextAdapt: false,
          );
        }
        return _buildMaterialApp(context);
      },
      child: const SizedBox.shrink(),
    );
  }

  Widget _buildMaterialApp(BuildContext context) {
    final delegate = l10n.S.delegate;

    // 监听 Cubit 变化，获取最新的 seedColor
    final state = context.watch<theme.AppThemeCubit>().state;
    Color? seedColor;
    if (state is theme.AppThemeInitial) {
      seedColor = state.seedColor;
    } else if (state is theme.AppThemeModified) {
      seedColor = state.seedColor;
    }

    return MaterialApp.router(
      debugShowCheckedModeBanner: false,
      onGenerateTitle: (context) => context.loc.appName,
      routerConfig: _router,
      theme: theme.getLightTheme(seedColor: seedColor),
      darkTheme: theme.getDarkTheme(seedColor: seedColor),
      themeMode: context.watch<theme.AppThemeCubit>().themeMode,
      locale: context.watch<theme.AppThemeCubit>().currentLocale,
      builder: (context, routerChild) {
        // EasyLoading 作为路由的外部包装
        final easyLoadingBuilder = EasyLoading.init();
        return easyLoadingBuilder(context, routerChild);
      },
      localizationsDelegates: [
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        delegate,
      ],
      supportedLocales: l10n.S.delegate.supportedLocales,
      localeListResolutionCallback: (locales, supportedLocales) {
        debug('locale list: $locales');
        if (locales == null || locales.isEmpty) {
          return supportedLocales.first;
        }
        for (final locale in locales) {
          // 精确匹配
          if (supportedLocales.contains(locale)) return locale;
          // 语言代码匹配（忽略国家代码差异，如 zh-CN → zh_CN）
          final match = supportedLocales.where(
            (sl) => sl.languageCode == locale.languageCode,
          );
          if (match.isNotEmpty) return match.first;
        }
        return supportedLocales.first;
      },
    );
  }
}

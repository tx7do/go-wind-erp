import 'package:dio/dio.dart';

import 'package:go_wind_erp/src/core/config/environments.dart';
import 'package:go_wind_erp/src/core/transport/http/interceptors/unified_auth_interceptor.dart';

/// 配置选项
void _configureOptions(Dio dio) {
  dio.options.baseUrl = Environments.apiBaseUrl;
  dio.options.connectTimeout = Environments.connectionTimeout;
  dio.options.receiveTimeout = Environments.receiveTimeout;
  dio.options.responseType = ResponseType.json;
  dio.options.contentType = Headers.jsonContentType;
}

/// 注册默认拦截器
void _configureInterceptors(Dio dio) {
  // 统一鉴权与错误拦截器：附带 Bearer 令牌（登录端点除外），并将 kratos
  // 错误体封装为 ApiException；鉴权类错误触发本地登出，非预期错误触发全局
  // 错误通知。主动令牌刷新由 SessionManager 负责，不在此处反应式处理。
  dio.interceptors.add(UnifiedAuthInterceptor());
}

Dio createDio() {
  final Dio dio = Dio();

  _configureOptions(dio);
  _configureInterceptors(dio);

  return dio;
}

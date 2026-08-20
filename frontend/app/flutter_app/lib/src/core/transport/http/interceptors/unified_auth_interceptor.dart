import 'package:dio/dio.dart';
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/src/core/repositories/user_auth_cache.dart'
    show UserAuthCache;
import 'package:go_wind_erp/src/core/transport/http/api_exception.dart'
    show ApiException;
import 'package:go_wind_erp/src/core/transport/http/global_error_notifier.dart'
    show GlobalErrorNotifier;

/// 登录端点路径——该请求不附带 Bearer 令牌（登录时尚无令牌）。
const String kLoginPath = '/app/v1/login';

/// 统一鉴权与错误拦截器。
///
/// - onRequest：为除登录端点外的所有请求附带 `Authorization: Bearer <access>`。
///   多租户上下文已在登录时经 `tenant_code` 绑定进 JWT，无需每请求附带租户头。
/// - onError：将 kratos 错误体解析为 [ApiException] 并附在 [DioException.error]
///   上透传；鉴权类错误（401/403-鉴权拒绝）触发本地登出（清除令牌 → 登录状态
///   通知器触发 → 路由重定向至登录）；非预期错误（5xx/网络/未知）触发全局
///   错误通知。不在 401 时做“反应式刷新”——刷新端点本身要求有效的访问令牌，
///   故刷新必须由 [SessionManager] 在令牌过期前主动进行。
class UnifiedAuthInterceptor extends Interceptor {
  UserAuthCache _cache() => GetIt.instance<UserAuthCache>();

  GlobalErrorNotifier _notifier() => GetIt.instance<GlobalErrorNotifier>();

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    if (options.path != kLoginPath) {
      final token = _cache().accessToken;
      if (token != null && token.isNotEmpty) {
        options.headers['Authorization'] = 'Bearer $token';
      }
    }
    handler.next(options);
  }

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    final api = ApiException.fromDioError(err);
    if (api.isAuthError) {
      // 会话失效：清除令牌，登录状态通知器将驱动路由重定向至登录页。
      await _cache().clearTokens();
    } else if (api.isUnexpected) {
      _notifier().fire();
    }
    final wrapped = DioException(
      requestOptions: err.requestOptions,
      response: err.response,
      type: err.type,
      error: api,
      stackTrace: err.stackTrace,
      message: err.message,
    );
    handler.next(wrapped);
  }
}

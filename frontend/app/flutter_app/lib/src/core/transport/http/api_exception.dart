import 'package:dio/dio.dart';

/// 传输层错误分类。
///
/// 由 [ApiException.fromDioError] 依据 HTTP 状态码与 kratos 错误体的
/// `reason` 字段判定。领域层据此映射为业务失败类型。
enum ApiExceptionCategory {
  /// 401，或 403 且 reason 为 UNAUTHORIZED/FORBIDDEN —— 会话失效。
  auth,

  /// 4xx 业务错误（凭证无效、参数错误等）。
  business,

  /// 5xx 服务端错误。
  server,

  /// 无响应：连接失败、超时、取消。
  network,

  /// 无法判定。
  unknown,
}

/// 统一的 API 异常。
///
/// 后端按 kratos 约定在非 2xx 响应中返回 `{code, reason, message, metadata}`
/// JSON。本类将该结构解析为带分类的异常，附加到 [DioException.error] 上
/// 透传给调用方，使领域层无需直接处理 [DioException] 细节。
///
/// 注意：成功响应后端不包 envelope，因此本类仅作用于错误路径。
class ApiException implements Exception {
  final int? statusCode;
  final String? reason;
  final String message;
  final ApiExceptionCategory category;

  bool get isAuthError => category == ApiExceptionCategory.auth;

  /// 是否为“非预期”错误（服务端/网络/未知）——此类错误统一向全局通知器上报，
  /// 由调用方 UI 专门处理的 4xx 业务错误不上报。
  bool get isUnexpected =>
      category == ApiExceptionCategory.server ||
      category == ApiExceptionCategory.network ||
      category == ApiExceptionCategory.unknown;

  ApiException({
    required this.statusCode,
    required this.reason,
    required this.message,
    required this.category,
  });

  /// 从 [DioException] 构造 [ApiException]。
  ///
  /// 若该 [DioException] 已由统一拦截器封装（即 [error] 已是 [ApiException]），
  /// 直接返回之，避免重复解析。
  static ApiException fromDioError(DioException err) {
    if (err.error is ApiException) {
      return err.error as ApiException;
    }
    final resp = err.response;
    if (resp == null) {
      final net = err.type == DioExceptionType.connectionTimeout ||
          err.type == DioExceptionType.sendTimeout ||
          err.type == DioExceptionType.receiveTimeout ||
          err.type == DioExceptionType.connectionError;
      return ApiException(
        statusCode: null,
        reason: null,
        message: err.message ?? 'network error',
        category: net
            ? ApiExceptionCategory.network
            : ApiExceptionCategory.unknown,
      );
    }
    String? reason;
    String message = err.message ?? 'unknown error';
    if (resp.data is Map<String, dynamic>) {
      final d = resp.data as Map<String, dynamic>;
      reason = d['reason'] as String?;
      final m = d['message'] as String?;
      if (m != null) message = m;
    }
    final code = resp.statusCode;
    ApiExceptionCategory cat;
    if (code == 401) {
      cat = ApiExceptionCategory.auth;
    } else if (code == 403 &&
        (reason == 'UNAUTHORIZED' || reason == 'FORBIDDEN')) {
      cat = ApiExceptionCategory.auth;
    } else if (code != null && code >= 400 && code < 500) {
      cat = ApiExceptionCategory.business;
    } else if (code != null && code >= 500) {
      cat = ApiExceptionCategory.server;
    } else {
      cat = ApiExceptionCategory.unknown;
    }
    return ApiException(
      statusCode: code,
      reason: reason,
      message: message,
      category: cat,
    );
  }
}

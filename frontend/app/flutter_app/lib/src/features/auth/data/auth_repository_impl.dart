import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show ApiClient, AuthenticationServiceV1LoginResponse;
import 'package:go_wind_erp/src/core/repositories/user_auth_cache.dart'
    show UserAuthCache;
import 'package:go_wind_erp/src/core/transport/http/api_exception.dart'
    show ApiException, ApiExceptionCategory;
import 'package:go_wind_erp/src/features/auth/data/auth_remote_data_source.dart'
    show AuthRemoteDataSource;
import 'package:go_wind_erp/src/features/auth/domain/auth_failure.dart';
import 'package:go_wind_erp/src/features/auth/domain/auth_repository.dart';
import 'package:go_wind_erp/src/features/auth/domain/auth_session.dart';
import 'package:go_wind_erp/src/features/auth/domain/login_credentials.dart';

/// [AuthRepository] 的 data 层实现。
///
/// 职责：调用 [AuthRemoteDataSource] 获取后端响应，将其映射为领域
/// [AuthSession] 并写入会话缓存；将传输层 [DioException]（已由统一
/// 拦截器封装为 [ApiException]）映射为 [AuthFailure] 子类抛出。
class AuthRepositoryImpl implements AuthRepository {
  final AuthRemoteDataSource _dataSource;
  final UserAuthCache _cache;

  AuthRepositoryImpl(this._dataSource, this._cache);

  @override
  Future<AuthSession> login(LoginCredentials credentials) async {
    try {
      final resp = await _dataSource.login(credentials);
      final session = _toSession(resp);
      await _cache.saveAuthInfo(
        session.accessToken,
        refreshToken: session.refreshToken,
      );
      return session;
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<AuthSession> refresh() async {
    final refreshToken = _cache.refreshToken;
    if (refreshToken == null || refreshToken.isEmpty) {
      throw const SessionExpiredFailure();
    }
    try {
      final resp = await _dataSource.refresh(refreshToken);
      final session = _toSession(resp);
      await _cache.saveAuthInfo(
        session.accessToken,
        refreshToken: session.refreshToken,
      );
      return session;
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<void> logout() async {
    try {
      await _dataSource.logout();
    } catch (_) {
      // 服务端撤销失败不阻塞本地登出。
    }
    await _cache.clearTokens();
  }

  AuthSession _toSession(AuthenticationServiceV1LoginResponse resp) {
    final access = resp.access_token;
    if (access == null || access.isEmpty) {
      throw const UnknownFailure();
    }
    return AuthSession(
      accessToken: access,
      refreshToken: resp.refresh_token,
    );
  }

  /// 将统一拦截器封装过的 [DioException]（其 `error` 为 [ApiException]）
  /// 映射为领域 [AuthFailure]。
  AuthFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);
    switch (api.category) {
      case ApiExceptionCategory.auth:
        return const SessionExpiredFailure();
      case ApiExceptionCategory.business:
        return const InvalidCredentialsFailure();
      case ApiExceptionCategory.server:
      case ApiExceptionCategory.network:
        return const NetworkFailure();
      case ApiExceptionCategory.unknown:
        return const UnknownFailure();
    }
  }
}

/// 供 [init.dart] 注册时构造 [AuthRepositoryImpl]。
AuthRepositoryImpl createAuthRepositoryImpl() {
  return AuthRepositoryImpl(
    AuthRemoteDataSource(GetIt.instance<ApiClient>()),
    GetIt.instance<UserAuthCache>(),
  );
}

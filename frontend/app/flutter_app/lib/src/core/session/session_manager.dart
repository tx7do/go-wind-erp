import 'dart:async';

import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/src/core/repositories/user_auth_cache.dart'
    show UserAuthCache;
import 'package:go_wind_erp/src/features/auth/domain/auth_failure.dart'
    show AuthFailure;
import 'package:go_wind_erp/src/features/auth/domain/auth_repository.dart'
    show AuthRepository;

/// 会话刷新管理器。
///
/// 后端 `/app/v1/refresh-token` 未经白名单豁免，要求请求附带“仍有效”的
/// 访问令牌。因此无法在 401（令牌已过期）时反应式刷新——必须在令牌过期前
/// 主动刷新。本类监听 [UserAuthCache.loginStateNotifier]：登录态为真时，
/// 依据访问令牌的过期时间调度一次 [AuthRepository.refresh]；登录态为假
/// （登出/会话失效）时取消调度。
///
/// 安全提前量 [lead] 取 60 秒。刷新失败一律清除令牌（登出）。
class SessionManager {
  static const Duration lead = Duration(seconds: 60);

  Timer? _timer;

  UserAuthCache get _cache => GetIt.instance<UserAuthCache>();
  AuthRepository get _auth => GetIt.instance<AuthRepository>();

  /// 开始监听登录状态并据当前状态调度。
  void start() {
    _cache.loginStateNotifier.addListener(_onChanged);
    _reschedule();
  }

  void _onChanged() => _reschedule();

  void _reschedule() {
    _timer?.cancel();
    _timer = null;
    if (!_cache.hasLogin) return;
    final exp = _cache.accessTokenExpiresAt;
    if (exp == null) return;
    final fireAt = exp.subtract(lead);
    final delay = fireAt.difference(DateTime.now());
    if (delay.isNegative || delay.inSeconds < 1) {
      _doRefresh();
      return;
    }
    _timer = Timer(delay, _doRefresh);
  }

  Future<void> _doRefresh() async {
    _timer = null;
    if (!_cache.hasLogin) return;
    try {
      await _auth.refresh();
      // 新令牌已写入缓存（含新过期时间），按新过期时间重新调度。
      _reschedule();
    } on AuthFailure {
      await _cache.clearTokens();
    } catch (_) {
      await _cache.clearTokens();
    }
  }
}

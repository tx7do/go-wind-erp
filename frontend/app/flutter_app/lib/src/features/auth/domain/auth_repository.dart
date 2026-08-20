import 'auth_failure.dart';
import 'auth_session.dart';
import 'login_credentials.dart';

/// 认证仓储抽象。
///
/// Domain 层接口，由 data 层 [AuthRepositoryImpl] 实现。
/// Presentation 层（[LoginCubit]）与 [SessionManager] 仅依赖此抽象，
/// 不感知具体的 HTTP / 生成客户端实现。
abstract class AuthRepository {
  /// 密码模式登录。
  ///
  /// 成功时令牌已被写入会话缓存并触发登录状态通知；
  /// 失败时抛出 [AuthFailure] 的具体子类。
  Future<AuthSession> login(LoginCredentials credentials);

  /// 刷新访问令牌。
  ///
  /// 依赖当前仍有效的访问令牌（由拦截器以 Bearer 头附带），
  /// 与刷新令牌一同提交。服务端原子地作废旧令牌对并签发新对。
  Future<AuthSession> refresh();

  /// 登出。
  ///
  /// 尽力通知服务端撤销令牌，随后无条件清除本地会话缓存。
  Future<void> logout();
}

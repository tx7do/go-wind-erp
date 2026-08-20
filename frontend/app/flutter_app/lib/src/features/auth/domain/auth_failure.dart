/// 认证领域失败类型。
///
/// 由 data 层 [AuthRepositoryImpl] 根据 [ApiException] 的分类映射产生。
/// 当前 UI 对所有登录失败统一展示同一条提示文案，细分类型保留以便后续细化。
sealed class AuthFailure implements Exception {
  const AuthFailure();
}

/// 会话失效（401 / 403-鉴权拒绝）：令牌缺失、过期或被撤销。
final class SessionExpiredFailure extends AuthFailure {
  const SessionExpiredFailure();
}

/// 凭证无效（400-BAD_REQUEST）：用户名/密码错误、租户编号无效等。
final class InvalidCredentialsFailure extends AuthFailure {
  const InvalidCredentialsFailure();
}

/// 网络或服务端错误（5xx / 连接失败 / 超时）。
final class NetworkFailure extends AuthFailure {
  const NetworkFailure();
}

/// 未分类错误。
final class UnknownFailure extends AuthFailure {
  const UnknownFailure();
}

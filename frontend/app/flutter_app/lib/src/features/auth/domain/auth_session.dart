/// 认证会话值对象。
///
/// 仅持有服务端签发的令牌字符串。令牌的过期时间不由领域层管理——
/// 传输/会话基础设施（[UserAuthCache] / [SessionManager]）会从令牌本体
/// 解析过期时间并据此调度刷新。
class AuthSession {
  final String accessToken;
  final String? refreshToken;

  const AuthSession({required this.accessToken, this.refreshToken});
}

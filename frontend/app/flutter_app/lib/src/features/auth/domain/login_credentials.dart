/// 登录凭证值对象。
///
/// 纯领域对象，不包含任何传输层细节（密码在此处以明文持有，
/// 加密发生在 data 层的 [AuthRemoteDataSource] 中）。
/// [tenantCode] 为空字符串表示平台登录（租户编号留空）。
class LoginCredentials {
  final String username;
  final String password;
  final String tenantCode;

  const LoginCredentials({
    required this.username,
    required this.password,
    this.tenantCode = '',
  });
}

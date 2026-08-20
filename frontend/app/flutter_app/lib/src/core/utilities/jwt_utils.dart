import 'package:jose/jose.dart';

import 'package:go_wind_erp/src/core/utilities/logger.dart' show fatal;

/// JWT 令牌工具类
class JwtUtils {
  JwtUtils._(); // Private constructor to prevent instantiation

  /// 验证token是否过期
  static bool isTokenExpired(String? token) {
    final exp = expiryOf(token);
    if (exp == null) {
      return false;
    }
    return exp.isBefore(DateTime.now());
  }

  /// 解析 JWT 的 `exp` 声明为 [DateTime]，无声明或解析失败返回 null。
  ///
  /// 仅供 [SessionManager] 据以调度主动刷新——不验证签名（签名由后端
  /// `AuthenticationService.ValidateToken` 校验）。
  static DateTime? expiryOf(String? token) {
    if (token == null || token.isEmpty) return null;
    try {
      final jwt = JsonWebToken.unverified(token);
      return jwt.claims.expiry;
    } catch (e) {
      fatal('parse jwt payload failed: $e');
      return null;
    }
  }
}

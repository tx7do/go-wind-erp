import 'package:encrypt/encrypt.dart' as encrypt;
import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show
        ApiClient,
        AuthenticationServiceClient,
        AuthenticationServiceV1GrantType,
        AuthenticationServiceV1LoginRequest,
        AuthenticationServiceV1LoginResponse;
import 'package:go_wind_erp/src/core/config/environments.dart';
import 'package:go_wind_erp/src/features/auth/domain/login_credentials.dart';

/// 认证远程数据源。
///
/// 适配 protoc-gen-dart-http 生成的 [AuthenticationServiceClient]，
/// 将领域 [LoginCredentials] 转换为后端 `/app/v1/*` 契约所需的请求体。
/// 密码在此处按后端约定做 AES-CBC 加密（密钥与 IV 均取自 `AES_KEY`，
/// 必须与后端 `crypto.DefaultAESKey` 一致）。
class AuthRemoteDataSource {
  final AuthenticationServiceClient _client;

  AuthRemoteDataSource(ApiClient api)
      : _client = api.authenticationService;

  /// 密码模式登录 → `POST /app/v1/login`。
  Future<AuthenticationServiceV1LoginResponse> login(
    LoginCredentials credentials,
  ) async {
    final request = AuthenticationServiceV1LoginRequest(
      grant_type: AuthenticationServiceV1GrantType.password,
      username: credentials.username,
      password: _encryptPassword(credentials.password),
      tenant_code:
          credentials.tenantCode.isEmpty ? null : credentials.tenantCode,
    );
    return _client.login(request);
  }

  /// 刷新令牌 → `POST /app/v1/refresh-token`。
  ///
  /// 访问令牌由统一鉴权拦截器以 Bearer 头附带；刷新令牌置于请求体。
  Future<AuthenticationServiceV1LoginResponse> refresh(
    String refreshToken,
  ) async {
    final request = AuthenticationServiceV1LoginRequest(
      grant_type: AuthenticationServiceV1GrantType.refreshToken,
      refresh_token: refreshToken,
    );
    return _client.refreshToken(request);
  }

  /// 登出 → `POST /app/v1/logout`。最佳努力，失败由仓储吞掉。
  Future<void> logout() async {
    await _client.logout({});
  }

  /// AES-CBC + PKCS7 加密密码，输出 Base64。
  ///
  /// 与后端 `tx7do/go-utils/crypto` 的 `AesEncrypt` 对齐：密钥与 IV 均为
  /// `AES_KEY`（16 字节，须与后端 `DefaultAESKey` 一致）。
  String _encryptPassword(String password) {
    final key = Environments.aesKey;
    final keyBytes = encrypt.Key.fromUtf8(key);
    final iv = encrypt.IV.fromUtf8(key);
    final encrypter = encrypt.Encrypter(
      encrypt.AES(keyBytes, mode: encrypt.AESMode.cbc, padding: 'PKCS7'),
    );
    return encrypter.encrypt(password, iv: iv).base64;
  }
}

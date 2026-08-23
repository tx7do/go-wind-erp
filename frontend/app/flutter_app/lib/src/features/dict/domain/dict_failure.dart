/// 字典查询领域失败类型。
///
/// 由 data 层 [DictRepositoryImpl] 根据 [ApiException] 的分类映射产生。
sealed class DictFailure implements Exception {
  const DictFailure();
}

/// 凭证/会话问题（401 / 403-鉴权拒绝）。
final class DictUnauthorizedFailure extends DictFailure {
  const DictUnauthorizedFailure();
}

/// 业务校验失败（400-BAD_REQUEST）：typeCode 缺失等。
final class DictInvalidInputFailure extends DictFailure {
  const DictInvalidInputFailure();
}

/// 网络或服务端错误（5xx / 连接失败 / 超时）。
final class DictNetworkFailure extends DictFailure {
  const DictNetworkFailure();
}

/// 未分类错误。
final class DictUnknownFailure extends DictFailure {
  const DictUnknownFailure();
}

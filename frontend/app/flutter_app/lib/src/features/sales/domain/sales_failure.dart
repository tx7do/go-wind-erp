/// 销售领域失败类型。
///
/// 由 data 层 [SalesRepositoryImpl] 根据 [ApiException] 的分类映射产生；
/// 退货数量超限等业务校验失败归入 [SalesInvalidInputFailure]。
sealed class SalesFailure implements Exception {
  const SalesFailure();
}

/// 凭证/会话问题（401 / 403-鉴权拒绝）。
final class SalesUnauthorizedFailure extends SalesFailure {
  const SalesUnauthorizedFailure();
}

/// 业务校验失败（400-BAD_REQUEST）：参数缺失、退数超已履约数等。
final class SalesInvalidInputFailure extends SalesFailure {
  const SalesInvalidInputFailure();
}

/// 网络或服务端错误（5xx / 连接失败 / 超时）。
final class SalesNetworkFailure extends SalesFailure {
  const SalesNetworkFailure();
}

/// 未分类错误。
final class SalesUnknownFailure extends SalesFailure {
  const SalesUnknownFailure();
}

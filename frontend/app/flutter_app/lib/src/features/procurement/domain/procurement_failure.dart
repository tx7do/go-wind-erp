/// 采购领域失败类型。
///
/// 由 data 层 [ProcurementRepositoryImpl] 根据 [ApiException] 分类映射产生。
sealed class ProcurementFailure implements Exception {
  const ProcurementFailure();
}

/// 凭证/会话问题（401 / 403-鉴权拒绝）。
final class ProcurementUnauthorizedFailure extends ProcurementFailure {
  const ProcurementUnauthorizedFailure();
}

/// 业务校验失败（400-BAD_REQUEST）。
final class ProcurementInvalidInputFailure extends ProcurementFailure {
  const ProcurementInvalidInputFailure();
}

/// 网络或服务端错误（5xx / 连接失败 / 超时）。
final class ProcurementNetworkFailure extends ProcurementFailure {
  const ProcurementNetworkFailure();
}

/// 未分类错误。
final class ProcurementUnknownFailure extends ProcurementFailure {
  const ProcurementUnknownFailure();
}

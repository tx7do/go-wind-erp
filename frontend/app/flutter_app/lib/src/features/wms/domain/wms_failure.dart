/// WMS 领域失败类型。
///
/// 由 data 层 [WmsRepositoryImpl] 根据 [ApiException] 的分类映射产生；
/// 提交流水时的业务校验失败（如数量算术不自洽）归入 [WmsInvalidInputFailure]。
sealed class WmsFailure implements Exception {
  const WmsFailure();
}

/// 凭证/会话问题（401 / 403-鉴权拒绝）。
final class WmsUnauthorizedFailure extends WmsFailure {
  const WmsUnauthorizedFailure();
}

/// 业务校验失败（400-BAD_REQUEST）：数量算术不自洽、参数缺失等。
final class WmsInvalidInputFailure extends WmsFailure {
  const WmsInvalidInputFailure();
}

/// 网络或服务端错误（5xx / 连接失败 / 超时）。
final class WmsNetworkFailure extends WmsFailure {
  const WmsNetworkFailure();
}

/// 未分类错误。
final class WmsUnknownFailure extends WmsFailure {
  const WmsUnknownFailure();
}

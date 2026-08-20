/// 看板领域失败类型。
///
/// 由 data 层 [DashboardRepositoryImpl] 根据 [ApiException] 的分类映射产生。
sealed class DashboardFailure implements Exception {
  const DashboardFailure();
}

/// 凭证/会话问题（401 / 403-鉴权拒绝）。
final class DashboardUnauthorizedFailure extends DashboardFailure {
  const DashboardUnauthorizedFailure();
}

/// 业务校验失败（400-BAD_REQUEST）。
final class DashboardInvalidInputFailure extends DashboardFailure {
  const DashboardInvalidInputFailure();
}

/// 网络或服务端错误（5xx / 连接失败 / 超时）。
final class DashboardNetworkFailure extends DashboardFailure {
  const DashboardNetworkFailure();
}

/// 未分类错误。
final class DashboardUnknownFailure extends DashboardFailure {
  const DashboardUnknownFailure();
}

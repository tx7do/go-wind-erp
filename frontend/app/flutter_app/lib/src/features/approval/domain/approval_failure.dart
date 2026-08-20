/// 审批领域失败类型。
///
/// 由 data 层 [ApprovalRepositoryImpl] 根据 [ApiException] 的分类映射产生；
/// 状态机拒绝（重复审批/撤销他人请求）归入 [ApprovalForbiddenFailure]。
sealed class ApprovalFailure implements Exception {
  const ApprovalFailure();
}

/// 凭证/会话问题（401 / 403-鉴权拒绝）。
final class ApprovalUnauthorizedFailure extends ApprovalFailure {
  const ApprovalUnauthorizedFailure();
}

/// 状态机或权限拒绝（403-FORBIDDEN）：非 PENDING 状态被审批、撤销他人请求。
final class ApprovalForbiddenFailure extends ApprovalFailure {
  const ApprovalForbiddenFailure();
}

/// 业务校验失败（400-BAD_REQUEST / 409-CONFLICT）：参数缺失、并发状态变更。
final class ApprovalInvalidInputFailure extends ApprovalFailure {
  const ApprovalInvalidInputFailure();
}

/// 网络或服务端错误（5xx / 连接失败 / 超时）。
final class ApprovalNetworkFailure extends ApprovalFailure {
  const ApprovalNetworkFailure();
}

/// 未分类错误。
final class ApprovalUnknownFailure extends ApprovalFailure {
  const ApprovalUnknownFailure();
}

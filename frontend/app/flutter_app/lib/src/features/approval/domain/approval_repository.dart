import 'package:go_wind_erp/src/features/approval/domain/approval_models.dart';

/// 审批中心仓储抽象。
///
/// presentation 层仅依赖本接口；实现见 data 层 [ApprovalRepositoryImpl]。
abstract class ApprovalRepository {
  /// 按状态筛选拉取审批列表（[filter] 为 null 或 all 时不过滤）。
  Future<List<ApprovalInfo>> listRequests(ApprovalFilter filter);

  /// 通过审批（仅 PENDING；[comment] 可选意见）。
  Future<void> approve(int id, {String? comment});

  /// 驳回审批（仅 PENDING；[comment] 可选理由）。
  Future<void> reject(int id, {String? comment});

  /// 撤销审批（仅 PENDING 且仅申请人本人）。
  Future<void> cancel(int id);
}

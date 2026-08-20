/// 审批领域模型。

/// 审批请求（列表/详情展示项）。
class ApprovalInfo {
  final int id;
  final String title;
  final String bizType;
  final String bizRef;
  final String summary;

  /// PENDING / APPROVED / REJECTED / CANCELLED（后端原始字符串）。
  final String status;

  final int applicantId;
  final int? approverId;
  final String? comment;
  final String? createdAt;

  const ApprovalInfo({
    required this.id,
    required this.title,
    required this.bizType,
    required this.bizRef,
    required this.summary,
    required this.status,
    required this.applicantId,
    this.approverId,
    this.comment,
    this.createdAt,
  });
}

/// 审批状态筛选。
enum ApprovalFilter { all, pending, approved, rejected, cancelled }

extension ApprovalFilterWire on ApprovalFilter {
  /// 作为 list filter 的状态值；all 不产生状态过滤。
  String? get statusValue => switch (this) {
        ApprovalFilter.all => null,
        ApprovalFilter.pending => 'PENDING',
        ApprovalFilter.approved => 'APPROVED',
        ApprovalFilter.rejected => 'REJECTED',
        ApprovalFilter.cancelled => 'CANCELLED',
      };
}

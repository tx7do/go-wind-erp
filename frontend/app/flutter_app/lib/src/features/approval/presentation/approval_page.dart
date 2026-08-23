import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:get_it/get_it.dart';

import 'package:go_wind_erp/generated/l10n.dart';
import 'package:go_wind_erp/src/features/approval/domain/approval_models.dart';
import 'package:go_wind_erp/src/features/approval/presentation/approval_cubit.dart';
import 'package:go_wind_erp/src/features/dict/domain/dict_failure.dart';
import 'package:go_wind_erp/src/features/dict/domain/dict_label_resolver.dart';
import 'package:go_wind_erp/src/features/dict/domain/dict_models.dart';
import 'package:go_wind_erp/src/features/dict/domain/dict_repository.dart';

/// 审批中心页。
///
/// 状态筛选（全部/待审批/已通过/已驳回/已取消）→ 请求列表；待审批项提供
/// 通过/驳回（弹窗填意见），申请人本人可撤销。
class ApprovalPage extends StatelessWidget {
  const ApprovalPage({super.key});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(loc.navApproval)),
      body: const _ApprovalBody(),
    );
  }
}

/// 审批列表主体。
///
/// 在 [initState] 拉取一次 `APPROVAL_BIZ_TYPE` 字典条目（条目内含全量 i18n
/// 标签，按 typeCode 进程内缓存），后续 [build] 中对 bizType 的标签解析为
/// 同步操作——经 [DictLabelResolver.labelByValue] 用当前 locale 从已缓存
/// 条目取展示文案，未命中返回空串。locale 变化会触发 rebuild，标签随之
/// 刷新，无需重新拉取字典。
class _ApprovalBody extends StatefulWidget {
  const _ApprovalBody();

  @override
  State<_ApprovalBody> createState() => _ApprovalBodyState();
}

class _ApprovalBodyState extends State<_ApprovalBody> {
  List<DictEntryInfo> _dictEntries = const [];
  bool _dictLoaded = false;

  @override
  void initState() {
    super.initState();
    _loadDictEntries();
  }

  Future<void> _loadDictEntries() async {
    try {
      final entries = await GetIt.instance<DictRepository>()
          .listByTypeCode('APPROVAL_BIZ_TYPE');
      if (!mounted) return;
      setState(() {
        _dictEntries = entries;
        _dictLoaded = true;
      });
    } on DictFailure {
      if (!mounted) return;
      setState(() {
        _dictEntries = const [];
        _dictLoaded = true;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    if (!_dictLoaded) {
      return const Center(child: CircularProgressIndicator());
    }
    return BlocBuilder<ApprovalCubit, ApprovalState>(
      builder: (context, state) {
        if (state is ApprovalReady) {
          return Column(
            children: [
              _buildFilterChips(context, loc, state),
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () => context.read<ApprovalCubit>().refresh(),
                  child: state.requests.isEmpty
                      ? ListView(
                          physics: const AlwaysScrollableScrollPhysics(),
                          children: [
                            Padding(
                              padding: const EdgeInsets.all(24),
                              child: Center(child: Text(loc.approvalEmpty)),
                            ),
                          ],
                        )
                      : ListView.builder(
                          physics: const AlwaysScrollableScrollPhysics(),
                          itemCount: state.requests.length,
                          itemBuilder: (_, i) => _buildTile(
                            context,
                            loc,
                            state.requests[i],
                            _dictEntries,
                            state.acting,
                          ),
                        ),
                ),
              ),
            ],
          );
        }
        if (state is ApprovalFailureState) {
          return Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(loc.loadFailed),
                const SizedBox(height: 12),
                FilledButton(
                  onPressed: () =>
                      context.read<ApprovalCubit>().load(ApprovalFilter.all),
                  child: Text(loc.retry),
                ),
              ],
            ),
          );
        }
        return const Center(child: CircularProgressIndicator());
      },
    );
  }

  Widget _buildFilterChips(
    BuildContext context,
    S loc,
    ApprovalReady state,
  ) {
    return SizedBox(
      height: 48,
      child: ListView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        children: [
          for (final filter in ApprovalFilter.values)
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: ChoiceChip(
                label: Text(_filterLabel(loc, filter)),
                selected: state.filter == filter,
                onSelected: (_) =>
                    context.read<ApprovalCubit>().setFilter(filter),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildTile(
    BuildContext context,
    S loc,
    ApprovalInfo item,
    List<DictEntryInfo> dictEntries,
    bool acting,
  ) {
    final isPending = item.status == 'PENDING';
    final bizTypeLabel = DictLabelResolver.labelByValue(
      item.bizType,
      dictEntries,
      Localizations.localeOf(context),
    );
    final parts = <String>[];
    if (bizTypeLabel.isNotEmpty) parts.add(bizTypeLabel);
    final ref = item.bizRef.trim();
    if (ref.isNotEmpty) parts.add(ref);
    final summary = item.summary.trim();
    final subtitleText = parts.isEmpty
        ? summary
        : summary.isEmpty
            ? parts.join(' · ')
            : '${parts.join(' · ')}\n$summary';
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: ListTile(
        title: Row(
          children: [
            Expanded(child: Text(item.title, overflow: TextOverflow.ellipsis)),
            _statusChip(loc, item.status),
          ],
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(
            subtitleText,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
        ),
        isThreeLine: true,
        trailing: isPending
            ? Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  IconButton(
                    tooltip: loc.approvalReject,
                    icon: const Icon(Icons.close),
                    onPressed: acting
                        ? null
                        : () => _confirmAction(
                              context,
                              loc,
                              loc.approvalReject,
                              (comment) => context
                                  .read<ApprovalCubit>()
                                  .reject(
                                    item.id,
                                    comment: comment,
                                    onRejected: (_) => _showRejected(
                                      context,
                                      loc,
                                    ),
                                  ),
                            ),
                  ),
                  IconButton(
                    tooltip: loc.approvalApprove,
                    icon: const Icon(Icons.check),
                    color: Colors.green,
                    onPressed: acting
                        ? null
                        : () => _confirmAction(
                              context,
                              loc,
                              loc.approvalApprove,
                              (comment) => context
                                  .read<ApprovalCubit>()
                                  .approve(
                                    item.id,
                                    comment: comment,
                                    onRejected: (_) => _showRejected(
                                      context,
                                      loc,
                                    ),
                                  ),
                            ),
                  ),
                ],
              )
            : null,
        onLongPress: isPending
            ? () => _confirmCancel(context, loc, item)
            : null,
      ),
    );
  }

  Widget _statusChip(S loc, String status) {
    final (label, color) = switch (status) {
      'PENDING' => (loc.approvalStatusPending, Colors.orange),
      'APPROVED' => (loc.approvalStatusApproved, Colors.green),
      'REJECTED' => (loc.approvalStatusRejected, Colors.red),
      'CANCELLED' => (loc.approvalStatusCancelled, Colors.grey),
      _ => ('', Colors.grey),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(label, style: TextStyle(color: color, fontSize: 12)),
    );
  }

  String _filterLabel(S loc, ApprovalFilter filter) => switch (filter) {
        ApprovalFilter.all => loc.approvalFilterAll,
        ApprovalFilter.pending => loc.approvalStatusPending,
        ApprovalFilter.approved => loc.approvalStatusApproved,
        ApprovalFilter.rejected => loc.approvalStatusRejected,
        ApprovalFilter.cancelled => loc.approvalStatusCancelled,
      };

  /// 弹窗填意见后执行审批动作；业务拒绝（状态机/所有权）以 SnackBar 提示。
  void _confirmAction(
    BuildContext context,
    S loc,
    String title,
    void Function(String? comment) onConfirm,
  ) {
    final controller = TextEditingController();
    showDialog<void>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(title),
        content: TextField(
          controller: controller,
          decoration: InputDecoration(
            labelText: loc.approvalComment,
            border: const OutlineInputBorder(),
            isDense: true,
          ),
          maxLines: 2,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: Text(loc.cancel),
          ),
          FilledButton(
            onPressed: () {
              Navigator.of(dialogContext).pop();
              onConfirm(controller.text.trim());
            },
            child: Text(loc.confirm),
          ),
        ],
      ),
    );
  }

  Future<void> _confirmCancel(
    BuildContext context,
    S loc,
    ApprovalInfo item,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(loc.approvalCancel),
        content: Text(loc.approvalCancelConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(loc.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(loc.confirm),
          ),
        ],
      ),
    );
    if (confirmed == true && context.mounted) {
      await context.read<ApprovalCubit>().cancel(
        item.id,
        onRejected: (_) => _showRejected(context, loc),
      );
    }
  }

  /// 业务拒绝（状态机/所有权）提示。
  void _showRejected(BuildContext context, S loc) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(loc.approvalActionRejected)));
  }
}

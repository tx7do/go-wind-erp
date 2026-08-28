import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';

import 'package:go_wind_erp/generated/l10n.dart';
import 'package:go_wind_erp/src/core/constants/router_paths.dart';
import 'package:go_wind_erp/src/features/sales/domain/sales_models.dart';
import 'package:go_wind_erp/src/features/sales/presentation/sales_cubit.dart';

/// 销售单列表页（移动端只读 + 状态筛选）。
///
/// 状态筛选（全部/草稿/待审批/已通过/已完结/已取消）→ 列表；点击进入详情页
/// （明细 + 履约进度 + 销售退货入口）。
class SalesPage extends StatelessWidget {
  const SalesPage({super.key});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(loc.navSales)),
      body: const _SalesBody(),
    );
  }
}

class _SalesBody extends StatelessWidget {
  const _SalesBody();

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return BlocBuilder<SalesCubit, SalesState>(
      builder: (context, state) {
        if (state is SalesReady) {
          return Column(
            children: [
              _buildFilterChips(context, loc, state),
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () =>
                      context.read<SalesCubit>().setFilter(state.filter),
                  child: state.orders.isEmpty
                      ? ListView(
                          physics: const AlwaysScrollableScrollPhysics(),
                          children: [
                            Padding(
                              padding: const EdgeInsets.all(24),
                              child: Center(child: Text(loc.salesEmpty)),
                            ),
                          ],
                        )
                      : ListView.builder(
                          physics: const AlwaysScrollableScrollPhysics(),
                          itemCount: state.orders.length,
                          itemBuilder: (_, i) => _buildTile(
                            context,
                            loc,
                            state.orders[i],
                          ),
                        ),
                ),
              ),
            ],
          );
        }
        if (state is SalesFailureState) {
          return Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(loc.loadFailed),
                const SizedBox(height: 12),
                FilledButton(
                  onPressed: () => context.read<SalesCubit>().load(),
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
    SalesReady state,
  ) {
    return SizedBox(
      height: 48,
      child: ListView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        children: [
          for (final filter in SalesFilter.values)
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: ChoiceChip(
                label: Text(_filterLabel(loc, filter)),
                selected: state.filter == filter,
                onSelected: (_) =>
                    context.read<SalesCubit>().setFilter(filter),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildTile(
    BuildContext context,
    S loc,
    SalesOrderInfo order,
  ) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: ListTile(
        title: Row(
          children: [
            Expanded(
              child: Text(
                order.soNumber,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            _statusChip(loc, order.status),
          ],
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(
            '${loc.salesCustomer}: ${order.customerCode}\n'
            '${loc.salesTotal}: ¥${(order.totalAmount / 100).toStringAsFixed(2)}',
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
        ),
        isThreeLine: true,
        onTap: () => context.push(
          '${AppRoutePath.salesDetailPrefix}${order.id}',
        ),
      ),
    );
  }

  Widget _statusChip(S loc, String status) {
    final (label, color) = switch (status) {
      'DRAFT' => (loc.salesStatusDraft, Colors.grey),
      'SUBMITTED' => (loc.salesStatusSubmitted, Colors.orange),
      'APPROVED' => (loc.salesStatusApproved, Colors.green),
      'REJECTED' => (loc.salesStatusRejected, Colors.red),
      'COMPLETED' => (loc.salesStatusCompleted, Colors.blue),
      'CANCELLED' => (loc.salesStatusCancelled, Colors.grey),
      _ => (status, Colors.grey),
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

  String _filterLabel(S loc, SalesFilter filter) => switch (filter) {
        SalesFilter.all => loc.salesFilterAll,
        SalesFilter.draft => loc.salesStatusDraft,
        SalesFilter.submitted => loc.salesStatusSubmitted,
        SalesFilter.approved => loc.salesStatusApproved,
        SalesFilter.completed => loc.salesStatusCompleted,
        SalesFilter.cancelled => loc.salesStatusCancelled,
      };
}

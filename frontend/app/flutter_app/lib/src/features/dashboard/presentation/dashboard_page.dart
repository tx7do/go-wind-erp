import 'dart:async' show unawaited;

import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/generated/l10n.dart';
import 'package:go_wind_erp/src/features/auth/domain/auth_repository.dart'
    show AuthRepository;
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_models.dart';
import 'package:go_wind_erp/src/features/dashboard/presentation/dashboard_cubit.dart';
import 'package:go_wind_erp/src/features/dashboard/presentation/dashboard_state.dart';

/// 经营看板页。
///
/// 库存域聚合总览：仓库数 / 在库 SKU 数 / 库存总量 / 流水数四张指标卡 +
/// 低库存清单（按数量升序）。下拉刷新；AppBar 保留登出入口以验证会话
/// 失效流程（清除令牌 → 路由重定向至登录）。
class DashboardPage extends StatelessWidget {
  const DashboardPage({super.key});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(loc.navDashboard),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: loc.logout,
            onPressed: () =>
                unawaited(GetIt.instance<AuthRepository>().logout()),
          ),
        ],
      ),
      body: BlocBuilder<DashboardCubit, DashboardState>(
        builder: (context, state) {
          if (state is DashboardReady) {
            return RefreshIndicator(
              onRefresh: () => context.read<DashboardCubit>().load(),
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.all(16),
                children: [
                  _buildMetricGrid(context, loc, state.overview),
                  const SizedBox(height: 16),
                  _buildLowStock(context, loc, state.overview.lowStockItems),
                ],
              ),
            );
          }
          if (state is DashboardFailureState) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(loc.loadFailed),
                  const SizedBox(height: 12),
                  FilledButton(
                    onPressed: () => context.read<DashboardCubit>().load(),
                    child: Text(loc.retry),
                  ),
                ],
              ),
            );
          }
          return const Center(child: CircularProgressIndicator());
        },
      ),
    );
  }

  Widget _buildMetricGrid(
    BuildContext context,
    S loc,
    InventoryOverviewInfo overview,
  ) {
    return GridView.count(
      crossAxisCount: 2,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      mainAxisSpacing: 12,
      crossAxisSpacing: 12,
      childAspectRatio: 1.6,
      children: [
        _MetricCard(
          label: loc.metricWarehouses,
          value: overview.warehouseCount.toString(),
          icon: Icons.warehouse,
          color: Colors.blue,
        ),
        _MetricCard(
          label: loc.metricSkus,
          value: overview.skuCount.toString(),
          icon: Icons.inventory_2,
          color: Colors.teal,
        ),
        _MetricCard(
          label: loc.metricTotalQuantity,
          value: overview.totalQuantity.toString(),
          icon: Icons.functions,
          color: Colors.indigo,
        ),
        _MetricCard(
          label: loc.metricMovements,
          value: overview.movementCount.toString(),
          icon: Icons.swap_vert,
          color: Colors.deepOrange,
        ),
      ],
    );
  }

  Widget _buildLowStock(BuildContext context, S loc, List<LowStockItem> items) {
    return Card(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              children: [
                const Icon(Icons.warning_amber, color: Colors.orange),
                const SizedBox(width: 8),
                Text(
                  loc.lowStockTitle,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ],
            ),
          ),
          if (items.isEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
              child: Text(loc.lowStockEmpty),
            )
          else
            for (final item in items)
              ListTile(
                dense: true,
                leading: Text(
                  item.quantity.toString(),
                  style: const TextStyle(
                    color: Colors.red,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                title: Text(item.productCode),
                subtitle: Text(item.locationId.toString()),
              ),
        ],
      ),
    );
  }
}

class _MetricCard extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;
  final Color color;

  const _MetricCard({
    required this.label,
    required this.value,
    required this.icon,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, color: color, size: 20),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    label,
                    style: Theme.of(context).textTheme.bodySmall,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              value,
              style: Theme.of(context)
                  .textTheme
                  .headlineSmall
                  ?.copyWith(fontWeight: FontWeight.bold),
            ),
          ],
        ),
      ),
    );
  }
}

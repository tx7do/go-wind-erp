import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:go_wind_erp/generated/l10n.dart';

/// 已登录状态的外壳：底部 [NavigationBar] 在四个模块间切换。
///
/// 看板/审批/仓储/销售；销售分支含详情子路由（退货入口在详情页）。
class HomeShell extends StatelessWidget {
  final StatefulNavigationShell navigationShell;

  const HomeShell({super.key, required this.navigationShell});

  @override
  Widget build(BuildContext context) {
    final loc = S.of(context);

    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: navigationShell.currentIndex,
        onDestinationSelected: (index) => navigationShell.goBranch(
          index,
          initialLocation: index == navigationShell.currentIndex,
        ),
        destinations: [
          NavigationDestination(
            icon: const Icon(Icons.dashboard_outlined),
            selectedIcon: const Icon(Icons.dashboard),
            label: loc.navDashboard,
          ),
          NavigationDestination(
            icon: const Icon(Icons.approval_outlined),
            selectedIcon: const Icon(Icons.approval),
            label: loc.navApproval,
          ),
          NavigationDestination(
            icon: const Icon(Icons.warehouse_outlined),
            selectedIcon: const Icon(Icons.warehouse),
            label: loc.navWms,
          ),
          NavigationDestination(
            icon: const Icon(Icons.sell_outlined),
            selectedIcon: const Icon(Icons.sell),
            label: loc.navSales,
          ),
        ],
      ),
    );
  }
}

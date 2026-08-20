import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:go_wind_erp/generated/l10n.dart';

/// 已登录状态的外壳：底部 [NavigationBar] 在三个模块间切换。
///
/// 各分支的具体页面（看板/審批/倉儲）当前为占位实现，待后端对应模块
/// 就位后替换。导航结构本身已稳定，后续仅需替换分支 builder。
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
        ],
      ),
    );
  }
}

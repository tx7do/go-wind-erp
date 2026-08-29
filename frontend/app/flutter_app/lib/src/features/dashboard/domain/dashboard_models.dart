/// 看板领域模型。
///
/// 纯 Dart 值对象；由 data 层从 `/app/v1/stock-quants:overview` 响应映射而来。

/// 库存经营总览。
class InventoryOverviewInfo {
  final int warehouseCount;
  final int skuCount;
  final int totalQuantity;
  final int movementCount;
  final List<LowStockItem> lowStockItems;

  const InventoryOverviewInfo({
    required this.warehouseCount,
    required this.skuCount,
    required this.totalQuantity,
    required this.movementCount,
    required this.lowStockItems,
  });
}

/// 低库存清单项。
///
/// 对应后端 `stock.quant` 的低库存条目（按 location + productCode 索引）。
class LowStockItem {
  final int locationId;
  final String productCode;
  final int quantity;

  const LowStockItem({
    required this.locationId,
    required this.productCode,
    required this.quantity,
  });
}


/// 财务经营汇总（移动端驾驶舱；金额单位分，展示层转元）。
class FinanceSummaryInfo {
  final int revenueMonth;
  final int cogsMonth;
  final int profitMonth;
  final int arBalance;
  final int apBalance;

  const FinanceSummaryInfo({
    required this.revenueMonth,
    required this.cogsMonth,
    required this.profitMonth,
    required this.arBalance,
    required this.apBalance,
  });

  /// 分转元（两位小数字符串）。
  static String yuan(int cents) => (cents / 100).toStringAsFixed(2);
}

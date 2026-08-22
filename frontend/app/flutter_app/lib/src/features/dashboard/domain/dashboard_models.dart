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

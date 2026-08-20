/// 看板领域模型。
///
/// 纯 Dart 值对象；由 data 层从 `/app/v1/inventories:overview` 响应映射而来。

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
class LowStockItem {
  final String warehouseCode;
  final String skuCode;
  final int quantity;
  final String status;

  const LowStockItem({
    required this.warehouseCode,
    required this.skuCode,
    required this.quantity,
    required this.status,
  });
}

/// WMS 领域模型。
///
/// 纯 Dart 值对象，不含传输层细节；由 data 层从生成客户端的响应映射而来。

/// 仓库（扫码场景的只读选择项）。
class WarehouseInfo {
  final String code;
  final String name;
  final String? location;

  const WarehouseInfo({required this.code, required this.name, this.location});
}

/// 库存记录（按仓库 + SKU 查询的结果）。
class InventoryInfo {
  final String warehouseCode;
  final String skuCode;
  final int quantity;

  /// AVAILABLE / LOCKED / QUARANTINED（后端原始字符串）。
  final String status;

  const InventoryInfo({
    required this.warehouseCode,
    required this.skuCode,
    required this.quantity,
    required this.status,
  });
}

/// 扫码支持的流水类型。
///
/// 移动端仅做入库/出库；调拨与调整属后台操作，不暴露给扫码端。
enum MovementKind { inbound, outbound }

extension MovementKindProto on MovementKind {
  /// 对应后端 `StockMovement.MovementType` 的 wire 值。
  String get wireName => this == MovementKind.inbound ? 'INBOUND' : 'OUTBOUND';
}

/// 待提交的出入库流水。
class StockMovementDraft {
  final String warehouseCode;
  final String skuCode;
  final MovementKind kind;

  /// 正为入、负为出（出库时由页面取负后传入）。
  final int delta;
  final int quantityBefore;
  final int quantityAfter;
  final String? remark;

  const StockMovementDraft({
    required this.warehouseCode,
    required this.skuCode,
    required this.kind,
    required this.delta,
    required this.quantityBefore,
    required this.quantityAfter,
    this.remark,
  });
}

/// 流水记录（历史列表展示项）。
class StockMovementRecord {
  final String movementType;
  final int delta;
  final int quantityBefore;
  final int quantityAfter;
  final String? createdAt;

  const StockMovementRecord({
    required this.movementType,
    required this.delta,
    required this.quantityBefore,
    required this.quantityAfter,
    this.createdAt,
  });
}

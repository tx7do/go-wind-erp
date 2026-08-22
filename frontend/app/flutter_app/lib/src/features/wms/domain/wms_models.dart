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

/// 库存量记录（按 location + productCode 查询的结果）。
///
/// 对应后端 `stock.quant`（只读；quantity 仅由拣货校验变更）。
class InventoryInfo {
  final int locationId;
  final String productCode;
  final int quantity;

  const InventoryInfo({
    required this.locationId,
    required this.productCode,
    required this.quantity,
  });
}

/// 待提交的内部调拨拣货单草稿。
///
/// 移动端仅可发起 INTERNAL 调拨：服务层按仓库推导 source/dest location，
/// 客户端提供 from/to 仓库编码（用于解析 locationId）与单条 move 计划。
/// 提交流程：create → confirm → validate。
class InternalTransferDraft {
  final String fromWarehouseCode;
  final String toWarehouseCode;
  final String productCode;
  final int plannedQuantity;

  const InternalTransferDraft({
    required this.fromWarehouseCode,
    required this.toWarehouseCode,
    required this.productCode,
    required this.plannedQuantity,
  });
}

/// 拣货单记录（历史列表展示项）。
///
/// 对应后端 `stock.picking` 的一等文档视图；`derivedState` 由子 moves 聚合
/// 计算（不存储）。
class PickingRecord {
  final String pickingNumber;
  final String pickingType;
  final String derivedState;
  final String? createdAt;

  const PickingRecord({
    required this.pickingNumber,
    required this.pickingType,
    required this.derivedState,
    this.createdAt,
  });
}

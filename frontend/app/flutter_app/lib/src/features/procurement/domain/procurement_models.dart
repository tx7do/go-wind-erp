/// 采购领域模型（移动端只读详情视图）。

/// 采购单明细（已收进度展示）。
class PurchaseOrderItemInfo {
  final int id;
  final String skuCode;
  final int quantity;
  final int receivedQuantity;
  final int unitPrice;
  final int amount;

  const PurchaseOrderItemInfo({
    required this.id,
    required this.skuCode,
    required this.quantity,
    required this.receivedQuantity,
    required this.unitPrice,
    required this.amount,
  });
}

/// 采购单（审批深链的目标详情）。
class PurchaseOrderInfo {
  final int id;
  final String poNumber;
  final String supplierCode;
  final String warehouseCode;
  final String status;
  final int totalAmount;
  final String? remark;
  final List<PurchaseOrderItemInfo> items;

  const PurchaseOrderInfo({
    required this.id,
    required this.poNumber,
    required this.supplierCode,
    required this.warehouseCode,
    required this.status,
    required this.totalAmount,
    this.remark,
    this.items = const [],
  });
}

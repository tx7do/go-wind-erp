/// 销售领域模型。
///
/// 纯 Dart 值对象，不含传输层细节；由 data 层从生成客户端的响应映射而来。

/// 销售单明细（履约进度展示 + 退货入口的粒度）。
class SalesOrderItemInfo {
  final int id;
  final String skuCode;
  final int quantity;
  final int fulfilledQuantity;
  final int unitPrice;
  final int amount;

  /// 可退数量 = 已履约数（退货从此扣减）。
  int get returnable => fulfilledQuantity;

  const SalesOrderItemInfo({
    required this.id,
    required this.skuCode,
    required this.quantity,
    required this.fulfilledQuantity,
    required this.unitPrice,
    required this.amount,
  });
}

/// 销售单（移动端只读视图 + 明细）。
class SalesOrderInfo {
  final int id;
  final String soNumber;
  final String customerCode;
  final String warehouseCode;
  final String status;
  final int totalAmount;
  final String? remark;
  final List<SalesOrderItemInfo> items;

  /// APPROVED/COMPLETED 且有已履约明细时可退货。
  bool get returnable =>
      (status == 'APPROVED' || status == 'COMPLETED') &&
      items.any((i) => i.fulfilledQuantity > 0);

  const SalesOrderInfo({
    required this.id,
    required this.soNumber,
    required this.customerCode,
    required this.warehouseCode,
    required this.status,
    required this.totalAmount,
    this.remark,
    this.items = const [],
  });
}

/// 待提交的销售退货（按明细逐条退）。
class SalesReturnDraft {
  final int salesOrderId;
  final int salesOrderItemId;
  final int quantity;

  const SalesReturnDraft({
    required this.salesOrderId,
    required this.salesOrderItemId,
    required this.quantity,
  });
}

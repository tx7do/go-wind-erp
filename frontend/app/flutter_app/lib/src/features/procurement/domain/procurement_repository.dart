import 'package:go_wind_erp/src/features/procurement/domain/procurement_models.dart';

/// 采购单仓储抽象（移动端只读详情）。
abstract class ProcurementRepository {
  /// 拉取采购单详情（含明细）。
  Future<PurchaseOrderInfo> getPurchaseOrder(int id);
}

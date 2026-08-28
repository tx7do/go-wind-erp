import 'package:go_wind_erp/src/features/sales/domain/sales_models.dart';

/// 销售单仓储抽象。
///
/// presentation 层仅依赖本接口；实现见 data 层 [SalesRepositoryImpl]。
abstract class SalesRepository {
  /// 拉取销售单列表（按状态过滤，status 为空表示全部）。
  Future<List<SalesOrderInfo>> listSalesOrders({
    String? status,
    int limit = 50,
  });

  /// 拉取销售单详情（含明细）。
  Future<SalesOrderInfo> getSalesOrder(int id);

  /// 提交销售退货（生成 INCOMING 退货拣货单；执行需在拣货列表确认+校验）。
  ///
  /// 客户端须校验退数 ≤ 已履约数；后端仍有原子守卫。
  Future<void> submitSalesReturn(SalesReturnDraft draft);
}

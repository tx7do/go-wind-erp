import 'package:go_wind_erp/src/features/wms/domain/wms_models.dart';

/// WMS 扫码仓储抽象。
///
/// presentation 层仅依赖本接口；实现见 data 层 [WmsRepositoryImpl]。
abstract class WmsRepository {
  /// 拉取仓库列表（扫码端只读，用于选择当前作业仓库）。
  Future<List<WarehouseInfo>> listWarehouses();

  /// 按仓库 + SKU 查询库存；未命中返回 null。
  Future<InventoryInfo?> findInventory(
    String warehouseCode,
    String skuCode,
  );

  /// 提交一条出入库流水。
  ///
  /// 后端强校验 `quantityBefore + delta == quantityAfter`，页面须按查得的
  /// 当前库存计算后传入；提交失败抛 [WmsFailure]。
  Future<void> submitMovement(StockMovementDraft draft);

  /// 拉取指定仓库 + SKU 的近期流水（默认 20 条）。
  Future<List<StockMovementRecord>> listMovements(
    String warehouseCode,
    String skuCode, {
    int limit = 20,
  });
}

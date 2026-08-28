import 'package:go_wind_erp/src/features/wms/domain/wms_models.dart';

/// WMS 扫码仓储抽象。
///
/// presentation 层仅依赖本接口；实现见 data 层 [WmsRepositoryImpl]。
abstract class WmsRepository {
  /// 拉取仓库列表（扫码端只读，用于选择当前作业仓库）。
  Future<List<WarehouseInfo>> listWarehouses();

  /// 按 location + productCode 查询库存量；未命中返回 null。
  ///
  /// 新 API 按 `locationId` 索引；仓库编码 → locationId 由 data 层解析。
  Future<InventoryInfo?> findInventory(
    String warehouseCode,
    String productCode,
  );

  /// 提交内部调拨拣货单：create → confirm → validate。
  ///
  /// 客户端须校验 from/to 仓库不同且数量不超当前库存；后端仍有守卫。
  /// 入库拣货单不再由客户端创建——服务层在采购单审批通过后自动生成。
  Future<void> submitInternalTransfer(InternalTransferDraft draft);

  /// 提交盘点拣货单：create(INVENTORY_ADJUSTMENT) → confirm → validate。
  ///
  /// 客户端须校验差异数非 0；后端按符号推导方向并有守卫。
  Future<void> submitStocktake(StocktakeDraft draft);

  /// 拉取近期拣货单（默认 20 条）。
  Future<List<PickingRecord>> listPickings({int limit = 20});
}

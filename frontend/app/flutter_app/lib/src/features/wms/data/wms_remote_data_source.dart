import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show
        ApiClient,
        InventoryServiceV1ConfirmStockPickingRequest,
        InventoryServiceV1CreateStockPickingRequest,
        InventoryServiceV1ListStockPickingResponse,
        InventoryServiceV1ListStockLotResponse,
        InventoryServiceV1ListWarehouseResponse,
        InventoryServiceV1StockMove,
        InventoryServiceV1StockPicking,
        InventoryServiceV1StockPicking$PickingType,
        InventoryServiceV1StockQuant,
        InventoryServiceV1ValidateStockPickingRequest,
        StockLotServiceClient,
        InventoryServiceV1Warehouse,
        PaginationPagingRequest,
        StockPickingServiceClient,
        StockQuantServiceClient,
        WarehouseServiceClient;
import 'package:go_wind_erp/src/features/wms/domain/wms_models.dart';

/// WMS 远程数据源。
///
/// 适配 protoc-gen-dart-http 生成的 stock_quant / stock_picking / warehouse
/// 三个客户端。库存查询与调拨均需把“仓库编码”解析为后端 `locationId`（取自
/// 仓库记录的 `receivingLocationId`——仓库创建时自动生成的内部 StockLocation）。
class WmsRemoteDataSource {
  final WarehouseServiceClient _warehouses;
  final StockQuantServiceClient _stockQuants;
  final StockPickingServiceClient _stockPickings;
  final StockLotServiceClient _stockLots;

  WmsRemoteDataSource(ApiClient api)
      : _warehouses = api.warehouseService,
        _stockQuants = api.stockQuantService,
        _stockPickings = api.stockPickingService,
        _stockLots = api.stockLotService;

  /// 仓库列表 → `GET /app/v1/warehouses`（不分页，扫码端仓库数量有限）。
  Future<InventoryServiceV1ListWarehouseResponse> listWarehouses() {
    return _warehouses.list(
      PaginationPagingRequest(noPaging: true),
    );
  }

  /// 库存量查询 → `GET /app/v1/stock-quants`（只读，列表后客户端按
  /// `locationId` + `productCode` 过滤）。
  ///
  /// 仓库编码 → `locationId` 解析自仓库列表的 `receivingLocationId`；解析
  /// 失败或无匹配条目时返回 null。
  Future<InventoryServiceV1StockQuant?> findInventory(
    String warehouseCode,
    String productCode,
  ) async {
    final locationId = await _resolveLocationId(warehouseCode);
    if (locationId == null) return null;
    final resp = await _stockQuants.list(
      PaginationPagingRequest(noPaging: true),
    );
    final items = resp.items ?? const <InventoryServiceV1StockQuant>[];
    for (final q in items) {
      if (q.locationId == locationId && q.productCode == productCode) {
        return q;
      }
    }
    return null;
  }

  /// 内部调拨 → create → confirm → validate。
  ///
  /// 客户端仅提供 from/to 仓库编码与单条 move 计划；picking 的
  /// `sourceLocationId`/`destinationLocationId` 由服务层按 picking_type +
  /// 仓库推导落库，客户端不提供。入库拣货单不在客户端创建（采购单审批通过
  /// 后由服务层自动生成）。
  Future<void> submitInternalTransfer(InternalTransferDraft draft) async {
    final sourceLocationId = await _resolveLocationId(draft.fromWarehouseCode);
    final destinationLocationId =
        await _resolveLocationId(draft.toWarehouseCode);
    if (sourceLocationId == null || destinationLocationId == null) return;

    final picking = InventoryServiceV1StockPicking(
      pickingType: InventoryServiceV1StockPicking$PickingType.internal,
      moves: [
        InventoryServiceV1StockMove(
          sourceLocationId: sourceLocationId,
          destinationLocationId: destinationLocationId,
          productCode: draft.productCode,
          plannedQuantity: draft.plannedQuantity,
        ),
      ],
    );
    final created = await _stockPickings.create(
      InventoryServiceV1CreateStockPickingRequest(data: picking),
    );
    final rawId = created['id'];
    final id = rawId is int ? rawId : int.tryParse(rawId?.toString() ?? '');
    if (id == null) {
      throw StateError('created picking missing id');
    }
    await _stockPickings.confirm(
      InventoryServiceV1ConfirmStockPickingRequest(id: id),
    );
    await _stockPickings.validate(
      InventoryServiceV1ValidateStockPickingRequest(id: id),
    );
  }

  /// 拣货单列表 → `GET /app/v1/stock-pickings`。
  Future<InventoryServiceV1ListStockPickingResponse> listPickings({
    int limit = 20,
  }) {
    return _stockPickings.list(
      PaginationPagingRequest(pageSize: limit),
    );
  }

  /// 批次余量 → `GET /app/v1/stock-lots`（按 SKU 过滤，扫码后核对效期）。
  Future<InventoryServiceV1ListStockLotResponse> listLots(String skuCode) {
    return _stockLots.list(
      PaginationPagingRequest(
        pageSize: 50,
        query: '{"skuCode":"$skuCode"}',
      ),
    );
  }

  /// 盘点 → create(INVENTORY_ADJUSTMENT) → confirm → validate 一键链。
  ///
  /// 客户端提供仓库编码与单条带符号差异 move；服务层按符号推导方向
  /// （盘盈 INVENTORY_LOSS→仓库 / 盘亏 仓库→INVENTORY_LOSS，盘亏数量
  /// 落库为绝对值）。
  Future<void> submitStocktake(StocktakeDraft draft) async {
    final picking = InventoryServiceV1StockPicking(
      pickingType:
          InventoryServiceV1StockPicking$PickingType.inventoryAdjustment,
      fromWarehouseCode: draft.warehouseCode,
      moves: [
        InventoryServiceV1StockMove(
          productCode: draft.productCode,
          plannedQuantity: draft.diff,
        ),
      ],
    );
    final created = await _stockPickings.create(
      InventoryServiceV1CreateStockPickingRequest(data: picking),
    );
    final rawId = created['id'];
    final id = rawId is int ? rawId : int.tryParse(rawId?.toString() ?? '');
    if (id == null) {
      throw StateError('created picking missing id');
    }
    await _stockPickings.confirm(
      InventoryServiceV1ConfirmStockPickingRequest(id: id),
    );
    await _stockPickings.validate(
      InventoryServiceV1ValidateStockPickingRequest(id: id),
    );
  }

  /// 仓库编码 → `locationId`（取自仓库记录的 `receivingLocationId`）。
  Future<int?> _resolveLocationId(String warehouseCode) async {
    final resp = await _warehouses.list(
      PaginationPagingRequest(noPaging: true),
    );
    final items = resp.items ?? const <InventoryServiceV1Warehouse>[];
    for (final w in items) {
      if (w.code == warehouseCode) return w.receivingLocationId;
    }
    return null;
  }

  /// 仓库响应 → 领域模型（丢弃编码/名称缺失的脏数据）。
  static List<WarehouseInfo> toWarehouseInfos(
    InventoryServiceV1ListWarehouseResponse resp,
  ) {
    final items = resp.items ?? const <InventoryServiceV1Warehouse>[];
    return items
        .where((w) => (w.code ?? '').isNotEmpty)
        .map(
          (w) => WarehouseInfo(
            code: w.code!,
            name: (w.name ?? '').isEmpty ? w.code! : w.name!,
            location: w.location,
          ),
        )
        .toList();
  }
}

import 'dart:convert' show jsonEncode;

import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show
        ApiClient,
        InventoryServiceClient,
        InventoryServiceV1CreateStockMovementRequest,
        InventoryServiceV1ListInventoryResponse,
        InventoryServiceV1ListStockMovementResponse,
        InventoryServiceV1ListWarehouseResponse,
        InventoryServiceV1StockMovement,
        InventoryServiceV1StockMovement$MovementType,
        InventoryServiceV1Warehouse,
        PaginationPagingRequest,
        StockMovementServiceClient,
        WarehouseServiceClient;
import 'package:go_wind_erp/src/features/wms/domain/wms_models.dart';

/// WMS 远程数据源。
///
/// 适配 protoc-gen-dart-http 生成的三个库存域客户端，构造 list 请求的
/// `filter` JSON（camelCase 字段名，与 go-crud pagination 过滤语法一致）。
class WmsRemoteDataSource {
  final WarehouseServiceClient _warehouses;
  final InventoryServiceClient _inventories;
  final StockMovementServiceClient _movements;

  WmsRemoteDataSource(ApiClient api)
      : _warehouses = api.warehouseService,
        _inventories = api.inventoryService,
        _movements = api.stockMovementService;

  /// 仓库列表 → `GET /app/v1/warehouses`（不分页，扫码端仓库数量有限）。
  Future<InventoryServiceV1ListWarehouseResponse> listWarehouses() {
    return _warehouses.list(
      PaginationPagingRequest(noPaging: true),
    );
  }

  /// 库存查询 → `GET /app/v1/inventories?filter={"warehouseCode":...,"skuCode":...}`。
  Future<InventoryServiceV1ListInventoryResponse> findInventory(
    String warehouseCode,
    String skuCode,
  ) {
    return _inventories.list(
      PaginationPagingRequest(
        filter: jsonEncode({
          'warehouseCode': warehouseCode,
          'skuCode': skuCode,
        }),
      ),
    );
  }

  /// 创建流水 → `POST /app/v1/stock-movements`。
  Future<Map<String, dynamic>> createMovement(StockMovementDraft draft) {
    final movement = InventoryServiceV1StockMovement(
      warehouseCode: draft.warehouseCode,
      skuCode: draft.skuCode,
      delta: draft.delta,
      movementType: InventoryServiceV1StockMovement$MovementType.fromString(
        draft.kind.wireName,
      ),
      quantityBefore: draft.quantityBefore,
      quantityAfter: draft.quantityAfter,
      remark: draft.remark,
    );
    return _movements.create(
      InventoryServiceV1CreateStockMovementRequest(data: movement),
    );
  }

  /// 流水列表 → `GET /app/v1/stock-movements?filter=...`。
  Future<InventoryServiceV1ListStockMovementResponse> listMovements(
    String warehouseCode,
    String skuCode,
    int limit,
  ) {
    return _movements.list(
      PaginationPagingRequest(
        pageSize: limit,
        filter: jsonEncode({
          'warehouseCode': warehouseCode,
          'skuCode': skuCode,
        }),
      ),
    );
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

import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/generated/api/app/service/v1/index.dart' show ApiClient;
import 'package:go_wind_erp/src/core/transport/http/api_exception.dart'
    show ApiException, ApiExceptionCategory;
import 'package:go_wind_erp/src/features/wms/data/wms_remote_data_source.dart';
import 'package:go_wind_erp/src/features/wms/domain/wms_failure.dart';
import 'package:go_wind_erp/src/features/wms/domain/wms_models.dart';
import 'package:go_wind_erp/src/features/wms/domain/wms_repository.dart';

/// [WmsRepository] 的 data 层实现。
///
/// 职责：调用 [WmsRemoteDataSource]，将响应映射为领域模型；将传输层
/// [DioException]（已由统一拦截器封装为 [ApiException]）映射为
/// [WmsFailure] 子类抛出。
class WmsRepositoryImpl implements WmsRepository {
  final WmsRemoteDataSource _dataSource;

  WmsRepositoryImpl(this._dataSource);

  @override
  Future<List<WarehouseInfo>> listWarehouses() async {
    try {
      final resp = await _dataSource.listWarehouses();
      return WmsRemoteDataSource.toWarehouseInfos(resp);
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<InventoryInfo?> findInventory(
    String warehouseCode,
    String skuCode,
  ) async {
    try {
      final resp = await _dataSource.findInventory(warehouseCode, skuCode);
      final items = resp.items ?? const [];
      if (items.isEmpty) return null;
      final inv = items.first;
      return InventoryInfo(
        warehouseCode: inv.warehouseCode ?? warehouseCode,
        skuCode: inv.skuCode ?? skuCode,
        quantity: inv.quantity ?? 0,
        status: inv.status?.toString() ?? 'AVAILABLE',
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<void> submitMovement(StockMovementDraft draft) async {
    try {
      await _dataSource.createMovement(draft);
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<void> transferStock({
    required String fromWarehouseCode,
    required String toWarehouseCode,
    required String skuCode,
    required int quantity,
    String? remark,
  }) async {
    try {
      await _dataSource.transferStock(
        fromWarehouseCode: fromWarehouseCode,
        toWarehouseCode: toWarehouseCode,
        skuCode: skuCode,
        quantity: quantity,
        remark: remark,
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<void> reverseMovement(int movementId, String reason) async {
    try {
      await _dataSource.reverseMovement(movementId, reason);
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<List<StockMovementRecord>> listMovements(
    String warehouseCode,
    String skuCode, {
    int limit = 20,
  }) async {
    try {
      final resp = await _dataSource.listMovements(
        warehouseCode,
        skuCode,
        limit,
      );
      final items = resp.items ?? const [];
      return items
          .map(
            (m) => StockMovementRecord(
              id: m.id ?? 0,
              movementType: m.movementType?.toString() ?? '',
              delta: m.delta ?? 0,
              quantityBefore: m.quantityBefore ?? 0,
              quantityAfter: m.quantityAfter ?? 0,
              createdAt: m.createdAt,
            ),
          )
          .toList();
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  /// 将统一拦截器封装过的 [DioException]（其 `error` 为 [ApiException]）
  /// 映射为领域 [WmsFailure]。
  WmsFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);
    switch (api.category) {
      case ApiExceptionCategory.auth:
        return const WmsUnauthorizedFailure();
      case ApiExceptionCategory.business:
        return const WmsInvalidInputFailure();
      case ApiExceptionCategory.server:
      case ApiExceptionCategory.network:
        return const WmsNetworkFailure();
      case ApiExceptionCategory.unknown:
        return const WmsUnknownFailure();
    }
  }
}

/// 供 [init.dart] 注册时构造 [WmsRepositoryImpl]。
WmsRepositoryImpl createWmsRepositoryImpl() {
  return WmsRepositoryImpl(
    WmsRemoteDataSource(GetIt.instance<ApiClient>()),
  );
}

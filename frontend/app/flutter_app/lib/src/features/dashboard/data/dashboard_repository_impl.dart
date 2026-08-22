import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show ApiClient, InventoryServiceV1GetStockQuantOverviewRequest;
import 'package:go_wind_erp/src/core/transport/http/api_exception.dart'
    show ApiException, ApiExceptionCategory;
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_failure.dart';
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_models.dart';
import 'package:go_wind_erp/src/features/dashboard/domain/dashboard_repository.dart';

/// [DashboardRepository] 的 data 层实现。
///
/// 直接调用生成客户端的 `stockQuantService.getOverview`，将响应映射为领域
/// 模型；将传输层 [DioException]（已由统一拦截器封装为 [ApiException]）
/// 映射为 [DashboardFailure] 子类抛出。看板为只读聚合，无需独立 data source
/// 类，仓储内联远程调用。
class DashboardRepositoryImpl implements DashboardRepository {
  final ApiClient _api;

  DashboardRepositoryImpl(this._api);

  @override
  Future<InventoryOverviewInfo> fetchOverview({
    int? lowStockThreshold,
    int? lowStockLimit,
  }) async {
    try {
      final resp = await _api.stockQuantService.getOverview(
        InventoryServiceV1GetStockQuantOverviewRequest(
          lowStockThreshold: lowStockThreshold,
          lowStockLimit: lowStockLimit,
        ),
      );
      final lowItems = resp.lowStockItems ?? const [];
      return InventoryOverviewInfo(
        warehouseCount: resp.warehouseCount ?? 0,
        skuCount: resp.skuCount ?? 0,
        totalQuantity: resp.totalQuantity ?? 0,
        movementCount: resp.movementCount ?? 0,
        lowStockItems: [
          for (final item in lowItems)
            LowStockItem(
              locationId: item.locationId ?? 0,
              productCode: item.productCode ?? '',
              quantity: item.quantity ?? 0,
            ),
        ],
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  /// 将统一拦截器封装过的 [DioException]（其 `error` 为 [ApiException]）
  /// 映射为领域 [DashboardFailure]。
  DashboardFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);
    switch (api.category) {
      case ApiExceptionCategory.auth:
        return const DashboardUnauthorizedFailure();
      case ApiExceptionCategory.business:
        return const DashboardInvalidInputFailure();
      case ApiExceptionCategory.server:
      case ApiExceptionCategory.network:
        return const DashboardNetworkFailure();
      case ApiExceptionCategory.unknown:
        return const DashboardUnknownFailure();
    }
  }
}

/// 供 [init.dart] 注册时构造 [DashboardRepositoryImpl]。
DashboardRepositoryImpl createDashboardRepositoryImpl() {
  return DashboardRepositoryImpl(GetIt.instance<ApiClient>());
}

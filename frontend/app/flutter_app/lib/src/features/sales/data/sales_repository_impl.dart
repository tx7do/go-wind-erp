import 'dart:convert' show jsonEncode;

import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show
        ApiClient,
        InventoryServiceV1CreateSalesReturnRequest,
        InventoryServiceV1SalesReturnItem,
        PaginationPagingRequest,
        SalesServiceV1GetSalesOrderRequest;
import 'package:go_wind_erp/src/core/transport/http/api_exception.dart'
    show ApiException, ApiExceptionCategory;
import 'package:go_wind_erp/src/features/sales/domain/sales_failure.dart';
import 'package:go_wind_erp/src/features/sales/domain/sales_models.dart';
import 'package:go_wind_erp/src/features/sales/domain/sales_repository.dart';

/// [SalesRepository] 的 data 层实现。
///
/// 只读列表/详情 + 销售退货动作，直接内联生成客户端调用（同审批中心模式）；
/// 将传输层 [DioException]（已由统一拦截器封装为 [ApiException]）映射为
/// [SalesFailure] 子类抛出。
class SalesRepositoryImpl implements SalesRepository {
  final ApiClient _api;

  SalesRepositoryImpl(this._api);

  @override
  Future<List<SalesOrderInfo>> listSalesOrders({
    String? status,
    int limit = 50,
  }) async {
    try {
      final filterJson = <String, String>{};
      if (status != null && status.isNotEmpty) {
        filterJson['status'] = status;
      }
      final resp = await _api.salesOrderService.list(
        PaginationPagingRequest(
          pageSize: limit,
          filter: filterJson.isEmpty ? null : jsonEncode(filterJson),
        ),
      );
      final items = resp.items ?? const [];
      return [
        for (final item in items)
          SalesOrderInfo(
            id: item.id ?? 0,
            soNumber: item.soNumber ?? '',
            customerCode: item.customerCode ?? '',
            warehouseCode: item.warehouseCode ?? '',
            status: item.status?.toString() ?? 'DRAFT',
            totalAmount: item.totalAmount ?? 0,
            remark: item.remark,
            items: const [],
          ),
      ];
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<SalesOrderInfo> getSalesOrder(int id) async {
    try {
      final so = await _api.salesOrderService.get(
        SalesServiceV1GetSalesOrderRequest(id: id),
      );
      return SalesOrderInfo(
        id: so.id ?? 0,
        soNumber: so.soNumber ?? '',
        customerCode: so.customerCode ?? '',
        warehouseCode: so.warehouseCode ?? '',
        status: so.status?.toString() ?? 'DRAFT',
        totalAmount: so.totalAmount ?? 0,
        remark: so.remark,
        items: [
          for (final it in so.items ?? const [])
            SalesOrderItemInfo(
              id: it.id ?? 0,
              skuCode: it.skuCode ?? '',
              quantity: it.quantity ?? 0,
              fulfilledQuantity: it.fulfilledQuantity ?? 0,
              unitPrice: it.unitPrice ?? 0,
              amount: it.amount ?? 0,
            ),
        ],
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<void> submitSalesReturn(SalesReturnDraft draft) async {
    try {
      await _api.stockPickingService.createSalesReturn(
        InventoryServiceV1CreateSalesReturnRequest(
          salesOrderId: draft.salesOrderId,
          items: [
            InventoryServiceV1SalesReturnItem(
              salesOrderItemId: draft.salesOrderItemId,
              quantity: draft.quantity,
            ),
          ],
        ),
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  /// 将统一拦截器封装过的 [DioException]（其 `error` 为 [ApiException]）
  /// 映射为领域 [SalesFailure]。
  SalesFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);
    switch (api.category) {
      case ApiExceptionCategory.auth:
        return const SalesUnauthorizedFailure();
      case ApiExceptionCategory.business:
        return const SalesInvalidInputFailure();
      case ApiExceptionCategory.server:
      case ApiExceptionCategory.network:
        return const SalesNetworkFailure();
      case ApiExceptionCategory.unknown:
        return const SalesUnknownFailure();
    }
  }
}

/// 供 [init.dart] 注册时构造 [SalesRepositoryImpl]。
SalesRepositoryImpl createSalesRepositoryImpl() {
  return SalesRepositoryImpl(GetIt.instance<ApiClient>());
}

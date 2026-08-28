import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show ApiClient, ProcurementServiceV1GetPurchaseOrderRequest;
import 'package:go_wind_erp/src/core/transport/http/api_exception.dart'
    show ApiException, ApiExceptionCategory;
import 'package:go_wind_erp/src/features/procurement/domain/procurement_failure.dart';
import 'package:go_wind_erp/src/features/procurement/domain/procurement_models.dart';
import 'package:go_wind_erp/src/features/procurement/domain/procurement_repository.dart';

/// [ProcurementRepository] 的 data 层实现（移动端只读详情）。
class ProcurementRepositoryImpl implements ProcurementRepository {
  final ApiClient _api;

  ProcurementRepositoryImpl(this._api);

  @override
  Future<PurchaseOrderInfo> getPurchaseOrder(int id) async {
    try {
      final po = await _api.purchaseOrderService.get(
        ProcurementServiceV1GetPurchaseOrderRequest(id: id),
      );
      return PurchaseOrderInfo(
        id: po.id ?? 0,
        poNumber: po.poNumber ?? '',
        supplierCode: po.supplierCode ?? '',
        warehouseCode: po.warehouseCode ?? '',
        status: po.status?.toString() ?? 'DRAFT',
        totalAmount: po.totalAmount ?? 0,
        remark: po.remark,
        items: [
          for (final it in po.items ?? const [])
            PurchaseOrderItemInfo(
              id: it.id ?? 0,
              skuCode: it.skuCode ?? '',
              quantity: it.quantity ?? 0,
              receivedQuantity: it.receivedQuantity ?? 0,
              unitPrice: it.unitPrice ?? 0,
              amount: it.amount ?? 0,
            ),
        ],
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  ProcurementFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);
    switch (api.category) {
      case ApiExceptionCategory.auth:
        return const ProcurementUnauthorizedFailure();
      case ApiExceptionCategory.business:
        return const ProcurementInvalidInputFailure();
      case ApiExceptionCategory.server:
      case ApiExceptionCategory.network:
        return const ProcurementNetworkFailure();
      case ApiExceptionCategory.unknown:
        return const ProcurementUnknownFailure();
    }
  }
}

/// 供 [init.dart] 注册时构造 [ProcurementRepositoryImpl]。
ProcurementRepositoryImpl createProcurementRepositoryImpl() {
  return ProcurementRepositoryImpl(GetIt.instance<ApiClient>());
}

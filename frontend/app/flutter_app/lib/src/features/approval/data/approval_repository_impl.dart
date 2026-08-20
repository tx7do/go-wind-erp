import 'dart:convert' show jsonEncode;

import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show
        ApiClient,
        ApprovalServiceV1ApproveApprovalRequestRequest,
        ApprovalServiceV1CancelApprovalRequestRequest,
        ApprovalServiceV1RejectApprovalRequestRequest,
        PaginationPagingRequest;
import 'package:go_wind_erp/src/core/transport/http/api_exception.dart'
    show ApiException, ApiExceptionCategory;
import 'package:go_wind_erp/src/features/approval/domain/approval_failure.dart';
import 'package:go_wind_erp/src/features/approval/domain/approval_models.dart';
import 'package:go_wind_erp/src/features/approval/domain/approval_repository.dart';

/// [ApprovalRepository] 的 data 层实现。
///
/// 只读列表 + 三个审批动作，直接内联生成客户端调用（同看板模式）；
/// 将传输层 [DioException]（已由统一拦截器封装为 [ApiException]）映射为
/// [ApprovalFailure] 子类抛出。
class ApprovalRepositoryImpl implements ApprovalRepository {
  final ApiClient _api;

  ApprovalRepositoryImpl(this._api);

  @override
  Future<List<ApprovalInfo>> listRequests(ApprovalFilter filter) async {
    try {
      final filterJson = <String, String>{};
      final status = filter.statusValue;
      if (status != null) {
        filterJson['status'] = status;
      }
      final resp = await _api.approvalRequestService.list(
        PaginationPagingRequest(
          pageSize: 50,
          filter: filterJson.isEmpty ? null : jsonEncode(filterJson),
        ),
      );
      final items = resp.items ?? const [];
      return [
        for (final item in items)
          ApprovalInfo(
            id: item.id ?? 0,
            title: item.title ?? '',
            bizType: item.bizType ?? '',
            bizRef: item.bizRef ?? '',
            summary: item.summary ?? '',
            status: item.status?.toString() ?? 'PENDING',
            applicantId: item.applicantId ?? 0,
            approverId: item.approverId,
            comment: item.comment,
            createdAt: item.createdAt,
          ),
      ];
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<void> approve(int id, {String? comment}) async {
    try {
      await _api.approvalRequestService.approve(
        ApprovalServiceV1ApproveApprovalRequestRequest(
          id: id,
          comment: (comment ?? '').isEmpty ? null : comment,
        ),
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<void> reject(int id, {String? comment}) async {
    try {
      await _api.approvalRequestService.reject(
        ApprovalServiceV1RejectApprovalRequestRequest(
          id: id,
          comment: (comment ?? '').isEmpty ? null : comment,
        ),
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  @override
  Future<void> cancel(int id) async {
    try {
      await _api.approvalRequestService.cancel(
        ApprovalServiceV1CancelApprovalRequestRequest(id: id),
      );
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  /// 将统一拦截器封装过的 [DioException]（其 `error` 为 [ApiException]）
  /// 映射为领域 [ApprovalFailure]。
  ApprovalFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);
    switch (api.category) {
      case ApiExceptionCategory.auth:
        return const ApprovalUnauthorizedFailure();
      case ApiExceptionCategory.business:
        // 403-FORBIDDEN 落在 business 类目下（kratos 4xx），细分状态机拒绝
        return const ApprovalForbiddenFailure();
      case ApiExceptionCategory.server:
      case ApiExceptionCategory.network:
        return const ApprovalNetworkFailure();
      case ApiExceptionCategory.unknown:
        return const ApprovalUnknownFailure();
    }
  }
}

/// 供 [init.dart] 注册时构造 [ApprovalRepositoryImpl]。
ApprovalRepositoryImpl createApprovalRepositoryImpl() {
  return ApprovalRepositoryImpl(GetIt.instance<ApiClient>());
}

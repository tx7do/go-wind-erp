import 'package:dio/dio.dart' show DioException;
import 'package:get_it/get_it.dart' show GetIt;

import 'package:go_wind_erp/generated/api/app/service/v1/index.dart' show ApiClient;
import 'package:go_wind_erp/src/core/transport/http/api_exception.dart'
    show ApiException, ApiExceptionCategory;
import 'package:go_wind_erp/src/features/dict/data/dict_remote_data_source.dart';
import 'package:go_wind_erp/src/features/dict/domain/dict_failure.dart';
import 'package:go_wind_erp/src/features/dict/domain/dict_models.dart';
import 'package:go_wind_erp/src/features/dict/domain/dict_repository.dart';

/// [DictRepository] 的 data 层实现。
///
/// 职责：调用 [DictRemoteDataSource]，将响应映射为领域模型；将传输层
/// [DioException]（已由统一拦截器封装为 [ApiException]）映射为
/// [DictFailure] 子类抛出。
///
/// 字典条目按 typeCode 进程内缓存（条目内含全量 i18n 标签，无需按语言
/// 重复拉取），与 admin 端 [getDictEntriesByTypeCode] 的缓存语义一致。
class DictRepositoryImpl implements DictRepository {
  final DictRemoteDataSource _dataSource;
  final Map<String, List<DictEntryInfo>> _cache = {};

  DictRepositoryImpl(this._dataSource);

  @override
  Future<List<DictEntryInfo>> listByTypeCode(String typeCode) async {
    final cached = _cache[typeCode];
    if (cached != null) return cached;
    try {
      final resp = await _dataSource.listByTypeCode(typeCode);
      final infos = DictRemoteDataSource.toEntryInfos(resp);
      _cache[typeCode] = infos;
      return infos;
    } on DioException catch (e) {
      throw _toFailure(e);
    }
  }

  /// 将统一拦截器封装过的 [DioException]（其 `error` 为 [ApiException]）
  /// 映射为领域 [DictFailure]。
  DictFailure _toFailure(DioException e) {
    final api = ApiException.fromDioError(e);
    switch (api.category) {
      case ApiExceptionCategory.auth:
        return const DictUnauthorizedFailure();
      case ApiExceptionCategory.business:
        return const DictInvalidInputFailure();
      case ApiExceptionCategory.server:
      case ApiExceptionCategory.network:
        return const DictNetworkFailure();
      case ApiExceptionCategory.unknown:
        return const DictUnknownFailure();
    }
  }
}

/// 供 [init.dart] 注册时构造 [DictRepositoryImpl]。
DictRepositoryImpl createDictRepositoryImpl() {
  return DictRepositoryImpl(
    DictRemoteDataSource(GetIt.instance<ApiClient>()),
  );
}

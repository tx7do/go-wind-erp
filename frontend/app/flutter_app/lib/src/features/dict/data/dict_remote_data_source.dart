import 'package:go_wind_erp/generated/api/app/service/v1/index.dart'
    show
        ApiClient,
        DictEntryLookupClient,
        DictServiceV1ListDictEntryByTypeCodeRequest,
        DictServiceV1ListDictEntryByTypeCodeResponse;
import 'package:go_wind_erp/src/features/dict/domain/dict_models.dart';

/// 字典远程数据源。
///
/// 适配 protoc-gen-dart-http 生成的 [DictEntryLookupClient]，仅封装
/// `GET /app/v1/dict/entries/by-type-code`（按类型编码查询启用项）。
class DictRemoteDataSource {
  final DictEntryLookupClient _dict;

  DictRemoteDataSource(ApiClient api) : _dict = api.dictEntryLookup;

  /// 按类型编码查询启用项（含各语言展示标签）。
  Future<DictServiceV1ListDictEntryByTypeCodeResponse> listByTypeCode(
    String typeCode,
  ) {
    return _dict.listByTypeCode(
      DictServiceV1ListDictEntryByTypeCodeRequest(typeCode: typeCode),
    );
  }

  /// 响应 → 领域模型：仅保留 entryValue + 各语言 entryLabel，丢弃其余字段。
  static List<DictEntryInfo> toEntryInfos(
    DictServiceV1ListDictEntryByTypeCodeResponse resp,
  ) {
    final items = resp.items ?? const [];
    final infos = <DictEntryInfo>[];
    for (final e in items) {
      final value = e.entryValue;
      final i18n = e.i18n;
      if (value == null || value.isEmpty || i18n == null || i18n.isEmpty) {
        continue;
      }
      final labels = <String, String>{};
      for (final entry in i18n.entries) {
        final label = entry.value.entryLabel;
        if (label != null && label.isNotEmpty) {
          labels[entry.key] = label;
        }
      }
      if (labels.isNotEmpty) {
        infos.add(DictEntryInfo(entryValue: value, labels: labels));
      }
    }
    return infos;
  }
}

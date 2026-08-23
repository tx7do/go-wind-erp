import 'package:go_wind_erp/src/features/dict/domain/dict_models.dart';

/// 字典只读查询抽象。
///
/// presentation 层仅依赖本接口；实现见 data 层 [DictRepositoryImpl]。
/// 仅暴露按类型编码查询启用项的只读语义，对应后端
/// `GET /app/v1/dict/entries/by-type-code`。
abstract class DictRepository {
  /// 按类型编码拉取启用项（含各语言展示标签）。
  ///
  /// 返回的条目仅含 entryValue + labels；未启用或不存在时返回空列表。
  Future<List<DictEntryInfo>> listByTypeCode(String typeCode);
}

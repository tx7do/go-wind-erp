/// 字典领域模型。
///
/// 纯 Dart 值对象，不含传输层细节；由 data 层从生成客户端的响应映射而来。
/// 仅保留展示所需字段，丢弃 id/typeId 等内部键。

/// 字典条目（展示项）。
///
/// 对应后端 `dict.entry` 的只读视图；`entryValue` 是后端编码值（如
/// `purchase_order`），`labels` 为各语言的展示标签（key 为 BCP-47 语言标签）。
class DictEntryInfo {
  final String entryValue;
  final Map<String, String> labels;

  const DictEntryInfo({required this.entryValue, required this.labels});
}

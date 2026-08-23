import 'dart:ui';

import 'package:go_wind_erp/src/features/dict/domain/dict_models.dart';

/// 字典标签解析器。
///
/// 移植自 admin 端 `getDictEntryLabelByValue`，关键差异：未匹配返回空串
/// （admin 返回原始值），杜绝编码值回退到 UI。语言标签格式按后端 i18n
/// map 的键约定归一化为 BCP-47 连字符（zh-CN / en-US）。
class DictLabelResolver {
  DictLabelResolver._();

  /// 将 [locale] 归一化为后端 i18n map 期望的 BCP-47 连字符标签。
  ///
  /// Flutter `Locale.toString()` 产出下划线形式（zh_CN）；后端字典 i18n
  /// map 的键为连字符形式（zh-CN）。此处统一转换为连字符。
  static String _normalizeLocale(Locale locale) {
    return locale.toString().replaceAll('_', '-');
  }

  /// 在 [entries] 中查找与 [value] 匹配的 [DictEntryInfo.entryValue]，
  /// 返回当前 [locale] 对应的展示标签。
  ///
  /// 查找失败（无匹配条目 / 无该语言标签）一律返回空串——不回退原始值。
  static String labelByValue(
    String? value,
    List<DictEntryInfo> entries,
    Locale locale,
  ) {
    if (value == null || value.isEmpty) return '';
    DictEntryInfo? matched;
    for (final e in entries) {
      if (e.entryValue == value) {
        matched = e;
        break;
      }
    }
    if (matched == null) return '';
    final key = _normalizeLocale(locale);
    return matched.labels[key] ?? '';
  }
}

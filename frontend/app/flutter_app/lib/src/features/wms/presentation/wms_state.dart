import 'package:equatable/equatable.dart';
import 'package:meta/meta.dart';

import 'package:go_wind_erp/src/features/wms/domain/wms_failure.dart';
import 'package:go_wind_erp/src/features/wms/domain/wms_models.dart';

/// WMS 扫码视图状态。
///
/// 仓库列表加载走 [WmsLoading]/[WmsReady]/[WmsLoadFailure]；之后的查询与
/// 提交是 [WmsReady] 内的瞬时动作，经 `lookingUp` / `submitting` / `message`
/// 字段表达，避免状态类爆炸。
@immutable
sealed class WmsState extends Equatable {
  const WmsState();
}

final class WmsInitial extends WmsState {
  const WmsInitial();

  @override
  List<Object?> get props => [];
}

final class WmsLoading extends WmsState {
  const WmsLoading();

  @override
  List<Object?> get props => [];
}

final class WmsLoadFailure extends WmsState {
  final WmsFailure failure;
  const WmsLoadFailure(this.failure);

  @override
  List<Object?> get props => [failure];
}

final class WmsReady extends WmsState {
  final List<WarehouseInfo> warehouses;

  /// 当前选中仓库编码；null 表示未选。
  final String? selectedWarehouseCode;

  /// 最近一次 SKU 查询结果；null 表示尚未查询或未命中。
  final InventoryInfo? inventory;

  /// 查询未命中（区别于“尚未查询”）。
  final bool lookupMiss;

  /// 当前查询的 SKU（用于回显与流水上下文）。
  final String currentSku;

  final List<StockMovementRecord> movements;

  final bool lookingUp;
  final bool submitting;

  /// 一次性反馈（提交成功/失败提示），展示后由 UI 清除。
  final String? message;

  const WmsReady({
    required this.warehouses,
    this.selectedWarehouseCode,
    this.inventory,
    this.lookupMiss = false,
    this.currentSku = '',
    this.movements = const [],
    this.lookingUp = false,
    this.submitting = false,
    this.message,
  });

  WmsReady copyWith({
    String? selectedWarehouseCode,
    InventoryInfo? inventory,
    bool? lookupMiss,
    String? currentSku,
    List<StockMovementRecord>? movements,
    bool? lookingUp,
    bool? submitting,
    String? message,
  }) {
    return WmsReady(
      warehouses: warehouses,
      selectedWarehouseCode: selectedWarehouseCode ?? this.selectedWarehouseCode,
      inventory: inventory ?? this.inventory,
      lookupMiss: lookupMiss ?? this.lookupMiss,
      currentSku: currentSku ?? this.currentSku,
      movements: movements ?? this.movements,
      lookingUp: lookingUp ?? this.lookingUp,
      submitting: submitting ?? this.submitting,
      message: message,
    );
  }

  @override
  List<Object?> get props => [
        warehouses,
        selectedWarehouseCode,
        inventory,
        lookupMiss,
        currentSku,
        movements,
        lookingUp,
        submitting,
        message,
      ];
}
